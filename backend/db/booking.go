package db

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/nikolaarden2000/software-design-project/backend/models"
)

const (
	queryGetRoomWindow = `
		SELECT r.available_from, r.available_to, l.timezone
		FROM rooms r
		JOIN locations l ON r.location_id = l.id
		WHERE r.id = $1`

	queryGetBookingsInRange = `
		SELECT start_time, end_time
		FROM bookings
		WHERE room_id = $1
			AND status <> 'canceled'
			AND start_time < $2
			AND end_time > $3`

	queryGetRoomTimezone = `
		SELECT l.timezone
		FROM rooms r
		JOIN locations l ON r.location_id = l.id
		WHERE r.id = $1`

	queryCreateBookingCTE = `
		WITH room_price AS (
				SELECT price FROM rooms WHERE id = $2
		)
		INSERT INTO bookings (user_id, room_id, start_time, end_time, total_price)
		SELECT $1, $2, $3, $4, price * $5
		FROM room_price
		RETURNING id`

	queryGetUserBookings = `
		SELECT
			b.id,
			r.id,
			COALESCE(r.images[1], ''),
			r.title,
			b.start_time,
			b.end_time,
			b.total_price,
			b.status,
			l.timezone
		FROM bookings b
		JOIN rooms r ON b.room_id = r.id
		JOIN locations l ON r.location_id = l.id
		WHERE b.user_id = $1
		ORDER BY b.id DESC`

	queryGetBookingForCancel = `
		SELECT status, start_time, end_time
		FROM bookings
		WHERE id = $1 AND user_id = $2`

	queryCancelBooking = `
		UPDATE bookings SET status = 'canceled'
		WHERE id = $1 AND user_id = $2`
)

type BookingRepo struct {
	q Querier
}

func NewBookingRepo(q Querier) *BookingRepo {
	return &BookingRepo{q: q}
}

func loadLocation(name string) (*time.Location, error) {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %q: %w", name, err)
	}
	return loc, nil
}

type booking struct {
	Start, End time.Time
}

func coalesceTime(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now()
	}
	return t
}

func generateHourlyStots(winStart, winEnd time.Time) []time.Time {
	var slots []time.Time
	for t := winStart; !t.Add(time.Hour).After(winEnd); t = t.Add(time.Hour) {
		slots = append(slots, t)
	}
	return slots
}

func (r *BookingRepo) GetRoomAvailability(ctx context.Context, roomID, days int, now time.Time) ([]models.DateAvailability, error) {
	if roomID <= 0 {
		return nil, ErrInvalidID
	}
	if days <= 0 {
		days = 7
	}
	now = coalesceTime(now)

	var availFrom, availTo time.Time
	var tzName string
	if err := r.q.QueryRow(ctx, queryGetRoomWindow, roomID).Scan(&availFrom, &availTo, &tzName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("GetRoomAvailability: select room: %w", err)
	}

	loc, err := loadLocation(tzName)
	if err != nil {
		return nil, fmt.Errorf("GetRoomAvailability: %w", err)
	}

	now = now.In(loc)
	startDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	endDate := startDate.AddDate(0, 0, days)

	rows, err := r.q.Query(ctx, queryGetBookingsInRange, roomID, endDate, now)
	if err != nil {
		return nil, fmt.Errorf("GetRoomAvailability: select bookings: %w", err)
	}
	defer rows.Close()

	bookings := make([]booking, 0)
	for rows.Next() {
		var s, e time.Time
		if err := rows.Scan(&s, &e); err != nil {
			return nil, fmt.Errorf("GetRoomAvailability: scan booking: %w", err)
		}
		bookings = append(bookings, booking{s, e})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetRoomAvailability: iterate bookings: %w", err)
	}

	sort.Slice(bookings, func(i, j int) bool {
		return bookings[i].Start.Before(bookings[j].Start)
	})

	results := make([]models.DateAvailability, 0, days)
	for d := 0; d < days; d++ {
		day := startDate.AddDate(0, 0, d)

		winStart := time.Date(day.Year(), day.Month(), day.Day(), availFrom.Hour(), availFrom.Minute(), availFrom.Second(), 0, loc)
		winEnd := time.Date(day.Year(), day.Month(), day.Day(), availTo.Hour(), availTo.Minute(), availTo.Second(), 0, loc)

		availableTimes := make([]string, 0)
		for _, slot := range generateHourlyStots(winStart, winEnd) {
			if !slot.After(now) {
				continue
			}
			slotEnd := slot.Add(time.Hour)
			overlaps := false
			for _, b := range bookings {
				if !b.Start.Before(slotEnd) {
					break
				}
				if b.End.After(slot) {
					overlaps = true
					break
				}
			}
			if !overlaps {
				availableTimes = append(availableTimes, slot.Format("15:04"))
			}
		}

		results = append(results, models.DateAvailability{
			Date:           day.Format("2006-01-02"),
			AvailableTimes: availableTimes,
		})
	}

	return results, nil
}

func (r *BookingRepo) CreateBooking(ctx context.Context, userID, roomID int, date string, slots []string, now time.Time) (int, error) {
	if userID <= 0 || roomID <= 0 {
		return 0, ErrInvalidID
	}

	now = coalesceTime(now)

	if len(slots) == 0 {
		return 0, ErrInvalidArgument
	}

	times := make([]time.Time, len(slots))
	for i, s := range slots {
		t, err := time.Parse("15:04", s)
		if err != nil {
			return 0, fmt.Errorf("invalid slot format: %s", s)
		}
		times[i] = t
	}
	for i := 1; i < len(times); i++ {
		if !times[i].Equal(times[i-1].Add(time.Hour)) {
			return 0, fmt.Errorf("slots must be consecutive")
		}
	}

	day, err := time.Parse("2006-01-02", date)
	if err != nil {
		return 0, fmt.Errorf("invalid date: %v", err)
	}

	var tzName string
	if err := r.q.QueryRow(ctx, queryGetRoomTimezone, roomID).Scan(&tzName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, fmt.Errorf("CreateBooking: fetch timezone: %w", err)
	}
	loc, err := loadLocation(tzName)
	if err != nil {
		return 0, fmt.Errorf("CreateBooking: %w", err)
	}

	startTime := time.Date(day.Year(), day.Month(), day.Day(), times[0].Hour(), times[0].Minute(), 0, 0, loc)
	endTime := startTime.Add(time.Duration(len(slots)) * time.Hour)

	if !startTime.After(now) {
		return 0, ErrInvalidArgument
	}

	var bookingID int
	err = r.q.QueryRow(ctx, queryCreateBookingCTE, userID, roomID, startTime, endTime, len(slots)).Scan(&bookingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.ConstraintName == "bookings_no_overlap" {
			return 0, ErrConflict
		}
		return 0, err
	}
	return bookingID, nil
}

func (r *BookingRepo) GetUserBookings(ctx context.Context, userID int, now time.Time) ([]models.BookingHistoryItem, error) {
	if userID <= 0 {
		return nil, ErrInvalidID
	}
	now = coalesceTime(now)

	rows, err := r.q.Query(ctx, queryGetUserBookings, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bookings := make([]models.BookingHistoryItem, 0)
	for rows.Next() {
		var b models.BookingHistoryItem
		var start, end time.Time
		var tzName string
		if err := rows.Scan(&b.ID, &b.RoomID, &b.ImageURL, &b.Title,
			&start, &end, &b.TotalPrice, &b.Status, &tzName); err != nil {
			return nil, err
		}

		loc, err := loadLocation(tzName)
		if err != nil {
			return nil, fmt.Errorf("GetUserBookings: booking %d: %w", b.ID, err)
		}

		b.Date = start.In(loc).Format("2006-01-02")
		b.StartTime = start.In(loc).Format("15:04")
		b.EndTime = end.In(loc).Format("15:04")
		b.Status = resolveStatus(b.Status, start, end, now)
		bookings = append(bookings, b)
	}
	return bookings, rows.Err()
}

func resolveStatus(baseStatus string, startTime, endTime, now time.Time) string {
	if baseStatus == "canceled" {
		return "canceled"
	}
	switch {
	case now.Before(startTime):
		return "booked"
	case now.After(endTime):
		return "finished"
	default:
		return "in_use"
	}
}

func (r *BookingRepo) CancelBooking(ctx context.Context, bookingID, userID int, now time.Time) error {
	if bookingID <= 0 || userID <= 0 {
		return ErrInvalidID
	}

	now = coalesceTime(now)

	var status string
	var startTime, endTime time.Time
	err := r.q.QueryRow(ctx, queryGetBookingForCancel, bookingID, userID).
		Scan(&status, &startTime, &endTime)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	if resolved := resolveStatus(status, startTime, endTime, now); resolved != "booked" {
		return ErrConflict
	}

	_, err = r.q.Exec(ctx, queryCancelBooking, bookingID, userID)
	return err
}
