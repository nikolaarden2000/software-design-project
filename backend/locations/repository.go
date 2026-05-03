package locations

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"gitlab.com/5130904-20104-teams/software-design-project/backend/db"
)

const (
	queryListLocations = `
		SELECT
			l.id,
			l.company_id,
			c.name AS company_name,
			l.city,
			(l.city || ', ' || l.street || ' ' || l.house_number) AS address,
			l.latitude,
			l.longitude,
			l.timezone
		FROM locations l
		JOIN companies c ON c.id = l.company_id
		WHERE ($1::int IS NULL OR l.company_id = $1)
		  AND ($2::text IS NULL OR l.city = $2)
		ORDER BY l.id ASC`

	queryCreateLocation = `
		INSERT INTO locations (
			company_id,
			city,
			street,
			house_number,
			latitude,
			longitude,
			timezone
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	queryListAdminLocations = `
		SELECT
			l.id,
			l.company_id,
			c.name AS company_name,
			l.city,
			(l.city || ', ' || l.street || ' ' || l.house_number) AS address,
			l.latitude,
			l.longitude,
			l.timezone,
			COUNT(r.id)::int AS rooms_count
		FROM locations l
		JOIN companies c ON c.id = l.company_id
		LEFT JOIN rooms r ON r.location_id = l.id
		WHERE (
			$1::bool
			OR EXISTS (
				SELECT 1
				FROM admin_locations al
				WHERE al.location_id = l.id
				  AND al.admin_id = $2
			)
		)
		GROUP BY l.id, l.company_id, c.name, l.city, l.street, l.house_number, l.latitude, l.longitude, l.timezone
		ORDER BY l.id ASC`
)

type Repository struct {
	q db.Querier
}

func NewRepository(q db.Querier) *Repository {
	return &Repository{q: q}
}

func (r *Repository) ListLocations(ctx context.Context, companyID *int, city *string) ([]Location, error) {
	rows, err := r.q.Query(ctx, queryListLocations, companyID, city)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]Location, 0)

	for rows.Next() {
		var location Location

		if err := rows.Scan(
			&location.ID,
			&location.CompanyID,
			&location.CompanyName,
			&location.City,
			&location.Address,
			&location.Lat,
			&location.Lng,
			&location.Timezone,
		); err != nil {
			return nil, err
		}

		result = append(result, location)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Repository) ListAdminLocations(ctx context.Context, adminID int, includeAll bool) ([]AdminLocation, error) {
	if adminID <= 0 && !includeAll {
		return nil, db.ErrInvalidID
	}

	rows, err := r.q.Query(ctx, queryListAdminLocations, includeAll, adminID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]AdminLocation, 0)

	for rows.Next() {
		var location AdminLocation

		if err := rows.Scan(
			&location.ID,
			&location.CompanyID,
			&location.CompanyName,
			&location.City,
			&location.Address,
			&location.Lat,
			&location.Lng,
			&location.Timezone,
			&location.RoomsCount,
		); err != nil {
			return nil, err
		}

		result = append(result, location)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Repository) CreateLocation(
	ctx context.Context,
	companyID int,
	city string,
	address string,
	lat float64,
	lng float64,
	timezone string,
) (*Location, error) {
	if companyID <= 0 {
		return nil, db.ErrInvalidID
	}

	city = strings.TrimSpace(city)
	address = strings.TrimSpace(address)
	timezone = strings.TrimSpace(timezone)

	if city == "" || address == "" {
		return nil, db.ErrInvalidArgument
	}

	if timezone == "" {
		timezone = "Europe/Moscow"
	}

	street, houseNumber := splitAddressForStorage(city, address)

	var id int
	err := r.q.QueryRow(
		ctx,
		queryCreateLocation,
		companyID,
		city,
		street,
		houseNumber,
		lat,
		lng,
		timezone,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	location, err := r.GetLocationByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return location, nil
}

func (r *Repository) GetLocationByID(ctx context.Context, id int) (*Location, error) {
	if id <= 0 {
		return nil, db.ErrInvalidID
	}

	rows, err := r.q.Query(ctx, `
		SELECT
			l.id,
			l.company_id,
			c.name AS company_name,
			l.city,
			(l.city || ', ' || l.street || ' ' || l.house_number) AS address,
			l.latitude,
			l.longitude,
			l.timezone
		FROM locations l
		JOIN companies c ON c.id = l.company_id
		WHERE l.id = $1
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, db.ErrNotFound
	}

	var location Location
	if err := rows.Scan(
		&location.ID,
		&location.CompanyID,
		&location.CompanyName,
		&location.City,
		&location.Address,
		&location.Lat,
		&location.Lng,
		&location.Timezone,
	); err != nil {
		return nil, err
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &location, nil
}

func (r *Repository) ExistsByID(ctx context.Context, id int) (bool, error) {
	if id <= 0 {
		return false, db.ErrInvalidID
	}

	var exists bool
	err := r.q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM locations WHERE id = $1)`, id).Scan(&exists)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}

		return false, err
	}

	return exists, nil
}

func splitAddressForStorage(city, address string) (string, string) {
	address = strings.TrimSpace(address)

	prefix := city + ","
	if strings.HasPrefix(address, prefix) {
		address = strings.TrimSpace(strings.TrimPrefix(address, prefix))
	}

	parts := strings.Fields(address)
	if len(parts) == 0 {
		return address, "-"
	}

	if len(parts) == 1 {
		return parts[0], "-"
	}

	houseNumber := parts[len(parts)-1]
	street := strings.Join(parts[:len(parts)-1], " ")

	return street, houseNumber
}
