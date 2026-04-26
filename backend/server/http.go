package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/nikolaarden2000/software-design-project/backend/auth"
	"github.com/nikolaarden2000/software-design-project/backend/db"
	"github.com/nikolaarden2000/software-design-project/backend/httpapi"
	"github.com/nikolaarden2000/software-design-project/backend/users"
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

type userDTO struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

func toUserDTO(u *users.User) userDTO {
	return userDTO{
		ID:       u.ID,
		Username: u.Username,
		Email:    u.Email,
		Role:     u.Role,
	}
}

func (s *Server) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_request", "Некорректное тело запроса")
		return
	}

	id, err := s.auth.Register(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrEmailExists):
			httpapi.WriteError(w, http.StatusConflict, "email_already_exists", "Пользователь с таким email уже существует")
		case errors.Is(err, auth.ErrEmptyUsername):
			httpapi.WriteError(w, http.StatusBadRequest, "empty_username", "Имя пользователя не должно быть пустым")
		case errors.Is(err, auth.ErrEmptyEmail):
			httpapi.WriteError(w, http.StatusBadRequest, "empty_email", "Email не должен быть пустым")
		case errors.Is(err, auth.ErrInvalidEmail):
			httpapi.WriteError(w, http.StatusBadRequest, "invalid_email", "Некорректный email")
		case errors.Is(err, auth.ErrPasswordTooShort):
			httpapi.WriteError(w, http.StatusBadRequest, "password_too_short", "Пароль слишком короткий")
		case errors.Is(err, auth.ErrPasswordTooLong):
			httpapi.WriteError(w, http.StatusBadRequest, "password_too_long", "Пароль слишком длинный")
		case errors.Is(err, auth.ErrPasswordInvalidChars):
			httpapi.WriteError(w, http.StatusBadRequest, "password_invalid_chars", "Пароль содержит недопустимые символы")
		default:
			log.Printf("[server] RegisterHandler: %v", err)
			httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		}
		return
	}

	u, err := s.auth.GetUserByID(r.Context(), id)
	if err != nil {
		log.Printf("[server] RegisterHandler: GetUserByID(%d): %v", id, err)
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		return
	}

	httpapi.WriteJSON(w, http.StatusCreated, map[string]any{
		"user": toUserDTO(u),
	})
}

func (s *Server) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_request", "Некорректное тело запроса")
		return
	}

	u, err := s.auth.Login(r.Context(), req.Email, req.Password, w)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrNoUser), errors.Is(err, auth.ErrInvalidCredentials):
			httpapi.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "Неверный email или пароль")
		case errors.Is(err, auth.ErrEmptyEmail):
			httpapi.WriteError(w, http.StatusBadRequest, "empty_email", "Email не должен быть пустым")
		default:
			log.Printf("[server] LoginHandler: %v", err)
			httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		}
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"user": toUserDTO(u),
	})
}

func (s *Server) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	s.auth.Logout(w, r)
	httpapi.WriteJSON(w, http.StatusNoContent, nil)
}

func (s *Server) MeHandler(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok || uid <= 0 {
		httpapi.WriteJSON(w, http.StatusOK, map[string]any{
			"authenticated": false,
			"user":          nil,
		})
		return
	}

	u, err := s.auth.GetUserByID(r.Context(), uid)
	if err != nil {
		log.Printf("[server] MeHandler: GetUserByID(%d): %v", uid, err)
		httpapi.WriteJSON(w, http.StatusOK, map[string]any{
			"authenticated": false,
			"user":          nil,
		})
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"user":          toUserDTO(u),
	})
}

func (s *Server) RoomsHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	city := q.Get("city")
	if city == "" {
		city = "Москва"
	}

	limit := parsePositiveInt(q.Get("limit"), 100)
	if limit > 100 {
		limit = 100
	}

	afterID := parsePositiveInt(q.Get("after_id"), 0)

	rooms, err := s.roomRepo.GetRoomsBatchByCity(r.Context(), afterID, limit, city)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrInvalidID):
			httpapi.WriteError(w, http.StatusBadRequest, "invalid_after_id", "Некорректный after_id")
		case errors.Is(err, db.ErrInvalidArgument):
			httpapi.WriteError(w, http.StatusBadRequest, "invalid_limit", "Некорректный limit")
		default:
			log.Printf("[server] RoomsHandler: %v", err)
			httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		}
		return
	}

	var nextAfterID any
	hasMore := len(rooms) == limit
	if len(rooms) > 0 {
		nextAfterID = rooms[len(rooms)-1].ID
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"items": rooms,
		"pagination": map[string]any{
			"limit":         limit,
			"next_after_id": nextAfterID,
			"has_more":      hasMore,
		},
	})
}

func (s *Server) RoomFiltersHandler(w http.ResponseWriter, r *http.Request) {
	city := r.URL.Query().Get("city")
	if city == "" {
		city = "Москва"
	}

	companies, err := s.roomRepo.GetCompaniesByCity(r.Context(), city)
	if err != nil {
		log.Printf("[server] RoomFiltersHandler: %v", err)
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"city":      city,
		"companies": companies,
	})
}

func (s *Server) RoomDetailsHandler(w http.ResponseWriter, r *http.Request) {
	roomID, ok := parsePathID(w, r, "id", "invalid_room_id", "Некорректный id комнаты")
	if !ok {
		return
	}

	data, err := s.roomRepo.GetRoomPageData(r.Context(), roomID)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrNotFound):
			httpapi.WriteError(w, http.StatusNotFound, "room_not_found", "Комната не найдена")
		case errors.Is(err, db.ErrInvalidID):
			httpapi.WriteError(w, http.StatusBadRequest, "invalid_room_id", "Некорректный id комнаты")
		default:
			log.Printf("[server] RoomDetailsHandler: GetRoomPageData(%d): %v", roomID, err)
			httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		}
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, data)
}

func (s *Server) RoomAvailabilityHandler(w http.ResponseWriter, r *http.Request) {
	roomID, ok := parsePathID(w, r, "id", "invalid_room_id", "Некорректный id комнаты")
	if !ok {
		return
	}

	days := parsePositiveInt(r.URL.Query().Get("days"), 7)
	if days > 31 {
		days = 31
	}

	availability, err := s.bookingRepo.GetRoomAvailability(r.Context(), roomID, days, time.Now())
	if err != nil {
		switch {
		case errors.Is(err, db.ErrNotFound):
			httpapi.WriteError(w, http.StatusNotFound, "room_not_found", "Комната не найдена")
		case errors.Is(err, db.ErrInvalidID):
			httpapi.WriteError(w, http.StatusBadRequest, "invalid_room_id", "Некорректный id комнаты")
		default:
			log.Printf("[server] RoomAvailabilityHandler: %v", err)
			httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		}
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"room_id": roomID,
		"dates":   availability,
	})
}

func (s *Server) BookingHandler(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok || uid <= 0 {
		httpapi.WriteError(w, http.StatusUnauthorized, "unauthorized", "Необходимо войти в аккаунт")
		return
	}

	var req bookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_request", "Некорректное тело запроса")
		return
	}

	bookingID, err := s.bookingRepo.CreateBooking(r.Context(), uid, req.RoomID, req.Date, req.Slots, time.Now())
	if err != nil {
		switch {
		case errors.Is(err, db.ErrConflict):
			httpapi.WriteError(w, http.StatusConflict, "slot_already_booked", "Выбранный слот уже забронирован")
		case errors.Is(err, db.ErrInvalidID):
			httpapi.WriteError(w, http.StatusBadRequest, "invalid_id", "Некорректный id пользователя или комнаты")
		case errors.Is(err, db.ErrInvalidArgument):
			httpapi.WriteError(w, http.StatusBadRequest, "invalid_booking_parameters", "Некорректные параметры бронирования")
		case errors.Is(err, db.ErrNotConsecutiveSlots):
			httpapi.WriteError(w, http.StatusBadRequest, "slots_must_be_consecutive", "Слоты должны идти подряд")
		case errors.Is(err, db.ErrNotFound):
			httpapi.WriteError(w, http.StatusNotFound, "room_not_found", "Комната не найдена")
		default:
			log.Printf("[server] BookingHandler: CreateBooking: %v", err)
			httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		}
		return
	}

	httpapi.WriteJSON(w, http.StatusCreated, map[string]any{
		"id":      bookingID,
		"room_id": req.RoomID,
		"date":    req.Date,
		"slots":   req.Slots,
		"status":  "booked",
	})
}

func (s *Server) MyBookingsHandler(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok || uid <= 0 {
		httpapi.WriteError(w, http.StatusUnauthorized, "unauthorized", "Необходимо войти в аккаунт")
		return
	}

	bookings, err := s.bookingRepo.GetUserBookings(r.Context(), uid, time.Now())
	if err != nil {
		log.Printf("[server] MyBookingsHandler: %v", err)
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"items": bookings,
	})
}

func (s *Server) CancelBookingHandler(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok || uid <= 0 {
		httpapi.WriteError(w, http.StatusUnauthorized, "unauthorized", "Необходимо войти в аккаунт")
		return
	}

	bookingID, ok := parsePathID(w, r, "id", "invalid_booking_id", "Некорректный id бронирования")
	if !ok {
		return
	}

	err := s.bookingRepo.CancelBooking(r.Context(), bookingID, uid, time.Now())
	if err != nil {
		switch {
		case errors.Is(err, db.ErrNotFound):
			httpapi.WriteError(w, http.StatusNotFound, "booking_not_found", "Бронирование не найдено")
		case errors.Is(err, db.ErrConflict):
			httpapi.WriteError(w, http.StatusConflict, "cannot_cancel_booking", "Нельзя отменить бронирование в текущем состоянии")
		case errors.Is(err, db.ErrInvalidID):
			httpapi.WriteError(w, http.StatusBadRequest, "invalid_booking_id", "Некорректный id бронирования")
		default:
			log.Printf("[server] CancelBookingHandler: CancelBooking: %v", err)
			httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		}
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"id":     bookingID,
		"status": "canceled",
	})
}

func parsePositiveInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return fallback
	}

	return value
}

func parsePathID(w http.ResponseWriter, r *http.Request, name, code, message string) (int, bool) {
	raw := r.PathValue(name)
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		httpapi.WriteError(w, http.StatusBadRequest, code, message)
		return 0, false
	}

	return id, true
}
