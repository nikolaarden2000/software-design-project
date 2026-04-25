package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gitlab.com/5130904-20104-teams/software-design-project/internal/auth"
	"gitlab.com/5130904-20104-teams/software-design-project/internal/db"
	"gitlab.com/5130904-20104-teams/software-design-project/internal/models"
	"gitlab.com/5130904-20104-teams/software-design-project/internal/server"
)

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Register(ctx context.Context, username, email, password string) (int, error) {
	args := m.Called(ctx, username, email, password)
	return args.Int(0), args.Error(1)
}

func (m *MockAuthService) Login(ctx context.Context, email, password string, w http.ResponseWriter) (string, error) {
	args := m.Called(ctx, email, password, w)
	return args.String(0), args.Error(1)
}

func (m *MockAuthService) Logout(w http.ResponseWriter, r *http.Request) {
	m.Called(w, r)
}

func (m *MockAuthService) GetUserByID(ctx context.Context, id int) (*models.User, error) {
	args := m.Called(ctx, id)
	var u *models.User
	if args.Get(0) != nil {
		u = args.Get(0).(*models.User)
	}
	return u, args.Error(1)
}

type MockRoomRepo struct {
	mock.Mock
}

func (m *MockRoomRepo) GetRoomsBatchByCity(ctx context.Context, lastID, limit int, city string) ([]models.Room, error) {
	args := m.Called(ctx, lastID, limit, city)
	return args.Get(0).([]models.Room), args.Error(1)
}

func (m *MockRoomRepo) GetCompaniesByCity(ctx context.Context, city string) ([]string, error) {
	args := m.Called(ctx, city)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockRoomRepo) GetRoomPageData(ctx context.Context, roomID int) (*models.RoomPageData, error) {
	args := m.Called(ctx, roomID)
	var d *models.RoomPageData
	if args.Get(0) != nil {
		d = args.Get(0).(*models.RoomPageData)
	}
	return d, args.Error(1)
}

type MockBookingRepo struct {
	mock.Mock
}

func (m *MockBookingRepo) GetRoomAvailability(ctx context.Context, roomID, days int, now time.Time) ([]models.DateAvailability, error) {
	args := m.Called(ctx, roomID, days, now)
	return args.Get(0).([]models.DateAvailability), args.Error(1)
}

func (m *MockBookingRepo) CreateBooking(ctx context.Context, userID, roomID int, date string, slots []string, now time.Time) (int, error) {
	args := m.Called(ctx, userID, roomID, date, slots, now)
	return args.Int(0), args.Error(1)
}

func (m *MockBookingRepo) GetUserBookings(ctx context.Context, userID int, now time.Time) ([]models.BookingHistoryItem, error) {
	args := m.Called(ctx, userID, now)
	return args.Get(0).([]models.BookingHistoryItem), args.Error(1)
}

func (m *MockBookingRepo) CancelBooking(ctx context.Context, bookingID, userID int, now time.Time) error {
	args := m.Called(ctx, bookingID, userID, now)
	return args.Error(0)
}

func setupServer(t *testing.T) (*server.Server, *MockAuthService, *MockRoomRepo, *MockBookingRepo) {
	authMock := new(MockAuthService)
	roomMock := new(MockRoomRepo)
	bookingMock := new(MockBookingRepo)

	tmpl := template.Must(template.New("home.html").Parse(`{{.Username}}`))
	template.Must(tmpl.New("auth.html").Parse(`auth`))
	template.Must(tmpl.New("me.html").Parse(`me`))
	template.Must(tmpl.New("room.html").Parse(`room`))

	srv := server.NewServer(authMock, roomMock, bookingMock, tmpl)
	return srv, authMock, roomMock, bookingMock
}

func TestRegisterHandler(t *testing.T) {
	srv, authMock, _, _ := setupServer(t)

	tests := []struct {
		name       string
		body       interface{}
		setupMock  func()
		wantStatus int
	}{
		{
			name: "Valid Registration",
			body: map[string]string{"username": "test", "email": "test@test.com", "password": "pass"},
			setupMock: func() {
				authMock.On("Register", mock.Anything, "test", "test@test.com", "pass").Return(1, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "Invalid JSON",
			body:       "invalid",
			setupMock:  func() {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "Email Exists",
			body: map[string]string{"username": "test", "email": "exists@test.com", "password": "pass"},
			setupMock: func() {
				authMock.On("Register", mock.Anything, "test", "exists@test.com", "pass").Return(0, auth.ErrEmailExists)
			},
			wantStatus: http.StatusConflict,
		},
		{
			name: "Internal Server Error",
			body: map[string]string{"username": "test", "email": "err@test.com", "password": "pass"},
			setupMock: func() {
				authMock.On("Register", mock.Anything, "test", "err@test.com", "pass").Return(0, errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			var b bytes.Buffer
			if s, ok := tt.body.(string); ok {
				b.WriteString(s)
			} else {
				json.NewEncoder(&b).Encode(tt.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/register", &b)
			w := httptest.NewRecorder()

			srv.RegisterHandler(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestLoginHandler(t *testing.T) {
	srv, authMock, _, _ := setupServer(t)

	tests := []struct {
		name       string
		body       interface{}
		setupMock  func()
		wantStatus int
	}{
		{
			name: "Valid Login",
			body: map[string]string{"email": "test@test.com", "password": "pass"},
			setupMock: func() {
				authMock.On("Login", mock.Anything, "test@test.com", "pass", mock.Anything).Return("session_id", nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "Invalid JSON",
			body:       "invalid",
			setupMock:  func() {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "Invalid Credentials",
			body: map[string]string{"email": "wrong@test.com", "password": "pass"},
			setupMock: func() {
				authMock.On("Login", mock.Anything, "wrong@test.com", "pass", mock.Anything).Return("", auth.ErrInvalidCredentials)
			},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			var b bytes.Buffer
			if s, ok := tt.body.(string); ok {
				b.WriteString(s)
			} else {
				json.NewEncoder(&b).Encode(tt.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/login", &b)
			w := httptest.NewRecorder()

			srv.LoginHandler(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestLogoutHandler(t *testing.T) {
	srv, authMock, _, _ := setupServer(t)
	authMock.On("Logout", mock.Anything, mock.Anything).Return()

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	w := httptest.NewRecorder()

	srv.LogoutHandler(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRoomHandler(t *testing.T) {
	srv, _, roomMock, _ := setupServer(t)

	tests := []struct {
		name       string
		path       string
		setupMock  func()
		wantStatus int
	}{
		{
			name: "Valid Room",
			path: "/room/1",
			setupMock: func() {
				roomMock.On("GetRoomPageData", mock.Anything, 1).Return(&models.RoomPageData{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "Invalid Room ID String",
			path:       "/room/abc",
			setupMock:  func() {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "Room Not Found",
			path: "/room/999",
			setupMock: func() {
				roomMock.On("GetRoomPageData", mock.Anything, 999).Return(nil, db.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			srv.RoomHandler(w, req)
			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestBookingHandler(t *testing.T) {
	srv, _, _, bookingMock := setupServer(t)

	tests := []struct {
		name       string
		userID     int
		body       interface{}
		setupMock  func()
		wantStatus int
	}{
		{
			name:       "Unauthorized (No user ID)",
			userID:     0,
			body:       map[string]interface{}{"room_id": 1, "date": "2024-05-01", "slots": []string{"10:00"}},
			setupMock:  func() {},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "Valid Booking",
			userID: 1,
			body:   map[string]interface{}{"room_id": 1, "date": "2024-05-02", "slots": []string{"10:00"}},
			setupMock: func() {
				bookingMock.On("CreateBooking", mock.Anything, 1, 1, "2024-05-02", []string{"10:00"}, mock.Anything).Return(1, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "Conflict (Already Booked)",
			userID: 1,
			body:   map[string]interface{}{"room_id": 1, "date": "2024-05-03", "slots": []string{"10:00"}},
			setupMock: func() {
				bookingMock.On("CreateBooking", mock.Anything, 1, 1, "2024-05-03", []string{"10:00"}, mock.Anything).Return(0, db.ErrConflict)
			},
			wantStatus: http.StatusConflict,
		},
		{
			name:   "Not Consecutive Slots",
			userID: 1,
			body:   map[string]interface{}{"room_id": 1, "date": "2024-05-04", "slots": []string{"10:00", "12:00"}},
			setupMock: func() {
				bookingMock.On("CreateBooking", mock.Anything, 1, 1, "2024-05-04", []string{"10:00", "12:00"}, mock.Anything).Return(0, db.ErrNotConsecutiveSlots)
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			var b bytes.Buffer
			json.NewEncoder(&b).Encode(tt.body)

			req := httptest.NewRequest(http.MethodPost, "/book", &b)
			if tt.userID > 0 {
				ctx := context.WithValue(req.Context(), auth.KeyUserID, tt.userID)
				req = req.WithContext(ctx)
			}
			w := httptest.NewRecorder()

			srv.BookingHandler(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestCancelBookingHandler(t *testing.T) {
	srv, _, _, bookingMock := setupServer(t)

	tests := []struct {
		name       string
		userID     int
		body       interface{}
		setupMock  func()
		wantStatus int
	}{
		{
			name:   "Valid Cancel",
			userID: 1,
			body:   map[string]interface{}{"booking_id": 1},
			setupMock: func() {
				bookingMock.On("CancelBooking", mock.Anything, 1, 1, mock.Anything).Return(nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "Invalid Booking ID (0)",
			userID:     1,
			body:       map[string]interface{}{"booking_id": 0},
			setupMock:  func() {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "Booking Not Found",
			userID: 1,
			body:   map[string]interface{}{"booking_id": 999},
			setupMock: func() {
				bookingMock.On("CancelBooking", mock.Anything, 999, 1, mock.Anything).Return(db.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			var b bytes.Buffer
			json.NewEncoder(&b).Encode(tt.body)

			req := httptest.NewRequest(http.MethodPost, "/cancel", &b)
			ctx := context.WithValue(req.Context(), auth.KeyUserID, tt.userID)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			srv.CancelBookingHandler(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestHomeHandler(t *testing.T) {
	t.Run("Anonymous User", func(t *testing.T) {
		srv, _, _, _ := setupServer(t)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		srv.HomeHandler(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Authenticated User", func(t *testing.T) {
		srv, authMock, _, _ := setupServer(t)
		authMock.On("GetUserByID", mock.Anything, 42).
			Return(&models.User{ID: 42, Username: "alice"}, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		ctx := context.WithValue(req.Context(), auth.KeyUserID, 42)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		srv.HomeHandler(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "alice")
	})

	t.Run("GetUserByID Error Falls Back Gracefully", func(t *testing.T) {
		srv, authMock, _, _ := setupServer(t)
		authMock.On("GetUserByID", mock.Anything, 7).
			Return(nil, errors.New("db down")).Once()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		ctx := context.WithValue(req.Context(), auth.KeyUserID, 7)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		srv.HomeHandler(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAuthHandler(t *testing.T) {
	srv, _, _, _ := setupServer(t)
	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	w := httptest.NewRecorder()
	srv.AuthHandler(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMeHandler(t *testing.T) {
	srv, _, _, _ := setupServer(t)
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	w := httptest.NewRecorder()
	srv.MeHandler(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLoginHandler_Extra(t *testing.T) {
	t.Run("User Not Found Returns Unauthorized", func(t *testing.T) {
		srv, authMock, _, _ := setupServer(t)
		authMock.On("Login", mock.Anything, "nouser@test.com", "pass", mock.Anything).
			Return("", auth.ErrNoUser).Once()

		body, _ := json.Marshal(map[string]string{"email": "nouser@test.com", "password": "pass"})
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
		w := httptest.NewRecorder()
		srv.LoginHandler(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Internal Server Error", func(t *testing.T) {
		srv, authMock, _, _ := setupServer(t)
		authMock.On("Login", mock.Anything, "boom@test.com", "pass", mock.Anything).
			Return("", errors.New("unexpected")).Once()

		body, _ := json.Marshal(map[string]string{"email": "boom@test.com", "password": "pass"})
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
		w := httptest.NewRecorder()
		srv.LoginHandler(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestRoomHandler_Extra(t *testing.T) {
	t.Run("ErrInvalidID Returns BadRequest", func(t *testing.T) {
		srv, _, roomMock, _ := setupServer(t)
		roomMock.On("GetRoomPageData", mock.Anything, 0).
			Return(nil, db.ErrInvalidID).Once()

		req := httptest.NewRequest(http.MethodGet, "/room/0", nil)
		w := httptest.NewRecorder()
		srv.RoomHandler(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Internal Server Error", func(t *testing.T) {
		srv, _, roomMock, _ := setupServer(t)
		roomMock.On("GetRoomPageData", mock.Anything, 5).
			Return(nil, errors.New("db down")).Once()

		req := httptest.NewRequest(http.MethodGet, "/room/5", nil)
		w := httptest.NewRecorder()
		srv.RoomHandler(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestBookingHandler_Extra(t *testing.T) {
	t.Run("Invalid JSON Returns BadRequest", func(t *testing.T) {
		srv, _, _, _ := setupServer(t)
		req := httptest.NewRequest(http.MethodPost, "/book", bytes.NewBufferString("not-json"))
		ctx := context.WithValue(req.Context(), auth.KeyUserID, 1)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		srv.BookingHandler(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("ErrInvalidArgument Returns BadRequest", func(t *testing.T) {
		srv, _, _, bookingMock := setupServer(t)
		bookingMock.On("CreateBooking", mock.Anything, 1, 1, "2024-06-01", []string{"10:00"}, mock.Anything).
			Return(0, db.ErrInvalidArgument).Once()

		body, _ := json.Marshal(map[string]interface{}{
			"room_id": 1, "date": "2024-06-01", "slots": []string{"10:00"},
		})
		req := httptest.NewRequest(http.MethodPost, "/book", bytes.NewReader(body))
		ctx := context.WithValue(req.Context(), auth.KeyUserID, 1)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		srv.BookingHandler(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Internal Server Error", func(t *testing.T) {
		srv, _, _, bookingMock := setupServer(t)
		bookingMock.On("CreateBooking", mock.Anything, 1, 2, "2024-06-02", []string{"11:00"}, mock.Anything).
			Return(0, errors.New("unexpected")).Once()

		body, _ := json.Marshal(map[string]interface{}{
			"room_id": 2, "date": "2024-06-02", "slots": []string{"11:00"},
		})
		req := httptest.NewRequest(http.MethodPost, "/book", bytes.NewReader(body))
		ctx := context.WithValue(req.Context(), auth.KeyUserID, 1)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		srv.BookingHandler(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestCancelBookingHandler_Extra(t *testing.T) {
	t.Run("Unauthorized (No User ID)", func(t *testing.T) {
		srv, _, _, _ := setupServer(t)
		body, _ := json.Marshal(map[string]interface{}{"booking_id": 1})
		req := httptest.NewRequest(http.MethodPost, "/cancel", bytes.NewReader(body))
		w := httptest.NewRecorder()
		srv.CancelBookingHandler(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Invalid JSON Returns BadRequest", func(t *testing.T) {
		srv, _, _, _ := setupServer(t)
		req := httptest.NewRequest(http.MethodPost, "/cancel", bytes.NewBufferString("not-json"))
		ctx := context.WithValue(req.Context(), auth.KeyUserID, 1)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		srv.CancelBookingHandler(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("ErrConflict (Cannot Cancel In Current State)", func(t *testing.T) {
		srv, _, _, bookingMock := setupServer(t)
		bookingMock.On("CancelBooking", mock.Anything, 10, 1, mock.Anything).
			Return(db.ErrConflict).Once()

		body, _ := json.Marshal(map[string]interface{}{"booking_id": 10})
		req := httptest.NewRequest(http.MethodPost, "/cancel", bytes.NewReader(body))
		ctx := context.WithValue(req.Context(), auth.KeyUserID, 1)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		srv.CancelBookingHandler(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("Internal Server Error", func(t *testing.T) {
		srv, _, _, bookingMock := setupServer(t)
		bookingMock.On("CancelBooking", mock.Anything, 20, 1, mock.Anything).
			Return(errors.New("db down")).Once()

		body, _ := json.Marshal(map[string]interface{}{"booking_id": 20})
		req := httptest.NewRequest(http.MethodPost, "/cancel", bytes.NewReader(body))
		ctx := context.WithValue(req.Context(), auth.KeyUserID, 1)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		srv.CancelBookingHandler(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
