package users

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

func userColumns() []string {
	return []string{"id", "name", "email", "password_hash", "role"}
}

func adminColumns() []string {
	return []string{"id", "name", "email", "role"}
}

func adminLocationColumns() []string {
	return []string{"id", "address", "company_name"}
}

func expectUserByID(mock pgxmock.PgxPoolIface, id int, userID int, username, email, passwordHash, role string) {
	mock.ExpectQuery(regexp.QuoteMeta(queryGetUserByID)).
		WithArgs(id).
		WillReturnRows(
			pgxmock.NewRows(userColumns()).
				AddRow(userID, username, email, passwordHash, role),
		)
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем классы валидных и невалидных ролей.
func TestIsValidRole_EquivalenceClasses(t *testing.T) {
	cases := []struct {
		name string
		role string
		want bool
	}{
		{"user role is valid", RoleUser, true},
		{"admin role is valid", RoleAdmin, true},
		{"superuser role is valid", RoleSuperuser, true},
		{"empty role is invalid", "", false},
		{"unknown role is invalid", "moderator", false},
		{"case-sensitive role is invalid", "ADMIN", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsValidRole(tc.role)

			if got != tc.want {
				t.Fatalf("IsValidRole(%q): got %v, want %v", tc.role, got, tc.want)
			}
		})
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем класс валидных ролей при создании пользователя.
func TestCreateUserWithRole_RoleEquivalenceClasses_ValidRoles(t *testing.T) {
	cases := []struct {
		name string
		role string
	}{
		{"user role", RoleUser},
		{"admin role", RoleAdmin},
		{"superuser role", RoleSuperuser},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newMock(t)
			repo := newUserRepo(mock)

			mock.ExpectQuery(regexp.QuoteMeta(queryCreateUserWithRole)).
				WithArgs("alice", "alice@example.com", "hash", tc.role).
				WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(42))

			id, err := repo.CreateUserWithRole(context.Background(), "alice", "alice@example.com", "hash", tc.role)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id != 42 {
				t.Fatalf("id: got %d, want 42", id)
			}
		})
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем два класса ролей: валидная роль (успешное создание) и невалидная роль (ошибка валидации до БД).
func TestCreateUserWithRole_EquivalenceClasses_ValidAndInvalidRole(t *testing.T) {
	cases := []struct {
		name      string
		role      string
		prepareDB func(mock pgxmock.PgxPoolIface)
		wantID    int
		wantErr   error
	}{
		{
			name:    "invalid role rejected before db call",
			role:    "moderator",
			wantErr: db.ErrInvalidArgument,
		},
		{
			name: "valid role returns new id",
			role: RoleAdmin,
			prepareDB: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(regexp.QuoteMeta(queryCreateUserWithRole)).
					WithArgs("alice", "alice@example.com", "hash", RoleAdmin).
					WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(42))
			},
			wantID: 42,
		},
		{
			name: "valid role returns new id",
			role: RoleUser,
			prepareDB: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(regexp.QuoteMeta(queryCreateUserWithRole)).
					WithArgs("alice", "alice@example.com", "hash", RoleUser).
					WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(100))
			},
			wantID: 100,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newMock(t)
			repo := newUserRepo(mock)
			if tc.prepareDB != nil {
				tc.prepareDB(mock)
			}

			id, err := repo.CreateUserWithRole(context.Background(), "alice", "alice@example.com", "hash", tc.role)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
			if id != tc.wantID {
				t.Fatalf("id: got %d, want %d", id, tc.wantID)
			}
		})
	}
}

// Техника тест-дизайна: предугадывание ошибок + обработка исключений.
// Проверяем типичные операционные сбои: конфликт по email и произвольная ошибка БД.
func TestCreateUserWithRole_ErrorGuessingAndExceptionHandling(t *testing.T) {
	dbErr := fmt.Errorf("insert failed")
	cases := []struct {
		name      string
		role      string
		prepareDB func(mock pgxmock.PgxPoolIface)
		wantErr   error
	}{
		{
			name: "taken email returns domain error",
			role: RoleUser,
			prepareDB: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(regexp.QuoteMeta(queryCreateUserWithRole)).
					WithArgs("alice", "alice@example.com", "hash", RoleUser).
					WillReturnRows(pgxmock.NewRows([]string{"id"}))
			},
			wantErr: db.ErrEmailTaken,
		},
		{
			name: "db error is propagated",
			role: RoleAdmin,
			prepareDB: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(regexp.QuoteMeta(queryCreateUserWithRole)).
					WithArgs("alice", "alice@example.com", "hash", RoleAdmin).
					WillReturnError(dbErr)
			},
			wantErr: dbErr,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newMock(t)
			repo := newUserRepo(mock)
			tc.prepareDB(mock)

			_, err := repo.CreateUserWithRole(context.Background(), "alice", "alice@example.com", "hash", tc.role)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем классы результата поиска по email: пользователь найден и пользователь отсутствует.
func TestGetUserByEmail_EquivalenceClasses_FoundAndNotFound(t *testing.T) {
	cases := []struct {
		name      string
		email     string
		prepareDB func(mock pgxmock.PgxPoolIface)
		wantErr   error
		wantUser  *User
	}{
		{
			name:  "user exists",
			email: "alice@example.com",
			prepareDB: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(regexp.QuoteMeta(queryGetUserByEmail)).
					WithArgs("alice@example.com").
					WillReturnRows(pgxmock.NewRows(userColumns()).AddRow(1, "alice", "alice@example.com", "hashed-pwd", RoleUser))
			},
			wantUser: &User{ID: 1, Username: "alice", Email: "alice@example.com", Password: "hashed-pwd", Role: RoleUser},
		},
		{
			name:  "user not found",
			email: "ghost@example.com",
			prepareDB: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(regexp.QuoteMeta(queryGetUserByEmail)).
					WithArgs("ghost@example.com").
					WillReturnRows(pgxmock.NewRows(userColumns()))
			},
			wantErr: db.ErrNotFound,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newMock(t)
			repo := newUserRepo(mock)
			tc.prepareDB(mock)
			got, err := repo.GetUserByEmail(context.Background(), tc.email)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
			if tc.wantUser != nil && (got.ID != tc.wantUser.ID || got.Email != tc.wantUser.Email || got.Role != tc.wantUser.Role) {
				t.Fatalf("unexpected user fields: %+v", got)
			}
		})
	}
}

// Техника тест-дизайна: обработка исключений.
// Проверяем, что ошибка БД при поиске по email пробрасывается без потери.
func TestGetUserByEmail_ExceptionHandling_DBErrorPropagates(t *testing.T) {
	mock := newMock(t)
	repo := newUserRepo(mock)
	dbErr := fmt.Errorf("timeout")
	mock.ExpectQuery(regexp.QuoteMeta(queryGetUserByEmail)).
		WithArgs("alice@example.com").
		WillReturnError(dbErr)

	_, err := repo.GetUserByEmail(context.Background(), "alice@example.com")
	if !errors.Is(err, dbErr) {
		t.Fatalf("got error %v, want %v", err, dbErr)
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем успешное получение пользователя по id.
func TestGetUserByID_Scenario_Success(t *testing.T) {
	mock := newMock(t)
	repo := newUserRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(queryGetUserByID)).
		WithArgs(7).
		WillReturnRows(
			pgxmock.NewRows(userColumns()).
				AddRow(7, "dave", "dave@example.com", "hashed-pwd", RoleAdmin),
		)

	u, err := repo.GetUserByID(context.Background(), 7)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != 7 || u.Username != "dave" || u.Email != "dave@example.com" || u.Password != "hashed-pwd" || u.Role != RoleAdmin {
		t.Fatalf("unexpected user fields: %+v", u)
	}
}

// Техника тест-дизайна: граничные значения.
// Проверяем недопустимые значения id около нижней границы.
func TestGetUserByID_BoundaryValues_InvalidIDsReturnErrInvalidID(t *testing.T) {
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
			repo := newUserRepo(mock)

			_, err := repo.GetUserByID(context.Background(), tc.id)

			if !errors.Is(err, db.ErrInvalidID) {
				t.Fatalf("got error %v, want %v", err, db.ErrInvalidID)
			}
		})
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем класс отсутствующего пользователя при поиске по id.
func TestGetUserByID_EquivalenceClasses_NotFoundReturnsErrNotFound(t *testing.T) {
	mock := newMock(t)
	repo := newUserRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(queryGetUserByID)).
		WithArgs(123).
		WillReturnRows(pgxmock.NewRows(userColumns()))

	_, err := repo.GetUserByID(context.Background(), 123)

	if !errors.Is(err, db.ErrNotFound) {
		t.Fatalf("got error %v, want %v", err, db.ErrNotFound)
	}
}

// Техника тест-дизайна: обработка исключений.
// Проверяем, что ошибка базы данных при поиске по id возвращается вызывающему коду.
func TestGetUserByID_ExceptionHandling_DBErrorPropagates(t *testing.T) {
	mock := newMock(t)
	repo := newUserRepo(mock)

	dbErr := fmt.Errorf("connection reset")
	mock.ExpectQuery(regexp.QuoteMeta(queryGetUserByID)).
		WithArgs(7).
		WillReturnError(dbErr)

	_, err := repo.GetUserByID(context.Background(), 7)

	if !errors.Is(err, dbErr) {
		t.Fatalf("got error %v, want %v", err, dbErr)
	}
}

// Техника тест-дизайна: таблица решений.
// Проверяем результат IsEmailTaken в зависимости от значения EXISTS.
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
			repo := newUserRepo(mock)

			mock.ExpectQuery(regexp.QuoteMeta(queryIsEmailTaken)).
				WithArgs("test@example.com").
				WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(tc.dbResult))

			got, err := repo.IsEmailTaken(context.Background(), "test@example.com")

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// Техника тест-дизайна: обработка исключений.
// Проверяем, что ошибка базы данных при проверке email возвращается вызывающему коду.
func TestIsEmailTaken_ExceptionHandling_DBErrorPropagates(t *testing.T) {
	mock := newMock(t)
	repo := newUserRepo(mock)

	dbErr := fmt.Errorf("query timeout")
	mock.ExpectQuery(regexp.QuoteMeta(queryIsEmailTaken)).
		WithArgs("test@example.com").
		WillReturnError(dbErr)

	got, err := repo.IsEmailTaken(context.Background(), "test@example.com")

	if got {
		t.Error("expected false on error, got true")
	}
	if !errors.Is(err, dbErr) {
		t.Fatalf("got error %v, want %v", err, dbErr)
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем получение списка локаций, назначенных администратору.
func TestListAdminLocations_Scenario_Success(t *testing.T) {
	mock := newMock(t)
	repo := newUserRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(queryListAdminLocations)).
		WithArgs(7).
		WillReturnRows(
			pgxmock.NewRows(adminLocationColumns()).
				AddRow(1, "Москва, Тверская 10", "Company A").
				AddRow(2, "Москва, Арбат 5", "Company B"),
		)

	locations, err := repo.ListAdminLocations(context.Background(), 7)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(locations) != 2 {
		t.Fatalf("got %d locations, want 2", len(locations))
	}
	if locations[0].ID != 1 || locations[0].Address != "Москва, Тверская 10" || locations[0].CompanyName != "Company A" {
		t.Fatalf("unexpected first location: %+v", locations[0])
	}
}

// Техника тест-дизайна: граничные значения.
// Проверяем недопустимые значения adminID около нижней границы.
func TestListAdminLocations_BoundaryValues_InvalidAdminIDsReturnErrInvalidID(t *testing.T) {
	cases := []struct {
		name    string
		adminID int
	}{
		{"zero admin id", 0},
		{"negative admin id", -1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newMock(t)
			repo := newUserRepo(mock)

			_, err := repo.ListAdminLocations(context.Background(), tc.adminID)

			if !errors.Is(err, db.ErrInvalidID) {
				t.Fatalf("got error %v, want %v", err, db.ErrInvalidID)
			}
		})
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем класс администратора без назначенных локаций.
func TestListAdminLocations_EquivalenceClasses_EmptyResultReturnsEmptySlice(t *testing.T) {
	mock := newMock(t)
	repo := newUserRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(queryListAdminLocations)).
		WithArgs(7).
		WillReturnRows(pgxmock.NewRows(adminLocationColumns()))

	locations, err := repo.ListAdminLocations(context.Background(), 7)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if locations == nil {
		t.Fatal("got nil, want non-nil empty slice")
	}
	if len(locations) != 0 {
		t.Fatalf("got %d locations, want 0", len(locations))
	}
}

// Техника тест-дизайна: обработка исключений.
// Проверяем, что ошибка базы данных при получении локаций администратора возвращается вызывающему коду.
func TestListAdminLocations_ExceptionHandling_DBErrorPropagates(t *testing.T) {
	mock := newMock(t)
	repo := newUserRepo(mock)

	dbErr := fmt.Errorf("locations query failed")
	mock.ExpectQuery(regexp.QuoteMeta(queryListAdminLocations)).
		WithArgs(7).
		WillReturnError(dbErr)

	_, err := repo.ListAdminLocations(context.Background(), 7)

	if !errors.Is(err, dbErr) {
		t.Fatalf("got error %v, want %v", err, dbErr)
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем список администраторов вместе с их назначенными локациями.
func TestListAdmins_Scenario_SuccessWithLocations(t *testing.T) {
	mock := newMock(t)
	repo := newUserRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(queryListAdmins)).
		WillReturnRows(
			pgxmock.NewRows(adminColumns()).
				AddRow(1, "admin1", "admin1@example.com", RoleAdmin).
				AddRow(2, "admin2", "admin2@example.com", RoleAdmin),
		)

	mock.ExpectQuery(regexp.QuoteMeta(queryListAdminLocations)).
		WithArgs(1).
		WillReturnRows(
			pgxmock.NewRows(adminLocationColumns()).
				AddRow(10, "Москва, Тверская 10", "Company A"),
		)

	mock.ExpectQuery(regexp.QuoteMeta(queryListAdminLocations)).
		WithArgs(2).
		WillReturnRows(
			pgxmock.NewRows(adminLocationColumns()).
				AddRow(20, "Казань, Баумана 1", "Company B").
				AddRow(21, "Казань, Кремлевская 2", "Company B"),
		)

	admins, err := repo.ListAdmins(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(admins) != 2 {
		t.Fatalf("got %d admins, want 2", len(admins))
	}

	if admins[0].ID != 1 || admins[0].Username != "admin1" || len(admins[0].Locations) != 1 {
		t.Fatalf("unexpected first admin: %+v", admins[0])
	}

	if admins[1].ID != 2 || admins[1].Username != "admin2" || len(admins[1].Locations) != 2 {
		t.Fatalf("unexpected second admin: %+v", admins[1])
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем класс результата без администраторов.
func TestListAdmins_EquivalenceClasses_EmptyResultReturnsEmptySlice(t *testing.T) {
	mock := newMock(t)
	repo := newUserRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(queryListAdmins)).
		WillReturnRows(pgxmock.NewRows(adminColumns()))

	admins, err := repo.ListAdmins(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if admins == nil {
		t.Fatal("got nil, want non-nil empty slice")
	}
	if len(admins) != 0 {
		t.Fatalf("got %d admins, want 0", len(admins))
	}
}

// Техника тест-дизайна: обработка исключений.
// Проверяем ошибку основного запроса списка администраторов.
func TestListAdmins_ExceptionHandling_ListAdminsDBErrorPropagates(t *testing.T) {
	mock := newMock(t)
	repo := newUserRepo(mock)

	dbErr := fmt.Errorf("admins query failed")
	mock.ExpectQuery(regexp.QuoteMeta(queryListAdmins)).
		WillReturnError(dbErr)

	_, err := repo.ListAdmins(context.Background())

	if !errors.Is(err, dbErr) {
		t.Fatalf("got error %v, want %v", err, dbErr)
	}
}

// Техника тест-дизайна: обработка исключений.
// Проверяем ошибку вложенного запроса локаций администратора.
func TestListAdmins_ExceptionHandling_ListAdminLocationsDBErrorPropagates(t *testing.T) {
	mock := newMock(t)
	repo := newUserRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(queryListAdmins)).
		WillReturnRows(
			pgxmock.NewRows(adminColumns()).
				AddRow(1, "admin1", "admin1@example.com", RoleAdmin),
		)

	dbErr := fmt.Errorf("locations query failed")
	mock.ExpectQuery(regexp.QuoteMeta(queryListAdminLocations)).
		WithArgs(1).
		WillReturnError(dbErr)

	_, err := repo.ListAdmins(context.Background())

	if !errors.Is(err, dbErr) {
		t.Fatalf("got error %v, want %v", err, dbErr)
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем успешное назначение администратора на локацию.
func TestAssignAdminToLocation_Scenario_Success(t *testing.T) {
	mock := newMock(t)
	repo := newUserRepo(mock)

	expectUserByID(mock, 7, 7, "admin", "admin@example.com", "hash", RoleAdmin)

	mock.ExpectExec(regexp.QuoteMeta(queryAssignAdminToLocation)).
		WithArgs(7, 10).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := repo.AssignAdminToLocation(context.Background(), 7, 10)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Техника тест-дизайна: граничные значения.
// Проверяем недопустимые adminID и locationID около нижней границы.
func TestAssignAdminToLocation_BoundaryValues_InvalidIDsReturnErrInvalidID(t *testing.T) {
	cases := []struct {
		name       string
		adminID    int
		locationID int
	}{
		{"admin id is zero", 0, 10},
		{"admin id is negative", -1, 10},
		{"location id is zero", 7, 0},
		{"location id is negative", 7, -1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newMock(t)
			repo := newUserRepo(mock)

			err := repo.AssignAdminToLocation(context.Background(), tc.adminID, tc.locationID)

			if !errors.Is(err, db.ErrInvalidID) {
				t.Fatalf("got error %v, want %v", err, db.ErrInvalidID)
			}
		})
	}
}

// Техника тест-дизайна: таблица решений.
// Проверяем, что назначение возможно только для пользователя с ролью admin.
func TestAssignAdminToLocation_DecisionTable_UserMustHaveAdminRole(t *testing.T) {
	cases := []struct {
		name    string
		role    string
		wantErr error
	}{
		{"admin role can be assigned", RoleAdmin, nil},
		{"user role cannot be assigned", RoleUser, db.ErrNotFound},
		{"superuser role cannot be assigned", RoleSuperuser, db.ErrNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newMock(t)
			repo := newUserRepo(mock)

			expectUserByID(mock, 7, 7, "person", "person@example.com", "hash", tc.role)

			if tc.wantErr == nil {
				mock.ExpectExec(regexp.QuoteMeta(queryAssignAdminToLocation)).
					WithArgs(7, 10).
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
			}

			err := repo.AssignAdminToLocation(context.Background(), 7, 10)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// Техника тест-дизайна: обработка исключений.
// Проверяем, что ошибка GetUserByID при назначении администратора возвращается вызывающему коду.
func TestAssignAdminToLocation_ExceptionHandling_GetUserByIDErrorPropagates(t *testing.T) {
	mock := newMock(t)
	repo := newUserRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(queryGetUserByID)).
		WithArgs(7).
		WillReturnRows(pgxmock.NewRows(userColumns()))

	err := repo.AssignAdminToLocation(context.Background(), 7, 10)

	if !errors.Is(err, db.ErrNotFound) {
		t.Fatalf("got error %v, want %v", err, db.ErrNotFound)
	}
}

// Техника тест-дизайна: таблица решений.
// Проверяем преобразование PostgreSQL-ошибок назначения администратора.
func TestMapAssignmentError_DecisionTable(t *testing.T) {
	plainErr := fmt.Errorf("plain error")
	duplicateErr := &pgconn.PgError{Code: "23505"}
	foreignKeyErr := &pgconn.PgError{Code: "23503"}
	unknownPgErr := &pgconn.PgError{Code: "99999"}

	cases := []struct {
		name    string
		err     error
		wantErr error
	}{
		{"plain error is returned as is", plainErr, plainErr},
		{"unique violation becomes conflict", duplicateErr, db.ErrConflict},
		{"foreign key violation becomes not found", foreignKeyErr, db.ErrNotFound},
		{"unknown pg error is returned as is", unknownPgErr, unknownPgErr},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mapAssignmentError(tc.err)

			if !errors.Is(got, tc.wantErr) {
				t.Fatalf("got error %v, want %v", got, tc.wantErr)
			}
		})
	}
}

// Техника тест-дизайна: обработка исключений.
// Проверяем, что конфликт назначения администратора на локацию возвращается как db.ErrConflict.
func TestAssignAdminToLocation_ExceptionHandling_DuplicateAssignmentReturnsErrConflict(t *testing.T) {
	mock := newMock(t)
	repo := newUserRepo(mock)

	expectUserByID(mock, 7, 7, "admin", "admin@example.com", "hash", RoleAdmin)

	mock.ExpectExec(regexp.QuoteMeta(queryAssignAdminToLocation)).
		WithArgs(7, 10).
		WillReturnError(&pgconn.PgError{Code: "23505"})

	err := repo.AssignAdminToLocation(context.Background(), 7, 10)

	if !errors.Is(err, db.ErrConflict) {
		t.Fatalf("got error %v, want %v", err, db.ErrConflict)
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем успешное удаление назначения администратора на локацию.
func TestDeleteAdminLocationAssignment_Scenario_Success(t *testing.T) {
	mock := newMock(t)
	repo := newUserRepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(queryDeleteAdminLocationAssignment)).
		WithArgs(7, 10).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err := repo.DeleteAdminLocationAssignment(context.Background(), 7, 10)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Техника тест-дизайна: граничные значения.
// Проверяем недопустимые adminID и locationID при удалении назначения.
func TestDeleteAdminLocationAssignment_BoundaryValues_InvalidIDsReturnErrInvalidID(t *testing.T) {
	cases := []struct {
		name       string
		adminID    int
		locationID int
	}{
		{"admin id is zero", 0, 10},
		{"admin id is negative", -1, 10},
		{"location id is zero", 7, 0},
		{"location id is negative", 7, -1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newMock(t)
			repo := newUserRepo(mock)

			err := repo.DeleteAdminLocationAssignment(context.Background(), tc.adminID, tc.locationID)

			if !errors.Is(err, db.ErrInvalidID) {
				t.Fatalf("got error %v, want %v", err, db.ErrInvalidID)
			}
		})
	}
}

// Техника тест-дизайна: таблица решений.
// Проверяем результат DELETE в зависимости от количества удалённых строк.
func TestDeleteAdminLocationAssignment_DecisionTable_RowsAffected(t *testing.T) {
	cases := []struct {
		name         string
		rowsAffected int64
		wantErr      error
	}{
		{"one row deleted means success", 1, nil},
		{"zero rows deleted means not found", 0, db.ErrNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := newMock(t)
			repo := newUserRepo(mock)

			mock.ExpectExec(regexp.QuoteMeta(queryDeleteAdminLocationAssignment)).
				WithArgs(7, 10).
				WillReturnResult(pgxmock.NewResult("DELETE", tc.rowsAffected))

			err := repo.DeleteAdminLocationAssignment(context.Background(), 7, 10)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// Техника тест-дизайна: обработка исключений.
// Проверяем, что ошибка базы данных при удалении назначения возвращается вызывающему коду.
func TestDeleteAdminLocationAssignment_ExceptionHandling_DBErrorPropagates(t *testing.T) {
	mock := newMock(t)
	repo := newUserRepo(mock)

	dbErr := fmt.Errorf("delete failed")
	mock.ExpectExec(regexp.QuoteMeta(queryDeleteAdminLocationAssignment)).
		WithArgs(7, 10).
		WillReturnError(dbErr)

	err := repo.DeleteAdminLocationAssignment(context.Background(), 7, 10)

	if !errors.Is(err, dbErr) {
		t.Fatalf("got error %v, want %v", err, dbErr)
	}
}
