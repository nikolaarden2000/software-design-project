package locations

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/nikolaarden2000/software-design-project/backend/db"
	pgxmock "github.com/pashagolub/pgxmock/v2"
)

func newLocationsMock(t *testing.T) pgxmock.PgxPoolIface {
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

func newLocationsRepo(mock pgxmock.PgxPoolIface) *Repository {
	return NewRepository(mock)
}

func locationColumns() []string {
	return []string{
		"id",
		"company_id",
		"company_name",
		"city",
		"address",
		"latitude",
		"longitude",
		"timezone",
	}
}

func adminLocationColumns() []string {
	return []string{
		"id",
		"company_id",
		"company_name",
		"city",
		"address",
		"latitude",
		"longitude",
		"timezone",
		"rooms_count",
	}
}

func getLocationByIDPattern() string {
	return `(?s).*FROM locations l.*JOIN companies c.*WHERE l\.id = \$1.*`
}

func existsLocationByIDQuery() string {
	return `SELECT EXISTS (SELECT 1 FROM locations WHERE id = $1)`
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем получение списка локаций без фильтров.
func TestListLocations_Scenario_SuccessWithoutFilters(t *testing.T) {
	mock := newLocationsMock(t)
	repo := newLocationsRepo(mock)

	var companyID *int
	var city *string

	mock.ExpectQuery(regexp.QuoteMeta(queryListLocations)).
		WithArgs(companyID, city).
		WillReturnRows(
			pgxmock.NewRows(locationColumns()).
				AddRow(1, 10, "Company A", "Москва", "Москва, Тверская 10", 55.75, 37.61, "Europe/Moscow").
				AddRow(2, 20, "Company B", "Казань", "Казань, Баумана 1", 55.79, 49.12, "Europe/Moscow"),
		)

	items, err := repo.ListLocations(context.Background(), companyID, city)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d locations, want 2", len(items))
	}

	if items[0].ID != 1 || items[0].CompanyID != 10 || items[0].CompanyName != "Company A" || items[0].City != "Москва" {
		t.Fatalf("unexpected first location: %+v", items[0])
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем класс запроса списка локаций с фильтрами company_id и city.
func TestListLocations_EquivalenceClasses_SuccessWithFilters(t *testing.T) {
	mock := newLocationsMock(t)
	repo := newLocationsRepo(mock)

	companyID := 10
	city := "Москва"

	mock.ExpectQuery(regexp.QuoteMeta(queryListLocations)).
		WithArgs(&companyID, &city).
		WillReturnRows(
			pgxmock.NewRows(locationColumns()).
				AddRow(1, 10, "Company A", "Москва", "Москва, Тверская 10", 55.75, 37.61, "Europe/Moscow"),
		)

	items, err := repo.ListLocations(context.Background(), &companyID, &city)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d locations, want 1", len(items))
	}
	if items[0].CompanyID != companyID || items[0].City != city {
		t.Fatalf("unexpected filtered item: %+v", items[0])
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем класс пустого результата.
func TestListLocations_EquivalenceClasses_EmptyResultReturnsEmptySlice(t *testing.T) {
	mock := newLocationsMock(t)
	repo := newLocationsRepo(mock)

	var companyID *int
	var city *string

	mock.ExpectQuery(regexp.QuoteMeta(queryListLocations)).
		WithArgs(companyID, city).
		WillReturnRows(pgxmock.NewRows(locationColumns()))

	items, err := repo.ListLocations(context.Background(), companyID, city)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if items == nil {
		t.Fatal("got nil, want non-nil empty slice")
	}
	if len(items) != 0 {
		t.Fatalf("got %d locations, want 0", len(items))
	}
}

// Техника тест-дизайна: обработка исключений.
// Проверяем, что ошибка запроса списка локаций возвращается вызывающему коду.
func TestListLocations_ExceptionHandling_QueryErrorPropagates(t *testing.T) {
	mock := newLocationsMock(t)
	repo := newLocationsRepo(mock)

	var companyID *int
	var city *string

	dbErr := fmt.Errorf("locations query failed")
	mock.ExpectQuery(regexp.QuoteMeta(queryListLocations)).
		WithArgs(companyID, city).
		WillReturnError(dbErr)

	_, err := repo.ListLocations(context.Background(), companyID, city)

	if !errors.Is(err, dbErr) {
		t.Fatalf("got error %v, want %v", err, dbErr)
	}
}

// Техника тест-дизайна: обработка исключений.
// Проверяем, что ошибка сканирования строки списка локаций возвращается вызывающему коду.
func TestListLocations_ExceptionHandling_ScanErrorPropagates(t *testing.T) {
	mock := newLocationsMock(t)
	repo := newLocationsRepo(mock)

	var companyID *int
	var city *string

	mock.ExpectQuery(regexp.QuoteMeta(queryListLocations)).
		WithArgs(companyID, city).
		WillReturnRows(
			pgxmock.NewRows(locationColumns()).
				AddRow("bad-id", 10, "Company A", "Москва", "Москва, Тверская 10", 55.75, 37.61, "Europe/Moscow"),
		)

	_, err := repo.ListLocations(context.Background(), companyID, city)

	if err == nil {
		t.Fatal("expected scan error, got nil")
	}
}

// Техника тест-дизайна: таблица решений.
// Проверяем результат ListAdminLocations в зависимости от includeAll и валидности adminID.
func TestListAdminLocations_DecisionTable_AdminIDValidationAndIncludeAll(t *testing.T) {
	cases := []struct {
		name        string
		adminID     int
		includeAll  bool
		expectQuery bool
		wantErr     error
	}{
		{"admin id is zero without includeAll", 0, false, false, db.ErrInvalidID},
		{"admin id is negative without includeAll", -1, false, false, db.ErrInvalidID},
		{"admin id is zero with includeAll", 0, true, true, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newLocationsMock(t)
			repo := newLocationsRepo(mock)

			if tc.expectQuery {
				mock.ExpectQuery(regexp.QuoteMeta(queryListAdminLocations)).
					WithArgs(tc.includeAll, tc.adminID).
					WillReturnRows(pgxmock.NewRows(adminLocationColumns()))
			}

			items, err := repo.ListAdminLocations(context.Background(), tc.adminID, tc.includeAll)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("got error %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if items == nil {
				t.Fatal("got nil, want non-nil empty slice")
			}
			if len(items) != 0 {
				t.Fatalf("got %d locations, want 0", len(items))
			}
		})
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем успешное получение локаций, доступных администратору.
func TestListAdminLocations_Scenario_Success(t *testing.T) {
	mock := newLocationsMock(t)
	repo := newLocationsRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(queryListAdminLocations)).
		WithArgs(false, 7).
		WillReturnRows(
			pgxmock.NewRows(adminLocationColumns()).
				AddRow(1, 10, "Company A", "Москва", "Москва, Тверская 10", 55.75, 37.61, "Europe/Moscow", 3).
				AddRow(2, 10, "Company A", "Москва", "Москва, Арбат 5", 55.74, 37.59, "Europe/Moscow", 0),
		)

	items, err := repo.ListAdminLocations(context.Background(), 7, false)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d locations, want 2", len(items))
	}

	if items[0].ID != 1 || items[0].RoomsCount != 3 {
		t.Fatalf("unexpected first admin location: %+v", items[0])
	}
	if items[1].ID != 2 || items[1].RoomsCount != 0 {
		t.Fatalf("unexpected second admin location: %+v", items[1])
	}
}

// Техника тест-дизайна: обработка исключений.
// Проверяем, что ошибка запроса admin-локаций возвращается вызывающему коду.
func TestListAdminLocations_ExceptionHandling_QueryErrorPropagates(t *testing.T) {
	mock := newLocationsMock(t)
	repo := newLocationsRepo(mock)

	dbErr := fmt.Errorf("admin locations query failed")
	mock.ExpectQuery(regexp.QuoteMeta(queryListAdminLocations)).
		WithArgs(false, 7).
		WillReturnError(dbErr)

	_, err := repo.ListAdminLocations(context.Background(), 7, false)

	if !errors.Is(err, dbErr) {
		t.Fatalf("got error %v, want %v", err, dbErr)
	}
}

// Техника тест-дизайна: обработка исключений.
// Проверяем, что ошибка сканирования строки admin-локации возвращается вызывающему коду.
func TestListAdminLocations_ExceptionHandling_ScanErrorPropagates(t *testing.T) {
	mock := newLocationsMock(t)
	repo := newLocationsRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(queryListAdminLocations)).
		WithArgs(false, 7).
		WillReturnRows(
			pgxmock.NewRows(adminLocationColumns()).
				AddRow(1, 10, "Company A", "Москва", "Москва, Тверская 10", 55.75, 37.61, "Europe/Moscow", "bad-count"),
		)

	_, err := repo.ListAdminLocations(context.Background(), 7, false)

	if err == nil {
		t.Fatal("expected scan error, got nil")
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем варианты разбора адреса для хранения в полях street и house_number.
func TestSplitAddressForStorage_EquivalenceClasses(t *testing.T) {
	cases := []struct {
		name            string
		city            string
		address         string
		wantStreet      string
		wantHouseNumber string
	}{
		{
			name:            "address with city prefix",
			city:            "Москва",
			address:         "Москва, Тверская 10",
			wantStreet:      "Тверская",
			wantHouseNumber: "10",
		},
		{
			name:            "address without city prefix",
			city:            "Москва",
			address:         "Арбат 5",
			wantStreet:      "Арбат",
			wantHouseNumber: "5",
		},
		{
			name:            "multiword street",
			city:            "Москва",
			address:         "Москва, Большая Никитская 12",
			wantStreet:      "Большая Никитская",
			wantHouseNumber: "12",
		},
		{
			name:            "single word address",
			city:            "Москва",
			address:         "Кремль",
			wantStreet:      "Кремль",
			wantHouseNumber: "-",
		},
		{
			name:            "empty address after trim",
			city:            "Москва",
			address:         "   ",
			wantStreet:      "",
			wantHouseNumber: "-",
		},
		{
			name:            "extra spaces are normalized",
			city:            "Москва",
			address:         "  Москва,   Новый   Арбат   21  ",
			wantStreet:      "Новый Арбат",
			wantHouseNumber: "21",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			street, houseNumber := splitAddressForStorage(tc.city, tc.address)

			if street != tc.wantStreet {
				t.Fatalf("street: got %q, want %q", street, tc.wantStreet)
			}
			if houseNumber != tc.wantHouseNumber {
				t.Fatalf("houseNumber: got %q, want %q", houseNumber, tc.wantHouseNumber)
			}
		})
	}
}

// Техника тест-дизайна: граничные значения.
// Проверяем недопустимые companyID около нижней границы.
func TestCreateLocation_BoundaryValues_InvalidCompanyIDReturnsErrInvalidID(t *testing.T) {
	cases := []struct {
		name      string
		companyID int
	}{
		{"company id is zero", 0},
		{"company id is negative", -1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newLocationsMock(t)
			repo := newLocationsRepo(mock)

			_, err := repo.CreateLocation(
				context.Background(),
				tc.companyID,
				"Москва",
				"Москва, Тверская 10",
				55.75,
				37.61,
				"Europe/Moscow",
			)

			if !errors.Is(err, db.ErrInvalidID) {
				t.Fatalf("got error %v, want %v", err, db.ErrInvalidID)
			}
		})
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем невалидные текстовые поля city и address.
func TestCreateLocation_EquivalenceClasses_EmptyCityOrAddressReturnsErrInvalidArgument(t *testing.T) {
	cases := []struct {
		name    string
		city    string
		address string
	}{
		{"empty city", "", "Москва, Тверская 10"},
		{"whitespace city", "   ", "Москва, Тверская 10"},
		{"empty address", "Москва", ""},
		{"whitespace address", "Москва", "   "},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newLocationsMock(t)
			repo := newLocationsRepo(mock)

			_, err := repo.CreateLocation(
				context.Background(),
				10,
				tc.city,
				tc.address,
				55.75,
				37.61,
				"Europe/Moscow",
			)

			if !errors.Is(err, db.ErrInvalidArgument) {
				t.Fatalf("got error %v, want %v", err, db.ErrInvalidArgument)
			}
		})
	}
}

// Техника тест-дизайна: таблица решений.
// Проверяем выбор timezone при создании локации: пустое значение заменяется значением по умолчанию, непустое используется после trim.
func TestCreateLocation_DecisionTable_TimezoneSelection(t *testing.T) {
	cases := []struct {
		name         string
		city         string
		address      string
		lat          float64
		lng          float64
		timezone     string
		wantID       int
		wantCity     string
		wantStreet   string
		wantHouse    string
		wantAddress  string
		wantTimezone string
	}{
		{
			name:         "default timezone",
			city:         "  Москва  ",
			address:      "  Москва, Тверская 10  ",
			lat:          55.75,
			lng:          37.61,
			timezone:     "   ",
			wantID:       100,
			wantCity:     "Москва",
			wantStreet:   "Тверская",
			wantHouse:    "10",
			wantAddress:  "Москва, Тверская 10",
			wantTimezone: "Europe/Moscow",
		},
		{
			name:         "explicit timezone",
			city:         "Казань",
			address:      "Казань, Баумана 1",
			lat:          55.79,
			lng:          49.12,
			timezone:     " Europe/Moscow ",
			wantID:       101,
			wantCity:     "Казань",
			wantStreet:   "Баумана",
			wantHouse:    "1",
			wantAddress:  "Казань, Баумана 1",
			wantTimezone: "Europe/Moscow",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newLocationsMock(t)
			repo := newLocationsRepo(mock)

			mock.ExpectQuery(regexp.QuoteMeta(queryCreateLocation)).
				WithArgs(10, tc.wantCity, tc.wantStreet, tc.wantHouse, tc.lat, tc.lng, tc.wantTimezone).
				WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(tc.wantID))

			mock.ExpectQuery(getLocationByIDPattern()).
				WithArgs(tc.wantID).
				WillReturnRows(
					pgxmock.NewRows(locationColumns()).
						AddRow(tc.wantID, 10, "Company A", tc.wantCity, tc.wantAddress, tc.lat, tc.lng, tc.wantTimezone),
				)

			location, err := repo.CreateLocation(
				context.Background(),
				10,
				tc.city,
				tc.address,
				tc.lat,
				tc.lng,
				tc.timezone,
			)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if location == nil {
				t.Fatal("got nil location")
			}
			if location.ID != tc.wantID || location.City != tc.wantCity || location.Address != tc.wantAddress || location.Timezone != tc.wantTimezone {
				t.Fatalf("unexpected location: %+v", location)
			}
		})
	}
}

// Техника тест-дизайна: обработка исключений.
// Проверяем, что ошибка INSERT при создании локации возвращается вызывающему коду.
func TestCreateLocation_ExceptionHandling_InsertErrorPropagates(t *testing.T) {
	mock := newLocationsMock(t)
	repo := newLocationsRepo(mock)

	dbErr := fmt.Errorf("insert location failed")
	mock.ExpectQuery(regexp.QuoteMeta(queryCreateLocation)).
		WithArgs(10, "Москва", "Тверская", "10", 55.75, 37.61, "Europe/Moscow").
		WillReturnError(dbErr)

	_, err := repo.CreateLocation(
		context.Background(),
		10,
		"Москва",
		"Москва, Тверская 10",
		55.75,
		37.61,
		"Europe/Moscow",
	)

	if !errors.Is(err, dbErr) {
		t.Fatalf("got error %v, want %v", err, dbErr)
	}
}

// Техника тест-дизайна: обработка исключений.
// Проверяем, что ошибка последующего GetLocationByID после INSERT возвращается вызывающему коду.
func TestCreateLocation_ExceptionHandling_GetCreatedLocationErrorPropagates(t *testing.T) {
	mock := newLocationsMock(t)
	repo := newLocationsRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(queryCreateLocation)).
		WithArgs(10, "Москва", "Тверская", "10", 55.75, 37.61, "Europe/Moscow").
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(100))

	mock.ExpectQuery(getLocationByIDPattern()).
		WithArgs(100).
		WillReturnRows(pgxmock.NewRows(locationColumns()))

	_, err := repo.CreateLocation(
		context.Background(),
		10,
		"Москва",
		"Москва, Тверская 10",
		55.75,
		37.61,
		"Europe/Moscow",
	)

	if !errors.Is(err, db.ErrNotFound) {
		t.Fatalf("got error %v, want %v", err, db.ErrNotFound)
	}
}

// Техника тест-дизайна: граничные значения.
// Проверяем недопустимые id около нижней границы.
func TestGetLocationByID_BoundaryValues_InvalidIDsReturnErrInvalidID(t *testing.T) {
	cases := []struct {
		name string
		id   int
	}{
		{"id is zero", 0},
		{"id is negative", -1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newLocationsMock(t)
			repo := newLocationsRepo(mock)

			_, err := repo.GetLocationByID(context.Background(), tc.id)

			if !errors.Is(err, db.ErrInvalidID) {
				t.Fatalf("got error %v, want %v", err, db.ErrInvalidID)
			}
		})
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем успешное получение локации по id.
func TestGetLocationByID_Scenario_Success(t *testing.T) {
	mock := newLocationsMock(t)
	repo := newLocationsRepo(mock)

	mock.ExpectQuery(getLocationByIDPattern()).
		WithArgs(100).
		WillReturnRows(
			pgxmock.NewRows(locationColumns()).
				AddRow(100, 10, "Company A", "Москва", "Москва, Тверская 10", 55.75, 37.61, "Europe/Moscow"),
		)

	location, err := repo.GetLocationByID(context.Background(), 100)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if location.ID != 100 || location.CompanyID != 10 || location.CompanyName != "Company A" {
		t.Fatalf("unexpected location: %+v", location)
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем класс отсутствующей локации.
func TestGetLocationByID_EquivalenceClasses_NotFoundReturnsErrNotFound(t *testing.T) {
	mock := newLocationsMock(t)
	repo := newLocationsRepo(mock)

	mock.ExpectQuery(getLocationByIDPattern()).
		WithArgs(999).
		WillReturnRows(pgxmock.NewRows(locationColumns()))

	_, err := repo.GetLocationByID(context.Background(), 999)

	if !errors.Is(err, db.ErrNotFound) {
		t.Fatalf("got error %v, want %v", err, db.ErrNotFound)
	}
}

// Техника тест-дизайна: обработка исключений.
// Проверяем, что ошибка запроса локации по id возвращается вызывающему коду.
func TestGetLocationByID_ExceptionHandling_QueryErrorPropagates(t *testing.T) {
	mock := newLocationsMock(t)
	repo := newLocationsRepo(mock)

	dbErr := fmt.Errorf("get location failed")
	mock.ExpectQuery(getLocationByIDPattern()).
		WithArgs(100).
		WillReturnError(dbErr)

	_, err := repo.GetLocationByID(context.Background(), 100)

	if !errors.Is(err, dbErr) {
		t.Fatalf("got error %v, want %v", err, dbErr)
	}
}

// Техника тест-дизайна: обработка исключений.
// Проверяем, что ошибка сканирования локации по id возвращается вызывающему коду.
func TestGetLocationByID_ExceptionHandling_ScanErrorPropagates(t *testing.T) {
	mock := newLocationsMock(t)
	repo := newLocationsRepo(mock)

	mock.ExpectQuery(getLocationByIDPattern()).
		WithArgs(100).
		WillReturnRows(
			pgxmock.NewRows(locationColumns()).
				AddRow("bad-id", 10, "Company A", "Москва", "Москва, Тверская 10", 55.75, 37.61, "Europe/Moscow"),
		)

	_, err := repo.GetLocationByID(context.Background(), 100)

	if err == nil {
		t.Fatal("expected scan error, got nil")
	}
}

// Техника тест-дизайна: граничные значения.
// Проверяем недопустимые id около нижней границы.
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
			mock := newLocationsMock(t)
			repo := newLocationsRepo(mock)

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

// Техника тест-дизайна: классы эквивалентности.
// Проверяем классы результата ExistsByID: локация существует и локация не существует.
func TestExistsByID_EquivalenceClasses_ExistsAndNotExists(t *testing.T) {
	cases := []struct {
		name     string
		dbResult bool
		want     bool
	}{
		{"location does not exist", false, false},
		{"location exists", true, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newLocationsMock(t)
			repo := newLocationsRepo(mock)

			mock.ExpectQuery(regexp.QuoteMeta(existsLocationByIDQuery())).
				WithArgs(100).
				WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(tc.dbResult))

			exists, err := repo.ExistsByID(context.Background(), 100)

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
// Проверяем, что ошибка запроса ExistsByID возвращается вызывающему коду.
func TestExistsByID_ExceptionHandling_DBErrorPropagates(t *testing.T) {
	mock := newLocationsMock(t)
	repo := newLocationsRepo(mock)

	dbErr := fmt.Errorf("exists location failed")
	mock.ExpectQuery(regexp.QuoteMeta(existsLocationByIDQuery())).
		WithArgs(100).
		WillReturnError(dbErr)

	exists, err := repo.ExistsByID(context.Background(), 100)

	if exists {
		t.Fatal("exists: got true, want false")
	}
	if !errors.Is(err, dbErr) {
		t.Fatalf("got error %v, want %v", err, dbErr)
	}
}
