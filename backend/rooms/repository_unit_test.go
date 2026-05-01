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

func expectApplyDueArchivedRooms(mock pgxmock.PgxPoolIface) {
	mock.ExpectExec(regexp.QuoteMeta(queryApplyDueArchivedRooms)).
		WithArgs(pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))
}

func expectLocationAccess(
	mock pgxmock.PgxPoolIface,
	locationID int,
	includeAll bool,
	adminID int,
	exists bool,
	accessible bool,
) {
	mock.ExpectQuery(`(?s).*SELECT.*FROM locations.*accessible.*`).
		WithArgs(locationID, includeAll, adminID).
		WillReturnRows(
			pgxmock.NewRows([]string{"exists", "accessible"}).
				AddRow(exists, accessible),
		)
}

func expectRoomAccess(
	mock pgxmock.PgxPoolIface,
	roomID int,
	includeAll bool,
	adminID int,
	exists bool,
	accessible bool,
) {
	mock.ExpectQuery(`(?s).*SELECT.*FROM rooms.*accessible.*`).
		WithArgs(roomID, includeAll, adminID).
		WillReturnRows(
			pgxmock.NewRows([]string{"exists", "accessible"}).
				AddRow(exists, accessible),
		)
}

func expectLastActiveOrFutureBookingEnd(
	mock pgxmock.PgxPoolIface,
	roomID int,
	now time.Time,
	endTime any,
) {
	mock.ExpectQuery(regexp.QuoteMeta(queryGetLastActiveOrFutureBookingEnd)).
		WithArgs(roomID, now.UTC()).
		WillReturnRows(
			pgxmock.NewRows([]string{"max"}).
				AddRow(endTime),
		)
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем два класса входных значений: допустимые статусы и недопустимые статусы.
func TestIsValidStatus_EquivalenceClasses(t *testing.T) {
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
			got := IsValidStatus(tc.status)
			if got != tc.want {
				t.Errorf("IsValidStatus(%q): got %v, want %v", tc.status, got, tc.want)
			}
		})
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем валидный объект формы и нормализацию текстовых полей.
func TestNormalizeAdminRoomInput_ValidInput_TrimsTextAndImages(t *testing.T) {
	input := validAdminRoomInput()
	input.Title = "  Переговорная  "
	input.Description = "  Описание  "
	input.AvailableFrom = " 09:00 "
	input.AvailableTo = " 18:00 "
	input.Images = []string{" /images/a.jpg ", " /images/b.jpg "}

	got, err := normalizeAdminRoomInput(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Title != "Переговорная" {
		t.Errorf("Title: got %q, want %q", got.Title, "Переговорная")
	}
	if got.Description != "Описание" {
		t.Errorf("Description: got %q, want %q", got.Description, "Описание")
	}
	if got.AvailableFrom != "09:00" {
		t.Errorf("AvailableFrom: got %q, want 09:00", got.AvailableFrom)
	}
	if got.AvailableTo != "18:00" {
		t.Errorf("AvailableTo: got %q, want 18:00", got.AvailableTo)
	}
	if got.Images[0] != "/images/a.jpg" || got.Images[1] != "/images/b.jpg" {
		t.Errorf("Images were not trimmed: %#v", got.Images)
	}
}

// Техника тест-дизайна: граничные значения.
// Проверяем значения около нижних границ: id, price, capacity, количество изображений.
func TestNormalizeAdminRoomInput_BoundaryValues(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*AdminRoomInput)
		wantErr error
	}{
		{
			name: "location_id = 0 is invalid",
			mutate: func(input *AdminRoomInput) {
				input.LocationID = 0
			},
			wantErr: db.ErrInvalidID,
		},
		{
			name: "location_id = 1 is valid",
			mutate: func(input *AdminRoomInput) {
				input.LocationID = 1
			},
			wantErr: nil,
		},
		{
			name: "price = 0 is invalid",
			mutate: func(input *AdminRoomInput) {
				input.Price = 0
			},
			wantErr: db.ErrInvalidArgument,
		},
		{
			name: "price = 1 is valid",
			mutate: func(input *AdminRoomInput) {
				input.Price = 1
			},
			wantErr: nil,
		},
		{
			name: "capacity = 0 is invalid",
			mutate: func(input *AdminRoomInput) {
				input.Capacity = 0
			},
			wantErr: db.ErrInvalidArgument,
		},
		{
			name: "capacity = 1 is valid",
			mutate: func(input *AdminRoomInput) {
				input.Capacity = 1
			},
			wantErr: nil,
		},
		{
			name: "5 images is valid",
			mutate: func(input *AdminRoomInput) {
				input.Images = []string{"1", "2", "3", "4", "5"}
			},
			wantErr: nil,
		},
		{
			name: "6 images is invalid",
			mutate: func(input *AdminRoomInput) {
				input.Images = []string{"1", "2", "3", "4", "5", "6"}
			},
			wantErr: db.ErrInvalidArgument,
		},
		{
			name: "nil images becomes empty slice",
			mutate: func(input *AdminRoomInput) {
				input.Images = nil
			},
			wantErr: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := validAdminRoomInput()
			tc.mutate(&input)

			got, err := normalizeAdminRoomInput(input)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}

			if tc.name == "nil images becomes empty slice" && got.Images == nil {
				t.Fatal("expected nil Images to be normalized to empty slice")
			}
		})
	}
}

// Техника тест-дизайна: граничные значения.
// Проверяем границу доступного интервала времени: from < to, from == to, from > to.
func TestNormalizeAdminRoomInput_TimeBoundaries(t *testing.T) {
	cases := []struct {
		name          string
		availableFrom string
		availableTo   string
		wantErr       error
	}{
		{"09:00 before 18:00 is valid", "09:00", "18:00", nil},
		{"09:00 equal 09:00 is invalid", "09:00", "09:00", db.ErrInvalidArgument},
		{"18:00 after 09:00 is invalid", "18:00", "09:00", db.ErrInvalidArgument},
		{"bad from format is invalid", "9 утра", "18:00", db.ErrInvalidArgument},
		{"bad to format is invalid", "09:00", "вечер", db.ErrInvalidArgument},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := validAdminRoomInput()
			input.AvailableFrom = tc.availableFrom
			input.AvailableTo = tc.availableTo

			_, err := normalizeAdminRoomInput(input)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем два класса sql.NullString: значение отсутствует и значение присутствует.
func TestStringPtrFromNull_EquivalenceClasses(t *testing.T) {
	cases := []struct {
		name  string
		value sql.NullString
		want  *string
	}{
		{
			name:  "null value returns nil",
			value: sql.NullString{Valid: false},
			want:  nil,
		},
		{
			name:  "valid value returns pointer",
			value: sql.NullString{String: "reason", Valid: true},
			want:  ptr("reason"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := stringPtrFromNull(tc.value)

			if tc.want == nil {
				if got != nil {
					t.Fatalf("got %q, want nil", *got)
				}
				return
			}

			if got == nil || *got != *tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// Техника тест-дизайна: таблица решений.
// Результат зависит от двух условий: существует ли локация и доступна ли она администратору.
func TestCheckLocationAccess_DecisionTable(t *testing.T) {
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

// Техника тест-дизайна: таблица решений.
// Результат зависит от двух условий: существует ли помещение и доступно ли оно администратору.
func TestCheckRoomAccess_DecisionTable(t *testing.T) {
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
		{"negative room id", -1, false, false, db.ErrInvalidID},
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

// Техника тест-дизайна: граничные значения.
// Проверяем границы параметров пагинации публичного каталога.
func TestGetRoomsBatchByCity_InputBoundaries(t *testing.T) {
	cases := []struct {
		name    string
		lastID  int
		limit   int
		wantErr error
	}{
		{"lastID = -1 is invalid", -1, 10, db.ErrInvalidID},
		{"lastID = 0 is valid", 0, 10, nil},
		{"limit = 0 is invalid", 0, 0, db.ErrInvalidArgument},
		{"limit = 1 is valid", 0, 1, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newRoomsMock(t)
			repo := newRoomsRepo(mock)

			if tc.wantErr == nil {
				expectApplyDueArchivedRooms(mock)
				mock.ExpectQuery(regexp.QuoteMeta(queryGetRoomsBatchByCity)).
					WithArgs(tc.lastID, "Москва", tc.limit).
					WillReturnRows(
						pgxmock.NewRows([]string{"id", "title", "company", "address", "capacity", "image", "price"}),
					)
			}

			_, err := repo.GetRoomsBatchByCity(context.Background(), tc.lastID, tc.limit, "Москва")

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// Техника тест-дизайна: основной позитивный сценарий.
// Проверяем корректное сканирование списка опубликованных помещений.
func TestGetRoomsBatchByCity_Success(t *testing.T) {
	mock := newRoomsMock(t)
	repo := newRoomsRepo(mock)

	expectApplyDueArchivedRooms(mock)
	mock.ExpectQuery(regexp.QuoteMeta(queryGetRoomsBatchByCity)).
		WithArgs(0, "Москва", 2).
		WillReturnRows(
			pgxmock.NewRows([]string{"id", "title", "company", "address", "capacity", "image", "price"}).
				AddRow(1, "Room A", "Company A", "Москва, Тверская, 1", 8, "/images/a.jpg", 1500).
				AddRow(2, "Room B", "Company B", "Москва, Арбат, 2", 12, "/images/b.jpg", 2000),
		)

	items, err := repo.GetRoomsBatchByCity(context.Background(), 0, 2, "Москва")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}

	if items[0].ID != 1 || items[0].Title != "Room A" || items[0].Price != 1500 {
		t.Fatalf("unexpected first item: %+v", items[0])
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем два класса статусов фильтра: валидный и невалидный.
func TestListAdminRooms_StatusEquivalenceClasses(t *testing.T) {
	cases := []struct {
		name    string
		status  *string
		wantErr error
	}{
		{"valid status", ptr(StatusDraft), nil},
		{"invalid status", ptr("deleted"), db.ErrInvalidArgument},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newRoomsMock(t)
			repo := newRoomsRepo(mock)

			expectApplyDueArchivedRooms(mock)

			if tc.wantErr == nil {
				mock.ExpectQuery(regexp.QuoteMeta(queryListAdminRooms)).
					WithArgs(false, 7, pgxmock.AnyArg(), StatusDraft).
					WillReturnRows(
						pgxmock.NewRows([]string{
							"id",
							"location_id",
							"title",
							"price",
							"capacity",
							"status",
							"rejection_reason",
							"created_at",
						}),
					)
			}

			_, err := repo.ListAdminRooms(context.Background(), 7, false, nil, tc.status)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// Техника тест-дизайна: основной позитивный сценарий.
// Проверяем, что список комнат администратора сканируется в модель AdminRoomListItem.
func TestListAdminRooms_Success(t *testing.T) {
	mock := newRoomsMock(t)
	repo := newRoomsRepo(mock)
	createdAt := time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC)

	expectApplyDueArchivedRooms(mock)
	mock.ExpectQuery(regexp.QuoteMeta(queryListAdminRooms)).
		WithArgs(false, 7, pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(
			pgxmock.NewRows([]string{
				"id",
				"location_id",
				"title",
				"price",
				"capacity",
				"status",
				"rejection_reason",
				"created_at",
			}).
				AddRow(1, 10, "Room A", 1500, 8, StatusDraft, nil, createdAt).
				AddRow(2, 10, "Room B", 2000, 12, StatusRejected, "bad photos", createdAt),
		)

	items, err := repo.ListAdminRooms(context.Background(), 7, false, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}

	if items[0].RejectionReason != nil {
		t.Fatalf("first item rejection reason: got %v, want nil", items[0].RejectionReason)
	}

	if items[1].RejectionReason == nil || *items[1].RejectionReason != "bad photos" {
		t.Fatalf("second item rejection reason: got %v, want bad photos", items[1].RejectionReason)
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем валидную причину отклонения и пустую/пробельную причину.
func TestRejectRoom_ReasonEquivalenceClasses(t *testing.T) {
	cases := []struct {
		name    string
		roomID  int
		reason  string
		wantErr error
	}{
		{"empty reason is invalid", 1, "", db.ErrInvalidArgument},
		{"whitespace reason is invalid", 1, "   ", db.ErrInvalidArgument},
		{"zero room id is invalid", 0, "reason", db.ErrInvalidID},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newRoomsMock(t)
			repo := newRoomsRepo(mock)

			err := repo.RejectRoom(context.Background(), tc.roomID, tc.reason)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// Техника тест-дизайна: основной позитивный сценарий.
// Проверяем успешное отклонение помещения с нормализацией причины.
func TestRejectRoom_Success(t *testing.T) {
	mock := newRoomsMock(t)
	repo := newRoomsRepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(queryRejectRoom)).
		WithArgs(5, "bad photos").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := repo.RejectRoom(context.Background(), 5, "  bad photos  ")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Техника тест-дизайна: таблица решений.
// Результат архивирования зависит от режима archive mode и наличия будущих бронирований.
func TestArchiveAdminRoom_DecisionTable(t *testing.T) {
	now := time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC)
	lastBookingEnd := now.Add(3 * time.Hour)

	cases := []struct {
		name              string
		mode              string
		lastBookingEnd    any
		expectImmediate   bool
		expectScheduled   bool
		wantErr           error
		wantStatus        string
		wantBookingClosed bool
	}{
		{
			name:            "immediate without future bookings archives room",
			mode:            ArchiveModeImmediate,
			lastBookingEnd:  nil,
			expectImmediate: true,
			wantStatus:      StatusArchived,
		},
		{
			name:           "immediate with future bookings returns conflict",
			mode:           ArchiveModeImmediate,
			lastBookingEnd: lastBookingEnd,
			wantErr:        db.ErrConflict,
		},
		{
			name:              "scheduled with future bookings disables booking and schedules archive",
			mode:              ArchiveModeScheduled,
			lastBookingEnd:    lastBookingEnd,
			expectScheduled:   true,
			wantStatus:        StatusPublished,
			wantBookingClosed: true,
		},
		{
			name:            "scheduled without future bookings archives immediately",
			mode:            ArchiveModeScheduled,
			lastBookingEnd:  nil,
			expectImmediate: true,
			wantStatus:      StatusArchived,
		},
		{
			name:            "empty mode defaults to immediate",
			mode:            "",
			lastBookingEnd:  nil,
			expectImmediate: true,
			wantStatus:      StatusArchived,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newRoomsMock(t)
			repo := newRoomsRepo(mock)

			expectRoomAccess(mock, 44, false, 7, true, true)
			expectLastActiveOrFutureBookingEnd(mock, 44, now, tc.lastBookingEnd)

			if tc.expectImmediate {
				mock.ExpectExec(regexp.QuoteMeta(queryArchiveAdminRoomImmediate)).
					WithArgs(44).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			}

			if tc.expectScheduled {
				mock.ExpectExec(regexp.QuoteMeta(queryScheduleAdminRoomArchive)).
					WithArgs(44, lastBookingEnd).
					WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			}

			result, err := repo.ArchiveAdminRoom(context.Background(), 7, false, 44, tc.mode, now)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}

			if tc.wantErr != nil {
				if result != nil {
					t.Fatalf("got result %+v, want nil", result)
				}
				return
			}

			if result == nil {
				t.Fatal("got nil result")
			}

			if result.ID != 44 {
				t.Fatalf("result id: got %d, want 44", result.ID)
			}

			if result.Status != tc.wantStatus {
				t.Fatalf("result status: got %q, want %q", result.Status, tc.wantStatus)
			}

			if result.BookingDisabled != tc.wantBookingClosed {
				t.Fatalf("BookingDisabled: got %v, want %v", result.BookingDisabled, tc.wantBookingClosed)
			}

			if tc.expectScheduled && result.ArchiveScheduledFor == nil {
				t.Fatal("expected ArchiveScheduledFor to be set")
			}
		})
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем валидные и невалидные режимы архивирования.
func TestArchiveAdminRoom_InvalidInputs(t *testing.T) {
	cases := []struct {
		name    string
		roomID  int
		mode    string
		wantErr error
	}{
		{"zero room id", 0, ArchiveModeImmediate, db.ErrInvalidID},
		{"negative room id", -1, ArchiveModeImmediate, db.ErrInvalidID},
		{"unknown archive mode", 1, "delete", db.ErrInvalidArgument},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newRoomsMock(t)
			repo := newRoomsRepo(mock)

			_, err := repo.ArchiveAdminRoom(context.Background(), 7, false, tc.roomID, tc.mode, fixedRoomTime(10))

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// Техника тест-дизайна: основной позитивный сценарий.
// Проверяем выполнение фонового обновления просроченных отложенных архивирований.
func TestApplyDueArchivedRooms_Success(t *testing.T) {
	mock := newRoomsMock(t)
	repo := newRoomsRepo(mock)

	now := fixedRoomTime(10)
	mock.ExpectExec(regexp.QuoteMeta(queryApplyDueArchivedRooms)).
		WithArgs(now.UTC()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 2))

	err := repo.ApplyDueArchivedRooms(context.Background(), now)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Техника тест-дизайна: обработка исключений.
// Проверяем, что ошибка базы данных не теряется и возвращается вызывающему коду.
func TestApplyDueArchivedRooms_DBError_PropagatesError(t *testing.T) {
	mock := newRoomsMock(t)
	repo := newRoomsRepo(mock)

	dbErr := errors.New("connection lost")
	mock.ExpectExec(regexp.QuoteMeta(queryApplyDueArchivedRooms)).
		WithArgs(pgxmock.AnyArg()).
		WillReturnError(dbErr)

	err := repo.ApplyDueArchivedRooms(context.Background(), fixedRoomTime(10))

	if !errors.Is(err, dbErr) {
		t.Fatalf("got error %v, want %v", err, dbErr)
	}
}

func ptr[T any](value T) *T {
	return &value
}

// Техника тест-дизайна: основной позитивный сценарий.
// Проверяем успешное получение списка компаний по городу.
func TestGetCompaniesByCity_Success(t *testing.T) {
	mock := newRoomsMock(t)
	repo := newRoomsRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(queryGetCompaniesByCity)).
		WithArgs("Москва").
		WillReturnRows(
			pgxmock.NewRows([]string{"name"}).
				AddRow("Company A").
				AddRow("Company B"),
		)

	companies, err := repo.GetCompaniesByCity(context.Background(), "Москва")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(companies) != 2 {
		t.Fatalf("got %d companies, want 2", len(companies))
	}

	if companies[0] != "Company A" || companies[1] != "Company B" {
		t.Fatalf("unexpected companies: %#v", companies)
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем два класса результата: список пустой и список непустой.
func TestGetCompaniesByCity_EmptyResult_ReturnsEmptySlice(t *testing.T) {
	mock := newRoomsMock(t)
	repo := newRoomsRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(queryGetCompaniesByCity)).
		WithArgs("Казань").
		WillReturnRows(pgxmock.NewRows([]string{"name"}))

	companies, err := repo.GetCompaniesByCity(context.Background(), "Казань")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if companies == nil {
		t.Fatal("got nil, want non-nil empty slice")
	}

	if len(companies) != 0 {
		t.Fatalf("got %d companies, want 0", len(companies))
	}
}

// Техника тест-дизайна: обработка исключений.
// Проверяем, что ошибка базы данных возвращается вызывающему коду.
func TestGetCompaniesByCity_DBError_PropagatesError(t *testing.T) {
	mock := newRoomsMock(t)
	repo := newRoomsRepo(mock)

	dbErr := errors.New("db timeout")
	mock.ExpectQuery(regexp.QuoteMeta(queryGetCompaniesByCity)).
		WithArgs("Москва").
		WillReturnError(dbErr)

	_, err := repo.GetCompaniesByCity(context.Background(), "Москва")

	if !errors.Is(err, dbErr) {
		t.Fatalf("got error %v, want %v", err, dbErr)
	}
}

// Техника тест-дизайна: граничные значения.
// Проверяем недопустимые значения идентификатора помещения: 0 и отрицательное значение.
func TestGetRoomPageData_InvalidRoomID_ReturnsErrInvalidID(t *testing.T) {
	cases := []struct {
		name   string
		roomID int
	}{
		{"roomID = 0", 0},
		{"roomID = -1", -1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newRoomsMock(t)
			repo := newRoomsRepo(mock)

			_, err := repo.GetRoomPageData(context.Background(), tc.roomID)

			if !errors.Is(err, db.ErrInvalidID) {
				t.Fatalf("got error %v, want %v", err, db.ErrInvalidID)
			}
		})
	}
}

// Техника тест-дизайна: основной позитивный сценарий.
// Проверяем успешное получение публичной карточки помещения и форматирование времени.
func TestGetRoomPageData_Success(t *testing.T) {
	mock := newRoomsMock(t)
	repo := newRoomsRepo(mock)

	availableFrom := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
	availableTo := time.Date(0, 1, 1, 18, 0, 0, 0, time.UTC)

	expectApplyDueArchivedRooms(mock)
	mock.ExpectQuery(regexp.QuoteMeta(queryGetRoomPageData)).
		WithArgs(15).
		WillReturnRows(
			pgxmock.NewRows([]string{
				"id",
				"title",
				"company",
				"address",
				"images",
				"price",
				"capacity",
				"available_from",
				"available_to",
				"description",
				"latitude",
				"longitude",
			}).
				AddRow(
					15,
					"Room A",
					"Company A",
					"Москва, Тверская 10",
					[]string{"/images/a.jpg"},
					1500,
					8,
					availableFrom,
					availableTo,
					"Описание",
					55.75,
					37.61,
				),
		)

	room, err := repo.GetRoomPageData(context.Background(), 15)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if room.ID != 15 || room.Title != "Room A" || room.Company != "Company A" {
		t.Fatalf("unexpected room: %+v", room)
	}

	if room.AvailableFrom != "09:00" || room.AvailableTo != "18:00" {
		t.Fatalf("unexpected availability: %s - %s", room.AvailableFrom, room.AvailableTo)
	}

	if room.Currency != "RUB" {
		t.Fatalf("currency: got %q, want RUB", room.Currency)
	}

	if room.MaxCapacity != room.Capacity {
		t.Fatalf("MaxCapacity: got %d, want %d", room.MaxCapacity, room.Capacity)
	}
}

// Техника тест-дизайна: обработка исключений.
// Проверяем преобразование отсутствующей строки БД в доменную ошибку db.ErrNotFound.
func TestGetRoomPageData_NotFound_ReturnsErrNotFound(t *testing.T) {
	mock := newRoomsMock(t)
	repo := newRoomsRepo(mock)

	expectApplyDueArchivedRooms(mock)
	mock.ExpectQuery(regexp.QuoteMeta(queryGetRoomPageData)).
		WithArgs(99).
		WillReturnRows(
			pgxmock.NewRows([]string{
				"id",
				"title",
				"company",
				"address",
				"images",
				"price",
				"capacity",
				"available_from",
				"available_to",
				"description",
				"latitude",
				"longitude",
			}),
		)

	_, err := repo.GetRoomPageData(context.Background(), 99)

	if !errors.Is(err, db.ErrNotFound) {
		t.Fatalf("got error %v, want %v", err, db.ErrNotFound)
	}
}

// Техника тест-дизайна: основной позитивный сценарий.
// Проверяем успешное получение admin-версии карточки помещения, включая archive-блок.
func TestGetAdminRoom_SuccessWithArchiveInfo(t *testing.T) {
	mock := newRoomsMock(t)
	repo := newRoomsRepo(mock)

	availableFrom := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
	availableTo := time.Date(0, 1, 1, 18, 0, 0, 0, time.UTC)
	scheduledFor := time.Date(2026, 5, 10, 18, 0, 0, 0, time.UTC)

	expectApplyDueArchivedRooms(mock)
	expectRoomAccess(mock, 15, false, 7, true, true)

	mock.ExpectQuery(regexp.QuoteMeta(queryGetAdminRoom)).
		WithArgs(15).
		WillReturnRows(
			pgxmock.NewRows([]string{
				"id",
				"location_id",
				"title",
				"description",
				"price",
				"capacity",
				"available_from",
				"available_to",
				"images",
				"status",
				"rejection_reason",
				"booking_disabled",
				"archive_scheduled_for",
				"has_active_or_future_bookings",
			}).
				AddRow(
					15,
					3,
					"Room A",
					"Описание",
					1500,
					8,
					availableFrom,
					availableTo,
					[]string{"/images/a.jpg"},
					StatusPublished,
					nil,
					true,
					scheduledFor,
					true,
				),
		)

	room, err := repo.GetAdminRoom(context.Background(), 7, false, 15)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if room.ID != 15 || room.LocationID != 3 {
		t.Fatalf("unexpected room ids: %+v", room)
	}

	if room.AvailableFrom != "09:00" || room.AvailableTo != "18:00" {
		t.Fatalf("unexpected time: %s - %s", room.AvailableFrom, room.AvailableTo)
	}

	if !room.Archive.BookingDisabled {
		t.Fatal("expected BookingDisabled = true")
	}

	if !room.Archive.HasActiveOrFutureBookings {
		t.Fatal("expected HasActiveOrFutureBookings = true")
	}

	if room.Archive.CanArchiveNow {
		t.Fatal("CanArchiveNow must be false when active/future bookings exist")
	}

	if room.Archive.ScheduledFor == nil {
		t.Fatal("expected ScheduledFor to be set")
	}
}

// Техника тест-дизайна: таблица решений.
// Проверяем вычисление can_archive_now в зависимости от статуса комнаты и наличия будущих броней.
func TestGetAdminRoom_CanArchiveNow_DecisionTable(t *testing.T) {
	cases := []struct {
		name                    string
		status                  string
		hasActiveFutureBookings bool
		wantCanArchiveNow       bool
	}{
		{
			name:                    "published without bookings can be archived",
			status:                  StatusPublished,
			hasActiveFutureBookings: false,
			wantCanArchiveNow:       true,
		},
		{
			name:                    "published with bookings cannot be archived now",
			status:                  StatusPublished,
			hasActiveFutureBookings: true,
			wantCanArchiveNow:       false,
		},
		{
			name:                    "already archived cannot be archived now",
			status:                  StatusArchived,
			hasActiveFutureBookings: false,
			wantCanArchiveNow:       false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newRoomsMock(t)
			repo := newRoomsRepo(mock)

			availableFrom := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
			availableTo := time.Date(0, 1, 1, 18, 0, 0, 0, time.UTC)

			expectApplyDueArchivedRooms(mock)
			expectRoomAccess(mock, 15, false, 7, true, true)

			mock.ExpectQuery(regexp.QuoteMeta(queryGetAdminRoom)).
				WithArgs(15).
				WillReturnRows(
					pgxmock.NewRows([]string{
						"id",
						"location_id",
						"title",
						"description",
						"price",
						"capacity",
						"available_from",
						"available_to",
						"images",
						"status",
						"rejection_reason",
						"booking_disabled",
						"archive_scheduled_for",
						"has_active_or_future_bookings",
					}).
						AddRow(
							15,
							3,
							"Room A",
							"Описание",
							1500,
							8,
							availableFrom,
							availableTo,
							[]string{},
							tc.status,
							nil,
							false,
							nil,
							tc.hasActiveFutureBookings,
						),
				)

			room, err := repo.GetAdminRoom(context.Background(), 7, false, 15)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if room.Archive.CanArchiveNow != tc.wantCanArchiveNow {
				t.Fatalf("CanArchiveNow: got %v, want %v", room.Archive.CanArchiveNow, tc.wantCanArchiveNow)
			}
		})
	}
}

// Техника тест-дизайна: основной позитивный сценарий.
// Проверяем создание комнаты администратором и сканирование результата INSERT.
func TestCreateAdminRoom_Success(t *testing.T) {
	mock := newRoomsMock(t)
	repo := newRoomsRepo(mock)

	input := validAdminRoomInput()
	createdAt := time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC)

	expectLocationAccess(mock, input.LocationID, false, 7, true, true)
	mock.ExpectQuery(regexp.QuoteMeta(queryCreateAdminRoom)).
		WithArgs(
			input.LocationID,
			input.Title,
			input.Description,
			input.Price,
			input.Capacity,
			input.AvailableFrom,
			input.AvailableTo,
			input.Images,
			7,
		).
		WillReturnRows(
			pgxmock.NewRows([]string{
				"id",
				"location_id",
				"title",
				"price",
				"capacity",
				"status",
				"rejection_reason",
				"created_at",
			}).
				AddRow(101, input.LocationID, input.Title, input.Price, input.Capacity, StatusDraft, nil, createdAt),
		)

	item, err := repo.CreateAdminRoom(context.Background(), 7, false, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if item.ID != 101 || item.Status != StatusDraft {
		t.Fatalf("unexpected created room: %+v", item)
	}
}

// Техника тест-дизайна: граничные значения.
// Проверяем, что creatorID = 0 не допускается.
func TestCreateAdminRoom_InvalidCreatorID_ReturnsErrInvalidID(t *testing.T) {
	mock := newRoomsMock(t)
	repo := newRoomsRepo(mock)

	_, err := repo.CreateAdminRoom(context.Background(), 0, false, validAdminRoomInput())

	if !errors.Is(err, db.ErrInvalidID) {
		t.Fatalf("got error %v, want %v", err, db.ErrInvalidID)
	}
}

// Техника тест-дизайна: основной позитивный сценарий.
// Проверяем успешное редактирование комнаты администратором.
func TestUpdateAdminRoom_Success(t *testing.T) {
	mock := newRoomsMock(t)
	repo := newRoomsRepo(mock)

	input := validAdminRoomInput()

	expectRoomAccess(mock, 44, false, 7, true, true)
	mock.ExpectExec(regexp.QuoteMeta(queryUpdateAdminRoom)).
		WithArgs(
			44,
			input.Title,
			input.Description,
			input.Price,
			input.Capacity,
			input.AvailableFrom,
			input.AvailableTo,
			input.Images,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := repo.UpdateAdminRoom(context.Background(), 7, false, 44, input)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Техника тест-дизайна: таблица решений.
// Проверяем исходы UPDATE в зависимости от количества изменённых строк.
func TestUpdateAdminRoom_RowsAffectedDecision(t *testing.T) {
	cases := []struct {
		name         string
		rowsAffected int64
		wantErr      error
	}{
		{"one row updated means success", 1, nil},
		{"zero rows updated means conflict", 0, db.ErrConflict},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newRoomsMock(t)
			repo := newRoomsRepo(mock)
			input := validAdminRoomInput()

			expectRoomAccess(mock, 44, false, 7, true, true)
			mock.ExpectExec(regexp.QuoteMeta(queryUpdateAdminRoom)).
				WithArgs(
					44,
					input.Title,
					input.Description,
					input.Price,
					input.Capacity,
					input.AvailableFrom,
					input.AvailableTo,
					input.Images,
				).
				WillReturnResult(pgxmock.NewResult("UPDATE", tc.rowsAffected))

			err := repo.UpdateAdminRoom(context.Background(), 7, false, 44, input)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// Техника тест-дизайна: таблица решений.
// Проверяем исходы отправки помещения на модерацию в зависимости от количества изменённых строк.
func TestSubmitAdminRoom_RowsAffectedDecision(t *testing.T) {
	cases := []struct {
		name         string
		rowsAffected int64
		wantErr      error
	}{
		{"one row updated means success", 1, nil},
		{"zero rows updated means conflict", 0, db.ErrConflict},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newRoomsMock(t)
			repo := newRoomsRepo(mock)

			expectRoomAccess(mock, 44, false, 7, true, true)
			mock.ExpectExec(regexp.QuoteMeta(querySubmitAdminRoom)).
				WithArgs(44).
				WillReturnResult(pgxmock.NewResult("UPDATE", tc.rowsAffected))

			err := repo.SubmitAdminRoom(context.Background(), 7, false, 44)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// Техника тест-дизайна: основной позитивный сценарий.
// Проверяем получение списка помещений, ожидающих модерации.
func TestListModerationRooms_Success(t *testing.T) {
	mock := newRoomsMock(t)
	repo := newRoomsRepo(mock)

	availableFrom := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
	availableTo := time.Date(0, 1, 1, 18, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta(queryListModerationRooms)).
		WillReturnRows(
			pgxmock.NewRows([]string{
				"id",
				"location_id",
				"company_name",
				"city",
				"address",
				"title",
				"description",
				"price",
				"capacity",
				"available_from",
				"available_to",
				"images",
				"status",
				"creator_id",
				"creator_name",
				"creator_email",
			}).
				AddRow(
					1,
					10,
					"Company A",
					"Москва",
					"Москва, Тверская 10",
					"Room A",
					"Описание",
					1500,
					8,
					availableFrom,
					availableTo,
					[]string{"/images/a.jpg"},
					StatusPending,
					7,
					"admin",
					"admin@mail.com",
				).
				AddRow(
					2,
					10,
					"Company A",
					"Москва",
					"Москва, Тверская 10",
					"Room B",
					"Описание",
					2000,
					12,
					availableFrom,
					availableTo,
					[]string{},
					StatusPending,
					nil,
					nil,
					nil,
				),
		)

	items, err := repo.ListModerationRooms(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}

	if items[0].CreatedBy == nil {
		t.Fatal("expected first room creator to be set")
	}

	if items[0].CreatedBy.Email != "admin@mail.com" {
		t.Fatalf("creator email: got %q, want admin@mail.com", items[0].CreatedBy.Email)
	}

	if items[1].CreatedBy != nil {
		t.Fatalf("second room creator: got %+v, want nil", items[1].CreatedBy)
	}

	if items[0].AvailableFrom != "09:00" || items[0].AvailableTo != "18:00" {
		t.Fatalf("unexpected formatted time: %s - %s", items[0].AvailableFrom, items[0].AvailableTo)
	}
}

// Техника тест-дизайна: обработка исключений.
// Проверяем, что ошибка запроса списка модерации возвращается вызывающему коду.
func TestListModerationRooms_DBError_PropagatesError(t *testing.T) {
	mock := newRoomsMock(t)
	repo := newRoomsRepo(mock)

	dbErr := errors.New("db is down")
	mock.ExpectQuery(regexp.QuoteMeta(queryListModerationRooms)).
		WillReturnError(dbErr)

	_, err := repo.ListModerationRooms(context.Background())

	if !errors.Is(err, dbErr) {
		t.Fatalf("got error %v, want %v", err, dbErr)
	}
}

// Техника тест-дизайна: таблица решений.
// Проверяем updateRoomModerationStatus через публичные методы ApproveRoom и ArchiveRoom.
func TestApproveAndArchiveRoom_UpdateStatusDecision(t *testing.T) {
	cases := []struct {
		name         string
		call         func(*Repository) error
		query        string
		rowsAffected int64
		existsOnMiss *bool
		wantErr      error
	}{
		{
			name:         "approve updates one row",
			query:        queryApproveRoom,
			rowsAffected: 1,
			call: func(repo *Repository) error {
				return repo.ApproveRoom(context.Background(), 5)
			},
			wantErr: nil,
		},
		{
			name:         "archive updates one row",
			query:        queryArchiveRoom,
			rowsAffected: 1,
			call: func(repo *Repository) error {
				return repo.ArchiveRoom(context.Background(), 5)
			},
			wantErr: nil,
		},
		{
			name:         "approve zero rows and room exists means conflict",
			query:        queryApproveRoom,
			rowsAffected: 0,
			existsOnMiss: ptr(true),
			call: func(repo *Repository) error {
				return repo.ApproveRoom(context.Background(), 5)
			},
			wantErr: db.ErrConflict,
		},
		{
			name:         "archive zero rows and room missing means not found",
			query:        queryArchiveRoom,
			rowsAffected: 0,
			existsOnMiss: ptr(false),
			call: func(repo *Repository) error {
				return repo.ArchiveRoom(context.Background(), 5)
			},
			wantErr: db.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newRoomsMock(t)
			repo := newRoomsRepo(mock)

			mock.ExpectExec(regexp.QuoteMeta(tc.query)).
				WithArgs(5).
				WillReturnResult(pgxmock.NewResult("UPDATE", tc.rowsAffected))

			if tc.existsOnMiss != nil {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS (SELECT 1 FROM rooms WHERE id = $1)`)).
					WithArgs(5).
					WillReturnRows(
						pgxmock.NewRows([]string{"exists"}).
							AddRow(*tc.existsOnMiss),
					)
			}

			err := tc.call(repo)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// Техника тест-дизайна: граничные значения.
// Проверяем недопустимые значения roomID для методов модерации.
func TestModerationMethods_InvalidRoomID_ReturnErrInvalidID(t *testing.T) {
	cases := []struct {
		name string
		call func(*Repository) error
	}{
		{
			name: "approve roomID = 0",
			call: func(repo *Repository) error {
				return repo.ApproveRoom(context.Background(), 0)
			},
		},
		{
			name: "archive roomID = -1",
			call: func(repo *Repository) error {
				return repo.ArchiveRoom(context.Background(), -1)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newRoomsMock(t)
			repo := newRoomsRepo(mock)

			err := tc.call(repo)

			if !errors.Is(err, db.ErrInvalidID) {
				t.Fatalf("got error %v, want %v", err, db.ErrInvalidID)
			}
		})
	}
}
