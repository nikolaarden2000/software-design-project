package users

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/nikolaarden2000/software-design-project/backend/db"
)

const (
	queryCreateUser = `
		INSERT INTO users (name, email, password_hash)
		VALUES ($1, $2, $3)
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
)

type Repository struct {
	q db.Querier
}

func NewRepository(q db.Querier) *Repository {
	return &Repository{q: q}
}

func (r *Repository) CreateUser(ctx context.Context, username, email, hashedPassword string) (int, error) {
	var id int
	err := r.q.QueryRow(ctx, queryCreateUser, username, email, hashedPassword).Scan(&id)
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
