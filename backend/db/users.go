package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/nikolaarden2000/software-design-project/backend/models"
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

type UserRepo struct {
	q Querier
}

func NewUserRepo(q Querier) *UserRepo {
	return &UserRepo{q: q}
}

func (r *UserRepo) CreateUser(ctx context.Context, username, email, hashedPassword string) (int, error) {
	var id int
	err := r.q.QueryRow(ctx, queryCreateUser, username, email, hashedPassword).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrEmailTaken
		}
		return 0, err
	}
	return id, nil
}

func (r *UserRepo) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	row := r.q.QueryRow(ctx, queryGetUserByEmail, email)
	return scanUser(row)
}

func (r *UserRepo) GetUserByID(ctx context.Context, id int) (*models.User, error) {
	if id <= 0 {
		return nil, ErrInvalidID
	}
	row := r.q.QueryRow(ctx, queryGetUserByID, id)
	return scanUser(row)
}

func scanUser(row pgx.Row) (*models.User, error) {
	var u models.User

	if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) IsEmailTaken(ctx context.Context, email string) (bool, error) {
	var exists bool

	if err := r.q.QueryRow(ctx, queryIsEmailTaken, email).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}
