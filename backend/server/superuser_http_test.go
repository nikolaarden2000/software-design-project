package server

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nikolaarden2000/software-design-project/backend/auth"
	"github.com/nikolaarden2000/software-design-project/backend/companies"
	"github.com/nikolaarden2000/software-design-project/backend/db"
	"github.com/nikolaarden2000/software-design-project/backend/locations"
	"github.com/nikolaarden2000/software-design-project/backend/rooms"
	"github.com/nikolaarden2000/software-design-project/backend/users"
)

type testCompanyRepo struct {
	listCompanies func(ctx context.Context) ([]companies.Company, error)
	createCompany func(ctx context.Context, name, description string) (*companies.Company, error)
	existsByID    func(ctx context.Context, id int) (bool, error)
}

func (m *testCompanyRepo) ListCompanies(ctx context.Context) ([]companies.Company, error) {
	if m.listCompanies != nil {
		return m.listCompanies(ctx)
	}
	return nil, errServerTestNotConfigured
}

func (m *testCompanyRepo) CreateCompany(ctx context.Context, name, description string) (*companies.Company, error) {
	if m.createCompany != nil {
		return m.createCompany(ctx, name, description)
	}
	return nil, errServerTestNotConfigured
}

func (m *testCompanyRepo) ExistsByID(ctx context.Context, id int) (bool, error) {
	if m.existsByID != nil {
		return m.existsByID(ctx, id)
	}
	return false, errServerTestNotConfigured
}

type testUserRepo struct {
	listAdmins                    func(ctx context.Context) ([]users.Admin, error)
	assignAdminToLocation         func(ctx context.Context, adminID, locationID int) error
	deleteAdminLocationAssignment func(ctx context.Context, adminID, locationID int) error
}

func (m *testUserRepo) ListAdmins(ctx context.Context) ([]users.Admin, error) {
	if m.listAdmins != nil {
		return m.listAdmins(ctx)
	}
	return nil, errServerTestNotConfigured
}

func (m *testUserRepo) AssignAdminToLocation(ctx context.Context, adminID, locationID int) error {
	if m.assignAdminToLocation != nil {
		return m.assignAdminToLocation(ctx, adminID, locationID)
	}
	return errServerTestNotConfigured
}

func (m *testUserRepo) DeleteAdminLocationAssignment(ctx context.Context, adminID, locationID int) error {
	if m.deleteAdminLocationAssignment != nil {
		return m.deleteAdminLocationAssignment(ctx, adminID, locationID)
	}
	return errServerTestNotConfigured
}

func newSuperuserHandlersTestServer(authSvc AuthService, userRepo UserRepo, companyRepo CompanyRepo, locationRepo LocationRepo, roomRepo RoomRepo) *Server {
	return &Server{
		auth:         authSvc,
		userRepo:     userRepo,
		companyRepo:  companyRepo,
		locationRepo: locationRepo,
		roomRepo:     roomRepo,
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем получение списка компаний.
func TestListCompaniesHandler_Scenario_Success(t *testing.T) {
	companyRepo := &testCompanyRepo{
		listCompanies: func(context.Context) ([]companies.Company, error) {
			return []companies.Company{{ID: 1, Name: "Company A", LocationsCount: 2}}, nil
		},
	}

	s := newSuperuserHandlersTestServer(nil, nil, companyRepo, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/superuser/companies", nil)
	w := httptest.NewRecorder()

	s.ListCompaniesHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	data := readServerDataMap(t, w)
	items, ok := data["items"].([]any)
	if !ok {
		t.Fatalf("items type: got %T, want []any", data["items"])
	}
	if len(items) != 1 {
		t.Fatalf("items length: got %d, want 1", len(items))
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем создание компании.
func TestCreateCompanyHandler_Scenario_Success(t *testing.T) {
	companyRepo := &testCompanyRepo{
		createCompany: func(_ context.Context, name, description string) (*companies.Company, error) {
			if name != "Company A" || description != "Description" {
				t.Fatalf("unexpected args: name=%q description=%q", name, description)
			}
			return &companies.Company{ID: 1, Name: name, Description: description}, nil
		},
	}

	s := newSuperuserHandlersTestServer(nil, nil, companyRepo, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/superuser/companies", bytes.NewBufferString(`{"name":"Company A","description":"Description"}`))
	w := httptest.NewRecorder()

	s.CreateCompanyHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusCreated)
	}
}

// Техника тест-дизайна: таблица решений.
// Проверяем соответствие ошибок CreateCompany HTTP-ответам.
func TestCreateCompanyHandler_DecisionTable_RepoErrors(t *testing.T) {
	cases := []struct {
		name       string
		repoErr    error
		wantStatus int
		wantCode   string
	}{
		{"invalid argument", db.ErrInvalidArgument, http.StatusBadRequest, "invalid_request"},
		{"conflict", db.ErrConflict, http.StatusConflict, "company_already_exists"},
		{"unknown error", errors.New("db failed"), http.StatusInternalServerError, "internal_error"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			companyRepo := &testCompanyRepo{
				createCompany: func(context.Context, string, string) (*companies.Company, error) {
					return nil, tc.repoErr
				},
			}

			s := newSuperuserHandlersTestServer(nil, nil, companyRepo, nil, nil)
			req := httptest.NewRequest(http.MethodPost, "/api/superuser/companies", bytes.NewBufferString(`{"name":"Company A","description":"Description"}`))
			w := httptest.NewRecorder()

			s.CreateCompanyHandler(w, req)

			if w.Code != tc.wantStatus {
				t.Fatalf("status: got %d, want %d", w.Code, tc.wantStatus)
			}

			apiErr := readServerError(t, w)
			if apiErr.Code != tc.wantCode {
				t.Fatalf("error code: got %q, want %q", apiErr.Code, tc.wantCode)
			}
		})
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем получение списка локаций с фильтрами.
func TestListLocationsHandler_Scenario_SuccessWithFilters(t *testing.T) {
	locationRepo := &testLocationRepo{
		listLocations: func(_ context.Context, companyID *int, city *string) ([]locations.Location, error) {
			if companyID == nil || *companyID != 10 {
				t.Fatalf("companyID: got %v, want 10", companyID)
			}
			if city == nil || *city != "Москва" {
				t.Fatalf("city: got %v, want Москва", city)
			}
			return []locations.Location{{ID: 1, CompanyID: 10, City: "Москва"}}, nil
		},
	}

	s := newSuperuserHandlersTestServer(nil, nil, nil, locationRepo, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/superuser/locations?company_id=10&city=Москва", nil)
	w := httptest.NewRecorder()

	s.ListLocationsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем создание локации после успешной проверки компании.
func TestCreateLocationHandler_Scenario_Success(t *testing.T) {
	companyRepo := &testCompanyRepo{
		existsByID: func(_ context.Context, id int) (bool, error) {
			if id != 10 {
				t.Fatalf("companyID: got %d, want 10", id)
			}
			return true, nil
		},
	}

	locationRepo := &testLocationRepo{
		createLocation: func(_ context.Context, companyID int, city string, address string, lat float64, lng float64, timezone string) (*locations.Location, error) {
			if companyID != 10 || city != "Москва" || address != "Москва, Тверская 10" {
				t.Fatalf("unexpected args: companyID=%d city=%q address=%q", companyID, city, address)
			}
			return &locations.Location{ID: 100, CompanyID: companyID, City: city, Address: address, Lat: lat, Lng: lng, Timezone: timezone}, nil
		},
	}

	s := newSuperuserHandlersTestServer(nil, nil, companyRepo, locationRepo, nil)
	body := `{"company_id":10,"city":"Москва","address":"Москва, Тверская 10","lat":55.75,"lng":37.61,"timezone":"Europe/Moscow"}`
	req := httptest.NewRequest(http.MethodPost, "/api/superuser/locations", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	s.CreateLocationHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusCreated)
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем создание локации для несуществующей компании.
func TestCreateLocationHandler_EquivalenceClasses_CompanyNotFound(t *testing.T) {
	companyRepo := &testCompanyRepo{
		existsByID: func(context.Context, int) (bool, error) {
			return false, nil
		},
	}

	s := newSuperuserHandlersTestServer(nil, nil, companyRepo, &testLocationRepo{}, nil)
	body := `{"company_id":10,"city":"Москва","address":"Москва, Тверская 10","lat":55.75,"lng":37.61,"timezone":"Europe/Moscow"}`
	req := httptest.NewRequest(http.MethodPost, "/api/superuser/locations", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	s.CreateLocationHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusNotFound)
	}

	apiErr := readServerError(t, w)
	if apiErr.Code != "company_not_found" {
		t.Fatalf("error code: got %q, want company_not_found", apiErr.Code)
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем получение списка администраторов.
func TestListAdminsHandler_Scenario_Success(t *testing.T) {
	userRepo := &testUserRepo{
		listAdmins: func(context.Context) ([]users.Admin, error) {
			return []users.Admin{{ID: 7, Username: "admin", Email: "admin@example.com", Role: users.RoleAdmin}}, nil
		},
	}

	s := newSuperuserHandlersTestServer(nil, userRepo, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/superuser/admins", nil)
	w := httptest.NewRecorder()

	s.ListAdminsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем создание администратора через auth.RegisterWithRole.
func TestCreateAdminHandler_Scenario_Success(t *testing.T) {
	authSvc := &testAuthService{
		registerWithRole: func(_ context.Context, username, email, password, role string) (int, error) {
			if username != "admin" || email != "admin@example.com" || password != "password1" || role != users.RoleAdmin {
				t.Fatalf("unexpected args: username=%q email=%q password=%q role=%q", username, email, password, role)
			}
			return 7, nil
		},
		getUserByID: func(_ context.Context, id int) (*users.User, error) {
			if id != 7 {
				t.Fatalf("id: got %d, want 7", id)
			}
			return &users.User{ID: 7, Username: "admin", Email: "admin@example.com", Role: users.RoleAdmin}, nil
		},
	}

	s := newSuperuserHandlersTestServer(authSvc, nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/superuser/admins", bytes.NewBufferString(`{"username":"admin","email":"admin@example.com","password":"password1"}`))
	w := httptest.NewRecorder()

	s.CreateAdminHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusCreated)
	}
}

// Техника тест-дизайна: таблица решений.
// Проверяем соответствие ошибок RegisterWithRole HTTP-ответам.
func TestCreateAdminHandler_DecisionTable_RegisterErrors(t *testing.T) {
	cases := []struct {
		name       string
		authErr    error
		wantStatus int
		wantCode   string
	}{
		{"email exists", auth.ErrEmailExists, http.StatusConflict, "email_already_exists"},
		{"invalid email", auth.ErrInvalidEmail, http.StatusBadRequest, "invalid_email"},
		{"empty email", auth.ErrEmptyEmail, http.StatusBadRequest, "invalid_request"},
		{"empty username", auth.ErrEmptyUsername, http.StatusBadRequest, "invalid_request"},
		{"password too short", auth.ErrPasswordTooShort, http.StatusBadRequest, "invalid_password"},
		{"password too long", auth.ErrPasswordTooLong, http.StatusBadRequest, "invalid_password"},
		{"password invalid chars", auth.ErrPasswordInvalidChars, http.StatusBadRequest, "invalid_password"},
		{"unknown error", errors.New("db failed"), http.StatusInternalServerError, "internal_error"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			authSvc := &testAuthService{
				registerWithRole: func(context.Context, string, string, string, string) (int, error) {
					return 0, tc.authErr
				},
			}

			s := newSuperuserHandlersTestServer(authSvc, nil, nil, nil, nil)
			req := httptest.NewRequest(http.MethodPost, "/api/superuser/admins", bytes.NewBufferString(`{"username":"admin","email":"admin@example.com","password":"password1"}`))
			w := httptest.NewRecorder()

			s.CreateAdminHandler(w, req)

			if w.Code != tc.wantStatus {
				t.Fatalf("status: got %d, want %d", w.Code, tc.wantStatus)
			}

			apiErr := readServerError(t, w)
			if apiErr.Code != tc.wantCode {
				t.Fatalf("error code: got %q, want %q", apiErr.Code, tc.wantCode)
			}
		})
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем назначение администратора на локацию.
func TestAssignAdminToLocationHandler_Scenario_Success(t *testing.T) {
	userRepo := &testUserRepo{
		assignAdminToLocation: func(_ context.Context, adminID, locationID int) error {
			if adminID != 7 || locationID != 10 {
				t.Fatalf("unexpected args: adminID=%d locationID=%d", adminID, locationID)
			}
			return nil
		},
	}

	s := newSuperuserHandlersTestServer(nil, userRepo, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/superuser/admins/7/locations", bytes.NewBufferString(`{"location_id":10}`))
	req.SetPathValue("admin_id", "7")
	w := httptest.NewRecorder()

	s.AssignAdminToLocationHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}
}

// Техника тест-дизайна: таблица решений.
// Проверяем соответствие ошибок AssignAdminToLocation HTTP-ответам.
func TestAssignAdminToLocationHandler_DecisionTable_RepoErrors(t *testing.T) {
	cases := []struct {
		name       string
		repoErr    error
		wantStatus int
		wantCode   string
	}{
		{"conflict", db.ErrConflict, http.StatusConflict, "assignment_already_exists"},
		{"not found", db.ErrNotFound, http.StatusNotFound, "admin_not_found"},
		{"invalid id", db.ErrInvalidID, http.StatusBadRequest, "invalid_request"},
		{"unknown error", errors.New("db failed"), http.StatusInternalServerError, "internal_error"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			userRepo := &testUserRepo{
				assignAdminToLocation: func(context.Context, int, int) error {
					return tc.repoErr
				},
			}

			s := newSuperuserHandlersTestServer(nil, userRepo, nil, nil, nil)
			req := httptest.NewRequest(http.MethodPost, "/api/superuser/admins/7/locations", bytes.NewBufferString(`{"location_id":10}`))
			req.SetPathValue("admin_id", "7")
			w := httptest.NewRecorder()

			s.AssignAdminToLocationHandler(w, req)

			if w.Code != tc.wantStatus {
				t.Fatalf("status: got %d, want %d", w.Code, tc.wantStatus)
			}

			apiErr := readServerError(t, w)
			if apiErr.Code != tc.wantCode {
				t.Fatalf("error code: got %q, want %q", apiErr.Code, tc.wantCode)
			}
		})
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем удаление назначения администратора на локацию.
func TestDeleteAdminLocationAssignmentHandler_Scenario_Success(t *testing.T) {
	userRepo := &testUserRepo{
		deleteAdminLocationAssignment: func(_ context.Context, adminID, locationID int) error {
			if adminID != 7 || locationID != 10 {
				t.Fatalf("unexpected args: adminID=%d locationID=%d", adminID, locationID)
			}
			return nil
		},
	}

	s := newSuperuserHandlersTestServer(nil, userRepo, nil, nil, nil)
	req := httptest.NewRequest(http.MethodDelete, "/api/superuser/admins/7/locations/10", nil)
	req.SetPathValue("admin_id", "7")
	req.SetPathValue("location_id", "10")
	w := httptest.NewRecorder()

	s.DeleteAdminLocationAssignmentHandler(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusNoContent)
	}
	if w.Body.String() != "" {
		t.Fatalf("body: got %q, want empty body", w.Body.String())
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем получение помещений на модерации.
func TestModerationRoomsHandler_Scenario_Success(t *testing.T) {
	roomRepo := &testRoomRepo{
		listModerationRooms: func(context.Context) ([]rooms.ModerationRoom, error) {
			return []rooms.ModerationRoom{{ID: 100, Title: "Room A", Status: rooms.StatusPending}}, nil
		},
	}

	s := newSuperuserHandlersTestServer(nil, nil, nil, nil, roomRepo)
	req := httptest.NewRequest(http.MethodGet, "/api/superuser/rooms/moderation", nil)
	w := httptest.NewRecorder()

	s.ModerationRoomsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем одобрение помещения.
func TestApproveRoomHandler_Scenario_Success(t *testing.T) {
	roomRepo := &testRoomRepo{
		approveRoom: func(_ context.Context, roomID int) error {
			if roomID != 100 {
				t.Fatalf("roomID: got %d, want 100", roomID)
			}
			return nil
		},
	}

	s := newSuperuserHandlersTestServer(nil, nil, nil, nil, roomRepo)
	req := httptest.NewRequest(http.MethodPost, "/api/superuser/rooms/100/approve", nil)
	req.SetPathValue("room_id", "100")
	w := httptest.NewRecorder()

	s.ApproveRoomHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	data := readServerDataMap(t, w)
	if data["status"] != rooms.StatusPublished {
		t.Fatalf("status: got %v, want %s", data["status"], rooms.StatusPublished)
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем отклонение помещения.
func TestRejectRoomHandler_Scenario_Success(t *testing.T) {
	roomRepo := &testRoomRepo{
		rejectRoom: func(_ context.Context, roomID int, reason string) error {
			if roomID != 100 || reason != "bad photos" {
				t.Fatalf("unexpected args: roomID=%d reason=%q", roomID, reason)
			}
			return nil
		},
	}

	s := newSuperuserHandlersTestServer(nil, nil, nil, nil, roomRepo)
	req := httptest.NewRequest(http.MethodPost, "/api/superuser/rooms/100/reject", bytes.NewBufferString(`{"reason":"bad photos"}`))
	req.SetPathValue("room_id", "100")
	w := httptest.NewRecorder()

	s.RejectRoomHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	data := readServerDataMap(t, w)
	if data["status"] != rooms.StatusRejected {
		t.Fatalf("status: got %v, want %s", data["status"], rooms.StatusRejected)
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем архивирование помещения суперпользователем.
func TestArchiveRoomHandler_Scenario_Success(t *testing.T) {
	roomRepo := &testRoomRepo{
		archiveRoom: func(_ context.Context, roomID int) error {
			if roomID != 100 {
				t.Fatalf("roomID: got %d, want 100", roomID)
			}
			return nil
		},
	}

	s := newSuperuserHandlersTestServer(nil, nil, nil, nil, roomRepo)
	req := httptest.NewRequest(http.MethodPost, "/api/superuser/rooms/100/archive", nil)
	req.SetPathValue("room_id", "100")
	w := httptest.NewRecorder()

	s.ArchiveRoomHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	data := readServerDataMap(t, w)
	if data["status"] != rooms.StatusArchived {
		t.Fatalf("status: got %v, want %s", data["status"], rooms.StatusArchived)
	}
}

// Техника тест-дизайна: таблица решений.
// Проверяем общий маппинг ошибок модерации помещения.
func TestApproveRoomHandler_DecisionTable_ModerationErrors(t *testing.T) {
	cases := []struct {
		name       string
		repoErr    error
		wantStatus int
		wantCode   string
	}{
		{"invalid id", db.ErrInvalidID, http.StatusBadRequest, "invalid_room_id"},
		{"not found", db.ErrNotFound, http.StatusNotFound, "room_not_found"},
		{"conflict", db.ErrConflict, http.StatusConflict, "cannot_approve_room"},
		{"unknown error", errors.New("db failed"), http.StatusInternalServerError, "internal_error"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			roomRepo := &testRoomRepo{
				approveRoom: func(context.Context, int) error {
					return tc.repoErr
				},
			}

			s := newSuperuserHandlersTestServer(nil, nil, nil, nil, roomRepo)
			req := httptest.NewRequest(http.MethodPost, "/api/superuser/rooms/100/approve", nil)
			req.SetPathValue("room_id", "100")
			w := httptest.NewRecorder()

			s.ApproveRoomHandler(w, req)

			if w.Code != tc.wantStatus {
				t.Fatalf("status: got %d, want %d", w.Code, tc.wantStatus)
			}

			apiErr := readServerError(t, w)
			if apiErr.Code != tc.wantCode {
				t.Fatalf("error code: got %q, want %q", apiErr.Code, tc.wantCode)
			}
		})
	}
}
