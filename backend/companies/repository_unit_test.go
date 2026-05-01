package companies

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/nikolaarden2000/software-design-project/backend/db"
	pgxmock "github.com/pashagolub/pgxmock/v2"
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

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем успешное получение списка компаний вместе с количеством локаций.
func TestListCompanies_Scenario_Success(t *testing.T) {
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

// Техника тест-дизайна: классы эквивалентности.
// Проверяем класс пустого результата: компаний нет, но возвращается не nil slice.
func TestListCompanies_EquivalenceClasses_EmptyResultReturnsEmptySlice(t *testing.T) {
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

// Техника тест-дизайна: обработка исключений.
// Проверяем, что ошибка запроса списка компаний возвращается вызывающему коду.
func TestListCompanies_ExceptionHandling_QueryErrorPropagates(t *testing.T) {
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

// Техника тест-дизайна: обработка исключений.
// Проверяем, что ошибка сканирования строки возвращается вызывающему коду.
func TestListCompanies_ExceptionHandling_ScanErrorPropagates(t *testing.T) {
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

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем успешное создание компании с нормализацией пробелов в name и description.
func TestCreateCompany_Scenario_SuccessTrimsInput(t *testing.T) {
	mock := newCompaniesMock(t)
	repo := newCompaniesRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(queryCreateCompany)).
		WithArgs("Company A", "Description A").
		WillReturnRows(
			pgxmock.NewRows(companyColumns()).
				AddRow(10, "Company A", "Description A", 0),
		)

	company, err := repo.CreateCompany(context.Background(), "  Company A  ", "  Description A  ")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if company == nil {
		t.Fatal("got nil company")
	}
	if company.ID != 10 || company.Name != "Company A" || company.Description != "Description A" || company.LocationsCount != 0 {
		t.Fatalf("unexpected company: %+v", company)
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем класс невалидных названий компании: пустая строка и строка из пробелов.
func TestCreateCompany_EquivalenceClasses_EmptyNameReturnsErrInvalidArgument(t *testing.T) {
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

// Техника тест-дизайна: классы эквивалентности.
// Проверяем допустимый класс пустого описания: description может быть пустым.
func TestCreateCompany_EquivalenceClasses_EmptyDescriptionIsAllowed(t *testing.T) {
	mock := newCompaniesMock(t)
	repo := newCompaniesRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(queryCreateCompany)).
		WithArgs("Company A", "").
		WillReturnRows(
			pgxmock.NewRows(companyColumns()).
				AddRow(10, "Company A", "", 0),
		)

	company, err := repo.CreateCompany(context.Background(), "Company A", "   ")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if company.Description != "" {
		t.Fatalf("description: got %q, want empty string", company.Description)
	}
}

// Техника тест-дизайна: таблица решений.
// Проверяем преобразование PostgreSQL unique violation в доменную ошибку конфликта.
func TestCreateCompany_DecisionTable_UniqueViolationReturnsErrConflict(t *testing.T) {
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

// Техника тест-дизайна: обработка исключений.
// Проверяем, что обычная ошибка базы данных при создании компании возвращается вызывающему коду.
func TestCreateCompany_ExceptionHandling_DBErrorPropagates(t *testing.T) {
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

// Техника тест-дизайна: граничные значения.
// Проверяем недопустимые значения id около нижней границы.
func TestExistsByID_BoundaryValues_InvalidIDsReturnErrInvalidID(t *testing.T) {
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

// Техника тест-дизайна: таблица решений.
// Проверяем результат ExistsByID в зависимости от значения SELECT EXISTS.
func TestExistsByID_DecisionTable(t *testing.T) {
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

// Техника тест-дизайна: обработка исключений.
// Проверяем, что ошибка базы данных при проверке существования компании возвращается вызывающему коду.
func TestExistsByID_ExceptionHandling_DBErrorPropagates(t *testing.T) {
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
