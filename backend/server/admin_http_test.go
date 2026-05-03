package server

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gitlab.com/5130904-20104-teams/software-design-project/backend/bookings"
	"gitlab.com/5130904-20104-teams/software-design-project/backend/db"
	"gitlab.com/5130904-20104-teams/software-design-project/backend/locations"
	"gitlab.com/5130904-20104-teams/software-design-project/backend/rooms"
	"gitlab.com/5130904-20104-teams/software-design-project/backend/users"
)

type testLocationRepo struct {
	listLocations      func(ctx context.Context, companyID *int, city *string) ([]locations.Location, error)
	listAdminLocations func(ctx context.Context, adminID int, includeAll bool) ([]locations.AdminLocation, error)
	createLocation     func(ctx context.Context, companyID int, city string, address string, lat float64, lng float64, timezone string) (*locations.Location, error)
	getLocationByID    func(ctx context.Context, id int) (*locations.Location, error)
	existsByID         func(ctx context.Context, id int) (bool, error)
}

func (m *testLocationRepo) ListLocations(ctx context.Context, companyID *int, city *string) ([]locations.Location, error) {
	if m.listLocations != nil {
		return m.listLocations(ctx, companyID, city)
	}
	return nil, errServerTestNotConfigured
}

func (m *testLocationRepo) ListAdminLocations(ctx context.Context, adminID int, includeAll bool) ([]locations.AdminLocation, error) {
	if m.listAdminLocations != nil {
		return m.listAdminLocations(ctx, adminID, includeAll)
	}
	return nil, errServerTestNotConfigured
}

func (m *testLocationRepo) CreateLocation(ctx context.Context, companyID int, city string, address string, lat float64, lng float64, timezone string) (*locations.Location, error) {
	if m.createLocation != nil {
		return m.createLocation(ctx, companyID, city, address, lat, lng, timezone)
	}
	return nil, errServerTestNotConfigured
}

func (m *testLocationRepo) GetLocationByID(ctx context.Context, id int) (*locations.Location, error) {
	if m.getLocationByID != nil {
		return m.getLocationByID(ctx, id)
	}
	return nil, errServerTestNotConfigured
}

func (m *testLocationRepo) ExistsByID(ctx context.Context, id int) (bool, error) {
	if m.existsByID != nil {
		return m.existsByID(ctx, id)
	}
	return false, errServerTestNotConfigured
}

func newAdminHandlersTestServer(locationRepo LocationRepo, roomRepo RoomRepo, bookingRepo BookingRepo) *Server {
	return &Server{
		locationRepo: locationRepo,
		roomRepo:     roomRepo,
		bookingRepo:  bookingRepo,
	}
}

func adminUser() *users.User {
	return &users.User{
		ID:       7,
		Username: "admin",
		Email:    "admin@example.com",
		Role:     users.RoleAdmin,
	}
}

func superuserUser() *users.User {
	return &users.User{
		ID:       1,
		Username: "superuser",
		Email:    "superuser@example.com",
		Role:     users.RoleSuperuser,
	}
}

func validAdminRoomJSON() string {
	return `{
		"location_id": 10,
		"title": "Room A",
		"description": "Description",
		"price": 1000,
		"capacity": 8,
		"available_from": "09:00",
		"available_to": "18:00",
		"images": ["/images/a.jpg"]
	}`
}

func TestAdminLocationsHandler_DecisionTable_RoleAccess(t *testing.T) {
	cases := []struct {
		name           string
		user           *users.User
		wantAdminID    int
		wantIncludeAll bool
	}{
		{"admin gets own locations", adminUser(), 7, false},
		{"superuser gets all locations", superuserUser(), 1, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			locationRepo := &testLocationRepo{
				listAdminLocations: func(_ context.Context, adminID int, includeAll bool) ([]locations.AdminLocation, error) {
					if adminID != tc.wantAdminID {
						t.Fatalf("adminID: got %d, want %d", adminID, tc.wantAdminID)
					}
					if includeAll != tc.wantIncludeAll {
						t.Fatalf("includeAll: got %v, want %v", includeAll, tc.wantIncludeAll)
					}
					return []locations.AdminLocation{}, nil
				},
			}

			s := newAdminHandlersTestServer(locationRepo, nil, nil)
			req := requestWithUser(httptest.NewRequest(http.MethodGet, "/api/admin/locations", nil), tc.user)
			w := httptest.NewRecorder()

			s.AdminLocationsHandler(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
			}
		})
	}
}

func TestAdminLocationsHandler_EquivalenceClasses_Unauthorized(t *testing.T) {
	s := newAdminHandlersTestServer(&testLocationRepo{}, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/admin/locations", nil)
	w := httptest.NewRecorder()

	s.AdminLocationsHandler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusUnauthorized)
	}

	apiErr := readServerError(t, w)
	if apiErr.Code != "unauthorized" {
		t.Fatalf("error code: got %q, want unauthorized", apiErr.Code)
	}
}

func TestAdminRoomsHandler_EquivalenceClasses_SuccessWithFilters(t *testing.T) {
	roomRepo := &testRoomRepo{
		listAdminRooms: func(_ context.Context, adminID int, includeAll bool, locationID *int, status *string) ([]rooms.AdminRoomListItem, error) {
			if adminID != 7 || includeAll {
				t.Fatalf("unexpected access args: adminID=%d includeAll=%v", adminID, includeAll)
			}
			if locationID == nil || *locationID != 10 {
				t.Fatalf("locationID: got %v, want 10", locationID)
			}
			if status == nil || *status != rooms.StatusDraft {
				t.Fatalf("status: got %v, want draft", status)
			}
			return []rooms.AdminRoomListItem{{ID: 100, LocationID: 10, Title: "Room A", Status: rooms.StatusDraft}}, nil
		},
	}

	s := newAdminHandlersTestServer(nil, roomRepo, nil)
	req := requestWithUser(httptest.NewRequest(http.MethodGet, "/api/admin/rooms?location_id=10&status=draft", nil), adminUser())
	w := httptest.NewRecorder()

	s.AdminRoomsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminRoomsHandler_DecisionTable_RepoErrors(t *testing.T) {
	cases := []struct {
		name       string
		repoErr    error
		wantStatus int
		wantCode   string
	}{
		{"forbidden", db.ErrForbidden, http.StatusForbidden, "forbidden"},
		{"location not found", db.ErrNotFound, http.StatusNotFound, "location_not_found"},
		{"invalid argument", db.ErrInvalidArgument, http.StatusBadRequest, "invalid_request"},
		{"unknown error", errors.New("db failed"), http.StatusInternalServerError, "internal_error"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			roomRepo := &testRoomRepo{
				listAdminRooms: func(context.Context, int, bool, *int, *string) ([]rooms.AdminRoomListItem, error) {
					return nil, tc.repoErr
				},
			}

			s := newAdminHandlersTestServer(nil, roomRepo, nil)
			req := requestWithUser(httptest.NewRequest(http.MethodGet, "/api/admin/rooms", nil), adminUser())
			w := httptest.NewRecorder()

			s.AdminRoomsHandler(w, req)

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

func TestAdminRoomDetailsHandler_EquivalenceClasses_Found(t *testing.T) {
	roomRepo := &testRoomRepo{
		getAdminRoom: func(_ context.Context, adminID int, includeAll bool, roomID int) (*rooms.AdminRoomDetails, error) {
			if adminID != 7 || includeAll || roomID != 100 {
				t.Fatalf("unexpected args: adminID=%d includeAll=%v roomID=%d", adminID, includeAll, roomID)
			}
			return &rooms.AdminRoomDetails{ID: 100, LocationID: 10, Title: "Room A", Status: rooms.StatusDraft}, nil
		},
	}

	s := newAdminHandlersTestServer(nil, roomRepo, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/admin/rooms/100", nil)
	req.SetPathValue("room_id", "100")
	req = requestWithUser(req, adminUser())
	w := httptest.NewRecorder()

	s.AdminRoomDetailsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestCreateAdminRoomHandler_Scenario_Success(t *testing.T) {
	roomRepo := &testRoomRepo{
		createAdminRoom: func(_ context.Context, creatorID int, includeAll bool, input rooms.AdminRoomInput) (*rooms.AdminRoomListItem, error) {
			if creatorID != 7 || includeAll {
				t.Fatalf("unexpected access args: creatorID=%d includeAll=%v", creatorID, includeAll)
			}
			if input.LocationID != 10 || input.Title != "Room A" {
				t.Fatalf("unexpected input: %+v", input)
			}
			return &rooms.AdminRoomListItem{ID: 100, LocationID: 10, Title: "Room A", Status: rooms.StatusDraft}, nil
		},
	}

	s := newAdminHandlersTestServer(nil, roomRepo, nil)
	req := requestWithUser(httptest.NewRequest(http.MethodPost, "/api/admin/rooms", bytes.NewBufferString(validAdminRoomJSON())), adminUser())
	w := httptest.NewRecorder()

	s.CreateAdminRoomHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestUpdateAdminRoomHandler_Scenario_Success(t *testing.T) {
	roomRepo := &testRoomRepo{
		updateAdminRoom: func(_ context.Context, adminID int, includeAll bool, roomID int, input rooms.AdminRoomInput) error {
			if adminID != 7 || includeAll || roomID != 100 {
				t.Fatalf("unexpected args: adminID=%d includeAll=%v roomID=%d", adminID, includeAll, roomID)
			}
			if input.Title != "Room A" {
				t.Fatalf("title: got %q, want Room A", input.Title)
			}
			return nil
		},
	}

	s := newAdminHandlersTestServer(nil, roomRepo, nil)
	req := httptest.NewRequest(http.MethodPatch, "/api/admin/rooms/100", bytes.NewBufferString(validAdminRoomJSON()))
	req.SetPathValue("room_id", "100")
	req = requestWithUser(req, adminUser())
	w := httptest.NewRecorder()

	s.UpdateAdminRoomHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	data := readServerDataMap(t, w)
	if data["status"] != rooms.StatusDraft {
		t.Fatalf("status: got %v, want %s", data["status"], rooms.StatusDraft)
	}
}

func TestSubmitAdminRoomHandler_StateTransitions_SubmitMovesRoomToPending(t *testing.T) {
	roomRepo := &testRoomRepo{
		submitAdminRoom: func(_ context.Context, adminID int, includeAll bool, roomID int) error {
			if adminID != 7 || includeAll || roomID != 100 {
				t.Fatalf("unexpected args: adminID=%d includeAll=%v roomID=%d", adminID, includeAll, roomID)
			}
			return nil
		},
	}

	s := newAdminHandlersTestServer(nil, roomRepo, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/rooms/100/submit", nil)
	req.SetPathValue("room_id", "100")
	req = requestWithUser(req, adminUser())
	w := httptest.NewRecorder()

	s.SubmitAdminRoomHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	data := readServerDataMap(t, w)
	if data["status"] != rooms.StatusPending {
		t.Fatalf("status: got %v, want %s", data["status"], rooms.StatusPending)
	}
}

func TestArchiveAdminRoomHandler_StateTransitions_ArchiveMovesRoomToArchived(t *testing.T) {
	roomRepo := &testRoomRepo{
		archiveAdminRoom: func(_ context.Context, adminID int, includeAll bool, roomID int, mode string, _ time.Time) (*rooms.AdminRoomArchiveResult, error) {
			if adminID != 7 || includeAll || roomID != 100 || mode != rooms.ArchiveModeImmediate {
				t.Fatalf("unexpected args: adminID=%d includeAll=%v roomID=%d mode=%q", adminID, includeAll, roomID, mode)
			}
			return &rooms.AdminRoomArchiveResult{ID: 100, Status: rooms.StatusArchived}, nil
		},
	}

	s := newAdminHandlersTestServer(nil, roomRepo, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/rooms/100/archive", bytes.NewBufferString(`{"mode":"immediate"}`))
	req.SetPathValue("room_id", "100")
	req = requestWithUser(req, adminUser())
	w := httptest.NewRecorder()

	s.ArchiveAdminRoomHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestArchiveAdminRoomHandler_DecisionTable_RepoErrors(t *testing.T) {
	cases := []struct {
		name       string
		repoErr    error
		wantStatus int
		wantCode   string
	}{
		{"invalid id", db.ErrInvalidID, http.StatusBadRequest, "invalid_room_id"},
		{"invalid argument", db.ErrInvalidArgument, http.StatusBadRequest, "invalid_request"},
		{"forbidden", db.ErrForbidden, http.StatusForbidden, "forbidden"},
		{"not found", db.ErrNotFound, http.StatusNotFound, "room_not_found"},
		{"conflict", db.ErrConflict, http.StatusConflict, "room_has_active_bookings"},
		{"unknown error", errors.New("db failed"), http.StatusInternalServerError, "internal_error"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			roomRepo := &testRoomRepo{
				archiveAdminRoom: func(context.Context, int, bool, int, string, time.Time) (*rooms.AdminRoomArchiveResult, error) {
					return nil, tc.repoErr
				},
			}

			s := newAdminHandlersTestServer(nil, roomRepo, nil)
			req := httptest.NewRequest(http.MethodPost, "/api/admin/rooms/100/archive", bytes.NewBufferString(`{"mode":"immediate"}`))
			req.SetPathValue("room_id", "100")
			req = requestWithUser(req, adminUser())
			w := httptest.NewRecorder()

			s.ArchiveAdminRoomHandler(w, req)

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

func TestAdminBookingsHandler_EquivalenceClasses_SuccessWithFilters(t *testing.T) {
	bookingRepo := &testBookingRepo{
		listAdminBookings: func(_ context.Context, adminID int, includeAll bool, locationID *int, roomID *int, status *string, _ time.Time) ([]bookings.AdminBookingItem, error) {
			if adminID != 7 || includeAll {
				t.Fatalf("unexpected access args: adminID=%d includeAll=%v", adminID, includeAll)
			}
			if locationID == nil || *locationID != 10 {
				t.Fatalf("locationID: got %v, want 10", locationID)
			}
			if roomID == nil || *roomID != 100 {
				t.Fatalf("roomID: got %v, want 100", roomID)
			}
			if status == nil || *status != "booked" {
				t.Fatalf("status: got %v, want booked", status)
			}
			return []bookings.AdminBookingItem{{ID: 500, RoomID: 100, Status: "booked"}}, nil
		},
	}

	s := newAdminHandlersTestServer(nil, nil, bookingRepo)
	req := requestWithUser(httptest.NewRequest(http.MethodGet, "/api/admin/bookings?location_id=10&room_id=100&status=booked", nil), adminUser())
	w := httptest.NewRecorder()

	s.AdminBookingsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestCancelAdminBookingHandler_StateTransitions_CancelMovesBookingToCanceled(t *testing.T) {
	bookingRepo := &testBookingRepo{
		cancelAdminBooking: func(_ context.Context, adminID int, includeAll bool, bookingID int, _ time.Time) error {
			if adminID != 7 || includeAll || bookingID != 500 {
				t.Fatalf("unexpected args: adminID=%d includeAll=%v bookingID=%d", adminID, includeAll, bookingID)
			}
			return nil
		},
	}

	s := newAdminHandlersTestServer(nil, nil, bookingRepo)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/bookings/500/cancel", nil)
	req.SetPathValue("booking_id", "500")
	req = requestWithUser(req, adminUser())
	w := httptest.NewRecorder()

	s.CancelAdminBookingHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	data := readServerDataMap(t, w)
	if data["status"] != "canceled" {
		t.Fatalf("status: got %v, want canceled", data["status"])
	}
}

func TestCancelAdminBookingHandler_DecisionTable_RepoErrors(t *testing.T) {
	cases := []struct {
		name       string
		repoErr    error
		wantStatus int
		wantCode   string
	}{
		{"invalid id", db.ErrInvalidID, http.StatusBadRequest, "invalid_booking_id"},
		{"forbidden", db.ErrForbidden, http.StatusForbidden, "forbidden"},
		{"not found", db.ErrNotFound, http.StatusNotFound, "booking_not_found"},
		{"conflict", db.ErrConflict, http.StatusConflict, "cannot_cancel_booking"},
		{"unknown error", errors.New("db failed"), http.StatusInternalServerError, "internal_error"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bookingRepo := &testBookingRepo{
				cancelAdminBooking: func(context.Context, int, bool, int, time.Time) error {
					return tc.repoErr
				},
			}

			s := newAdminHandlersTestServer(nil, nil, bookingRepo)
			req := httptest.NewRequest(http.MethodPost, "/api/admin/bookings/500/cancel", nil)
			req.SetPathValue("booking_id", "500")
			req = requestWithUser(req, adminUser())
			w := httptest.NewRecorder()

			s.CancelAdminBookingHandler(w, req)

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
