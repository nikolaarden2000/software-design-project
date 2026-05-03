package bookings

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	pgxmock "github.com/pashagolub/pgxmock/v2"
	"gitlab.com/5130904-20104-teams/software-design-project/backend/db"
)

func newMock(t *testing.T) pgxmock.PgxPoolIface {
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

func newBookingRepo(mock pgxmock.PgxPoolIface) *Repository {
	return NewRepository(mock)
}

func fixed(hour int) time.Time {
	return time.Date(2024, 1, 15, hour, 0, 0, 0, time.UTC)
}

var roomWindowCols = []string{"available_from", "available_to", "timezone"}

func expectRoomWindow(mock pgxmock.PgxPoolIface, roomID int, from, to time.Time, tz string) {
	mock.ExpectQuery(regexp.QuoteMeta(queryGetRoomWindow)).
		WithArgs(roomID).
		WillReturnRows(pgxmock.NewRows(roomWindowCols).AddRow(from, to, tz))
}

func expectBookingsQuery(mock pgxmock.PgxPoolIface, roomID int, endDate, now time.Time, rows *pgxmock.Rows) {
	mock.ExpectQuery(regexp.QuoteMeta(queryGetBookingsInRange)).
		WithArgs(roomID, endDate, now).
		WillReturnRows(rows)
}

func noBookingRows() *pgxmock.Rows {
	return pgxmock.NewRows([]string{"start_time", "end_time"})
}

var bookingCols = []string{"b_id", "r_id", "image_url", "title", "start_time", "end_time", "total_price", "status", "timezone"}

var adminBookingCols = []string{
	"id",
	"room_id",
	"room_title",
	"location_id",
	"location_address",
	"user_id",
	"user_email",
	"user_username",
	"start_time",
	"end_time",
	"total_price",
	"status",
	"timezone",
}

func intPtr(value int) *int {
	return &value
}

func TestResolveStatus_StateTransitions(t *testing.T) {
	start := fixed(10)
	end := fixed(12)

	cases := []struct {
		name       string
		baseStatus string
		now        time.Time
		want       string
	}{
		{"canceled is absorbing state", "canceled", fixed(9), "canceled"},
		{"before start is booked", "booked", fixed(9), "booked"},
		{"after end is finished", "booked", fixed(13), "finished"},
		{"between start and end is in_use", "booked", fixed(11), "in_use"},
		{"now equals start is in_use", "booked", start, "in_use"},
		{"now equals end is in_use", "booked", end, "in_use"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveStatus(tc.baseStatus, start, end, tc.now)

			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGetRoomAvailability_BoundaryValues_InvalidRoomIDReturnsErrInvalidID(t *testing.T) {
	cases := []struct {
		name string
		id   int
	}{
		{"zero id", 0},
		{"negative id", -1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newMock(t)

			_, err := newBookingRepo(mock).GetRoomAvailability(context.Background(), tc.id, 1, fixed(8))

			if !errors.Is(err, db.ErrInvalidID) {
				t.Errorf("expected db.ErrInvalidID, got: %v", err)
			}
		})
	}
}

func TestGetRoomAvailability_EquivalenceClasses_RoomNotFoundReturnsErrNotFound(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta(queryGetRoomWindow)).
		WithArgs(99).
		WillReturnRows(pgxmock.NewRows(roomWindowCols))

	_, err := newBookingRepo(mock).GetRoomAvailability(context.Background(), 99, 1, fixed(8))

	if !errors.Is(err, db.ErrNotFound) {
		t.Fatalf("expected db.ErrNotFound, got: %v", err)
	}
}

func TestGetRoomAvailability_ErrorGuessing_InvalidTimezoneReturnsError(t *testing.T) {
	mock := newMock(t)
	availFrom := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
	availTo := time.Date(0, 1, 1, 18, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta(queryGetRoomWindow)).
		WithArgs(1).
		WillReturnRows(pgxmock.NewRows(roomWindowCols).AddRow(availFrom, availTo, "Not/ATimezone"))

	_, err := newBookingRepo(mock).GetRoomAvailability(context.Background(), 1, 1, fixed(8))

	if err == nil {
		t.Fatal("expected error for invalid timezone, got nil")
	}
}

func TestGetRoomAvailability_ErrorGuessing_RoomQueryErrorPropagates(t *testing.T) {
	mock := newMock(t)
	dbErr := fmt.Errorf("timeout")

	mock.ExpectQuery(regexp.QuoteMeta(queryGetRoomWindow)).
		WithArgs(1).
		WillReturnError(dbErr)

	_, err := newBookingRepo(mock).GetRoomAvailability(context.Background(), 1, 1, fixed(8))

	if !errors.Is(err, dbErr) {
		t.Fatalf("expected dbErr, got: %v", err)
	}
}

func TestGetRoomAvailability_BoundaryValues_DefaultDaysWhenZero(t *testing.T) {
	mock := newMock(t)
	now := fixed(8)
	endDate := time.Date(2024, 1, 22, 0, 0, 0, 0, time.UTC)
	availFrom := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
	availTo := time.Date(0, 1, 1, 10, 0, 0, 0, time.UTC)

	expectRoomWindow(mock, 1, availFrom, availTo, "UTC")
	expectBookingsQuery(mock, 1, endDate, now, noBookingRows())

	results, err := newBookingRepo(mock).GetRoomAvailability(context.Background(), 1, 0, now)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 7 {
		t.Errorf("expected 7 days, got %d", len(results))
	}
}

func TestGetRoomAvailability_EquivalenceClasses_NoBookingsAllSlotsAvailable(t *testing.T) {
	mock := newMock(t)
	now := fixed(8)
	endDate := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)
	availFrom := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
	availTo := time.Date(0, 1, 1, 18, 0, 0, 0, time.UTC)

	expectRoomWindow(mock, 1, availFrom, availTo, "UTC")
	expectBookingsQuery(mock, 1, endDate, now, noBookingRows())

	results, err := newBookingRepo(mock).GetRoomAvailability(context.Background(), 1, 1, now)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results[0].AvailableTimes) != 9 {
		t.Errorf("expected 9 slots, got %d: %v", len(results[0].AvailableTimes), results[0].AvailableTimes)
	}
	if results[0].AvailableTimes[0] != "09:00" {
		t.Errorf("first slot: got %q, want 09:00", results[0].AvailableTimes[0])
	}
}

func TestGetRoomAvailability_EquivalenceClasses_PastSlotsTodayAreFiltered(t *testing.T) {
	mock := newMock(t)
	now := fixed(14)
	endDate := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)
	availFrom := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
	availTo := time.Date(0, 1, 1, 18, 0, 0, 0, time.UTC)

	expectRoomWindow(mock, 1, availFrom, availTo, "UTC")
	expectBookingsQuery(mock, 1, endDate, now, noBookingRows())

	results, err := newBookingRepo(mock).GetRoomAvailability(context.Background(), 1, 1, now)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	times := results[0].AvailableTimes
	if len(times) != 3 {
		t.Errorf("expected 3 future slots, got %d: %v", len(times), times)
	}
	if times[0] != "15:00" {
		t.Errorf("first available slot: got %q, want 15:00", times[0])
	}
}

func TestGetRoomAvailability_EquivalenceClasses_BookingBlocksSlots(t *testing.T) {
	mock := newMock(t)
	now := fixed(8)
	endDate := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)
	availFrom := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
	availTo := time.Date(0, 1, 1, 18, 0, 0, 0, time.UTC)
	bStart := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	bEnd := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	expectRoomWindow(mock, 1, availFrom, availTo, "UTC")
	expectBookingsQuery(
		mock,
		1,
		endDate,
		now,
		pgxmock.NewRows([]string{"start_time", "end_time"}).AddRow(bStart, bEnd),
	)

	results, err := newBookingRepo(mock).GetRoomAvailability(context.Background(), 1, 1, now)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	times := results[0].AvailableTimes
	if len(times) != 7 {
		t.Errorf("expected 7 slots, got %d: %v", len(times), times)
	}
	for _, ts := range times {
		if ts == "10:00" || ts == "11:00" {
			t.Errorf("slot %q should be blocked", ts)
		}
	}
}

func TestGetRoomAvailability_EquivalenceClasses_MultipleBookingsBlockSlots(t *testing.T) {
	mock := newMock(t)
	now := fixed(8)
	endDate := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)
	availFrom := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
	availTo := time.Date(0, 1, 1, 18, 0, 0, 0, time.UTC)

	expectRoomWindow(mock, 1, availFrom, availTo, "UTC")
	expectBookingsQuery(
		mock,
		1,
		endDate,
		now,
		pgxmock.NewRows([]string{"start_time", "end_time"}).
			AddRow(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC), time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC)).
			AddRow(time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC), time.Date(2024, 1, 15, 16, 0, 0, 0, time.UTC)),
	)

	results, err := newBookingRepo(mock).GetRoomAvailability(context.Background(), 1, 1, now)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	times := results[0].AvailableTimes
	if len(times) != 7 {
		t.Errorf("expected 7 slots, got %d: %v", len(times), times)
	}
}

func TestGetRoomAvailability_ErrorGuessing_BookingsQueryErrorPropagates(t *testing.T) {
	mock := newMock(t)
	now := fixed(8)
	dbErr := fmt.Errorf("connection reset")
	availFrom := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
	availTo := time.Date(0, 1, 1, 18, 0, 0, 0, time.UTC)

	expectRoomWindow(mock, 1, availFrom, availTo, "UTC")
	mock.ExpectQuery(regexp.QuoteMeta(queryGetBookingsInRange)).
		WithArgs(1, pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(dbErr)

	_, err := newBookingRepo(mock).GetRoomAvailability(context.Background(), 1, 1, now)

	if !errors.Is(err, dbErr) {
		t.Fatalf("expected dbErr, got: %v", err)
	}
}

func TestCreateBooking_Scenario_Success(t *testing.T) {
	mock := newMock(t)
	now := fixed(8)
	msk, _ := time.LoadLocation("Europe/Moscow")
	startTime := time.Date(2024, 1, 16, 10, 0, 0, 0, msk)
	endTime := startTime.Add(2 * time.Hour)

	mock.ExpectQuery(regexp.QuoteMeta(queryGetRoomTimezone)).
		WithArgs(5).
		WillReturnRows(pgxmock.NewRows([]string{"timezone"}).AddRow("Europe/Moscow"))
	mock.ExpectQuery(regexp.QuoteMeta(queryCreateBookingCTE)).
		WithArgs(3, 5, startTime, endTime, 2).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(42))

	id, err := newBookingRepo(mock).CreateBooking(
		context.Background(),
		3,
		5,
		"2024-01-16",
		[]string{"10:00", "11:00"},
		now,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 42 {
		t.Errorf("id: got %d, want 42", id)
	}
}

func TestCreateBooking_ErrorGuessing_OverlapConflictReturnsErrConflict(t *testing.T) {
	mock := newMock(t)
	now := fixed(8)

	mock.ExpectQuery(regexp.QuoteMeta(queryGetRoomTimezone)).
		WithArgs(5).
		WillReturnRows(pgxmock.NewRows([]string{"timezone"}).AddRow("UTC"))
	mock.ExpectQuery(regexp.QuoteMeta(queryCreateBookingCTE)).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(&pgconn.PgError{ConstraintName: "bookings_no_overlap"})

	_, err := newBookingRepo(mock).CreateBooking(
		context.Background(),
		3,
		5,
		"2024-01-16",
		[]string{"10:00"},
		now,
	)

	if !errors.Is(err, db.ErrConflict) {
		t.Fatalf("expected db.ErrConflict, got: %v", err)
	}
}

func TestCreateBooking_EquivalenceClasses_RoomNotFoundOnTimezoneQueryReturnsErrNotFound(t *testing.T) {
	mock := newMock(t)

	mock.ExpectQuery(regexp.QuoteMeta(queryGetRoomTimezone)).
		WithArgs(99).
		WillReturnRows(pgxmock.NewRows([]string{"timezone"}))

	_, err := newBookingRepo(mock).CreateBooking(
		context.Background(),
		3,
		99,
		"2024-01-16",
		[]string{"10:00"},
		fixed(8),
	)

	if !errors.Is(err, db.ErrNotFound) {
		t.Fatalf("expected db.ErrNotFound, got: %v", err)
	}
}

func TestCreateBooking_ErrorGuessing_InvalidTimezoneFromDBReturnsError(t *testing.T) {
	mock := newMock(t)

	mock.ExpectQuery(regexp.QuoteMeta(queryGetRoomTimezone)).
		WithArgs(5).
		WillReturnRows(pgxmock.NewRows([]string{"timezone"}).AddRow("Not/ATimezone"))

	_, err := newBookingRepo(mock).CreateBooking(
		context.Background(),
		3,
		5,
		"2024-01-16",
		[]string{"10:00"},
		fixed(8),
	)

	if err == nil {
		t.Fatal("expected error for invalid timezone, got nil")
	}
}

func TestCreateBooking_EquivalenceClasses_PastStartTimeReturnsErrInvalidArgument(t *testing.T) {
	mock := newMock(t)
	now := fixed(12)

	mock.ExpectQuery(regexp.QuoteMeta(queryGetRoomTimezone)).
		WithArgs(5).
		WillReturnRows(pgxmock.NewRows([]string{"timezone"}).AddRow("UTC"))

	_, err := newBookingRepo(mock).CreateBooking(
		context.Background(),
		3,
		5,
		"2024-01-15",
		[]string{"10:00"},
		now,
	)

	if !errors.Is(err, db.ErrInvalidArgument) {
		t.Fatalf("expected db.ErrInvalidArgument, got: %v", err)
	}
}

func TestCreateBooking_ErrorGuessing_DBErrorPropagates(t *testing.T) {
	mock := newMock(t)
	dbErr := fmt.Errorf("connection timeout")
	now := fixed(8)

	mock.ExpectQuery(regexp.QuoteMeta(queryGetRoomTimezone)).
		WithArgs(5).
		WillReturnRows(pgxmock.NewRows([]string{"timezone"}).AddRow("UTC"))
	mock.ExpectQuery(regexp.QuoteMeta(queryCreateBookingCTE)).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(dbErr)

	_, err := newBookingRepo(mock).CreateBooking(
		context.Background(),
		3,
		5,
		"2024-01-16",
		[]string{"10:00"},
		now,
	)

	if !errors.Is(err, dbErr) {
		t.Fatalf("expected dbErr, got: %v", err)
	}
}

func TestCreateBooking_InputValidationEquivalenceClasses_NoDBCalls(t *testing.T) {
	now := fixed(8)

	cases := []struct {
		name    string
		userID  int
		roomID  int
		date    string
		slots   []string
		checkFn func(error) bool
	}{
		{
			name:   "zero userID",
			userID: 0,
			roomID: 5,
			date:   "2024-01-16",
			slots:  []string{"10:00"},
			checkFn: func(err error) bool {
				return errors.Is(err, db.ErrInvalidID)
			},
		},
		{
			name:   "negative roomID",
			userID: 3,
			roomID: -1,
			date:   "2024-01-16",
			slots:  []string{"10:00"},
			checkFn: func(err error) bool {
				return errors.Is(err, db.ErrInvalidID)
			},
		},
		{
			name:   "empty slots",
			userID: 3,
			roomID: 5,
			date:   "2024-01-16",
			slots:  []string{},
			checkFn: func(err error) bool {
				return errors.Is(err, db.ErrInvalidArgument)
			},
		},
		{
			name:   "bad slot format",
			userID: 3,
			roomID: 5,
			date:   "2024-01-16",
			slots:  []string{"25:99"},
			checkFn: func(err error) bool {
				return err != nil
			},
		},
		{
			name:   "non-consecutive slots",
			userID: 3,
			roomID: 5,
			date:   "2024-01-16",
			slots:  []string{"10:00", "12:00"},
			checkFn: func(err error) bool {
				return errors.Is(err, db.ErrNotConsecutiveSlots)
			},
		},
		{
			name:   "bad date format",
			userID: 3,
			roomID: 5,
			date:   "15-01-2024",
			slots:  []string{"10:00"},
			checkFn: func(err error) bool {
				return err != nil
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newMock(t)

			_, err := newBookingRepo(mock).CreateBooking(
				context.Background(),
				tc.userID,
				tc.roomID,
				tc.date,
				tc.slots,
				now,
			)

			if !tc.checkFn(err) {
				t.Errorf("unexpected error value: %v", err)
			}
		})
	}
}

func TestCancelBooking_StateTransitions_BookedCanBeCanceled(t *testing.T) {
	mock := newMock(t)
	now := fixed(8)

	mock.ExpectQuery(regexp.QuoteMeta(queryGetBookingForCancel)).
		WithArgs(1, 10).
		WillReturnRows(pgxmock.NewRows([]string{"status", "start_time", "end_time"}).
			AddRow("booked", fixed(10), fixed(11)))
	mock.ExpectExec(regexp.QuoteMeta(queryCancelBooking)).
		WithArgs(1, 10).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	if err := newBookingRepo(mock).CancelBooking(context.Background(), 1, 10, now); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCancelBooking_StateTransitions_NonCancellableStatesReturnErrConflict(t *testing.T) {
	now := fixed(12)

	cases := []struct {
		name     string
		dbStatus string
		start    time.Time
		end      time.Time
	}{
		{"active booking resolves to in_use", "booked", now.Add(-30 * time.Minute), now.Add(30 * time.Minute)},
		{"past booking resolves to finished", "booked", now.Add(-2 * time.Hour), now.Add(-time.Hour)},
		{"already canceled remains canceled", "canceled", now.Add(time.Hour), now.Add(2 * time.Hour)},
		{"booked in database but start already passed", "booked", now.Add(-time.Hour), now.Add(time.Hour)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newMock(t)

			mock.ExpectQuery(regexp.QuoteMeta(queryGetBookingForCancel)).
				WithArgs(1, 10).
				WillReturnRows(pgxmock.NewRows([]string{"status", "start_time", "end_time"}).
					AddRow(tc.dbStatus, tc.start, tc.end))

			err := newBookingRepo(mock).CancelBooking(context.Background(), 1, 10, now)

			if !errors.Is(err, db.ErrConflict) {
				t.Errorf("expected db.ErrConflict, got: %v", err)
			}
		})
	}
}

func TestCancelBooking_EquivalenceClasses_NotFoundReturnsErrNotFound(t *testing.T) {
	mock := newMock(t)

	mock.ExpectQuery(regexp.QuoteMeta(queryGetBookingForCancel)).
		WithArgs(999, 10).
		WillReturnRows(pgxmock.NewRows([]string{"status", "start_time", "end_time"}))

	err := newBookingRepo(mock).CancelBooking(context.Background(), 999, 10, fixed(8))

	if !errors.Is(err, db.ErrNotFound) {
		t.Fatalf("expected db.ErrNotFound, got: %v", err)
	}
}

func TestCancelBooking_ErrorGuessing_DBErrorPropagates(t *testing.T) {
	mock := newMock(t)
	dbErr := fmt.Errorf("connection reset")

	mock.ExpectQuery(regexp.QuoteMeta(queryGetBookingForCancel)).
		WithArgs(1, 10).
		WillReturnError(dbErr)

	err := newBookingRepo(mock).CancelBooking(context.Background(), 1, 10, fixed(8))

	if !errors.Is(err, dbErr) {
		t.Fatalf("expected dbErr, got: %v", err)
	}
}

func TestCancelBooking_BoundaryValues_InvalidIDsReturnErrInvalidID(t *testing.T) {
	cases := []struct {
		name      string
		bookingID int
		userID    int
	}{
		{"zero bookingID", 0, 1},
		{"negative bookingID", -1, 1},
		{"zero userID", 1, 0},
		{"negative userID", 1, -1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newMock(t)

			err := newBookingRepo(mock).CancelBooking(context.Background(), tc.bookingID, tc.userID, fixed(8))

			if !errors.Is(err, db.ErrInvalidID) {
				t.Errorf("expected db.ErrInvalidID, got: %v", err)
			}
		})
	}
}

func TestGetUserBookings_EquivalenceClasses_ValidTimezoneFormatsTimes(t *testing.T) {
	mock := newMock(t)
	now := fixed(6)

	msk, _ := time.LoadLocation("Europe/Moscow")
	start := time.Date(2024, 1, 15, 10, 0, 0, 0, msk).UTC()
	end := start.Add(time.Hour)

	mock.ExpectQuery(regexp.QuoteMeta(queryGetUserBookings)).
		WithArgs(7).
		WillReturnRows(pgxmock.NewRows(bookingCols).
			AddRow(1, 10, "img.jpg", "Room A", start, end, 1000, "booked", "Europe/Moscow"))

	bookings, err := newBookingRepo(mock).GetUserBookings(context.Background(), 7, now)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bookings) != 1 {
		t.Fatalf("expected 1 booking, got %d", len(bookings))
	}
	if bookings[0].StartTime != "10:00" {
		t.Errorf("StartTime: got %q, want 10:00", bookings[0].StartTime)
	}
	if bookings[0].Status != "booked" {
		t.Errorf("Status: got %q, want booked", bookings[0].Status)
	}
}

func TestGetUserBookings_EquivalenceClasses_NoBookingsReturnsEmptySlice(t *testing.T) {
	mock := newMock(t)

	mock.ExpectQuery(regexp.QuoteMeta(queryGetUserBookings)).
		WithArgs(7).
		WillReturnRows(pgxmock.NewRows(bookingCols))

	bookings, err := newBookingRepo(mock).GetUserBookings(context.Background(), 7, fixed(8))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bookings == nil {
		t.Fatal("expected non-nil empty slice, got nil")
	}
}

func TestGetUserBookings_ErrorGuessing_DBErrorPropagates(t *testing.T) {
	mock := newMock(t)
	dbErr := fmt.Errorf("connection refused")

	mock.ExpectQuery(regexp.QuoteMeta(queryGetUserBookings)).
		WithArgs(7).
		WillReturnError(dbErr)

	_, err := newBookingRepo(mock).GetUserBookings(context.Background(), 7, fixed(8))

	if !errors.Is(err, dbErr) {
		t.Fatalf("expected dbErr, got: %v", err)
	}
}

func TestGetUserBookings_ErrorGuessing_InvalidTimezoneInRowReturnsError(t *testing.T) {
	mock := newMock(t)
	start := fixed(10)
	end := fixed(11)

	mock.ExpectQuery(regexp.QuoteMeta(queryGetUserBookings)).
		WithArgs(7).
		WillReturnRows(pgxmock.NewRows(bookingCols).
			AddRow(1, 10, "", "Room A", start, end, 1000, "booked", "Bad/Zone"))

	_, err := newBookingRepo(mock).GetUserBookings(context.Background(), 7, fixed(8))

	if err == nil {
		t.Fatal("expected error for invalid timezone, got nil")
	}
}

func TestGetUserBookings_BoundaryValues_InvalidUserIDReturnsErrInvalidID(t *testing.T) {
	cases := []struct {
		name   string
		userID int
	}{
		{"zero userID", 0},
		{"negative userID", -1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newMock(t)

			_, err := newBookingRepo(mock).GetUserBookings(context.Background(), tc.userID, fixed(8))

			if !errors.Is(err, db.ErrInvalidID) {
				t.Errorf("expected db.ErrInvalidID, got: %v", err)
			}
		})
	}
}

func TestDBStatusForFilter_EquivalenceClasses(t *testing.T) {
	cases := []struct {
		name   string
		status string
		want   string
	}{
		{"booked stays booked", "booked", "booked"},
		{"in_use maps to booked", "in_use", "booked"},
		{"finished maps to booked", "finished", "booked"},
		{"canceled stays canceled", "canceled", "canceled"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := dbStatusForFilter(tc.status)

			if got != tc.want {
				t.Fatalf("dbStatusForFilter(%q): got %q, want %q", tc.status, got, tc.want)
			}
		})
	}
}

func TestListAdminBookings_BoundaryValues_InvalidAdminIDReturnsErrInvalidID(t *testing.T) {
	cases := []struct {
		name    string
		adminID int
	}{
		{"admin id is zero", 0},
		{"admin id is negative", -1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newMock(t)
			repo := newBookingRepo(mock)

			_, err := repo.ListAdminBookings(context.Background(), tc.adminID, false, nil, nil, nil, fixed(8))

			if !errors.Is(err, db.ErrInvalidID) {
				t.Fatalf("got error %v, want %v", err, db.ErrInvalidID)
			}
		})
	}
}

func TestListAdminBookings_BoundaryValues_InvalidFilterIDsReturnErrInvalidID(t *testing.T) {
	cases := []struct {
		name       string
		locationID *int
		roomID     *int
	}{
		{"location id is zero", intPtr(0), nil},
		{"location id is negative", intPtr(-1), nil},
		{"room id is zero", nil, intPtr(0)},
		{"room id is negative", nil, intPtr(-1)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newMock(t)
			repo := newBookingRepo(mock)

			_, err := repo.ListAdminBookings(context.Background(), 7, false, tc.locationID, tc.roomID, nil, fixed(8))

			if !errors.Is(err, db.ErrInvalidID) {
				t.Fatalf("got error %v, want %v", err, db.ErrInvalidID)
			}
		})
	}
}

func TestListAdminBookings_EquivalenceClasses_InvalidStatusReturnsErrInvalidArgument(t *testing.T) {
	mock := newMock(t)
	repo := newBookingRepo(mock)
	status := "unknown"

	_, err := repo.ListAdminBookings(context.Background(), 7, false, nil, nil, &status, fixed(8))

	if !errors.Is(err, db.ErrInvalidArgument) {
		t.Fatalf("got error %v, want %v", err, db.ErrInvalidArgument)
	}
}

func TestListAdminBookings_EquivalenceClasses_SuccessWithFilters(t *testing.T) {
	mock := newMock(t)
	repo := newBookingRepo(mock)
	now := fixed(6)

	locationID := 10
	roomID := 5
	status := "booked"

	msk, _ := time.LoadLocation("Europe/Moscow")
	start := time.Date(2024, 1, 15, 10, 0, 0, 0, msk).UTC()
	end := start.Add(time.Hour)

	mock.ExpectQuery(regexp.QuoteMeta(queryListAdminBookings)).
		WithArgs(false, 7, locationID, roomID, "booked").
		WillReturnRows(pgxmock.NewRows(adminBookingCols).
			AddRow(1, 5, "Room A", 10, "Москва, Тверская 10", 3, "user@example.com", "alice", start, end, 1000, "booked", "Europe/Moscow"))

	items, err := repo.ListAdminBookings(context.Background(), 7, false, &locationID, &roomID, &status, now)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	if items[0].ID != 1 || items[0].RoomID != 5 || items[0].LocationID != 10 {
		t.Fatalf("unexpected item: %+v", items[0])
	}
	if items[0].StartTime != "10:00" || items[0].Status != "booked" {
		t.Fatalf("unexpected formatted fields: %+v", items[0])
	}
}

func TestListAdminBookings_EquivalenceClasses_EmptyResultReturnsEmptySlice(t *testing.T) {
	mock := newMock(t)
	repo := newBookingRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(queryListAdminBookings)).
		WithArgs(false, 7, nil, nil, nil).
		WillReturnRows(pgxmock.NewRows(adminBookingCols))

	items, err := repo.ListAdminBookings(context.Background(), 7, false, nil, nil, nil, fixed(8))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if items == nil {
		t.Fatal("got nil, want non-nil empty slice")
	}
	if len(items) != 0 {
		t.Fatalf("got %d items, want 0", len(items))
	}
}

func TestListAdminBookings_StateTransitions_StatusFilterUsesResolvedStatus(t *testing.T) {
	mock := newMock(t)
	repo := newBookingRepo(mock)

	now := fixed(8)
	status := "in_use"
	msk, _ := time.LoadLocation("Europe/Moscow")

	activeStart := time.Date(2024, 1, 15, 10, 0, 0, 0, msk).UTC()
	activeEnd := activeStart.Add(2 * time.Hour)
	futureStart := time.Date(2024, 1, 15, 15, 0, 0, 0, msk).UTC()
	futureEnd := futureStart.Add(time.Hour)

	mock.ExpectQuery(regexp.QuoteMeta(queryListAdminBookings)).
		WithArgs(false, 7, nil, nil, "booked").
		WillReturnRows(pgxmock.NewRows(adminBookingCols).
			AddRow(1, 5, "Room A", 10, "Москва, Тверская 10", 3, "user@example.com", "alice", activeStart, activeEnd, 1000, "booked", "Europe/Moscow").
			AddRow(2, 6, "Room B", 10, "Москва, Арбат 5", 4, "bob@example.com", "bob", futureStart, futureEnd, 2000, "booked", "Europe/Moscow"))

	items, err := repo.ListAdminBookings(context.Background(), 7, false, nil, nil, &status, now)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	if items[0].ID != 1 || items[0].Status != "in_use" {
		t.Fatalf("unexpected item: %+v", items[0])
	}
}

func TestListAdminBookings_DecisionTable_IncludeAllAllowsZeroAdminID(t *testing.T) {
	mock := newMock(t)
	repo := newBookingRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(queryListAdminBookings)).
		WithArgs(true, 0, nil, nil, nil).
		WillReturnRows(pgxmock.NewRows(adminBookingCols))

	items, err := repo.ListAdminBookings(context.Background(), 0, true, nil, nil, nil, fixed(8))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if items == nil {
		t.Fatal("got nil, want non-nil empty slice")
	}
	if len(items) != 0 {
		t.Fatalf("got %d items, want 0", len(items))
	}
}

func TestListAdminBookings_ErrorGuessing_QueryErrorPropagates(t *testing.T) {
	mock := newMock(t)
	repo := newBookingRepo(mock)

	dbErr := fmt.Errorf("admin bookings query failed")
	mock.ExpectQuery(regexp.QuoteMeta(queryListAdminBookings)).
		WithArgs(false, 7, nil, nil, nil).
		WillReturnError(dbErr)

	_, err := repo.ListAdminBookings(context.Background(), 7, false, nil, nil, nil, fixed(8))

	if !errors.Is(err, dbErr) {
		t.Fatalf("got error %v, want %v", err, dbErr)
	}
}

func TestListAdminBookings_ErrorGuessing_ScanErrorPropagates(t *testing.T) {
	mock := newMock(t)
	repo := newBookingRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(queryListAdminBookings)).
		WithArgs(false, 7, nil, nil, nil).
		WillReturnRows(pgxmock.NewRows(adminBookingCols).
			AddRow("bad-id", 5, "Room A", 10, "Москва, Тверская 10", 3, "user@example.com", "alice", fixed(10), fixed(11), 1000, "booked", "UTC"))

	_, err := repo.ListAdminBookings(context.Background(), 7, false, nil, nil, nil, fixed(8))

	if err == nil {
		t.Fatal("expected scan error, got nil")
	}
}

func TestListAdminBookings_ErrorGuessing_InvalidTimezoneInRowReturnsError(t *testing.T) {
	mock := newMock(t)
	repo := newBookingRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(queryListAdminBookings)).
		WithArgs(false, 7, nil, nil, nil).
		WillReturnRows(pgxmock.NewRows(adminBookingCols).
			AddRow(1, 5, "Room A", 10, "Москва, Тверская 10", 3, "user@example.com", "alice", fixed(10), fixed(11), 1000, "booked", "Bad/Zone"))

	_, err := repo.ListAdminBookings(context.Background(), 7, false, nil, nil, nil, fixed(8))

	if err == nil {
		t.Fatal("expected error for invalid timezone, got nil")
	}
}

func TestCancelAdminBooking_BoundaryValues_InvalidIDsReturnErrInvalidID(t *testing.T) {
	cases := []struct {
		name       string
		adminID    int
		includeAll bool
		bookingID  int
	}{
		{"zero bookingID", 7, false, 0},
		{"negative bookingID", 7, false, -1},
		{"zero adminID without includeAll", 0, false, 1},
		{"negative adminID without includeAll", -1, false, 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newMock(t)
			repo := newBookingRepo(mock)

			err := repo.CancelAdminBooking(context.Background(), tc.adminID, tc.includeAll, tc.bookingID, fixed(8))

			if !errors.Is(err, db.ErrInvalidID) {
				t.Fatalf("got error %v, want %v", err, db.ErrInvalidID)
			}
		})
	}
}

func TestCancelAdminBooking_StateTransitions_BookedCanBeCanceled(t *testing.T) {
	mock := newMock(t)
	repo := newBookingRepo(mock)
	now := fixed(8)

	mock.ExpectQuery(regexp.QuoteMeta(queryGetAdminBookingForCancel)).
		WithArgs(1, false, 7).
		WillReturnRows(pgxmock.NewRows([]string{"status", "start_time", "end_time", "accessible"}).
			AddRow("booked", fixed(10), fixed(11), true))
	mock.ExpectExec(regexp.QuoteMeta(queryCancelAdminBooking)).
		WithArgs(1).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := repo.CancelAdminBooking(context.Background(), 7, false, 1, now)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCancelAdminBooking_EquivalenceClasses_NotFoundReturnsErrNotFound(t *testing.T) {
	mock := newMock(t)
	repo := newBookingRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(queryGetAdminBookingForCancel)).
		WithArgs(999, false, 7).
		WillReturnRows(pgxmock.NewRows([]string{"status", "start_time", "end_time", "accessible"}))

	err := repo.CancelAdminBooking(context.Background(), 7, false, 999, fixed(8))

	if !errors.Is(err, db.ErrNotFound) {
		t.Fatalf("got error %v, want %v", err, db.ErrNotFound)
	}
}

func TestCancelAdminBooking_ErrorGuessing_QueryErrorPropagates(t *testing.T) {
	mock := newMock(t)
	repo := newBookingRepo(mock)

	dbErr := fmt.Errorf("admin cancel query failed")
	mock.ExpectQuery(regexp.QuoteMeta(queryGetAdminBookingForCancel)).
		WithArgs(1, false, 7).
		WillReturnError(dbErr)

	err := repo.CancelAdminBooking(context.Background(), 7, false, 1, fixed(8))

	if !errors.Is(err, dbErr) {
		t.Fatalf("got error %v, want %v", err, dbErr)
	}
}

func TestCancelAdminBooking_EquivalenceClasses_InaccessibleBookingReturnsErrForbidden(t *testing.T) {
	mock := newMock(t)
	repo := newBookingRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(queryGetAdminBookingForCancel)).
		WithArgs(1, false, 7).
		WillReturnRows(pgxmock.NewRows([]string{"status", "start_time", "end_time", "accessible"}).
			AddRow("booked", fixed(10), fixed(11), false))

	err := repo.CancelAdminBooking(context.Background(), 7, false, 1, fixed(8))

	if !errors.Is(err, db.ErrForbidden) {
		t.Fatalf("got error %v, want %v", err, db.ErrForbidden)
	}
}

func TestCancelAdminBooking_StateTransitions_NonCancellableStatesReturnErrConflict(t *testing.T) {
	now := fixed(12)

	cases := []struct {
		name     string
		dbStatus string
		start    time.Time
		end      time.Time
	}{
		{"active booking resolves to in_use", "booked", now.Add(-30 * time.Minute), now.Add(30 * time.Minute)},
		{"past booking resolves to finished", "booked", now.Add(-2 * time.Hour), now.Add(-time.Hour)},
		{"already canceled remains canceled", "canceled", now.Add(time.Hour), now.Add(2 * time.Hour)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newMock(t)
			repo := newBookingRepo(mock)

			mock.ExpectQuery(regexp.QuoteMeta(queryGetAdminBookingForCancel)).
				WithArgs(1, false, 7).
				WillReturnRows(pgxmock.NewRows([]string{"status", "start_time", "end_time", "accessible"}).
					AddRow(tc.dbStatus, tc.start, tc.end, true))

			err := repo.CancelAdminBooking(context.Background(), 7, false, 1, now)

			if !errors.Is(err, db.ErrConflict) {
				t.Fatalf("got error %v, want %v", err, db.ErrConflict)
			}
		})
	}
}

func TestCancelAdminBooking_ErrorGuessing_UpdateErrorPropagates(t *testing.T) {
	mock := newMock(t)
	repo := newBookingRepo(mock)
	updateErr := fmt.Errorf("update failed")

	mock.ExpectQuery(regexp.QuoteMeta(queryGetAdminBookingForCancel)).
		WithArgs(1, false, 7).
		WillReturnRows(pgxmock.NewRows([]string{"status", "start_time", "end_time", "accessible"}).
			AddRow("booked", fixed(10), fixed(11), true))
	mock.ExpectExec(regexp.QuoteMeta(queryCancelAdminBooking)).
		WithArgs(1).
		WillReturnError(updateErr)

	err := repo.CancelAdminBooking(context.Background(), 7, false, 1, fixed(8))

	if !errors.Is(err, updateErr) {
		t.Fatalf("got error %v, want %v", err, updateErr)
	}
}

func TestCancelAdminBooking_EquivalenceClasses_UpdateZeroRowsReturnsErrNotFound(t *testing.T) {
	mock := newMock(t)
	repo := newBookingRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(queryGetAdminBookingForCancel)).
		WithArgs(1, false, 7).
		WillReturnRows(pgxmock.NewRows([]string{"status", "start_time", "end_time", "accessible"}).
			AddRow("booked", fixed(10), fixed(11), true))
	mock.ExpectExec(regexp.QuoteMeta(queryCancelAdminBooking)).
		WithArgs(1).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err := repo.CancelAdminBooking(context.Background(), 7, false, 1, fixed(8))

	if !errors.Is(err, db.ErrNotFound) {
		t.Fatalf("got error %v, want %v", err, db.ErrNotFound)
	}
}
