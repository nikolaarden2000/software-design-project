package users

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/nikolaarden2000/software-design-project/backend/db"
	pgxmock "github.com/pashagolub/pgxmock/v2"
)

func newMock(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unfulfilled mock expectations: %v", err)
		}
	})
	return mock
}

func newUserRepo(mock pgxmock.PgxPoolIface) *Repository {
	return NewRepository(mock)
}

// CreateUser
func TestCreateUser_Success(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta(queryCreateUser)).
		WithArgs("alice", "alice@example.com", "hashed-pwd").
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(123))

	id, err := newUserRepo(mock).CreateUser(context.Background(), "alice", "alice@example.com", "hashed-pwd")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 123 {
		t.Errorf("id: got %d, want 123", id)
	}
}

func TestCreateUser_DuplicateEmail_ReturnsErrEmailTaken(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery(regexp.QuoteMeta(queryCreateUser)).
		WithArgs("bob", "taken@example.com", "hashed-pwd").
		WillReturnRows(pgxmock.NewRows([]string{"id"}))

	_, err := newUserRepo(mock).CreateUser(context.Background(), "bob", "taken@example.com", "hashed-pwd")

	if !errors.Is(err, db.ErrEmailTaken) {
		t.Fatalf("expected db.ErrEmailTaken, got: %v", err)
	}
}

func TestCreateUser_DBError_PropagatesError(t *testing.T) {
	mock := newMock(t)
	dbErr := fmt.Errorf("connection refused")
	mock.ExpectQuery(regexp.QuoteMeta(queryCreateUser)).
		WithArgs("carol", "carol@example.com", "hashed-pwd").
		WillReturnError(dbErr)

	_, err := newUserRepo(mock).CreateUser(context.Background(), "carol", "carol@example.com", "hashed-pwd")

	if !errors.Is(err, dbErr) {
		t.Fatalf("expected wrapped dbErr, got: %v", err)
	}
}

// GetUserByEmail
func TestGetUserByEmail_Success(t *testing.T) {
	mock := newMock(t)
	cols := []string{"id", "name", "email", "password_hash", "role"}
	mock.ExpectQuery(regexp.QuoteMeta(queryGetUserByEmail)).
		WithArgs("alice@example.com").
		WillReturnRows(pgxmock.NewRows(cols).AddRow(1, "alice", "alice@example.com", "hashed-pwd", "client"))

	u, err := newUserRepo(mock).GetUserByEmail(context.Background(), "alice@example.com")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != 1 || u.Username != "alice" || u.Email != "alice@example.com" || u.Password != "hashed-pwd" || u.Role != "client" {
		t.Errorf("unexpected user fields: %+v", u)
	}
}

func TestGetUserByEmail_NotFound_ReturnsErrNotFound(t *testing.T) {
	mock := newMock(t)
	cols := []string{"id", "name", "email", "password_hash", "role"}
	mock.ExpectQuery(regexp.QuoteMeta(queryGetUserByEmail)).
		WithArgs("ghost@example.com").
		WillReturnRows(pgxmock.NewRows(cols))

	_, err := newUserRepo(mock).GetUserByEmail(context.Background(), "ghost@example.com")

	if !errors.Is(err, db.ErrNotFound) {
		t.Fatalf("expected db.ErrNotFound, got: %v", err)
	}
}

func TestGetUserByEmail_DBError_PropagatesError(t *testing.T) {
	mock := newMock(t)
	dbErr := fmt.Errorf("timeout")
	mock.ExpectQuery(regexp.QuoteMeta(queryGetUserByEmail)).
		WithArgs("alice@example.com").
		WillReturnError(dbErr)

	_, err := newUserRepo(mock).GetUserByEmail(context.Background(), "alice@example.com")

	if !errors.Is(err, dbErr) {
		t.Fatalf("expected wrapped dbErr, got: %v", err)
	}
}

// GetUserByID

func TestGetUserByID_Success(t *testing.T) {
	mock := newMock(t)
	cols := []string{"id", "name", "email", "password_hash", "role"}
	mock.ExpectQuery(regexp.QuoteMeta(queryGetUserByID)).
		WithArgs(7).
		WillReturnRows(pgxmock.NewRows(cols).AddRow(7, "dave", "dave@example.com", "hashed-pwd", "client"))

	u, err := newUserRepo(mock).GetUserByID(context.Background(), 7)
	t.Log(u)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != 7 || u.Username != "dave" || u.Email != "dave@example.com" || u.Password != "hashed-pwd" || u.Role != "client" {
		t.Errorf("unexpected user fields: %+v", u)
	}
}

func TestGetUserByID_NotFound_ReturnsErrNotFound(t *testing.T) {
	mock := newMock(t)
	cols := []string{"id", "name", "email", "password_hash", "role"}
	mock.ExpectQuery(regexp.QuoteMeta(queryGetUserByID)).
		WithArgs(123).
		WillReturnRows(pgxmock.NewRows(cols))

	_, err := newUserRepo(mock).GetUserByID(context.Background(), 123)

	if !errors.Is(err, db.ErrNotFound) {
		t.Fatalf("expected db.ErrNotFound, got: %v", err)
	}
}

func TestGetUserByID_DBError_PropagatesError(t *testing.T) {
	mock := newMock(t)
	dbErr := fmt.Errorf("connection reset")
	mock.ExpectQuery(regexp.QuoteMeta(queryGetUserByID)).
		WithArgs(7).
		WillReturnError(dbErr)

	_, err := newUserRepo(mock).GetUserByID(context.Background(), 7)

	if !errors.Is(err, dbErr) {
		t.Fatalf("expected wrapped dbErr, got: %v", err)
	}
}

func TestGetUserByID_BoundaryIDs_ReturnErrInvalidID(t *testing.T) {
	cases := []struct {
		name string
		id   int
	}{
		{"zero id", 0},
		{"negative id", -1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newMock(t)

			_, err := newUserRepo(mock).GetUserByID(context.Background(), tc.id)

			if !errors.Is(err, db.ErrInvalidID) {
				t.Errorf("[%s] expected db.ErrInvalidID, got: %v", tc.name, err)
			}
		})
	}
}

// IsEmailTaken

func TestIsEmailTaken_DecisionTable(t *testing.T) {
	cases := []struct {
		name     string
		dbResult bool
		want     bool
	}{
		{"free email returns false", false, false},
		{"taken email returns true", true, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newMock(t)
			mock.ExpectQuery(regexp.QuoteMeta(queryIsEmailTaken)).
				WithArgs("test@example.com").
				WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(tc.dbResult))

			got, err := newUserRepo(mock).IsEmailTaken(context.Background(), "test@example.com")

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsEmailTaken_DBError_PropagatesError(t *testing.T) {
	mock := newMock(t)
	dbErr := fmt.Errorf("query timeout")
	mock.ExpectQuery(regexp.QuoteMeta(queryIsEmailTaken)).
		WithArgs("test@example.com").
		WillReturnError(dbErr)

	got, err := newUserRepo(mock).IsEmailTaken(context.Background(), "test@example.com")

	if got {
		t.Error("expected false on error, got true")
	}
	if !errors.Is(err, dbErr) {
		t.Fatalf("expected wrapped dbErr, got: %v", err)
	}
}
