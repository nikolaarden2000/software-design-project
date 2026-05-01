package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nikolaarden2000/software-design-project/backend/auth"
	"github.com/nikolaarden2000/software-design-project/backend/bookings"
	"github.com/nikolaarden2000/software-design-project/backend/db"
	"github.com/nikolaarden2000/software-design-project/backend/httpapi"
	"github.com/nikolaarden2000/software-design-project/backend/rooms"
	"github.com/nikolaarden2000/software-design-project/backend/users"
)

var errServerTestNotConfigured = errors.New("server test mock: method not configured")

type testAuthService struct {
	register         func(ctx context.Context, username, email, password string) (int, error)
	registerWithRole func(ctx context.Context, username, email, password, role string) (int, error)
	login            func(ctx context.Context, email, password string, w http.ResponseWriter) (*users.User, error)
	logout           func(w http.ResponseWriter, r *http.Request)
	getUserByID      func(ctx context.Context, id int) (*users.User, error)
}

func (m *testAuthService) Register(ctx context.Context, username, email, password string) (int, error) {
	if m.register != nil {
		return m.register(ctx, username, email, password)
	}
	return 0, errServerTestNotConfigured
}

func (m *testAuthService) RegisterWithRole(ctx context.Context, username, email, password, role string) (int, error) {
	if m.registerWithRole != nil {
		return m.registerWithRole(ctx, username, email, password, role)
	}
	return 0, errServerTestNotConfigured
}

func (m *testAuthService) Login(ctx context.Context, email, password string, w http.ResponseWriter) (*users.User, error) {
	if m.login != nil {
		return m.login(ctx, email, password, w)
	}
	return nil, errServerTestNotConfigured
}

func (m *testAuthService) Logout(w http.ResponseWriter, r *http.Request) {
	if m.logout != nil {
		m.logout(w, r)
	}
}

func (m *testAuthService) GetUserByID(ctx context.Context, id int) (*users.User, error) {
	if m.getUserByID != nil {
		return m.getUserByID(ctx, id)
	}
	return nil, errServerTestNotConfigured
}

type testRoomRepo struct {
	getRoomsBatchByCity func(ctx context.Context, lastID, limit int, city string) ([]rooms.Room, error)
	getCompaniesByCity  func(ctx context.Context, city string) ([]string, error)
	getRoomPageData     func(ctx context.Context, roomID int) (*rooms.RoomPageData, error)

	listAdminRooms   func(ctx context.Context, adminID int, includeAll bool, locationID *int, status *string) ([]rooms.AdminRoomListItem, error)
	getAdminRoom     func(ctx context.Context, adminID int, includeAll bool, roomID int) (*rooms.AdminRoomDetails, error)
	createAdminRoom  func(ctx context.Context, creatorID int, includeAll bool, input rooms.AdminRoomInput) (*rooms.AdminRoomListItem, error)
	updateAdminRoom  func(ctx context.Context, adminID int, includeAll bool, roomID int, input rooms.AdminRoomInput) error
	submitAdminRoom  func(ctx context.Context, adminID int, includeAll bool, roomID int) error
	archiveAdminRoom func(ctx context.Context, adminID int, includeAll bool, roomID int, mode string, now time.Time) (*rooms.AdminRoomArchiveResult, error)

	listModerationRooms func(ctx context.Context) ([]rooms.ModerationRoom, error)
	approveRoom         func(ctx context.Context, roomID int) error
	rejectRoom          func(ctx context.Context, roomID int, reason string) error
	archiveRoom         func(ctx context.Context, roomID int) error
}

func (m *testRoomRepo) GetRoomsBatchByCity(ctx context.Context, lastID, limit int, city string) ([]rooms.Room, error) {
	if m.getRoomsBatchByCity != nil {
		return m.getRoomsBatchByCity(ctx, lastID, limit, city)
	}
	return nil, errServerTestNotConfigured
}

func (m *testRoomRepo) GetCompaniesByCity(ctx context.Context, city string) ([]string, error) {
	if m.getCompaniesByCity != nil {
		return m.getCompaniesByCity(ctx, city)
	}
	return nil, errServerTestNotConfigured
}

func (m *testRoomRepo) GetRoomPageData(ctx context.Context, roomID int) (*rooms.RoomPageData, error) {
	if m.getRoomPageData != nil {
		return m.getRoomPageData(ctx, roomID)
	}
	return nil, errServerTestNotConfigured
}

func (m *testRoomRepo) ListAdminRooms(ctx context.Context, adminID int, includeAll bool, locationID *int, status *string) ([]rooms.AdminRoomListItem, error) {
	if m.listAdminRooms != nil {
		return m.listAdminRooms(ctx, adminID, includeAll, locationID, status)
	}
	return nil, errServerTestNotConfigured
}

func (m *testRoomRepo) GetAdminRoom(ctx context.Context, adminID int, includeAll bool, roomID int) (*rooms.AdminRoomDetails, error) {
	if m.getAdminRoom != nil {
		return m.getAdminRoom(ctx, adminID, includeAll, roomID)
	}
	return nil, errServerTestNotConfigured
}

func (m *testRoomRepo) CreateAdminRoom(ctx context.Context, creatorID int, includeAll bool, input rooms.AdminRoomInput) (*rooms.AdminRoomListItem, error) {
	if m.createAdminRoom != nil {
		return m.createAdminRoom(ctx, creatorID, includeAll, input)
	}
	return nil, errServerTestNotConfigured
}

func (m *testRoomRepo) UpdateAdminRoom(ctx context.Context, adminID int, includeAll bool, roomID int, input rooms.AdminRoomInput) error {
	if m.updateAdminRoom != nil {
		return m.updateAdminRoom(ctx, adminID, includeAll, roomID, input)
	}
	return errServerTestNotConfigured
}

func (m *testRoomRepo) SubmitAdminRoom(ctx context.Context, adminID int, includeAll bool, roomID int) error {
	if m.submitAdminRoom != nil {
		return m.submitAdminRoom(ctx, adminID, includeAll, roomID)
	}
	return errServerTestNotConfigured
}

func (m *testRoomRepo) ArchiveAdminRoom(ctx context.Context, adminID int, includeAll bool, roomID int, mode string, now time.Time) (*rooms.AdminRoomArchiveResult, error) {
	if m.archiveAdminRoom != nil {
		return m.archiveAdminRoom(ctx, adminID, includeAll, roomID, mode, now)
	}
	return nil, errServerTestNotConfigured
}

func (m *testRoomRepo) ListModerationRooms(ctx context.Context) ([]rooms.ModerationRoom, error) {
	if m.listModerationRooms != nil {
		return m.listModerationRooms(ctx)
	}
	return nil, errServerTestNotConfigured
}

func (m *testRoomRepo) ApproveRoom(ctx context.Context, roomID int) error {
	if m.approveRoom != nil {
		return m.approveRoom(ctx, roomID)
	}
	return errServerTestNotConfigured
}

func (m *testRoomRepo) RejectRoom(ctx context.Context, roomID int, reason string) error {
	if m.rejectRoom != nil {
		return m.rejectRoom(ctx, roomID, reason)
	}
	return errServerTestNotConfigured
}

func (m *testRoomRepo) ArchiveRoom(ctx context.Context, roomID int) error {
	if m.archiveRoom != nil {
		return m.archiveRoom(ctx, roomID)
	}
	return errServerTestNotConfigured
}

type testBookingRepo struct {
	getRoomAvailability func(ctx context.Context, roomID, days int, now time.Time) ([]rooms.DateAvailability, error)
	createBooking       func(ctx context.Context, userID, roomID int, date string, slots []string, now time.Time) (int, error)
	getUserBookings     func(ctx context.Context, userID int, now time.Time) ([]bookings.BookingHistoryItem, error)
	cancelBooking       func(ctx context.Context, bookingID, userID int, now time.Time) error

	listAdminBookings  func(ctx context.Context, adminID int, includeAll bool, locationID *int, roomID *int, status *string, now time.Time) ([]bookings.AdminBookingItem, error)
	cancelAdminBooking func(ctx context.Context, adminID int, includeAll bool, bookingID int, now time.Time) error
}

func (m *testBookingRepo) GetRoomAvailability(ctx context.Context, roomID, days int, now time.Time) ([]rooms.DateAvailability, error) {
	if m.getRoomAvailability != nil {
		return m.getRoomAvailability(ctx, roomID, days, now)
	}
	return nil, errServerTestNotConfigured
}

func (m *testBookingRepo) CreateBooking(ctx context.Context, userID, roomID int, date string, slots []string, now time.Time) (int, error) {
	if m.createBooking != nil {
		return m.createBooking(ctx, userID, roomID, date, slots, now)
	}
	return 0, errServerTestNotConfigured
}

func (m *testBookingRepo) GetUserBookings(ctx context.Context, userID int, now time.Time) ([]bookings.BookingHistoryItem, error) {
	if m.getUserBookings != nil {
		return m.getUserBookings(ctx, userID, now)
	}
	return nil, errServerTestNotConfigured
}

func (m *testBookingRepo) CancelBooking(ctx context.Context, bookingID, userID int, now time.Time) error {
	if m.cancelBooking != nil {
		return m.cancelBooking(ctx, bookingID, userID, now)
	}
	return errServerTestNotConfigured
}

func (m *testBookingRepo) ListAdminBookings(ctx context.Context, adminID int, includeAll bool, locationID *int, roomID *int, status *string, now time.Time) ([]bookings.AdminBookingItem, error) {
	if m.listAdminBookings != nil {
		return m.listAdminBookings(ctx, adminID, includeAll, locationID, roomID, status, now)
	}
	return nil, errServerTestNotConfigured
}

func (m *testBookingRepo) CancelAdminBooking(ctx context.Context, adminID int, includeAll bool, bookingID int, now time.Time) error {
	if m.cancelAdminBooking != nil {
		return m.cancelAdminBooking(ctx, adminID, includeAll, bookingID, now)
	}
	return errServerTestNotConfigured
}

func newHTTPHandlersTestServer(authSvc AuthService, roomRepo RoomRepo, bookingRepo BookingRepo) *Server {
	return &Server{
		auth:        authSvc,
		roomRepo:    roomRepo,
		bookingRepo: bookingRepo,
	}
}

func requestWithUser(req *http.Request, u *users.User) *http.Request {
	ctx := context.WithValue(req.Context(), auth.KeyUser, u)
	return req.WithContext(ctx)
}

func readServerResponse(t *testing.T, w *httptest.ResponseRecorder) httpapi.Response {
	t.Helper()

	var body httpapi.Response
	if err := json.NewDecoder(w.Result().Body).Decode(&body); err != nil {
		t.Fatalf("Decode response body: %v", err)
	}

	return body
}

func readServerDataMap(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	body := readServerResponse(t, w)

	data, ok := body.Data.(map[string]any)
	if !ok {
		t.Fatalf("Data type: got %T, want map[string]any", body.Data)
	}

	return data
}

func readServerError(t *testing.T, w *httptest.ResponseRecorder) *httpapi.APIError {
	t.Helper()

	body := readServerResponse(t, w)

	if body.Error == nil {
		t.Fatal("expected API error, got nil")
	}

	return body.Error
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем преобразование внутренней модели пользователя в DTO без password_hash.
func TestToUserDTO_Scenario_MapsPublicFields(t *testing.T) {
	u := &users.User{
		ID:       7,
		Username: "alice",
		Email:    "alice@example.com",
		Password: "secret-hash",
		Role:     users.RoleAdmin,
	}

	dto := toUserDTO(u)

	if dto.ID != 7 || dto.Username != "alice" || dto.Email != "alice@example.com" || dto.Role != users.RoleAdmin {
		t.Fatalf("unexpected dto: %+v", dto)
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем успешную регистрацию: handler вызывает auth.Register, затем GetUserByID и возвращает user DTO.
func TestRegisterHandler_Scenario_Success(t *testing.T) {
	authSvc := &testAuthService{
		register: func(_ context.Context, username, email, password string) (int, error) {
			if username != "alice" || email != "alice@example.com" || password != "password1" {
				t.Fatalf("unexpected register args: username=%q email=%q password=%q", username, email, password)
			}
			return 42, nil
		},
		getUserByID: func(_ context.Context, id int) (*users.User, error) {
			if id != 42 {
				t.Fatalf("GetUserByID id: got %d, want 42", id)
			}
			return &users.User{ID: 42, Username: "alice", Email: "alice@example.com", Role: users.RoleUser}, nil
		},
	}
	s := newHTTPHandlersTestServer(authSvc, nil, nil)

	body := bytes.NewBufferString(`{"username":"alice","email":"alice@example.com","password":"password1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/register", body)
	w := httptest.NewRecorder()

	s.RegisterHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusCreated)
	}

	data := readServerDataMap(t, w)
	userData, ok := data["user"].(map[string]any)
	if !ok {
		t.Fatalf("user type: got %T, want map[string]any", data["user"])
	}
	if userData["email"] != "alice@example.com" {
		t.Fatalf("email: got %v, want alice@example.com", userData["email"])
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем класс некорректного JSON body.
func TestRegisterHandler_EquivalenceClasses_InvalidJSONReturnsBadRequest(t *testing.T) {
	s := newHTTPHandlersTestServer(&testAuthService{}, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewBufferString(`{bad json`))
	w := httptest.NewRecorder()

	s.RegisterHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}

	apiErr := readServerError(t, w)
	if apiErr.Code != "invalid_request" {
		t.Fatalf("error code: got %q, want invalid_request", apiErr.Code)
	}
}

// Техника тест-дизайна: таблица решений.
// Проверяем соответствие ошибок auth.Register HTTP-статусам и API-кодам.
func TestRegisterHandler_DecisionTable_RegisterErrors(t *testing.T) {
	cases := []struct {
		name       string
		authErr    error
		wantStatus int
		wantCode   string
	}{
		{"email already exists", auth.ErrEmailExists, http.StatusConflict, "email_already_exists"},
		{"empty username", auth.ErrEmptyUsername, http.StatusBadRequest, "empty_username"},
		{"empty email", auth.ErrEmptyEmail, http.StatusBadRequest, "empty_email"},
		{"invalid email", auth.ErrInvalidEmail, http.StatusBadRequest, "invalid_email"},
		{"password too short", auth.ErrPasswordTooShort, http.StatusBadRequest, "password_too_short"},
		{"password too long", auth.ErrPasswordTooLong, http.StatusBadRequest, "password_too_long"},
		{"password invalid chars", auth.ErrPasswordInvalidChars, http.StatusBadRequest, "password_invalid_chars"},
		{"unknown error", errors.New("db failed"), http.StatusInternalServerError, "internal_error"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			authSvc := &testAuthService{
				register: func(context.Context, string, string, string) (int, error) {
					return 0, tc.authErr
				},
			}
			s := newHTTPHandlersTestServer(authSvc, nil, nil)

			req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewBufferString(`{"username":"alice","email":"alice@example.com","password":"password1"}`))
			w := httptest.NewRecorder()

			s.RegisterHandler(w, req)

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
// Проверяем успешный login и возврат user DTO.
func TestLoginHandler_Scenario_Success(t *testing.T) {
	authSvc := &testAuthService{
		login: func(_ context.Context, email, password string, w http.ResponseWriter) (*users.User, error) {
			if email != "alice@example.com" || password != "password1" {
				t.Fatalf("unexpected login args: email=%q password=%q", email, password)
			}
			http.SetCookie(w, &http.Cookie{Name: "session_id", Value: "session"})
			return &users.User{ID: 7, Username: "alice", Email: email, Role: users.RoleUser}, nil
		},
	}
	s := newHTTPHandlersTestServer(authSvc, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewBufferString(`{"email":"alice@example.com","password":"password1"}`))
	w := httptest.NewRecorder()

	s.LoginHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	data := readServerDataMap(t, w)
	userData, ok := data["user"].(map[string]any)
	if !ok {
		t.Fatalf("user type: got %T, want map[string]any", data["user"])
	}
	if userData["role"] != users.RoleUser {
		t.Fatalf("role: got %v, want %s", userData["role"], users.RoleUser)
	}
}

// Техника тест-дизайна: таблица решений.
// Проверяем соответствие ошибок auth.Login HTTP-статусам и API-кодам.
func TestLoginHandler_DecisionTable_LoginErrors(t *testing.T) {
	cases := []struct {
		name       string
		authErr    error
		wantStatus int
		wantCode   string
	}{
		{"no user", auth.ErrNoUser, http.StatusUnauthorized, "invalid_credentials"},
		{"invalid credentials", auth.ErrInvalidCredentials, http.StatusUnauthorized, "invalid_credentials"},
		{"empty email", auth.ErrEmptyEmail, http.StatusBadRequest, "empty_email"},
		{"unknown error", errors.New("db failed"), http.StatusInternalServerError, "internal_error"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			authSvc := &testAuthService{
				login: func(context.Context, string, string, http.ResponseWriter) (*users.User, error) {
					return nil, tc.authErr
				},
			}
			s := newHTTPHandlersTestServer(authSvc, nil, nil)

			req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewBufferString(`{"email":"alice@example.com","password":"password1"}`))
			w := httptest.NewRecorder()

			s.LoginHandler(w, req)

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
// Проверяем logout: вызывается auth.Logout и возвращается 204 без тела.
func TestLogoutHandler_Scenario_ReturnsNoContent(t *testing.T) {
	logoutCalled := false
	authSvc := &testAuthService{
		logout: func(http.ResponseWriter, *http.Request) {
			logoutCalled = true
		},
	}
	s := newHTTPHandlersTestServer(authSvc, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
	w := httptest.NewRecorder()

	s.LogoutHandler(w, req)

	if !logoutCalled {
		t.Fatal("auth.Logout must be called")
	}
	if w.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusNoContent)
	}
	if w.Body.String() != "" {
		t.Fatalf("body: got %q, want empty body", w.Body.String())
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем два класса /me: пользователь отсутствует и пользователь есть в контексте.
func TestMeHandler_EquivalenceClasses(t *testing.T) {
	t.Run("anonymous user", func(t *testing.T) {
		s := newHTTPHandlersTestServer(nil, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		w := httptest.NewRecorder()

		s.MeHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
		}

		data := readServerDataMap(t, w)
		if data["authenticated"] != false {
			t.Fatalf("authenticated: got %v, want false", data["authenticated"])
		}
		if data["user"] != nil {
			t.Fatalf("user: got %v, want nil", data["user"])
		}
	})

	t.Run("authenticated user", func(t *testing.T) {
		s := newHTTPHandlersTestServer(nil, nil, nil)
		u := &users.User{ID: 7, Username: "alice", Email: "alice@example.com", Role: users.RoleUser}

		req := requestWithUser(httptest.NewRequest(http.MethodGet, "/api/me", nil), u)
		w := httptest.NewRecorder()

		s.MeHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
		}

		data := readServerDataMap(t, w)
		if data["authenticated"] != true {
			t.Fatalf("authenticated: got %v, want true", data["authenticated"])
		}
	})
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем каталог комнат, значения по умолчанию и pagination-блок.
func TestRoomsHandler_Scenario_SuccessWithDefaultCityAndPagination(t *testing.T) {
	roomRepo := &testRoomRepo{
		getRoomsBatchByCity: func(_ context.Context, lastID, limit int, city string) ([]rooms.Room, error) {
			if lastID != 0 {
				t.Fatalf("lastID: got %d, want 0", lastID)
			}
			if limit != 2 {
				t.Fatalf("limit: got %d, want 2", limit)
			}
			if city != "Москва" {
				t.Fatalf("city: got %q, want Москва", city)
			}
			return []rooms.Room{{ID: 1}, {ID: 2}}, nil
		},
	}
	s := newHTTPHandlersTestServer(nil, roomRepo, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/rooms?limit=2", nil)
	w := httptest.NewRecorder()

	s.RoomsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	data := readServerDataMap(t, w)
	pagination, ok := data["pagination"].(map[string]any)
	if !ok {
		t.Fatalf("pagination type: got %T, want map[string]any", data["pagination"])
	}

	if pagination["has_more"] != true {
		t.Fatalf("has_more: got %v, want true", pagination["has_more"])
	}
	if pagination["next_after_id"].(float64) != 2 {
		t.Fatalf("next_after_id: got %v, want 2", pagination["next_after_id"])
	}
}

// Техника тест-дизайна: граничные значения.
// Проверяем верхнюю границу limit: значение больше 100 ограничивается до 100.
func TestRoomsHandler_BoundaryValues_LimitIsCappedTo100(t *testing.T) {
	roomRepo := &testRoomRepo{
		getRoomsBatchByCity: func(_ context.Context, _ int, limit int, _ string) ([]rooms.Room, error) {
			if limit != 100 {
				t.Fatalf("limit: got %d, want 100", limit)
			}
			return []rooms.Room{}, nil
		},
	}
	s := newHTTPHandlersTestServer(nil, roomRepo, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/rooms?limit=500", nil)
	w := httptest.NewRecorder()

	s.RoomsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}
}

// Техника тест-дизайна: таблица решений.
// Проверяем соответствие ошибок roomRepo.GetRoomsBatchByCity HTTP-ответам.
func TestRoomsHandler_DecisionTable_RepoErrors(t *testing.T) {
	cases := []struct {
		name       string
		repoErr    error
		wantStatus int
		wantCode   string
	}{
		{"invalid after id", db.ErrInvalidID, http.StatusBadRequest, "invalid_after_id"},
		{"invalid limit", db.ErrInvalidArgument, http.StatusBadRequest, "invalid_limit"},
		{"unknown error", errors.New("db failed"), http.StatusInternalServerError, "internal_error"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			roomRepo := &testRoomRepo{
				getRoomsBatchByCity: func(context.Context, int, int, string) ([]rooms.Room, error) {
					return nil, tc.repoErr
				},
			}
			s := newHTTPHandlersTestServer(nil, roomRepo, nil)

			req := httptest.NewRequest(http.MethodGet, "/api/rooms", nil)
			w := httptest.NewRecorder()

			s.RoomsHandler(w, req)

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
// Проверяем получение фильтров каталога по городу.
func TestRoomFiltersHandler_Scenario_Success(t *testing.T) {
	roomRepo := &testRoomRepo{
		getCompaniesByCity: func(_ context.Context, city string) ([]string, error) {
			if city != "Казань" {
				t.Fatalf("city: got %q, want Казань", city)
			}
			return []string{"Company A", "Company B"}, nil
		},
	}
	s := newHTTPHandlersTestServer(nil, roomRepo, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/rooms/filters?city=Казань", nil)
	w := httptest.NewRecorder()

	s.RoomFiltersHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	data := readServerDataMap(t, w)
	if data["city"] != "Казань" {
		t.Fatalf("city: got %v, want Казань", data["city"])
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем получение публичной карточки комнаты.
func TestRoomDetailsHandler_Scenario_Success(t *testing.T) {
	roomRepo := &testRoomRepo{
		getRoomPageData: func(_ context.Context, roomID int) (*rooms.RoomPageData, error) {
			if roomID != 10 {
				t.Fatalf("roomID: got %d, want 10", roomID)
			}
			return &rooms.RoomPageData{ID: 10, Title: "Room A"}, nil
		},
	}
	s := newHTTPHandlersTestServer(nil, roomRepo, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/rooms/10", nil)
	req.SetPathValue("id", "10")
	w := httptest.NewRecorder()

	s.RoomDetailsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}
}

// Техника тест-дизайна: таблица решений.
// Проверяем соответствие ошибок получения карточки комнаты HTTP-ответам.
func TestRoomDetailsHandler_DecisionTable_RepoErrors(t *testing.T) {
	cases := []struct {
		name       string
		repoErr    error
		wantStatus int
		wantCode   string
	}{
		{"not found", db.ErrNotFound, http.StatusNotFound, "room_not_found"},
		{"invalid id", db.ErrInvalidID, http.StatusBadRequest, "invalid_room_id"},
		{"unknown error", errors.New("db failed"), http.StatusInternalServerError, "internal_error"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			roomRepo := &testRoomRepo{
				getRoomPageData: func(context.Context, int) (*rooms.RoomPageData, error) {
					return nil, tc.repoErr
				},
			}
			s := newHTTPHandlersTestServer(nil, roomRepo, nil)

			req := httptest.NewRequest(http.MethodGet, "/api/rooms/10", nil)
			req.SetPathValue("id", "10")
			w := httptest.NewRecorder()

			s.RoomDetailsHandler(w, req)

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

// Техника тест-дизайна: граничные значения.
// Проверяем верхнюю границу days: значение больше 31 ограничивается до 31.
func TestRoomAvailabilityHandler_BoundaryValues_DaysIsCappedTo31(t *testing.T) {
	bookingRepo := &testBookingRepo{
		getRoomAvailability: func(_ context.Context, roomID, days int, _ time.Time) ([]rooms.DateAvailability, error) {
			if roomID != 10 {
				t.Fatalf("roomID: got %d, want 10", roomID)
			}
			if days != 31 {
				t.Fatalf("days: got %d, want 31", days)
			}
			return []rooms.DateAvailability{{Date: "2024-01-15", AvailableTimes: []string{"10:00"}}}, nil
		},
	}
	s := newHTTPHandlersTestServer(nil, nil, bookingRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/rooms/10/availability?days=100", nil)
	req.SetPathValue("id", "10")
	w := httptest.NewRecorder()

	s.RoomAvailabilityHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем неавторизованный запрос на создание бронирования.
func TestBookingHandler_EquivalenceClasses_UnauthorizedReturns401(t *testing.T) {
	s := newHTTPHandlersTestServer(nil, nil, &testBookingRepo{})

	req := httptest.NewRequest(http.MethodPost, "/api/bookings", bytes.NewBufferString(`{"room_id":10,"date":"2024-01-16","slots":["10:00"]}`))
	w := httptest.NewRecorder()

	s.BookingHandler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusUnauthorized)
	}

	apiErr := readServerError(t, w)
	if apiErr.Code != "unauthorized" {
		t.Fatalf("error code: got %q, want unauthorized", apiErr.Code)
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем успешное создание бронирования авторизованным пользователем.
func TestBookingHandler_Scenario_Success(t *testing.T) {
	bookingRepo := &testBookingRepo{
		createBooking: func(_ context.Context, userID, roomID int, date string, slots []string, _ time.Time) (int, error) {
			if userID != 7 || roomID != 10 || date != "2024-01-16" {
				t.Fatalf("unexpected args: userID=%d roomID=%d date=%q", userID, roomID, date)
			}
			if len(slots) != 2 || slots[0] != "10:00" || slots[1] != "11:00" {
				t.Fatalf("unexpected slots: %#v", slots)
			}
			return 100, nil
		},
	}
	s := newHTTPHandlersTestServer(nil, nil, bookingRepo)

	u := &users.User{ID: 7, Username: "alice", Email: "alice@example.com", Role: users.RoleUser}
	req := httptest.NewRequest(http.MethodPost, "/api/bookings", bytes.NewBufferString(`{"room_id":10,"date":"2024-01-16","slots":["10:00","11:00"]}`))
	req = requestWithUser(req, u)
	w := httptest.NewRecorder()

	s.BookingHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusCreated)
	}

	data := readServerDataMap(t, w)
	if data["status"] != "booked" {
		t.Fatalf("status: got %v, want booked", data["status"])
	}
	if data["id"].(float64) != 100 {
		t.Fatalf("id: got %v, want 100", data["id"])
	}
}

// Техника тест-дизайна: таблица решений.
// Проверяем соответствие ошибок CreateBooking HTTP-ответам.
func TestBookingHandler_DecisionTable_CreateBookingErrors(t *testing.T) {
	cases := []struct {
		name       string
		repoErr    error
		wantStatus int
		wantCode   string
	}{
		{"slot already booked", db.ErrConflict, http.StatusConflict, "slot_already_booked"},
		{"invalid id", db.ErrInvalidID, http.StatusBadRequest, "invalid_id"},
		{"invalid argument", db.ErrInvalidArgument, http.StatusBadRequest, "invalid_booking_parameters"},
		{"not consecutive", db.ErrNotConsecutiveSlots, http.StatusBadRequest, "slots_must_be_consecutive"},
		{"room not found", db.ErrNotFound, http.StatusNotFound, "room_not_found"},
		{"unknown error", errors.New("db failed"), http.StatusInternalServerError, "internal_error"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bookingRepo := &testBookingRepo{
				createBooking: func(context.Context, int, int, string, []string, time.Time) (int, error) {
					return 0, tc.repoErr
				},
			}
			s := newHTTPHandlersTestServer(nil, nil, bookingRepo)

			u := &users.User{ID: 7, Username: "alice", Email: "alice@example.com", Role: users.RoleUser}
			req := httptest.NewRequest(http.MethodPost, "/api/bookings", bytes.NewBufferString(`{"room_id":10,"date":"2024-01-16","slots":["10:00"]}`))
			req = requestWithUser(req, u)
			w := httptest.NewRecorder()

			s.BookingHandler(w, req)

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
// Проверяем получение истории бронирований текущего пользователя.
func TestMyBookingsHandler_Scenario_Success(t *testing.T) {
	bookingRepo := &testBookingRepo{
		getUserBookings: func(_ context.Context, userID int, _ time.Time) ([]bookings.BookingHistoryItem, error) {
			if userID != 7 {
				t.Fatalf("userID: got %d, want 7", userID)
			}
			return []bookings.BookingHistoryItem{{ID: 1, RoomID: 10, Status: "booked"}}, nil
		},
	}
	s := newHTTPHandlersTestServer(nil, nil, bookingRepo)

	u := &users.User{ID: 7, Username: "alice", Email: "alice@example.com", Role: users.RoleUser}
	req := requestWithUser(httptest.NewRequest(http.MethodGet, "/api/me/bookings", nil), u)
	w := httptest.NewRecorder()

	s.MyBookingsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем успешную отмену бронирования текущим пользователем.
func TestCancelBookingHandler_Scenario_Success(t *testing.T) {
	bookingRepo := &testBookingRepo{
		cancelBooking: func(_ context.Context, bookingID, userID int, _ time.Time) error {
			if bookingID != 100 || userID != 7 {
				t.Fatalf("unexpected args: bookingID=%d userID=%d", bookingID, userID)
			}
			return nil
		},
	}
	s := newHTTPHandlersTestServer(nil, nil, bookingRepo)

	u := &users.User{ID: 7, Username: "alice", Email: "alice@example.com", Role: users.RoleUser}
	req := httptest.NewRequest(http.MethodPost, "/api/bookings/100/cancel", nil)
	req.SetPathValue("id", "100")
	req = requestWithUser(req, u)
	w := httptest.NewRecorder()

	s.CancelBookingHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	data := readServerDataMap(t, w)
	if data["status"] != "canceled" {
		t.Fatalf("status: got %v, want canceled", data["status"])
	}
}

// Техника тест-дизайна: таблица решений.
// Проверяем соответствие ошибок CancelBooking HTTP-ответам.
func TestCancelBookingHandler_DecisionTable_CancelErrors(t *testing.T) {
	cases := []struct {
		name       string
		repoErr    error
		wantStatus int
		wantCode   string
	}{
		{"not found", db.ErrNotFound, http.StatusNotFound, "booking_not_found"},
		{"conflict", db.ErrConflict, http.StatusConflict, "cannot_cancel_booking"},
		{"invalid id", db.ErrInvalidID, http.StatusBadRequest, "invalid_booking_id"},
		{"unknown error", errors.New("db failed"), http.StatusInternalServerError, "internal_error"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bookingRepo := &testBookingRepo{
				cancelBooking: func(context.Context, int, int, time.Time) error {
					return tc.repoErr
				},
			}
			s := newHTTPHandlersTestServer(nil, nil, bookingRepo)

			u := &users.User{ID: 7, Username: "alice", Email: "alice@example.com", Role: users.RoleUser}
			req := httptest.NewRequest(http.MethodPost, "/api/bookings/100/cancel", nil)
			req.SetPathValue("id", "100")
			req = requestWithUser(req, u)
			w := httptest.NewRecorder()

			s.CancelBookingHandler(w, req)

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
