package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Техника тест-дизайна: сценарное тестирование, позитивный сценарий.
// Проверяем успешную запись JSON-ответа с data.
func TestWriteJSON_Scenario_WritesDataResponse(t *testing.T) {
	w := httptest.NewRecorder()

	WriteJSON(w, http.StatusOK, map[string]any{
		"id":   10,
		"name": "test",
	})

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want %d", res.StatusCode, http.StatusOK)
	}

	if got := res.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type: got %q, want application/json", got)
	}

	var body Response
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("Decode response body: %v", err)
	}

	if body.Error != nil {
		t.Fatalf("Error: got %+v, want nil", body.Error)
	}

	data, ok := body.Data.(map[string]any)
	if !ok {
		t.Fatalf("Data type: got %T, want map[string]any", body.Data)
	}

	if data["name"] != "test" {
		t.Fatalf("name: got %v, want test", data["name"])
	}

	if data["id"].(float64) != 10 {
		t.Fatalf("id: got %v, want 10", data["id"])
	}
}

// Техника тест-дизайна: граничные значения.
// Проверяем особый граничный HTTP-статус 204, при котором тело ответа не записывается.
func TestWriteJSON_BoundaryValues_NoContentWritesNoBody(t *testing.T) {
	w := httptest.NewRecorder()

	WriteJSON(w, http.StatusNoContent, map[string]any{
		"id": 10,
	})

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status: got %d, want %d", res.StatusCode, http.StatusNoContent)
	}

	if got := res.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type: got %q, want application/json", got)
	}

	if body := w.Body.String(); body != "" {
		t.Fatalf("body: got %q, want empty body", body)
	}
}

// Техника тест-дизайна: сценарное тестирование, негативный сценарий.
// Проверяем запись JSON-ответа с error.
func TestWriteError_Scenario_WritesErrorResponse(t *testing.T) {
	w := httptest.NewRecorder()

	WriteError(w, http.StatusBadRequest, "invalid_request", "Некорректный запрос")

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d", res.StatusCode, http.StatusBadRequest)
	}

	if got := res.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type: got %q, want application/json", got)
	}

	var body Response
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("Decode response body: %v", err)
	}

	if body.Data != nil {
		t.Fatalf("Data: got %+v, want nil", body.Data)
	}

	if body.Error == nil {
		t.Fatal("Error: got nil, want APIError")
	}

	if body.Error.Code != "invalid_request" {
		t.Fatalf("error code: got %q, want invalid_request", body.Error.Code)
	}

	if body.Error.Message != "Некорректный запрос" {
		t.Fatalf("error message: got %q, want Некорректный запрос", body.Error.Message)
	}
}
