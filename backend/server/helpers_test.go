package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nikolaarden2000/software-design-project/backend/auth"
	"github.com/nikolaarden2000/software-design-project/backend/httpapi"
	"github.com/nikolaarden2000/software-design-project/backend/users"
)

func readAPIError(t *testing.T, w *httptest.ResponseRecorder) *httpapi.APIError {
	t.Helper()

	var body httpapi.Response
	if err := json.NewDecoder(w.Result().Body).Decode(&body); err != nil {
		t.Fatalf("Decode response body: %v", err)
	}

	if body.Error == nil {
		t.Fatal("expected API error, got nil")
	}

	return body.Error
}

// Техника тест-дизайна: граничные значения.
// Проверяем пустую строку, ноль, положительное число, отрицательное число и нечисловое значение.
func TestParsePositiveInt_BoundaryValues(t *testing.T) {
	cases := []struct {
		name     string
		raw      string
		fallback int
		want     int
	}{
		{"empty uses fallback", "", 20, 20},
		{"positive number is returned", "15", 20, 15},
		{"zero is returned", "0", 20, 0},
		{"negative uses fallback", "-1", 20, 20},
		{"non-number uses fallback", "abc", 20, 20},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parsePositiveInt(tc.raw, tc.fallback)

			if got != tc.want {
				t.Fatalf("parsePositiveInt(%q, %d): got %d, want %d", tc.raw, tc.fallback, got, tc.want)
			}
		})
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем успешный разбор path-параметра id.
func TestParsePathID_EquivalenceClasses_ValidID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/rooms/10", nil)
	req.SetPathValue("room_id", "10")
	w := httptest.NewRecorder()

	id, ok := parsePathID(w, req, "room_id", "invalid_room_id", "Некорректный идентификатор помещения")

	if !ok {
		t.Fatal("ok: got false, want true")
	}
	if id != 10 {
		t.Fatalf("id: got %d, want 10", id)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want default 200 because no error was written", w.Code)
	}
}

// Техника тест-дизайна: граничные значения.
// Проверяем недопустимые path id около нижней границы и нечисловое значение.
func TestParsePathID_BoundaryValues_InvalidIDWritesBadRequest(t *testing.T) {
	cases := []struct {
		name string
		raw  string
	}{
		{"empty id", ""},
		{"zero id", "0"},
		{"negative id", "-1"},
		{"non-number id", "abc"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/rooms/"+tc.raw, nil)
			req.SetPathValue("room_id", tc.raw)
			w := httptest.NewRecorder()

			id, ok := parsePathID(w, req, "room_id", "invalid_room_id", "Некорректный идентификатор помещения")

			if ok {
				t.Fatal("ok: got true, want false")
			}
			if id != 0 {
				t.Fatalf("id: got %d, want 0", id)
			}
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status: got %d, want %d", w.Code, http.StatusBadRequest)
			}

			apiErr := readAPIError(t, w)
			if apiErr.Code != "invalid_room_id" {
				t.Fatalf("error code: got %q, want invalid_room_id", apiErr.Code)
			}
		})
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем отсутствующий query-параметр, пустой параметр и валидное положительное число.
func TestOptionalIntQuery_EquivalenceClasses_ValidValues(t *testing.T) {
	cases := []struct {
		name      string
		targetURL string
		wantNil   bool
		wantValue int
	}{
		{"missing param returns nil", "/api/admin/rooms", true, 0},
		{"empty param returns nil", "/api/admin/rooms?location_id=", true, 0},
		{"whitespace param returns nil", "/api/admin/rooms?location_id=+++", true, 0},
		{"valid positive param returns pointer", "/api/admin/rooms?location_id=15", false, 15},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.targetURL, nil)
			w := httptest.NewRecorder()

			value, ok := optionalIntQuery(w, req, "location_id", "invalid_location_id", "Некорректный идентификатор локации")

			if !ok {
				t.Fatal("ok: got false, want true")
			}

			if tc.wantNil {
				if value != nil {
					t.Fatalf("value: got %v, want nil", *value)
				}
				return
			}

			if value == nil {
				t.Fatal("value: got nil, want pointer")
			}
			if *value != tc.wantValue {
				t.Fatalf("value: got %d, want %d", *value, tc.wantValue)
			}
		})
	}
}

// Техника тест-дизайна: граничные значения.
// Проверяем недопустимые значения optional int около нижней границы.
func TestOptionalIntQuery_BoundaryValues_InvalidValuesWriteBadRequest(t *testing.T) {
	cases := []struct {
		name      string
		targetURL string
	}{
		{"zero is invalid", "/api/admin/rooms?location_id=0"},
		{"negative is invalid", "/api/admin/rooms?location_id=-1"},
		{"non-number is invalid", "/api/admin/rooms?location_id=abc"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.targetURL, nil)
			w := httptest.NewRecorder()

			value, ok := optionalIntQuery(w, req, "location_id", "invalid_location_id", "Некорректный идентификатор локации")

			if ok {
				t.Fatal("ok: got true, want false")
			}
			if value != nil {
				t.Fatalf("value: got %v, want nil", *value)
			}
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status: got %d, want %d", w.Code, http.StatusBadRequest)
			}

			apiErr := readAPIError(t, w)
			if apiErr.Code != "invalid_location_id" {
				t.Fatalf("error code: got %q, want invalid_location_id", apiErr.Code)
			}
		})
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем отсутствующую строку, строку из пробелов и непустое значение с trim.
func TestOptionalStringQuery_EquivalenceClasses(t *testing.T) {
	cases := []struct {
		name      string
		targetURL string
		wantNil   bool
		wantValue string
	}{
		{"missing param returns nil", "/api/admin/rooms", true, ""},
		{"empty param returns nil", "/api/admin/rooms?status=", true, ""},
		{"whitespace param returns nil", "/api/admin/rooms?status=+++", true, ""},
		{"value is trimmed", "/api/admin/rooms?status=++draft++", false, "draft"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.targetURL, nil)

			value := optionalStringQuery(req, "status")

			if tc.wantNil {
				if value != nil {
					t.Fatalf("value: got %q, want nil", *value)
				}
				return
			}

			if value == nil {
				t.Fatal("value: got nil, want pointer")
			}
			if *value != tc.wantValue {
				t.Fatalf("value: got %q, want %q", *value, tc.wantValue)
			}
		})
	}
}

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем успешное получение текущего пользователя из контекста запроса.
func TestCurrentUserFromRequest_EquivalenceClasses_UserExists(t *testing.T) {
	wantUser := &users.User{
		ID:       7,
		Username: "admin",
		Email:    "admin@example.com",
		Role:     users.RoleAdmin,
	}

	ctx := context.WithValue(context.Background(), auth.KeyUser, wantUser)
	req := httptest.NewRequest(http.MethodGet, "/api/admin", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	got, ok := currentUserFromRequest(w, req)

	if !ok {
		t.Fatal("ok: got false, want true")
	}
	if got != wantUser {
		t.Fatalf("user: got %+v, want %+v", got, wantUser)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want default 200 because no error was written", w.Code)
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем запрос без пользователя в контексте.
func TestCurrentUserFromRequest_EquivalenceClasses_NoUserWritesUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/admin", nil)
	w := httptest.NewRecorder()

	got, ok := currentUserFromRequest(w, req)

	if ok {
		t.Fatal("ok: got true, want false")
	}
	if got != nil {
		t.Fatalf("user: got %+v, want nil", got)
	}
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusUnauthorized)
	}

	apiErr := readAPIError(t, w)
	if apiErr.Code != "unauthorized" {
		t.Fatalf("error code: got %q, want unauthorized", apiErr.Code)
	}
}

// Техника тест-дизайна: классы эквивалентности.
// Проверяем случай, когда в контексте лежит значение неправильного типа.
func TestCurrentUserFromRequest_EquivalenceClasses_WrongContextValueWritesUnauthorized(t *testing.T) {
	ctx := context.WithValue(context.Background(), auth.KeyUser, "not-a-user")
	req := httptest.NewRequest(http.MethodGet, "/api/admin", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	got, ok := currentUserFromRequest(w, req)

	if ok {
		t.Fatal("ok: got true, want false")
	}
	if got != nil {
		t.Fatalf("user: got %+v, want nil", got)
	}
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusUnauthorized)
	}
}
