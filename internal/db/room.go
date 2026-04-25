package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"gitlab.com/5130904-20104-teams/software-design-project/internal/models"
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

type RoomRepo struct {
	q Querier
}

func NewRoomRepo(q Querier) *RoomRepo {
	return &RoomRepo{q: q}
}

func (r *RoomRepo) GetRoomsBatchByCity(ctx context.Context, lastID, limit int, city string) ([]models.Room, error) {
	if lastID < 0 {
		return nil, ErrInvalidID
	}
	if limit <= 0 {
		return nil, ErrInvalidArgument
	}

	rows, err := r.q.Query(ctx, queryGetRoomsBatchByCity, lastID, city, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rooms := make([]models.Room, 0)
	for rows.Next() {
		var room models.Room
		err := rows.Scan(&room.ID, &room.Title, &room.Company, &room.Address, &room.Capacity, &room.ImageURL, &room.Price)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return rooms, nil
}

func (r *RoomRepo) GetCompaniesByCity(ctx context.Context, city string) ([]string, error) {
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

func (r *RoomRepo) GetRoomPageData(ctx context.Context, roomID int) (*models.RoomPageData, error) {
	if roomID <= 0 {
		return nil, ErrInvalidID
	}

	var d models.RoomPageData
	var availFrom, availTo time.Time

	err := r.q.QueryRow(ctx, queryGetRoomPageData, roomID).Scan(
		&d.ID, &d.Title, &d.Company, &d.Address,
		&d.Images, &d.Price, &d.Capacity,
		&availFrom, &availTo,
		&d.Description, &d.Lat, &d.Lng,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get room page data: %w", err)
	}
	d.AvailableFrom = availFrom.Format("15:04")
	d.AvailableTo = availTo.Format("15:04")
	d.MaxCapacity = d.Capacity

	return &d, nil
}
