package rooms

import (
	"context"
	"errors"
	"fmt"
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
		WHERE r.id > $1 AND l.city=$2
		ORDER BY r.id ASC
		LIMIT $3`

	queryGetCompaniesByCity = `
		SELECT c.name
		FROM companies c
		WHERE EXISTS (
			SELECT 1 FROM locations l
			WHERE l.company_id = c.id
			AND l.city = $1
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
		WHERE r.id = $1`
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
