package rooms

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nikolaarden2000/software-design-project/backend/db"
)

const (
	queryGetRoomsBatchByCity = `
		SELECT r.id, r.title, c.name, CONCAT(l.city, ', ', l.street, ', ', l.house_number), r.capacity, r.images[1], r.price
		FROM rooms r
		JOIN locations l ON r.location_id = l.id
		JOIN companies c ON l.company_id = c.id
		WHERE r.id > $1
		  AND l.city = $2
		  AND r.status = 'published'
		ORDER BY r.id ASC
		LIMIT $3`

	queryGetCompaniesByCity = `
		SELECT c.name
		FROM companies c
		WHERE EXISTS (
			SELECT 1
			FROM locations l
			JOIN rooms r ON r.location_id = l.id
			WHERE l.company_id = c.id
			  AND l.city = $1
			  AND r.status = 'published'
		)`

	queryGetRoomPageData = `
		SELECT r.id,
		  r.title,
		  c.name AS company,
		  (l.city || ', ' || l.street || ' ' || l.house_number) AS address,
		  r.images,
		  r.price,
		  r.capacity,
		  r.available_from,
		  r.available_to,
		  COALESCE(r.description, '') AS description,
		  l.latitude,
		  l.longitude
		FROM rooms r
		JOIN locations l ON r.location_id = l.id
		JOIN companies c ON l.company_id = c.id
		WHERE r.id = $1
		  AND r.status = 'published'`

	queryListAdminRooms = `
		SELECT
			r.id,
			r.location_id,
			r.title,
			r.price,
			r.capacity,
			r.status::text,
			r.rejection_reason,
			r.created_at
		FROM rooms r
		WHERE (
			$1::bool
			OR EXISTS (
				SELECT 1
				FROM admin_locations al
				WHERE al.location_id = r.location_id
				  AND al.admin_id = $2
			)
		)
		  AND ($3::int IS NULL OR r.location_id = $3)
		  AND ($4::room_status IS NULL OR r.status = $4::room_status)
		ORDER BY r.id DESC`

	queryGetAdminRoom = `
		SELECT
			r.id,
			r.location_id,
			r.title,
			COALESCE(r.description, ''),
			r.price,
			r.capacity,
			r.available_from,
			r.available_to,
			r.images,
			r.status::text,
			r.rejection_reason
		FROM rooms r
		WHERE r.id = $1`

	queryCreateAdminRoom = `
		INSERT INTO rooms (
			location_id,
			title,
			description,
			price,
			capacity,
			available_from,
			available_to,
			images,
			status,
			created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'draft', $9)
		RETURNING id, location_id, title, price, capacity, status::text, rejection_reason, created_at`

	queryUpdateAdminRoom = `
		UPDATE rooms
		SET
			title = $2,
			description = $3,
			price = $4,
			capacity = $5,
			available_from = $6,
			available_to = $7,
			images = $8,
			status = 'draft',
			rejection_reason = NULL,
			updated_at = now()
		WHERE id = $1
		  AND status IN ('draft', 'rejected')`

	querySubmitAdminRoom = `
		UPDATE rooms
		SET
			status = 'pending',
			rejection_reason = NULL,
			updated_at = now()
		WHERE id = $1
		  AND status IN ('draft', 'rejected')`
)

type Repository struct {
	q db.Querier
}

func NewRepository(q db.Querier) *Repository {
	return &Repository{q: q}
}

func (r *Repository) GetRoomsBatchByCity(ctx context.Context, lastID, limit int, city string) ([]Room, error) {
	if lastID < 0 {
		return nil, db.ErrInvalidID
	}
	if limit <= 0 {
		return nil, db.ErrInvalidArgument
	}

	rows, err := r.q.Query(ctx, queryGetRoomsBatchByCity, lastID, city, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]Room, 0)
	for rows.Next() {
		var room Room
		err := rows.Scan(
			&room.ID,
			&room.Title,
			&room.Company,
			&room.Address,
			&room.Capacity,
			&room.ImageURL,
			&room.Price,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, room)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Repository) GetCompaniesByCity(ctx context.Context, city string) ([]string, error) {
	rows, err := r.q.Query(ctx, queryGetCompaniesByCity, city)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	companies := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		companies = append(companies, name)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return companies, nil
}

func (r *Repository) GetRoomPageData(ctx context.Context, roomID int) (*RoomPageData, error) {
	if roomID <= 0 {
		return nil, db.ErrInvalidID
	}

	var d RoomPageData
	var availFrom, availTo time.Time

	err := r.q.QueryRow(ctx, queryGetRoomPageData, roomID).Scan(
		&d.ID,
		&d.Title,
		&d.Company,
		&d.Address,
		&d.Images,
		&d.Price,
		&d.Capacity,
		&availFrom,
		&availTo,
		&d.Description,
		&d.Lat,
		&d.Lng,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, db.ErrNotFound
		}
		return nil, fmt.Errorf("get room page data: %w", err)
	}

	d.AvailableFrom = availFrom.Format("15:04")
	d.AvailableTo = availTo.Format("15:04")
	d.MaxCapacity = d.Capacity
	d.Currency = "RUB"

	return &d, nil
}

func (r *Repository) ListAdminRooms(ctx context.Context, adminID int, includeAll bool, locationID *int, status *string) ([]AdminRoomListItem, error) {
	if adminID <= 0 && !includeAll {
		return nil, db.ErrInvalidID
	}

	if status != nil && !IsValidStatus(*status) {
		return nil, db.ErrInvalidArgument
	}

	if locationID != nil {
		if err := r.checkLocationAccess(ctx, *locationID, adminID, includeAll); err != nil {
			return nil, err
		}
	}

	var locationParam any
	if locationID != nil {
		locationParam = *locationID
	}

	var statusParam any
	if status != nil {
		statusParam = *status
	}

	rows, err := r.q.Query(ctx, queryListAdminRooms, includeAll, adminID, locationParam, statusParam)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]AdminRoomListItem, 0)

	for rows.Next() {
		item, err := scanAdminRoomListItem(rows)
		if err != nil {
			return nil, err
		}

		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Repository) GetAdminRoom(ctx context.Context, adminID int, includeAll bool, roomID int) (*AdminRoomDetails, error) {
	if roomID <= 0 {
		return nil, db.ErrInvalidID
	}

	if err := r.checkRoomAccess(ctx, roomID, adminID, includeAll); err != nil {
		return nil, err
	}

	var room AdminRoomDetails
	var availableFrom, availableTo time.Time
	var rejectionReason sql.NullString

	err := r.q.QueryRow(ctx, queryGetAdminRoom, roomID).Scan(
		&room.ID,
		&room.LocationID,
		&room.Title,
		&room.Description,
		&room.Price,
		&room.Capacity,
		&availableFrom,
		&availableTo,
		&room.Images,
		&room.Status,
		&rejectionReason,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, db.ErrNotFound
		}
		return nil, err
	}

	room.AvailableFrom = availableFrom.Format("15:04")
	room.AvailableTo = availableTo.Format("15:04")
	room.RejectionReason = stringPtrFromNull(rejectionReason)

	return &room, nil
}

func (r *Repository) CreateAdminRoom(ctx context.Context, creatorID int, includeAll bool, input AdminRoomInput) (*AdminRoomListItem, error) {
	if creatorID <= 0 {
		return nil, db.ErrInvalidID
	}

	normalized, err := normalizeAdminRoomInput(input)
	if err != nil {
		return nil, err
	}

	if err := r.checkLocationAccess(ctx, normalized.LocationID, creatorID, includeAll); err != nil {
		return nil, err
	}

	row := r.q.QueryRow(
		ctx,
		queryCreateAdminRoom,
		normalized.LocationID,
		normalized.Title,
		normalized.Description,
		normalized.Price,
		normalized.Capacity,
		normalized.AvailableFrom,
		normalized.AvailableTo,
		normalized.Images,
		creatorID,
	)

	item, err := scanAdminRoomListItem(row)
	if err != nil {
		return nil, err
	}

	return &item, nil
}

func (r *Repository) UpdateAdminRoom(ctx context.Context, adminID int, includeAll bool, roomID int, input AdminRoomInput) error {
	if roomID <= 0 {
		return db.ErrInvalidID
	}

	normalized, err := normalizeAdminRoomInput(input)
	if err != nil {
		return err
	}

	if err := r.checkRoomAccess(ctx, roomID, adminID, includeAll); err != nil {
		return err
	}

	tag, err := r.q.Exec(
		ctx,
		queryUpdateAdminRoom,
		roomID,
		normalized.Title,
		normalized.Description,
		normalized.Price,
		normalized.Capacity,
		normalized.AvailableFrom,
		normalized.AvailableTo,
		normalized.Images,
	)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return db.ErrConflict
	}

	return nil
}

func (r *Repository) SubmitAdminRoom(ctx context.Context, adminID int, includeAll bool, roomID int) error {
	if roomID <= 0 {
		return db.ErrInvalidID
	}

	if err := r.checkRoomAccess(ctx, roomID, adminID, includeAll); err != nil {
		return err
	}

	tag, err := r.q.Exec(ctx, querySubmitAdminRoom, roomID)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return db.ErrConflict
	}

	return nil
}

func (r *Repository) checkLocationAccess(ctx context.Context, locationID, adminID int, includeAll bool) error {
	if locationID <= 0 {
		return db.ErrInvalidID
	}

	var exists bool
	var accessible bool

	err := r.q.QueryRow(ctx, `
		SELECT
			EXISTS (
				SELECT 1
				FROM locations
				WHERE id = $1
			) AS exists,
			EXISTS (
				SELECT 1
				FROM locations l
				WHERE l.id = $1
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
	`, locationID, includeAll, adminID).Scan(&exists, &accessible)
	if err != nil {
		return err
	}

	if !exists {
		return db.ErrNotFound
	}

	if !accessible {
		return db.ErrForbidden
	}

	return nil
}

func (r *Repository) checkRoomAccess(ctx context.Context, roomID, adminID int, includeAll bool) error {
	if roomID <= 0 {
		return db.ErrInvalidID
	}

	var exists bool
	var accessible bool

	err := r.q.QueryRow(ctx, `
		SELECT
			EXISTS (
				SELECT 1
				FROM rooms
				WHERE id = $1
			) AS exists,
			EXISTS (
				SELECT 1
				FROM rooms room
				WHERE room.id = $1
				  AND (
				  	$2::bool
				  	OR EXISTS (
				  		SELECT 1
				  		FROM admin_locations al
				  		WHERE al.location_id = room.location_id
				  		  AND al.admin_id = $3
				  	)
				  )
			) AS accessible
	`, roomID, includeAll, adminID).Scan(&exists, &accessible)
	if err != nil {
		return err
	}

	if !exists {
		return db.ErrNotFound
	}

	if !accessible {
		return db.ErrForbidden
	}

	return nil
}

func normalizeAdminRoomInput(input AdminRoomInput) (AdminRoomInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.AvailableFrom = strings.TrimSpace(input.AvailableFrom)
	input.AvailableTo = strings.TrimSpace(input.AvailableTo)

	if input.LocationID <= 0 {
		return AdminRoomInput{}, db.ErrInvalidID
	}

	if input.Title == "" || input.Description == "" {
		return AdminRoomInput{}, db.ErrInvalidArgument
	}

	if input.Price <= 0 || input.Capacity <= 0 {
		return AdminRoomInput{}, db.ErrInvalidArgument
	}

	availableFrom, err := time.Parse("15:04", input.AvailableFrom)
	if err != nil {
		return AdminRoomInput{}, db.ErrInvalidArgument
	}

	availableTo, err := time.Parse("15:04", input.AvailableTo)
	if err != nil {
		return AdminRoomInput{}, db.ErrInvalidArgument
	}

	if !availableFrom.Before(availableTo) {
		return AdminRoomInput{}, db.ErrInvalidArgument
	}

	if len(input.Images) > 5 {
		return AdminRoomInput{}, db.ErrInvalidArgument
	}

	if input.Images == nil {
		input.Images = []string{}
	}

	for i := range input.Images {
		input.Images[i] = strings.TrimSpace(input.Images[i])
	}

	return input, nil
}

func scanAdminRoomListItem(row pgx.Row) (AdminRoomListItem, error) {
	var item AdminRoomListItem
	var rejectionReason sql.NullString
	var createdAt time.Time

	err := row.Scan(
		&item.ID,
		&item.LocationID,
		&item.Title,
		&item.Price,
		&item.Capacity,
		&item.Status,
		&rejectionReason,
		&createdAt,
	)
	if err != nil {
		return AdminRoomListItem{}, err
	}

	item.RejectionReason = stringPtrFromNull(rejectionReason)
	item.CreatedAt = createdAt.UTC().Format(time.RFC3339)

	return item, nil
}

func stringPtrFromNull(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}

	result := value.String
	return &result
}
