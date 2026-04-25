package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"gitlab.com/5130904-20104-teams/software-design-project/internal/auth"
	"gitlab.com/5130904-20104-teams/software-design-project/internal/db"
	"gitlab.com/5130904-20104-teams/software-design-project/internal/models"
)

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type bookingRequest struct {
	RoomID int      `json:"room_id"`
	Date   string   `json:"date"`
	Slots  []string `json:"slots"`
}

type cancelRequest struct {
	BookingID int `json:"booking_id"`
}

func (s *Server) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	_, err := s.auth.Register(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrEmailExists):
			http.Error(w, "email already exists", http.StatusConflict)
		default:
			log.Printf("[server] RegisterHandler: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	_, err := s.auth.Login(r.Context(), req.Email, req.Password, w)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrNoUser), errors.Is(err, auth.ErrInvalidCredentials):
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
		default:
			log.Printf("[server] LoginHandler: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	s.auth.Logout(w, r)
	w.WriteHeader(http.StatusOK)
}

type userAuth struct {
	Auth     bool
	Username string
}

func (s *Server) getUserAuth(r *http.Request) (user userAuth) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok || uid <= 0 {
		return
	}

	u, err := s.auth.GetUserByID(r.Context(), uid)
	if err != nil {
		log.Printf("[server] getUserAuth: GetUserByID(%d): %v", uid, err)
		return
	}
	user.Auth = true
	user.Username = u.Username
	return
}

func (s *Server) HomeHandler(w http.ResponseWriter, r *http.Request) {
	s.tmplHTML.ExecuteTemplate(w, "home.html", s.getUserAuth(r))
}

func (s *Server) AuthHandler(w http.ResponseWriter, r *http.Request) {
	s.tmplHTML.ExecuteTemplate(w, "auth.html", s.getUserAuth(r))
}

func (s *Server) MeHandler(w http.ResponseWriter, r *http.Request) {
	s.tmplHTML.ExecuteTemplate(w, "me.html", s.getUserAuth(r))
}

func (s *Server) RoomHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/room/"):]
	roomID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid room id", http.StatusBadRequest)
		return
	}

	data, err := s.roomRepo.GetRoomPageData(r.Context(), roomID)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrNotFound):
			http.Error(w, "room not found", http.StatusNotFound)
		case errors.Is(err, db.ErrInvalidID):
			http.Error(w, "invalid room id", http.StatusBadRequest)
		default:
			log.Printf("[server] RoomHandler: GetRoomPageData(%d): %v", roomID, err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	type templateData struct {
		models.RoomPageData
		userAuth
	}
	td := templateData{
		RoomPageData: *data,
		userAuth:     s.getUserAuth(r),
	}

	if err := s.tmplHTML.ExecuteTemplate(w, "room.html", td); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) BookingHandler(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok || uid <= 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req bookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	_, err := s.bookingRepo.CreateBooking(r.Context(), uid, req.RoomID, req.Date, req.Slots, time.Now())
	if err != nil {
		switch {
		case errors.Is(err, db.ErrConflict):
			http.Error(w, "slot already booked", http.StatusConflict)
		case errors.Is(err, db.ErrInvalidArgument):
			http.Error(w, "invalid booking parameters", http.StatusBadRequest)
		case errors.Is(err, db.ErrNotConsecutiveSlots):
			http.Error(w, "slots must be consecutive", http.StatusBadRequest)
		default:
			log.Printf("[server] BookingHandler: CreateBooking: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) CancelBookingHandler(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok || uid <= 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req cancelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.BookingID <= 0 {
		http.Error(w, "invalid booking_id", http.StatusBadRequest)
		return
	}

	err := s.bookingRepo.CancelBooking(r.Context(), req.BookingID, uid, time.Now())
	if err != nil {
		switch {
		case errors.Is(err, db.ErrNotFound):
			http.Error(w, "booking not found", http.StatusNotFound)
		case errors.Is(err, db.ErrConflict):
			http.Error(w, "cannot cancel booking in current state", http.StatusConflict)
		default:
			log.Printf("[server] CancelBookingHandler: CancelBooking: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusOK)
}
