package bookings

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/nikolaarden2000/software-design-project/backend/db"
	"github.com/nikolaarden2000/software-design-project/backend/rooms"
)

const (
	queryGetRoomWindow = `
		SELECT r.available_from, r.available_to, l.timezone
		FROM rooms r
		JOIN locations l ON r.location_id = l.id
		WHERE r.id = $1
			AND r.status = 'published'
			AND r.booking_disabled = false`

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
		WHERE r.id = $1
			AND r.status = 'published'
			AND r.booking_disabled = false`

	queryCreateBookingCTE = `
		WITH room_price AS (
			SELECT price
			FROM rooms
			WHERE id = $2
				AND status = 'published'
				AND booking_disabled = false
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

	queryListAdminBookings = `
		SELECT
			b.id,
			r.id AS room_id,
			r.title AS room_title,
			l.id AS location_id,
			(l.city || ', ' || l.street || ' ' || l.house_number) AS location_address,
			u.id AS user_id,
			u.email AS user_email,
			u.name AS user_username,
			b.start_time,
			b.end_time,
			b.total_price,
			b.status::text,
			l.timezone
		FROM bookings b
		JOIN rooms r ON r.id = b.room_id
		JOIN locations l ON l.id = r.location_id
		JOIN users u ON u.id = b.user_id
		WHERE (
			$1::bool
			OR EXISTS (
				SELECT 1
				FROM admin_locations al
				WHERE al.location_id = l.id
				  AND al.admin_id = $2
			)
		)
		  AND ($3::int IS NULL OR l.id = $3)
		  AND ($4::text IS NULL OR b.status::text = $4)
		ORDER BY b.id DESC`

	queryGetAdminBookingForCancel = `
		SELECT
			b.status::text,
			b.start_time,
			b.end_time,
			EXISTS (
				SELECT 1
				FROM bookings b2
				JOIN rooms r ON r.id = b2.room_id
				JOIN locations l ON l.id = r.location_id
				WHERE b2.id = $1
				  AND (
				  	$2::bool
				  	OR EXISTS (
				  		SELECT 1
				  		FROM admin_locations al
				  		WHERE al.location_id = l.id
				  		  AND al.admin_id = $3
				  	)
				  )
			) AS accessible
		FROM bookings b
		WHERE b.id = $1`

	queryCancelAdminBooking = `
		UPDATE bookings
		SET status = 'canceled'
		WHERE id = $1`
)

type Repository struct {
	q db.Querier
}

func NewRepository(q db.Querier) *Repository {
	return &Repository{q: q}
}

type booking struct {
	Start time.Time
	End   time.Time
}

func loadLocation(name string) (*time.Location, error) {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %q: %w", name, err)
	}
	return loc, nil
}

func coalesceTime(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now()
	}
	return t
}

func generateHourlySlots(winStart, winEnd time.Time) []time.Time {
	var slots []time.Time
	for t := winStart; !t.Add(time.Hour).After(winEnd); t = t.Add(time.Hour) {
		slots = append(slots, t)
	}
	return slots
}

func (r *Repository) GetRoomAvailability(ctx context.Context, roomID, days int, now time.Time) ([]rooms.DateAvailability, error) {
	if roomID <= 0 {
		return nil, db.ErrInvalidID
	}
	if days <= 0 {
		days = 7
	}

	now = coalesceTime(now)

	var availFrom, availTo time.Time
	var tzName string
	if err := r.q.QueryRow(ctx, queryGetRoomWindow, roomID).Scan(&availFrom, &availTo, &tzName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, db.ErrNotFound
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

	existingBookings := make([]booking, 0)
	for rows.Next() {
		var start, end time.Time
		if err := rows.Scan(&start, &end); err != nil {
			return nil, fmt.Errorf("GetRoomAvailability: scan booking: %w", err)
		}
		existingBookings = append(existingBookings, booking{
			Start: start,
			End:   end,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetRoomAvailability: iterate bookings: %w", err)
	}

	sort.Slice(existingBookings, func(i, j int) bool {
		return existingBookings[i].Start.Before(existingBookings[j].Start)
	})

	results := make([]rooms.DateAvailability, 0, days)
	for d := 0; d < days; d++ {
		day := startDate.AddDate(0, 0, d)

		winStart := time.Date(day.Year(), day.Month(), day.Day(), availFrom.Hour(), availFrom.Minute(), availFrom.Second(), 0, loc)
		winEnd := time.Date(day.Year(), day.Month(), day.Day(), availTo.Hour(), availTo.Minute(), availTo.Second(), 0, loc)

		availableTimes := make([]string, 0)
		for _, slot := range generateHourlySlots(winStart, winEnd) {
			if !slot.After(now) {
				continue
			}

			slotEnd := slot.Add(time.Hour)
			overlaps := false

			for _, b := range existingBookings {
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

		results = append(results, rooms.DateAvailability{
			Date:           day.Format("2006-01-02"),
			AvailableTimes: availableTimes,
		})
	}

	return results, nil
}

func (r *Repository) CreateBooking(ctx context.Context, userID, roomID int, date string, slots []string, now time.Time) (int, error) {
	if userID <= 0 || roomID <= 0 {
		return 0, db.ErrInvalidID
	}

	now = coalesceTime(now)

	if len(slots) == 0 {
		return 0, db.ErrInvalidArgument
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
			return 0, db.ErrNotConsecutiveSlots
		}
	}

	day, err := time.Parse("2006-01-02", date)
	if err != nil {
		return 0, fmt.Errorf("invalid date: %w", err)
	}

	var tzName string
	if err := r.q.QueryRow(ctx, queryGetRoomTimezone, roomID).Scan(&tzName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, db.ErrNotFound
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
		return 0, db.ErrInvalidArgument
	}

	var bookingID int
	err = r.q.QueryRow(ctx, queryCreateBookingCTE, userID, roomID, startTime, endTime, len(slots)).Scan(&bookingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, db.ErrNotFound
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.ConstraintName == "bookings_no_overlap" {
			return 0, db.ErrConflict
		}

		return 0, err
	}

	return bookingID, nil
}

func (r *Repository) GetUserBookings(ctx context.Context, userID int, now time.Time) ([]BookingHistoryItem, error) {
	if userID <= 0 {
		return nil, db.ErrInvalidID
	}

	now = coalesceTime(now)

	rows, err := r.q.Query(ctx, queryGetUserBookings, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]BookingHistoryItem, 0)
	for rows.Next() {
		var b BookingHistoryItem
		var start, end time.Time
		var tzName string

		if err := rows.Scan(
			&b.ID,
			&b.RoomID,
			&b.ImageURL,
			&b.Title,
			&start,
			&end,
			&b.TotalPrice,
			&b.Status,
			&tzName,
		); err != nil {
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

		result = append(result, b)
	}

	return result, rows.Err()
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

func (r *Repository) CancelBooking(ctx context.Context, bookingID, userID int, now time.Time) error {
	if bookingID <= 0 || userID <= 0 {
		return db.ErrInvalidID
	}

	now = coalesceTime(now)

	var status string
	var startTime, endTime time.Time
	err := r.q.QueryRow(ctx, queryGetBookingForCancel, bookingID, userID).
		Scan(&status, &startTime, &endTime)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.ErrNotFound
		}
		return err
	}

	if resolved := resolveStatus(status, startTime, endTime, now); resolved != "booked" {
		return db.ErrConflict
	}

	_, err = r.q.Exec(ctx, queryCancelBooking, bookingID, userID)
	return err
}

func (r *Repository) ListAdminBookings(
	ctx context.Context,
	adminID int,
	includeAll bool,
	locationID *int,
	status *string,
	now time.Time,
) ([]AdminBookingItem, error) {
	if adminID <= 0 && !includeAll {
		return nil, db.ErrInvalidID
	}

	now = coalesceTime(now)

	var locationParam any
	if locationID != nil {
		if *locationID <= 0 {
			return nil, db.ErrInvalidID
		}
		locationParam = *locationID
	}

	var statusParam any
	if status != nil {
		switch *status {
		case "booked", "canceled", "in_use", "finished":
			statusParam = dbStatusForFilter(*status)
		default:
			return nil, db.ErrInvalidArgument
		}
	}

	rows, err := r.q.Query(ctx, queryListAdminBookings, includeAll, adminID, locationParam, statusParam)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]AdminBookingItem, 0)

	for rows.Next() {
		var item AdminBookingItem
		var start, end time.Time
		var baseStatus string
		var tzName string

		if err := rows.Scan(
			&item.ID,
			&item.RoomID,
			&item.RoomTitle,
			&item.LocationID,
			&item.LocationAddress,
			&item.UserID,
			&item.UserEmail,
			&item.UserUsername,
			&start,
			&end,
			&item.TotalPrice,
			&baseStatus,
			&tzName,
		); err != nil {
			return nil, err
		}

		loc, err := loadLocation(tzName)
		if err != nil {
			return nil, fmt.Errorf("ListAdminBookings: booking %d: %w", item.ID, err)
		}

		item.Date = start.In(loc).Format("2006-01-02")
		item.StartTime = start.In(loc).Format("15:04")
		item.EndTime = end.In(loc).Format("15:04")
		item.Status = resolveStatus(baseStatus, start, end, now)

		if status != nil && item.Status != *status {
			continue
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *Repository) CancelAdminBooking(ctx context.Context, adminID int, includeAll bool, bookingID int, now time.Time) error {
	if bookingID <= 0 {
		return db.ErrInvalidID
	}

	if adminID <= 0 && !includeAll {
		return db.ErrInvalidID
	}

	now = coalesceTime(now)

	var baseStatus string
	var startTime, endTime time.Time
	var accessible bool

	err := r.q.QueryRow(ctx, queryGetAdminBookingForCancel, bookingID, includeAll, adminID).
		Scan(&baseStatus, &startTime, &endTime, &accessible)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.ErrNotFound
		}
		return err
	}

	if !accessible {
		return db.ErrForbidden
	}

	if resolved := resolveStatus(baseStatus, startTime, endTime, now); resolved != "booked" {
		return db.ErrConflict
	}

	tag, err := r.q.Exec(ctx, queryCancelAdminBooking, bookingID)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return db.ErrNotFound
	}

	return nil
}

func dbStatusForFilter(status string) string {
	switch status {
	case "booked", "in_use", "finished":
		return "booked"
	default:
		return status
	}
}
