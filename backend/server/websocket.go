package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/nikolaarden2000/software-design-project/backend/auth"
	"github.com/nikolaarden2000/software-design-project/backend/models"
)

type bookingWSRequest struct {
	RoomID int `json:"room_id"`
}

type bookingWSResponse struct {
	RoomID int                       `json:"room_id"`
	Dates  []models.DateAvailability `json:"dates"`
}

func (s *Server) HomeWSHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[home](%v): could not upgrade from http to websocket: %v\n", r.RemoteAddr, err)
		return
	}
	defer conn.Close()

	log.Printf("[home](%v): websocket connection established", r.RemoteAddr)

	_, cityBytes, err := conn.ReadMessage()
	if err != nil {
		log.Printf("[home](%v): failed to read city: %v\n", r.RemoteAddr, err)
		return
	}
	city := string(cityBytes)
	log.Printf("[home](%v): requested city: %s", r.RemoteAddr, city)

	ctx := r.Context()
	companies, err := s.roomRepo.GetCompaniesByCity(ctx, city)
	if err != nil {
		log.Printf("[home](%v): error fetching companies: %v\n", r.RemoteAddr, err)
		conn.WriteJSON([]string{})
		return
	}

	if len(companies) == 0 {
		log.Printf("[home](%v): no companies found in %s", r.RemoteAddr, city)
		conn.WriteJSON([]string{})
		return
	}

	if err := conn.WriteJSON(companies); err != nil {
		log.Printf("[home](%v): failed to send companies: %v\n", r.RemoteAddr, err)
		return
	}
	log.Printf("[home](%v): sent %d companies", r.RemoteAddr, len(companies))

	lastID := 0

	for {
		_, nBytes, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[home](%v): connection closed or read error: %v\n", r.RemoteAddr, err)
			break
		}
		n, err := strconv.Atoi(string(nBytes))
		if err != nil || n <= 0 {
			log.Printf("[home](%v): invalid number of rooms requested: %s", r.RemoteAddr, string(nBytes))
			break
		}

		rooms, err := s.roomRepo.GetRoomsBatchByCity(ctx, lastID, n, city)
		if err != nil {
			log.Printf("[home](%v): error fetching rooms: %v\n", r.RemoteAddr, err)
			break
		}

		if len(rooms) == 0 {
			log.Printf("[home](%v): no more rooms to send", r.RemoteAddr)
			conn.WriteJSON([]any{})
			break
		}

		lastID = rooms[len(rooms)-1].ID

		if err := conn.WriteJSON(rooms); err != nil {
			log.Printf("[home](%v): failed to send rooms: %v\n", r.RemoteAddr, err)
			break
		}
		log.Printf("[home](%v): sent %d rooms (lastID=%d)", r.RemoteAddr, len(rooms), lastID)
	}
}

func (s *Server) BookingWSHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[booking](%v): could not upgrade from http to websocket: %v\n", r.RemoteAddr, err)
		return
	}
	defer conn.Close()

	log.Printf("[booking](%v): websocket connection established", r.RemoteAddr)

	_, msg, err := conn.ReadMessage()
	if err != nil {
		log.Printf("[booking](%v): failed to read message: %v", r.RemoteAddr, err)
		return
	}

	var req bookingWSRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		log.Printf("[booking](%v): invalid request: %v", r.RemoteAddr, err)
		return
	}
	log.Printf("[booking](%v): requested availability for room %d", r.RemoteAddr, req.RoomID)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	availability, err := s.bookingRepo.GetRoomAvailability(ctx, req.RoomID, 7, time.Now())
	if err != nil {
		log.Printf("[booking](%v): error fetching availability for room %d: %v", r.RemoteAddr, req.RoomID, err)
		return
	}

	resp := bookingWSResponse{
		RoomID: req.RoomID,
		Dates:  availability,
	}
	if err := conn.WriteJSON(resp); err != nil {
		log.Printf("[booking](%v): failed to send availability for room %d: %v", r.RemoteAddr, req.RoomID, err)
		return
	}
	log.Printf("[booking](%v): sent availability for room %d", r.RemoteAddr, req.RoomID)
}

func (s *Server) MeWSHandler(w http.ResponseWriter, r *http.Request) {
	uid, ok := r.Context().Value(auth.KeyUserID).(int)
	if !ok || uid <= 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[booking history](%v): could not upgrade to websocket: %v\n", r.RemoteAddr, err)
		return
	}
	defer conn.Close()

	log.Printf("[booking history](%v): websocket connection established for user %d", r.RemoteAddr, uid)

	bookings, err := s.bookingRepo.GetUserBookings(r.Context(), uid, time.Now())
	if err != nil {
		log.Printf("[booking history](%v): failed to fetch bookings for user %d: %v", r.RemoteAddr, uid, err)
		conn.WriteJSON([]any{})
		return
	}

	if err := conn.WriteJSON(bookings); err != nil {
		log.Printf("[booking history](%v): failed to send bookings for user %d: %v", r.RemoteAddr, uid, err)
		return
	}
	log.Printf("[booking history](%v): sent %d bookings to user %d", r.RemoteAddr, len(bookings), uid)
}
