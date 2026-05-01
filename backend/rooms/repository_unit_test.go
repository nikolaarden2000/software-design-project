package rooms

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/nikolaarden2000/software-design-project/backend/db"
	pgxmock "github.com/pashagolub/pgxmock/v2"
)

func newRoomsMock(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unfulfilled mock expectations: %v", err)
		}
	})
	return mock
}

func newRoomsRepo(mock pgxmock.PgxPoolIface) *Repository {
	return NewRepository(mock)
}

func fixedRoomTime(hour int) time.Time {
	return time.Date(2026, 4, 27, hour, 0, 0, 0, time.UTC)
}

func validAdminRoomInput() AdminRoomInput {
	return AdminRoomInput{
		LocationID:    1,
		Title:         "Переговорная",
		Description:   "Описание помещения",
		Price:         1000,
		Capacity:      8,
		AvailableFrom: "09:00",
		AvailableTo:   "18:00",
		Images:        []string{"/images/room.jpg"},
	}
}

func ptr[T any](value T) *T {
	return &value
}

func expectApplyDueArchivedRooms(mock pgxmock.PgxPoolIface) {
	mock.ExpectExec(regexp.QuoteMeta(queryApplyDueArchivedRooms)).
		WithArgs(pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))
}

func expectLocationAccess(mock pgxmock.PgxPoolIface, locationID int, includeAll bool, adminID int, exists bool, accessible bool) {
	mock.ExpectQuery(`(?s).*SELECT.*FROM locations.*accessible.*`).
		WithArgs(locationID, includeAll, adminID).
		WillReturnRows(pgxmock.NewRows([]string{"exists", "accessible"}).AddRow(exists, accessible))
}

func expectRoomAccess(mock pgxmock.PgxPoolIface, roomID int, includeAll bool, adminID int, exists bool, accessible bool) {
	mock.ExpectQuery(`(?s).*SELECT.*FROM rooms.*accessible.*`).
		WithArgs(roomID, includeAll, adminID).
		WillReturnRows(pgxmock.NewRows([]string{"exists", "accessible"}).AddRow(exists, accessible))
}

func expectLastActiveOrFutureBookingEnd(mock pgxmock.PgxPoolIface, roomID int, now time.Time, endTime any) {
	mock.ExpectQuery(regexp.QuoteMeta(queryGetLastActiveOrFutureBookingEnd)).
		WithArgs(roomID, now.UTC()).
		WillReturnRows(pgxmock.NewRows([]string{"max"}).AddRow(endTime))
}

// Техника тест-дизайна: Классы эквивалентности.
// Проверяем валидные статусы (один класс) и невалидные статусы (неизвестный, пустой, неверный регистр).
func TestIsValidStatus(t *testing.T) {
	cases := []struct {
		name   string
		status string
		want   bool
	}{
		{"draft is valid", StatusDraft, true},
		{"pending is valid", StatusPending, true},
		{"published is valid", StatusPublished, true},
		{"rejected is valid", StatusRejected, true},
		{"archived is valid", StatusArchived, true},
		{"empty status is invalid", "", false},
		{"unknown status is invalid", "deleted", false},
		{"case-sensitive invalid status", "PUBLISHED", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsValidStatus(tc.status); got != tc.want {
				t.Errorf("IsValidStatus(%q): got %v, want %v", tc.status, got, tc.want)
			}
		})
	}
}

// Техника тест-дизайна: Классы эквивалентности и Граничные значения.
// Проверяем граничные значения (0, 1 для чисел; 5, 6 для массивов),
// границы временных интервалов (from < to, from == to, from > to),
// а также классы валидной нормализации (усечение пробелов) и невалидных форматов.
func TestNormalizeAdminRoomInput(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*AdminRoomInput)
		wantErr error
		check   func(*testing.T, AdminRoomInput)
	}{
		{
			name: "valid input trims text and images",
			mutate: func(input *AdminRoomInput) {
				input.Title = "  Переговорная  "
				input.Description = "  Описание  "
				input.AvailableFrom = " 09:00 "
				input.AvailableTo = " 18:00 "
				input.Images = []string{" /images/a.jpg ", " /images/b.jpg "}
			},
			wantErr: nil,
			check: func(t *testing.T, got AdminRoomInput) {
				if got.Title != "Переговорная" {
					t.Errorf("Title: got %q, want %q", got.Title, "Переговорная")
				}
				if got.AvailableFrom != "09:00" {
					t.Errorf("AvailableFrom: got %q, want 09:00", got.AvailableFrom)
				}
				if len(got.Images) == 2 && got.Images[0] != "/images/a.jpg" {
					t.Errorf("Images not trimmed")
				}
			},
		},
		{"location_id = 0 is invalid", func(i *AdminRoomInput) { i.LocationID = 0 }, db.ErrInvalidID, nil},
		{"price = 0 is invalid", func(i *AdminRoomInput) { i.Price = 0 }, db.ErrInvalidArgument, nil},
		{"price = 1 is valid", func(i *AdminRoomInput) { i.Price = 1 }, nil, nil},
		{"capacity = 0 is invalid", func(i *AdminRoomInput) { i.Capacity = 0 }, db.ErrInvalidArgument, nil},
		{"5 images is valid", func(i *AdminRoomInput) { i.Images = []string{"1", "2", "3", "4", "5"} }, nil, nil},
		{"6 images is invalid", func(i *AdminRoomInput) { i.Images = []string{"1", "2", "3", "4", "5", "6"} }, db.ErrInvalidArgument, nil},
		{"nil images becomes empty slice", func(i *AdminRoomInput) { i.Images = nil }, nil, func(t *testing.T, got AdminRoomInput) {
			if got.Images == nil {
				t.Fatal("expected nil Images to be normalized to empty slice")
			}
		}},
		{"09:00 equal 09:00 is invalid", func(i *AdminRoomInput) { i.AvailableFrom, i.AvailableTo = "09:00", "09:00" }, db.ErrInvalidArgument, nil},
		{"18:00 after 09:00 is invalid", func(i *AdminRoomInput) { i.AvailableFrom, i.AvailableTo = "18:00", "09:00" }, db.ErrInvalidArgument, nil},
		{"bad from format is invalid", func(i *AdminRoomInput) { i.AvailableFrom = "9 утра" }, db.ErrInvalidArgument, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := validAdminRoomInput()
			tc.mutate(&input)
			got, err := normalizeAdminRoomInput(input)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
			if err == nil && tc.check != nil {
				tc.check(t, got)
			}
		})
	}
}

// Техника тест-дизайна: Классы эквивалентности.
func TestStringPtrFromNull(t *testing.T) {
	cases := []struct {
		name  string
		value sql.NullString
		want  *string
	}{
		{"null value returns nil", sql.NullString{Valid: false}, nil},
		{"valid value returns pointer", sql.NullString{String: "reason", Valid: true}, ptr("reason")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := stringPtrFromNull(tc.value)
			if tc.want == nil && got != nil {
				t.Fatalf("got %q, want nil", *got)
			}
			if tc.want != nil && (got == nil || *got != *tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// Техника тест-дизайна: Таблица решений и Граничные значения.
// Граничные значения: проверка locationID <= 0.
// Таблица решений: комбинации существования (exists) и доступности (accessible) в БД.
func TestCheckLocationAccess(t *testing.T) {
	cases := []struct {
		name       string
		locationID int
		exists     bool
		accessible bool
		wantErr    error
	}{
		{"location exists and accessible", 10, true, true, nil},
		{"location does not exist", 10, false, false, db.ErrNotFound},
		{"location exists but forbidden", 10, true, false, db.ErrForbidden},
		{"zero location id", 0, false, false, db.ErrInvalidID},
		{"negative location id", -1, false, false, db.ErrInvalidID},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newRoomsMock(t)
			repo := newRoomsRepo(mock)
			if tc.locationID > 0 {
				expectLocationAccess(mock, tc.locationID, false, 7, tc.exists, tc.accessible)
			}
			err := repo.checkLocationAccess(context.Background(), tc.locationID, 7, false)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// Техника тест-дизайна: Таблица решений и Граничные значения.
func TestCheckRoomAccess(t *testing.T) {
	cases := []struct {
		name       string
		roomID     int
		exists     bool
		accessible bool
		wantErr    error
	}{
		{"room exists and accessible", 20, true, true, nil},
		{"room does not exist", 20, false, false, db.ErrNotFound},
		{"room exists but forbidden", 20, true, false, db.ErrForbidden},
		{"zero room id", 0, false, false, db.ErrInvalidID},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newRoomsMock(t)
			repo := newRoomsRepo(mock)
			if tc.roomID > 0 {
				expectRoomAccess(mock, tc.roomID, false, 7, tc.exists, tc.accessible)
			}
			err := repo.checkRoomAccess(context.Background(), tc.roomID, 7, false)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// Техника тест-дизайна: Классы эквивалентности и Граничные значения.
// Границы: limit <= 0, lastID < 0. Класс эквивалентности: успешная выборка списка.
func TestGetRoomsBatchByCity(t *testing.T) {
	cases := []struct {
		name    string
		lastID  int
		limit   int
		setup   func(pgxmock.PgxPoolIface)
		wantErr error
		check   func(*testing.T, []Room)
	}{
		{"lastID = -1 is invalid", -1, 10, func(m pgxmock.PgxPoolIface) {}, db.ErrInvalidID, nil},
		{"limit = 0 is invalid", 0, 0, func(m pgxmock.PgxPoolIface) {}, db.ErrInvalidArgument, nil},
		{
			name:   "success returns mapped items",
			lastID: 0,
			limit:  2,
			setup: func(m pgxmock.PgxPoolIface) {
				expectApplyDueArchivedRooms(m)
				m.ExpectQuery(regexp.QuoteMeta(queryGetRoomsBatchByCity)).
					WithArgs(0, "Москва", 2).
					WillReturnRows(pgxmock.NewRows([]string{"id", "title", "company", "address", "capacity", "image", "price"}).
						AddRow(1, "Room A", "Company A", "Москва", 8, "/img.jpg", 1500))
			},
			wantErr: nil,
			check: func(t *testing.T, items []Room) {
				if len(items) != 1 || items[0].Title != "Room A" {
					t.Fatalf("unexpected items: %+v", items)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newRoomsMock(t)
			repo := newRoomsRepo(mock)
			tc.setup(mock)
			items, err := repo.GetRoomsBatchByCity(context.Background(), tc.lastID, tc.limit, "Москва")
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
			if tc.check != nil {
				tc.check(t, items)
			}
		})
	}
}

// Техника тест-дизайна: Классы эквивалентности.
// Проверка валидного/невалидного статусов фильтрации и успешного маппинга ответа БД.
func TestListAdminRooms(t *testing.T) {
	cases := []struct {
		name    string
		status  *string
		setup   func(pgxmock.PgxPoolIface)
		wantErr error
		check   func(*testing.T, []AdminRoomListItem)
	}{
		{"invalid status filter", ptr("deleted"), func(m pgxmock.PgxPoolIface) { expectApplyDueArchivedRooms(m) }, db.ErrInvalidArgument, nil},
		{
			name:   "success with valid mapping",
			status: nil,
			setup: func(m pgxmock.PgxPoolIface) {
				expectApplyDueArchivedRooms(m)
				createdAt := time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC)
				m.ExpectQuery(regexp.QuoteMeta(queryListAdminRooms)).
					WithArgs(false, 7, pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnRows(pgxmock.NewRows([]string{"id", "loc", "title", "price", "cap", "status", "rej", "created"}).
						AddRow(1, 10, "Room A", 1500, 8, StatusDraft, nil, createdAt).
						AddRow(2, 10, "Room B", 2000, 12, StatusRejected, "bad", createdAt))
			},
			wantErr: nil,
			check: func(t *testing.T, items []AdminRoomListItem) {
				if len(items) != 2 || items[1].RejectionReason == nil || *items[1].RejectionReason != "bad" {
					t.Fatalf("unexpected items mapping: %+v", items)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newRoomsMock(t)
			repo := newRoomsRepo(mock)
			tc.setup(mock)
			items, err := repo.ListAdminRooms(context.Background(), 7, false, nil, tc.status)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
			if tc.check != nil {
				tc.check(t, items)
			}
		})
	}
}

// Техника тест-дизайна: Классы эквивалентности и Граничные значения.
// Границы: roomID <= 0. Классы: пустая причина, причина из пробелов, валидное отклонение.
func TestRejectRoom(t *testing.T) {
	cases := []struct {
		name    string
		roomID  int
		reason  string
		setup   func(pgxmock.PgxPoolIface)
		wantErr error
	}{
		{"zero room id", 0, "reason", nil, db.ErrInvalidID},
		{"empty reason", 1, "", nil, db.ErrInvalidArgument},
		{"whitespace reason", 1, "   ", nil, db.ErrInvalidArgument},
		{
			name:   "success normalizes reason",
			roomID: 5,
			reason: "  bad photos  ",
			setup: func(m pgxmock.PgxPoolIface) {
				m.ExpectExec(regexp.QuoteMeta(queryRejectRoom)).WithArgs(5, "bad photos").WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			},
			wantErr: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newRoomsMock(t)
			repo := newRoomsRepo(mock)
			if tc.setup != nil {
				tc.setup(mock)
			}
			err := repo.RejectRoom(context.Background(), tc.roomID, tc.reason)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// Техника тест-дизайна: Таблица решений и Граничные значения.
// Границы: roomID <= 0.
// Таблица решений: комбинации режимов архивирования и наличия будущих броней.
func TestArchiveAdminRoom(t *testing.T) {
	now := time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC)
	lastBookingEnd := now.Add(3 * time.Hour)

	cases := []struct {
		name              string
		roomID            int
		mode              string
		lastBookingEnd    any
		expectImmediate   bool
		expectScheduled   bool
		wantErr           error
		wantStatus        string
		wantBookingClosed bool
	}{
		{"zero room id", 0, ArchiveModeImmediate, nil, false, false, db.ErrInvalidID, "", false},
		{"unknown mode", 44, "delete", nil, false, false, db.ErrInvalidArgument, "", false},
		{"immediate without future bookings", 44, ArchiveModeImmediate, nil, true, false, nil, StatusArchived, false},
		{"immediate with future bookings returns conflict", 44, ArchiveModeImmediate, lastBookingEnd, false, false, db.ErrConflict, "", false},
		{"scheduled with future bookings disables booking", 44, ArchiveModeScheduled, lastBookingEnd, false, true, nil, StatusPublished, true},
		{"empty mode defaults to immediate", 44, "", nil, true, false, nil, StatusArchived, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newRoomsMock(t)
			repo := newRoomsRepo(mock)

			if tc.roomID > 0 && tc.wantErr != db.ErrInvalidArgument {
				expectRoomAccess(mock, tc.roomID, false, 7, true, true)
				expectLastActiveOrFutureBookingEnd(mock, tc.roomID, now, tc.lastBookingEnd)
			}
			if tc.expectImmediate {
				mock.ExpectExec(regexp.QuoteMeta(queryArchiveAdminRoomImmediate)).WithArgs(tc.roomID).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			}
			if tc.expectScheduled {
				mock.ExpectExec(regexp.QuoteMeta(queryScheduleAdminRoomArchive)).WithArgs(tc.roomID, lastBookingEnd).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			}

			result, err := repo.ArchiveAdminRoom(context.Background(), 7, false, tc.roomID, tc.mode, now)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil {
				if result.Status != tc.wantStatus {
					t.Fatalf("status: got %q, want %q", result.Status, tc.wantStatus)
				}
				if result.BookingDisabled != tc.wantBookingClosed {
					t.Fatalf("booking disabled: got %v", result.BookingDisabled)
				}
			}
		})
	}
}

// Техника тест-дизайна: Классы эквивалентности.
// Проверяем успешное применение архивации БД и корректный проброс ошибки (Error handling).
func TestApplyDueArchivedRooms(t *testing.T) {
	now := fixedRoomTime(10)
	dbErr := errors.New("db timeout")

	cases := []struct {
		name    string
		setup   func(pgxmock.PgxPoolIface)
		wantErr error
	}{
		{"success updates rows", func(m pgxmock.PgxPoolIface) {
			m.ExpectExec(regexp.QuoteMeta(queryApplyDueArchivedRooms)).WithArgs(now.UTC()).WillReturnResult(pgxmock.NewResult("UPDATE", 2))
		}, nil},
		{"db error propagates", func(m pgxmock.PgxPoolIface) {
			m.ExpectExec(regexp.QuoteMeta(queryApplyDueArchivedRooms)).WithArgs(now.UTC()).WillReturnError(dbErr)
		}, dbErr},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newRoomsMock(t)
			repo := newRoomsRepo(mock)
			tc.setup(mock)
			err := repo.ApplyDueArchivedRooms(context.Background(), now)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// Техника тест-дизайна: Классы эквивалентности.
// Классы ответов базы данных: непустой результат, пустой результат (возврат пустого среза), ошибка.
func TestGetCompaniesByCity(t *testing.T) {
	dbErr := errors.New("timeout")
	cases := []struct {
		name    string
		setup   func(pgxmock.PgxPoolIface)
		wantErr error
		wantLen int
	}{
		{
			name: "success returns list",
			setup: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(regexp.QuoteMeta(queryGetCompaniesByCity)).WithArgs("Москва").
					WillReturnRows(pgxmock.NewRows([]string{"name"}).AddRow("Company A").AddRow("Company B"))
			},
			wantErr: nil,
			wantLen: 2,
		},
		{
			name: "empty result returns non-nil empty slice",
			setup: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(regexp.QuoteMeta(queryGetCompaniesByCity)).WithArgs("Москва").WillReturnRows(pgxmock.NewRows([]string{"name"}))
			},
			wantErr: nil,
			wantLen: 0,
		},
		{
			name: "db error propagates",
			setup: func(m pgxmock.PgxPoolIface) {
				m.ExpectQuery(regexp.QuoteMeta(queryGetCompaniesByCity)).WithArgs("Москва").WillReturnError(dbErr)
			},
			wantErr: dbErr,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newRoomsMock(t)
			repo := newRoomsRepo(mock)
			tc.setup(mock)
			companies, err := repo.GetCompaniesByCity(context.Background(), "Москва")
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
			if err == nil && len(companies) != tc.wantLen {
				t.Fatalf("len: got %d, want %d", len(companies), tc.wantLen)
			}
		})
	}
}

// Техника тест-дизайна: Классы эквивалентности и Граничные значения.
// Границы: roomID <= 0. Классы: найдено, не найдено.
func TestGetRoomPageData(t *testing.T) {
	cases := []struct {
		name    string
		roomID  int
		setup   func(pgxmock.PgxPoolIface)
		wantErr error
	}{
		{"invalid room id", 0, func(m pgxmock.PgxPoolIface) {}, db.ErrInvalidID},
		{"not found returns domain error", 99, func(m pgxmock.PgxPoolIface) {
			expectApplyDueArchivedRooms(m)
			m.ExpectQuery(regexp.QuoteMeta(queryGetRoomPageData)).WithArgs(99).
				WillReturnRows(pgxmock.NewRows([]string{"id", "title", "company", "address", "images", "price", "capacity", "available_from", "available_to", "description", "latitude", "longitude"}))
		}, db.ErrNotFound},
		{"success populates fields", 15, func(m pgxmock.PgxPoolIface) {
			expectApplyDueArchivedRooms(m)
			m.ExpectQuery(regexp.QuoteMeta(queryGetRoomPageData)).WithArgs(15).
				WillReturnRows(pgxmock.NewRows([]string{"id", "title", "company", "address", "images", "price", "capacity", "available_from", "available_to", "description", "latitude", "longitude"}).
					AddRow(15, "Room", "C", "A", []string{}, 1, 1, time.Now(), time.Now(), "D", 1.0, 1.0))
		}, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newRoomsMock(t)
			repo := newRoomsRepo(mock)
			tc.setup(mock)
			_, err := repo.GetRoomPageData(context.Background(), tc.roomID)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// Техника тест-дизайна: Таблица решений и Граничные значения.
// Границы: roomID <= 0. Таблица решений: влияние статуса и будущих броней на флаг CanArchiveNow.
func TestGetAdminRoom(t *testing.T) {
	cases := []struct {
		name                    string
		roomID                  int
		status                  string
		hasActiveFutureBookings bool
		wantCanArchiveNow       bool
		wantErr                 error
	}{
		{"invalid room id", 0, "", false, false, db.ErrInvalidID},
		{"published without bookings can be archived", 15, StatusPublished, false, true, nil},
		{"published with bookings cannot be archived", 15, StatusPublished, true, false, nil},
		{"archived cannot be archived now", 15, StatusArchived, false, false, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newRoomsMock(t)
			repo := newRoomsRepo(mock)

			if tc.roomID > 0 {
				expectApplyDueArchivedRooms(mock)
				expectRoomAccess(mock, tc.roomID, false, 7, true, true)
				mock.ExpectQuery(regexp.QuoteMeta(queryGetAdminRoom)).WithArgs(tc.roomID).
					WillReturnRows(pgxmock.NewRows([]string{"id", "loc", "title", "desc", "price", "cap", "from", "to", "img", "status", "rej", "disabled", "sched", "has_bookings"}).
						AddRow(tc.roomID, 3, "Room", "D", 1, 1, time.Now(), time.Now(), []string{}, tc.status, nil, false, nil, tc.hasActiveFutureBookings))
			}

			room, err := repo.GetAdminRoom(context.Background(), 7, false, tc.roomID)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
			if err == nil && room.Archive.CanArchiveNow != tc.wantCanArchiveNow {
				t.Fatalf("CanArchiveNow: got %v, want %v", room.Archive.CanArchiveNow, tc.wantCanArchiveNow)
			}
		})
	}
}

// Техника тест-дизайна: Классы эквивалентности и Граничные значения.
// Границы: creatorID <= 0.
func TestCreateAdminRoom(t *testing.T) {
	cases := []struct {
		name      string
		creatorID int
		setup     func(pgxmock.PgxPoolIface)
		wantErr   error
	}{
		{"invalid creator id", 0, func(m pgxmock.PgxPoolIface) {}, db.ErrInvalidID},
		{"success creates room", 7, func(m pgxmock.PgxPoolIface) {
			expectLocationAccess(m, 1, false, 7, true, true)
			m.ExpectQuery(regexp.QuoteMeta(queryCreateAdminRoom)).WithArgs(1, "Переговорная", "Описание помещения", 1000, 8, "09:00", "18:00", []string{"/images/room.jpg"}, 7).
				WillReturnRows(pgxmock.NewRows([]string{"id", "loc", "title", "price", "cap", "status", "rej", "created"}).
					AddRow(101, 1, "Переговорная", 1000, 8, StatusDraft, nil, time.Now()))
		}, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newRoomsMock(t)
			repo := newRoomsRepo(mock)
			tc.setup(mock)
			_, err := repo.CreateAdminRoom(context.Background(), tc.creatorID, false, validAdminRoomInput())
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// Техника тест-дизайна: Классы эквивалентности и Граничные значения.
// Границы: roomID <= 0.
// Классы (результаты UPDATE): обновлена 1 строка (успех) и обновлено 0 строк (ErrConflict).
func TestUpdateAdminRoom(t *testing.T) {
	cases := []struct {
		name         string
		roomID       int
		rowsAffected int64
		wantErr      error
	}{
		{"invalid room id", 0, 0, db.ErrInvalidID},
		{"success on 1 row updated", 44, 1, nil},
		{"conflict on 0 rows updated", 44, 0, db.ErrConflict},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newRoomsMock(t)
			repo := newRoomsRepo(mock)
			input := validAdminRoomInput()

			if tc.roomID > 0 {
				expectRoomAccess(mock, tc.roomID, false, 7, true, true)
				mock.ExpectExec(regexp.QuoteMeta(queryUpdateAdminRoom)).
					WithArgs(tc.roomID, input.Title, input.Description, input.Price, input.Capacity, input.AvailableFrom, input.AvailableTo, input.Images).
					WillReturnResult(pgxmock.NewResult("UPDATE", tc.rowsAffected))
			}

			err := repo.UpdateAdminRoom(context.Background(), 7, false, tc.roomID, input)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// Техника тест-дизайна: Классы эквивалентности и Граничные значения.
// Аналогично UpdateAdminRoom — классы затронутых строк.
func TestSubmitAdminRoom(t *testing.T) {
	cases := []struct {
		name         string
		roomID       int
		rowsAffected int64
		wantErr      error
	}{
		{"invalid room id", 0, 0, db.ErrInvalidID},
		{"success on 1 row updated", 44, 1, nil},
		{"conflict on 0 rows updated", 44, 0, db.ErrConflict},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newRoomsMock(t)
			repo := newRoomsRepo(mock)
			if tc.roomID > 0 {
				expectRoomAccess(mock, tc.roomID, false, 7, true, true)
				mock.ExpectExec(regexp.QuoteMeta(querySubmitAdminRoom)).WithArgs(tc.roomID).WillReturnResult(pgxmock.NewResult("UPDATE", tc.rowsAffected))
			}
			err := repo.SubmitAdminRoom(context.Background(), 7, false, tc.roomID)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// Техника тест-дизайна: Классы эквивалентности.
func TestListModerationRooms(t *testing.T) {
	cases := []struct {
		name    string
		setup   func(pgxmock.PgxPoolIface)
		wantErr error
	}{
		{"success handles null creators", func(m pgxmock.PgxPoolIface) {
			m.ExpectQuery(regexp.QuoteMeta(queryListModerationRooms)).
				WillReturnRows(pgxmock.NewRows([]string{"id", "loc", "comp", "city", "addr", "title", "desc", "price", "cap", "from", "to", "img", "status", "cr_id", "cr_name", "cr_email"}).
					AddRow(1, 10, "A", "MOW", "Addr", "Title", "Desc", 1, 1, time.Now(), time.Now(), []string{}, StatusPending, nil, nil, nil))
		}, nil},
		{"db error propagates", func(m pgxmock.PgxPoolIface) {
			m.ExpectQuery(regexp.QuoteMeta(queryListModerationRooms)).WillReturnError(errors.New("db error"))
		}, errors.New("db error")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newRoomsMock(t)
			repo := newRoomsRepo(mock)
			tc.setup(mock)
			_, err := repo.ListModerationRooms(context.Background())
			// Use string compare for exact mock error check or Is if wrapped properly
			if err != nil && tc.wantErr != nil && err.Error() != tc.wantErr.Error() {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// Техника тест-дизайна: Классы эквивалентности и Граничные значения.
// Границы: roomID <= 0.
// Классы: успешное обновление (1 строка), конфликт (0 строк, комната существует), не найдено (0 строк, комнаты нет).
func TestModerationUpdateStatusMethods(t *testing.T) {
	cases := []struct {
		name         string
		call         func(*Repository) error
		query        string
		roomID       int
		rowsAffected int64
		existsOnMiss *bool
		wantErr      error
	}{
		{"approve invalid id", func(r *Repository) error { return r.ApproveRoom(context.Background(), 0) }, "", 0, 0, nil, db.ErrInvalidID},
		{"archive invalid id", func(r *Repository) error { return r.ArchiveRoom(context.Background(), -1) }, "", -1, 0, nil, db.ErrInvalidID},
		{"approve updates one row", func(r *Repository) error { return r.ApproveRoom(context.Background(), 5) }, queryApproveRoom, 5, 1, nil, nil},
		{"archive updates one row", func(r *Repository) error { return r.ArchiveRoom(context.Background(), 5) }, queryArchiveRoom, 5, 1, nil, nil},
		{"approve zero rows conflict", func(r *Repository) error { return r.ApproveRoom(context.Background(), 5) }, queryApproveRoom, 5, 0, ptr(true), db.ErrConflict},
		{"archive zero rows not found", func(r *Repository) error { return r.ArchiveRoom(context.Background(), 5) }, queryArchiveRoom, 5, 0, ptr(false), db.ErrNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newRoomsMock(t)
			repo := newRoomsRepo(mock)

			if tc.roomID > 0 {
				mock.ExpectExec(regexp.QuoteMeta(tc.query)).WithArgs(tc.roomID).WillReturnResult(pgxmock.NewResult("UPDATE", tc.rowsAffected))
				if tc.existsOnMiss != nil {
					mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS`)).WithArgs(tc.roomID).WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(*tc.existsOnMiss))
				}
			}

			err := tc.call(repo)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
		})
	}
}
