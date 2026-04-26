package db

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	pgxmock "github.com/pashagolub/pgxmock/v2"
)

func newBookingRepo(mock pgxmock.PgxPoolIface) *BookingRepo {
	return NewBookingRepo(mock)
}

func fixed(hour int) time.Time {
	return time.Date(2024, 1, 15, hour, 0, 0, 0, time.UTC)
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
		{"canceled: absorbing state", "canceled", fixed(9), "canceled"},
		{"before start -> booked", "booked", fixed(9), "booked"},
		{"after end -> finished", "booked", fixed(13), "finished"},
		{"between -> in_use", "booked", fixed(11), "in_use"},
		{"now == start -> in_use (BVA)", "booked", start, "in_use"},
		{"now == end -> in_use (BVA)", "booked", end, "in_use"},
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

// GetRoomAvailability

func TestGetRoomAvailability_InvalidRoomID_ReturnsErrInvalidID(t *testing.T) {
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

			if !errors.Is(err, ErrInvalidID) {
				t.Errorf("expected ErrInvalidID, got: %v", err)
			}
		})
	}
}

func TestGetRoomAvailability_RoomNotFound_ReturnsErrNotFound(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta(queryGetRoomWindow)).
		WithArgs(99).
		WillReturnRows(pgxmock.NewRows(roomWindowCols))

	_, err := newBookingRepo(mock).GetRoomAvailability(context.Background(), 99, 1, fixed(8))

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestGetRoomAvailability_InvalidTimezone_ReturnsError(t *testing.T) {
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

func TestGetRoomAvailability_RoomQueryError_PropagatesError(t *testing.T) {
	mock := newMock(t)
	dbErr := fmt.Errorf("timeout")
	mock.ExpectQuery(regexp.QuoteMeta(queryGetRoomWindow)).
		WithArgs(1).WillReturnError(dbErr)

	_, err := newBookingRepo(mock).GetRoomAvailability(context.Background(), 1, 1, fixed(8))

	if !errors.Is(err, dbErr) {
		t.Fatalf("expected dbErr, got: %v", err)
	}
}

func TestGetRoomAvailability_DefaultDays_WhenZero(t *testing.T) {
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

func TestGetRoomAvailability_NormalWindow_NoBookings_AllSlotsAvailable(t *testing.T) {
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

func TestGetRoomAvailability_PastSlotsToday_AreFiltered(t *testing.T) {
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

func TestGetRoomAvailability_WithBooking_BlocksSlots(t *testing.T) {
	mock := newMock(t)
	now := fixed(8)
	endDate := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)
	availFrom := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
	availTo := time.Date(0, 1, 1, 18, 0, 0, 0, time.UTC)
	bStart := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	bEnd := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	expectRoomWindow(mock, 1, availFrom, availTo, "UTC")
	expectBookingsQuery(mock, 1, endDate, now,
		pgxmock.NewRows([]string{"start_time", "end_time"}).AddRow(bStart, bEnd))

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

func TestGetRoomAvailability_MultipleBookings_EarlyBreakCorrect(t *testing.T) {
	mock := newMock(t)
	now := fixed(8)
	endDate := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)
	availFrom := time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC)
	availTo := time.Date(0, 1, 1, 18, 0, 0, 0, time.UTC)

	expectRoomWindow(mock, 1, availFrom, availTo, "UTC")
	expectBookingsQuery(mock, 1, endDate, now,
		pgxmock.NewRows([]string{"start_time", "end_time"}).
			AddRow(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC), time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC)).
			AddRow(time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC), time.Date(2024, 1, 15, 16, 0, 0, 0, time.UTC)))

	results, err := newBookingRepo(mock).GetRoomAvailability(context.Background(), 1, 1, now)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	times := results[0].AvailableTimes
	if len(times) != 7 {
		t.Errorf("expected 7 slots (9 - 2), got %d: %v", len(times), times)
	}
}

func TestGetRoomAvailability_BookingsQueryError_PropagatesError(t *testing.T) {
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

// CreateBooking

func TestCreateBooking_Success(t *testing.T) {
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

	id, err := newBookingRepo(mock).CreateBooking(context.Background(), 3, 5,
		"2024-01-16", []string{"10:00", "11:00"}, now)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 42 {
		t.Errorf("id: got %d, want 42", id)
	}
}

func TestCreateBooking_Conflict_ReturnsErrConflict(t *testing.T) {
	mock := newMock(t)
	now := fixed(8)

	mock.ExpectQuery(regexp.QuoteMeta(queryGetRoomTimezone)).
		WithArgs(5).
		WillReturnRows(pgxmock.NewRows([]string{"timezone"}).AddRow("UTC"))
	mock.ExpectQuery(regexp.QuoteMeta(queryCreateBookingCTE)).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(&pgconn.PgError{ConstraintName: "bookings_no_overlap"})

	_, err := newBookingRepo(mock).CreateBooking(context.Background(), 3, 5,
		"2024-01-16", []string{"10:00"}, now)

	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict, got: %v", err)
	}
}

func TestCreateBooking_RoomNotFound_OnTimezoneQuery_ReturnsErrNotFound(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta(queryGetRoomTimezone)).
		WithArgs(99).
		WillReturnRows(pgxmock.NewRows([]string{"timezone"}))

	_, err := newBookingRepo(mock).CreateBooking(context.Background(), 3, 99,
		"2024-01-16", []string{"10:00"}, fixed(8))

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestCreateBooking_InvalidTimezoneFromDB_ReturnsError(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta(queryGetRoomTimezone)).
		WithArgs(5).
		WillReturnRows(pgxmock.NewRows([]string{"timezone"}).AddRow("Not/ATimezone"))

	_, err := newBookingRepo(mock).CreateBooking(context.Background(), 3, 5,
		"2024-01-16", []string{"10:00"}, fixed(8))

	if err == nil {
		t.Fatal("expected error for invalid timezone, got nil")
	}
}

func TestCreateBooking_PastStartTime_ReturnsErrInvalidArgument(t *testing.T) {
	mock := newMock(t)
	now := fixed(12)
	mock.ExpectQuery(regexp.QuoteMeta(queryGetRoomTimezone)).
		WithArgs(5).
		WillReturnRows(pgxmock.NewRows([]string{"timezone"}).AddRow("UTC"))

	_, err := newBookingRepo(mock).CreateBooking(context.Background(), 3, 5,
		"2024-01-15", []string{"10:00"}, now)

	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("expected ErrInvalidArgument, got: %v", err)
	}
}

func TestCreateBooking_DBError_PropagatesError(t *testing.T) {
	mock := newMock(t)
	dbErr := fmt.Errorf("connection timeout")
	now := fixed(8)

	mock.ExpectQuery(regexp.QuoteMeta(queryGetRoomTimezone)).
		WithArgs(5).
		WillReturnRows(pgxmock.NewRows([]string{"timezone"}).AddRow("UTC"))
	mock.ExpectQuery(regexp.QuoteMeta(queryCreateBookingCTE)).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(dbErr)

	_, err := newBookingRepo(mock).CreateBooking(context.Background(), 3, 5,
		"2024-01-16", []string{"10:00"}, now)

	if !errors.Is(err, dbErr) {
		t.Fatalf("expected dbErr, got: %v", err)
	}
}

func TestCreateBooking_InputValidation_NoDBCalls(t *testing.T) {
	now := fixed(8)
	cases := []struct {
		name    string
		userID  int
		roomID  int
		date    string
		slots   []string
		checkFn func(error) bool
	}{
		{"zero userID", 0, 5, "2024-01-16", []string{"10:00"},
			func(e error) bool { return errors.Is(e, ErrInvalidID) }},
		{"negative roomID", 3, -1, "2024-01-16", []string{"10:00"},
			func(e error) bool { return errors.Is(e, ErrInvalidID) }},
		{"empty slots", 3, 5, "2024-01-16", []string{},
			func(e error) bool { return errors.Is(e, ErrInvalidArgument) }},
		{"bad slot format", 3, 5, "2024-01-16", []string{"25:99"},
			func(e error) bool { return e != nil }},
		{"non-consecutive", 3, 5, "2024-01-16", []string{"10:00", "12:00"},
			func(e error) bool { return e != nil }},
		{"bad date format", 3, 5, "15-01-2024", []string{"10:00"},
			func(e error) bool { return e != nil }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newMock(t)
			_, err := newBookingRepo(mock).CreateBooking(context.Background(),
				tc.userID, tc.roomID, tc.date, tc.slots, now)
			if !tc.checkFn(err) {
				t.Errorf("unexpected error value: %v", err)
			}
		})
	}
}

// CancelBooking

func TestCancelBooking_BookedStatus_Success(t *testing.T) {
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

func TestCancelBooking_NonCancellableStates_ReturnErrConflict(t *testing.T) {
	now := fixed(12)
	cases := []struct {
		name     string
		dbStatus string
		start    time.Time
		end      time.Time
	}{
		{"in_use: active", "booked", now.Add(-30 * time.Minute), now.Add(30 * time.Minute)},
		{"finished: past", "booked", now.Add(-2 * time.Hour), now.Add(-time.Hour)},
		{"already canceled", "canceled", now.Add(time.Hour), now.Add(2 * time.Hour)},
		{"booked in DB but start passed (p.3)", "booked", now.Add(-time.Hour), now.Add(time.Hour)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newMock(t)
			mock.ExpectQuery(regexp.QuoteMeta(queryGetBookingForCancel)).
				WithArgs(1, 10).
				WillReturnRows(pgxmock.NewRows([]string{"status", "start_time", "end_time"}).
					AddRow(tc.dbStatus, tc.start, tc.end))

			if err := newBookingRepo(mock).CancelBooking(context.Background(), 1, 10, now); !errors.Is(err, ErrConflict) {
				t.Errorf("[%s] expected ErrConflict, got: %v", tc.name, err)
			}
		})
	}
}

func TestCancelBooking_NotFound_ReturnsErrNotFound(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta(queryGetBookingForCancel)).
		WithArgs(999, 10).
		WillReturnRows(pgxmock.NewRows([]string{"status", "start_time", "end_time"}))

	if err := newBookingRepo(mock).CancelBooking(context.Background(), 999, 10, fixed(8)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestCancelBooking_DBError_PropagatesError(t *testing.T) {
	mock := newMock(t)
	dbErr := fmt.Errorf("connection reset")
	mock.ExpectQuery(regexp.QuoteMeta(queryGetBookingForCancel)).
		WithArgs(1, 10).WillReturnError(dbErr)

	if err := newBookingRepo(mock).CancelBooking(context.Background(), 1, 10, fixed(8)); !errors.Is(err, dbErr) {
		t.Fatalf("expected dbErr, got: %v", err)
	}
}

func TestCancelBooking_InvalidIDs_ReturnErrInvalidID(t *testing.T) {
	cases := []struct {
		name     string
		bid, uid int
	}{
		{"zero bookingID", 0, 1}, {"negative bookingID", -1, 1},
		{"zero userID", 1, 0}, {"negative userID", 1, -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newMock(t)
			if err := newBookingRepo(mock).CancelBooking(context.Background(), tc.bid, tc.uid, fixed(8)); !errors.Is(err, ErrInvalidID) {
				t.Errorf("expected ErrInvalidID, got: %v", err)
			}
		})
	}
}

// GetUserBookings

var bookingCols = []string{"b_id", "r_id", "image_url", "title", "start_time", "end_time", "total_price", "status", "timezone"}

func TestGetUserBookings_Success_TimesFormattedInRoomTimezone(t *testing.T) {
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
		t.Errorf("StartTime: got %q, want 10:00 (MSK)", bookings[0].StartTime)
	}
	if bookings[0].Status != "booked" {
		t.Errorf("Status: got %q, want booked", bookings[0].Status)
	}
}

func TestGetUserBookings_NoBookings_ReturnsEmptySlice(t *testing.T) {
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

func TestGetUserBookings_DBError_PropagatesError(t *testing.T) {
	mock := newMock(t)
	dbErr := fmt.Errorf("connection refused")
	mock.ExpectQuery(regexp.QuoteMeta(queryGetUserBookings)).
		WithArgs(7).WillReturnError(dbErr)

	if _, err := newBookingRepo(mock).GetUserBookings(context.Background(), 7, fixed(8)); !errors.Is(err, dbErr) {
		t.Fatalf("expected dbErr, got: %v", err)
	}
}

func TestGetUserBookings_InvalidTimezoneInRow_ReturnsError(t *testing.T) {
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

func TestGetUserBookings_InvalidUserID_ReturnsErrInvalidID(t *testing.T) {
	for _, id := range []int{0, -1} {
		mock := newMock(t)
		if _, err := newBookingRepo(mock).GetUserBookings(context.Background(), id, fixed(8)); !errors.Is(err, ErrInvalidID) {
			t.Errorf("id=%d: expected ErrInvalidID, got: %v", id, err)
		}
	}
}
