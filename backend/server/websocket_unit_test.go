package server_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/nikolaarden2000/software-design-project/backend/auth"
	"github.com/nikolaarden2000/software-design-project/backend/models"
)

func makeWSURL(s *httptest.Server, path string) string {
	return "ws" + strings.TrimPrefix(s.URL, "http") + path
}

func TestHomeWSHandler(t *testing.T) {
	srv, _, roomMock, _ := setupServer(t)

	ts := httptest.NewServer(http.HandlerFunc(srv.HomeWSHandler))
	defer ts.Close()

	wsURL := makeWSURL(ts, "/")

	t.Run("Valid Sequence", func(t *testing.T) {
		roomMock.ExpectedCalls = nil
		roomMock.On("GetCompaniesByCity", mock.Anything, "Moscow").Return([]string{"CompA", "CompB"}, nil).Once()

		rooms := []models.Room{{ID: 1, Title: "R1"}, {ID: 2, Title: "R2"}}
		roomMock.On("GetRoomsBatchByCity", mock.Anything, 0, 2, "Moscow").Return(rooms, nil).Once()

		roomMock.On("GetRoomsBatchByCity", mock.Anything, 2, 2, "Moscow").Return([]models.Room{}, nil).Once()

		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		assert.NoError(t, err)
		defer conn.Close()

		err = conn.WriteMessage(websocket.TextMessage, []byte("Moscow"))
		assert.NoError(t, err)

		var companies []string
		err = conn.ReadJSON(&companies)
		assert.NoError(t, err)
		assert.Equal(t, []string{"CompA", "CompB"}, companies)

		err = conn.WriteMessage(websocket.TextMessage, []byte("2"))
		assert.NoError(t, err)

		var readRooms []models.Room
		err = conn.ReadJSON(&readRooms)
		assert.NoError(t, err)
		assert.Len(t, readRooms, 2)
		assert.Equal(t, 2, readRooms[1].ID)

		err = conn.WriteMessage(websocket.TextMessage, []byte("2"))
		assert.NoError(t, err)

		var emptyRooms []any
		err = conn.ReadJSON(&emptyRooms)
		assert.NoError(t, err)
		assert.Len(t, emptyRooms, 0)
	})

	t.Run("Invalid City / No Companies", func(t *testing.T) {
		roomMock.ExpectedCalls = nil
		roomMock.On("GetCompaniesByCity", mock.Anything, "Nowhere").Return([]string{}, nil).Once()

		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		assert.NoError(t, err)
		defer conn.Close()

		err = conn.WriteMessage(websocket.TextMessage, []byte("Nowhere"))
		assert.NoError(t, err)

		var companies []string
		err = conn.ReadJSON(&companies)
		assert.NoError(t, err)
		assert.Empty(t, companies)
	})
}

func TestBookingWSHandler(t *testing.T) {
	srv, _, _, bookingMock := setupServer(t)

	ts := httptest.NewServer(http.HandlerFunc(srv.BookingWSHandler))
	defer ts.Close()

	wsURL := makeWSURL(ts, "/")

	t.Run("Valid Availability Request", func(t *testing.T) {
		bookingMock.ExpectedCalls = nil

		availability := []models.DateAvailability{
			{Date: "2025-09-18", AvailableTimes: []string{"09:00", "11:00"}},
		}

		bookingMock.On("GetRoomAvailability", mock.Anything, 1, 7, mock.Anything).Return(availability, nil).Once()

		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		assert.NoError(t, err)
		defer conn.Close()

		reqBody := map[string]int{"room_id": 1}
		err = conn.WriteJSON(reqBody)
		assert.NoError(t, err)

		var resp struct {
			RoomID int                       `json:"room_id"`
			Dates  []models.DateAvailability `json:"dates"`
		}
		err = conn.ReadJSON(&resp)
		assert.NoError(t, err)
		assert.Equal(t, 1, resp.RoomID)
		assert.Len(t, resp.Dates, 1)
	})
}

func TestMeWSHandler(t *testing.T) {
	srv, _, _, bookingMock := setupServer(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), auth.KeyUserID, 1)
		srv.MeWSHandler(w, r.WithContext(ctx))
	})

	ts := httptest.NewServer(handler)
	defer ts.Close()

	wsURL := makeWSURL(ts, "/")

	t.Run("Fetch Bookings", func(t *testing.T) {
		bookingMock.ExpectedCalls = nil

		history := []models.BookingHistoryItem{
			{ID: 1, RoomID: 10, Title: "Комната переговоров", Date: "2025-09-18", StartTime: "10:00", EndTime: "11:00", TotalPrice: 500, Status: "booked"},
			{ID: 2, RoomID: 20, Title: "Лаундж-зона", Date: "2025-09-19", StartTime: "12:00", EndTime: "13:00", TotalPrice: 700, Status: "canceled"},
		}

		bookingMock.On("GetUserBookings", mock.Anything, 1, mock.Anything).Return(history, nil).Once()

		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		assert.NoError(t, err)
		defer conn.Close()

		var readHistory []models.BookingHistoryItem
		err = conn.ReadJSON(&readHistory)
		assert.NoError(t, err)
		assert.Len(t, readHistory, 2)
		assert.Equal(t, "booked", readHistory[0].Status)
		assert.Equal(t, "canceled", readHistory[1].Status)
	})
}

func TestHomeWSHandler_Extra(t *testing.T) {
	srv, _, roomMock, _ := setupServer(t)

	ts := httptest.NewServer(http.HandlerFunc(srv.HomeWSHandler))
	defer ts.Close()
	wsURL := makeWSURL(ts, "/")

	t.Run("GetCompaniesByCity Error Sends Empty List", func(t *testing.T) {
		roomMock.ExpectedCalls = nil
		roomMock.On("GetCompaniesByCity", mock.Anything, "Broken").
			Return([]string{}, errors.New("db error")).Once()

		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		assert.NoError(t, err)
		defer conn.Close()

		err = conn.WriteMessage(websocket.TextMessage, []byte("Broken"))
		assert.NoError(t, err)

		var companies []string
		err = conn.ReadJSON(&companies)
		assert.NoError(t, err)
		assert.Empty(t, companies)
	})

	t.Run("Invalid Rooms Count Closes Connection", func(t *testing.T) {
		roomMock.ExpectedCalls = nil
		roomMock.On("GetCompaniesByCity", mock.Anything, "Moscow").
			Return([]string{"CompA"}, nil).Once()

		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		assert.NoError(t, err)
		defer conn.Close()

		err = conn.WriteMessage(websocket.TextMessage, []byte("Moscow"))
		assert.NoError(t, err)

		var companies []string
		err = conn.ReadJSON(&companies)
		assert.NoError(t, err)

		err = conn.WriteMessage(websocket.TextMessage, []byte("abc"))
		assert.NoError(t, err)

		_, _, err = conn.ReadMessage()
		assert.Error(t, err)
	})

	t.Run("Zero Rooms Count Closes Connection", func(t *testing.T) {
		roomMock.ExpectedCalls = nil
		roomMock.On("GetCompaniesByCity", mock.Anything, "Moscow").
			Return([]string{"CompA"}, nil).Once()

		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		assert.NoError(t, err)
		defer conn.Close()

		err = conn.WriteMessage(websocket.TextMessage, []byte("Moscow"))
		assert.NoError(t, err)

		var companies []string
		err = conn.ReadJSON(&companies)
		assert.NoError(t, err)

		err = conn.WriteMessage(websocket.TextMessage, []byte("0"))
		assert.NoError(t, err)

		_, _, err = conn.ReadMessage()
		assert.Error(t, err)
	})

	t.Run("GetRoomsBatchByCity Error Closes Connection", func(t *testing.T) {
		roomMock.ExpectedCalls = nil
		roomMock.On("GetCompaniesByCity", mock.Anything, "Moscow").
			Return([]string{"CompA"}, nil).Once()
		roomMock.On("GetRoomsBatchByCity", mock.Anything, 0, 5, "Moscow").
			Return([]models.Room{}, errors.New("db error")).Once()

		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		assert.NoError(t, err)
		defer conn.Close()

		err = conn.WriteMessage(websocket.TextMessage, []byte("Moscow"))
		assert.NoError(t, err)

		var companies []string
		err = conn.ReadJSON(&companies)
		assert.NoError(t, err)

		err = conn.WriteMessage(websocket.TextMessage, []byte("5"))
		assert.NoError(t, err)

		_, _, err = conn.ReadMessage()
		assert.Error(t, err)
	})
}

func TestBookingWSHandler_Extra(t *testing.T) {
	srv, _, _, bookingMock := setupServer(t)

	ts := httptest.NewServer(http.HandlerFunc(srv.BookingWSHandler))
	defer ts.Close()
	wsURL := makeWSURL(ts, "/")

	t.Run("Invalid JSON Closes Connection", func(t *testing.T) {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		assert.NoError(t, err)
		defer conn.Close()

		err = conn.WriteMessage(websocket.TextMessage, []byte("not-json"))
		assert.NoError(t, err)

		_, _, err = conn.ReadMessage()
		assert.Error(t, err)
	})

	t.Run("GetRoomAvailability Error Closes Connection", func(t *testing.T) {
		bookingMock.ExpectedCalls = nil
		bookingMock.On("GetRoomAvailability", mock.Anything, 99, 7, mock.Anything).
			Return([]models.DateAvailability{}, errors.New("db error")).Once()

		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		assert.NoError(t, err)
		defer conn.Close()

		err = conn.WriteJSON(map[string]int{"room_id": 99})
		assert.NoError(t, err)

		_, _, err = conn.ReadMessage()
		assert.Error(t, err)
	})
}

func TestMeWSHandler_Extra(t *testing.T) {
	t.Run("Unauthorized Returns HTTP 401", func(t *testing.T) {
		srv, _, _, _ := setupServer(t)

		ts := httptest.NewServer(http.HandlerFunc(srv.MeWSHandler))
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/")
		assert.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("GetUserBookings Error Sends Empty List", func(t *testing.T) {
		srv, _, _, bookingMock := setupServer(t)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), auth.KeyUserID, 5)
			srv.MeWSHandler(w, r.WithContext(ctx))
		})
		ts := httptest.NewServer(handler)
		defer ts.Close()

		bookingMock.ExpectedCalls = nil
		bookingMock.On("GetUserBookings", mock.Anything, 5, mock.Anything).
			Return([]models.BookingHistoryItem{}, errors.New("db error")).Once()

		conn, _, err := websocket.DefaultDialer.Dial(makeWSURL(ts, "/"), nil)
		assert.NoError(t, err)
		defer conn.Close()

		var result []models.BookingHistoryItem
		err = conn.ReadJSON(&result)
		assert.NoError(t, err)
		assert.Empty(t, result)
	})
}
