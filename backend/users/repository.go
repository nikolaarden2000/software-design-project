package users

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/nikolaarden2000/software-design-project/backend/db"
)

const (
	queryCreateUserWithRole = `
		INSERT INTO users (name, email, password_hash, role)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (email) DO NOTHING
		RETURNING id`

	queryGetUserByEmail = `
		SELECT id, name, email, password_hash, role
		FROM users
		WHERE email = $1`

	queryGetUserByID = `
		SELECT id, name, email, password_hash, role
		FROM users
		WHERE id = $1`

	queryIsEmailTaken = `
		SELECT EXISTS (SELECT 1 FROM users WHERE email = $1)`

	queryListAdmins = `
		SELECT id, name, email, role
		FROM users
		WHERE role = 'admin'
		ORDER BY id ASC`

	queryListAdminLocations = `
		SELECT
			l.id,
			(l.city || ', ' || l.street || ' ' || l.house_number) AS address,
			c.name AS company_name
		FROM admin_locations al
		JOIN locations l ON l.id = al.location_id
		JOIN companies c ON c.id = l.company_id
		WHERE al.admin_id = $1
		ORDER BY l.id ASC`

	queryAssignAdminToLocation = `
		INSERT INTO admin_locations (admin_id, location_id)
		VALUES ($1, $2)`

	queryDeleteAdminLocationAssignment = `
		DELETE FROM admin_locations
		WHERE admin_id = $1 AND location_id = $2`
)

type Repository struct {
	q db.Querier
}

func NewRepository(q db.Querier) *Repository {
	return &Repository{q: q}
}

func (r *Repository) CreateUserWithRole(ctx context.Context, username, email, hashedPassword, role string) (int, error) {
	if !IsValidRole(role) {
		return 0, db.ErrInvalidArgument
	}

	var id int
	err := r.q.QueryRow(ctx, queryCreateUserWithRole, username, email, hashedPassword, role).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, db.ErrEmailTaken
		}
		return 0, err
	}

	return id, nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	row := r.q.QueryRow(ctx, queryGetUserByEmail, email)
	return scanUser(row)
}

func (r *Repository) GetUserByID(ctx context.Context, id int) (*User, error) {
	if id <= 0 {
		return nil, db.ErrInvalidID
	}

	row := r.q.QueryRow(ctx, queryGetUserByID, id)
	return scanUser(row)
}

func (r *Repository) IsEmailTaken(ctx context.Context, email string) (bool, error) {
	var exists bool

	if err := r.q.QueryRow(ctx, queryIsEmailTaken, email).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}

func scanUser(row pgx.Row) (*User, error) {
	var u User

	if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, db.ErrNotFound
		}
		return nil, err
	}

	return &u, nil
}

func (r *Repository) ListAdmins(ctx context.Context) ([]Admin, error) {
	rows, err := r.q.Query(ctx, queryListAdmins)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	admins := make([]Admin, 0)

	for rows.Next() {
		var admin Admin

		if err := rows.Scan(
			&admin.ID,
			&admin.Username,
			&admin.Email,
			&admin.Role,
		); err != nil {
			return nil, err
		}

		locations, err := r.ListAdminLocations(ctx, admin.ID)
		if err != nil {
			return nil, err
		}

		admin.Locations = locations
		admins = append(admins, admin)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return admins, nil
}

func (r *Repository) ListAdminLocations(ctx context.Context, adminID int) ([]AdminLocation, error) {
	if adminID <= 0 {
		return nil, db.ErrInvalidID
	}

	rows, err := r.q.Query(ctx, queryListAdminLocations, adminID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	locations := make([]AdminLocation, 0)

	for rows.Next() {
		var location AdminLocation

		if err := rows.Scan(
			&location.ID,
			&location.Address,
			&location.CompanyName,
		); err != nil {
			return nil, err
		}

		locations = append(locations, location)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return locations, nil
}

func (r *Repository) AssignAdminToLocation(ctx context.Context, adminID, locationID int) error {
	if adminID <= 0 || locationID <= 0 {
		return db.ErrInvalidID
	}

	admin, err := r.GetUserByID(ctx, adminID)
	if err != nil {
		return err
	}

	if admin.Role != RoleAdmin {
		return db.ErrNotFound
	}

	_, err = r.q.Exec(ctx, queryAssignAdminToLocation, adminID, locationID)
	if err != nil {
		return mapAssignmentError(err)
	}

	return nil
}

func (r *Repository) DeleteAdminLocationAssignment(ctx context.Context, adminID, locationID int) error {
	if adminID <= 0 || locationID <= 0 {
		return db.ErrInvalidID
	}

	tag, err := r.q.Exec(ctx, queryDeleteAdminLocationAssignment, adminID, locationID)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return db.ErrNotFound
	}

	return nil
}

func mapAssignmentError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case "23505":
		return db.ErrConflict
	case "23503":
		return db.ErrNotFound
	default:
		return err
	}
}
