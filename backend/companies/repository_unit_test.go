package companies

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	pgxmock "github.com/pashagolub/pgxmock/v2"
	"gitlab.com/5130904-20104-teams/software-design-project/backend/db"
)

func newCompaniesMock(t *testing.T) pgxmock.PgxPoolIface {
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

func newCompaniesRepo(mock pgxmock.PgxPoolIface) *Repository {
	return NewRepository(mock)
}

func companyColumns() []string {
	return []string{"id", "name", "description", "locations_count"}
}

func TestListCompanies_EquivalencePartitioning_NonEmptyResult(t *testing.T) {
	mock := newCompaniesMock(t)
	repo := newCompaniesRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(queryListCompanies)).
		WillReturnRows(
			pgxmock.NewRows(companyColumns()).
				AddRow(1, "Company A", "Description A", 2).
				AddRow(2, "Company B", "Description B", 0),
		)

	items, err := repo.ListCompanies(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d companies, want 2", len(items))
	}

	if items[0].ID != 1 || items[0].Name != "Company A" || items[0].Description != "Description A" || items[0].LocationsCount != 2 {
		t.Fatalf("unexpected first company: %+v", items[0])
	}

	if items[1].ID != 2 || items[1].Name != "Company B" || items[1].LocationsCount != 0 {
		t.Fatalf("unexpected second company: %+v", items[1])
	}
}

func TestListCompanies_EquivalencePartitioning_EmptyResult(t *testing.T) {
	mock := newCompaniesMock(t)
	repo := newCompaniesRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(queryListCompanies)).
		WillReturnRows(pgxmock.NewRows(companyColumns()))

	items, err := repo.ListCompanies(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if items == nil {
		t.Fatal("got nil, want non-nil empty slice")
	}
	if len(items) != 0 {
		t.Fatalf("got %d companies, want 0", len(items))
	}
}

func TestListCompanies_ErrorGuessing_QueryError(t *testing.T) {
	mock := newCompaniesMock(t)
	repo := newCompaniesRepo(mock)

	dbErr := fmt.Errorf("companies query failed")
	mock.ExpectQuery(regexp.QuoteMeta(queryListCompanies)).
		WillReturnError(dbErr)

	_, err := repo.ListCompanies(context.Background())

	if !errors.Is(err, dbErr) {
		t.Fatalf("got error %v, want %v", err, dbErr)
	}
}

func TestListCompanies_ErrorGuessing_ScanError(t *testing.T) {
	mock := newCompaniesMock(t)
	repo := newCompaniesRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(queryListCompanies)).
		WillReturnRows(
			pgxmock.NewRows(companyColumns()).
				AddRow("bad-id", "Company A", "Description A", 1),
		)

	_, err := repo.ListCompanies(context.Background())

	if err == nil {
		t.Fatal("expected scan error, got nil")
	}
}

func TestCreateCompany_EquivalencePartitioning_Success(t *testing.T) {
	cases := []struct {
		name       string
		inName     string
		inDesc     string
		expectName string
		expectDesc string
		mockID     int
	}{
		{"success with spaces trimming", "  Company A  ", "  Description A  ", "Company A", "Description A", 10},
		{"success with empty description allowed", "Company B", "   ", "Company B", "", 11},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newCompaniesMock(t)
			repo := newCompaniesRepo(mock)

			mock.ExpectQuery(regexp.QuoteMeta(queryCreateCompany)).
				WithArgs(tc.expectName, tc.expectDesc).
				WillReturnRows(
					pgxmock.NewRows(companyColumns()).
						AddRow(tc.mockID, tc.expectName, tc.expectDesc, 0),
				)

			company, err := repo.CreateCompany(context.Background(), tc.inName, tc.inDesc)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if company == nil {
				t.Fatal("got nil company")
			}
			if company.ID != tc.mockID || company.Name != tc.expectName || company.Description != tc.expectDesc || company.LocationsCount != 0 {
				t.Fatalf("unexpected company: %+v", company)
			}
		})
	}
}

func TestCreateCompany_EquivalencePartitioning_EmptyName(t *testing.T) {
	cases := []struct {
		name        string
		companyName string
	}{
		{"empty name", ""},
		{"whitespace name", "   "},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newCompaniesMock(t)
			repo := newCompaniesRepo(mock)

			_, err := repo.CreateCompany(context.Background(), tc.companyName, "description")

			if !errors.Is(err, db.ErrInvalidArgument) {
				t.Fatalf("got error %v, want %v", err, db.ErrInvalidArgument)
			}
		})
	}
}

func TestCreateCompany_ErrorGuessing_UniqueViolation(t *testing.T) {
	mock := newCompaniesMock(t)
	repo := newCompaniesRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(queryCreateCompany)).
		WithArgs("Company A", "Description A").
		WillReturnError(&pgconn.PgError{Code: "23505"})

	_, err := repo.CreateCompany(context.Background(), "Company A", "Description A")

	if !errors.Is(err, db.ErrConflict) {
		t.Fatalf("got error %v, want %v", err, db.ErrConflict)
	}
}

func TestCreateCompany_ErrorGuessing_DBError(t *testing.T) {
	mock := newCompaniesMock(t)
	repo := newCompaniesRepo(mock)

	dbErr := fmt.Errorf("insert failed")
	mock.ExpectQuery(regexp.QuoteMeta(queryCreateCompany)).
		WithArgs("Company A", "Description A").
		WillReturnError(dbErr)

	_, err := repo.CreateCompany(context.Background(), "Company A", "Description A")

	if !errors.Is(err, dbErr) {
		t.Fatalf("got error %v, want %v", err, dbErr)
	}
}

func TestExistsByID_BoundaryValueAnalysis_InvalidIDs(t *testing.T) {
	cases := []struct {
		name string
		id   int
	}{
		{"id is zero", 0},
		{"id is negative", -1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newCompaniesMock(t)
			repo := newCompaniesRepo(mock)

			exists, err := repo.ExistsByID(context.Background(), tc.id)

			if exists {
				t.Fatal("exists: got true, want false")
			}
			if !errors.Is(err, db.ErrInvalidID) {
				t.Fatalf("got error %v, want %v", err, db.ErrInvalidID)
			}
		})
	}
}

func TestExistsByID_EquivalencePartitioning_ValidIDs(t *testing.T) {
	cases := []struct {
		name     string
		dbResult bool
		want     bool
	}{
		{"company does not exist", false, false},
		{"company exists", true, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newCompaniesMock(t)
			repo := newCompaniesRepo(mock)

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS (SELECT 1 FROM companies WHERE id = $1)`)).
				WithArgs(10).
				WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(tc.dbResult))

			exists, err := repo.ExistsByID(context.Background(), 10)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if exists != tc.want {
				t.Fatalf("exists: got %v, want %v", exists, tc.want)
			}
		})
	}
}

func TestExistsByID_ErrorGuessing_DBError(t *testing.T) {
	mock := newCompaniesMock(t)
	repo := newCompaniesRepo(mock)

	dbErr := fmt.Errorf("exists query failed")
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS (SELECT 1 FROM companies WHERE id = $1)`)).
		WithArgs(10).
		WillReturnError(dbErr)

	exists, err := repo.ExistsByID(context.Background(), 10)

	if exists {
		t.Fatal("exists: got true, want false")
	}
	if !errors.Is(err, dbErr) {
		t.Fatalf("got error %v, want %v", err, dbErr)
	}
}
