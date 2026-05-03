package companies

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"gitlab.com/5130904-20104-teams/software-design-project/backend/db"
)

const (
	queryListCompanies = `
		SELECT
			c.id,
			c.name,
			c.description,
			COUNT(l.id)::int AS locations_count
		FROM companies c
		LEFT JOIN locations l ON l.company_id = c.id
		GROUP BY c.id, c.name, c.description
		ORDER BY c.id ASC`

	queryCreateCompany = `
		INSERT INTO companies (name, description)
		VALUES ($1, $2)
		RETURNING id, name, description, 0::int AS locations_count`
)

type Repository struct {
	q db.Querier
}

func NewRepository(q db.Querier) *Repository {
	return &Repository{q: q}
}

func (r *Repository) ListCompanies(ctx context.Context) ([]Company, error) {
	rows, err := r.q.Query(ctx, queryListCompanies)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]Company, 0)

	for rows.Next() {
		var company Company

		if err := rows.Scan(
			&company.ID,
			&company.Name,
			&company.Description,
			&company.LocationsCount,
		); err != nil {
			return nil, err
		}

		result = append(result, company)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Repository) CreateCompany(ctx context.Context, name, description string) (*Company, error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)

	if name == "" {
		return nil, db.ErrInvalidArgument
	}

	var company Company

	err := r.q.QueryRow(ctx, queryCreateCompany, name, description).Scan(
		&company.ID,
		&company.Name,
		&company.Description,
		&company.LocationsCount,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, db.ErrConflict
		}

		return nil, err
	}

	return &company, nil
}

func (r *Repository) ExistsByID(ctx context.Context, id int) (bool, error) {
	if id <= 0 {
		return false, db.ErrInvalidID
	}

	var exists bool
	err := r.q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM companies WHERE id = $1)`, id).Scan(&exists)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}

		return false, err
	}

	return exists, nil
}
