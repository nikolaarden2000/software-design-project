package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/nikolaarden2000/software-design-project/backend/auth"
	"github.com/nikolaarden2000/software-design-project/backend/db"
	"github.com/nikolaarden2000/software-design-project/backend/httpapi"
	"github.com/nikolaarden2000/software-design-project/backend/rooms"
	"github.com/nikolaarden2000/software-design-project/backend/users"
)

type createCompanyRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type createLocationRequest struct {
	CompanyID int     `json:"company_id"`
	City      string  `json:"city"`
	Address   string  `json:"address"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	Timezone  string  `json:"timezone"`
}

type createAdminRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type assignAdminLocationRequest struct {
	LocationID int `json:"location_id"`
}

type rejectRoomRequest struct {
	Reason string `json:"reason"`
}

func (s *Server) ListCompaniesHandler(w http.ResponseWriter, r *http.Request) {
	companies, err := s.companyRepo.ListCompanies(r.Context())
	if err != nil {
		log.Printf("[server] ListCompaniesHandler: %v", err)
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"items": companies,
	})
}

func (s *Server) CreateCompanyHandler(w http.ResponseWriter, r *http.Request) {
	var req createCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_request", "Некорректное тело запроса")
		return
	}

	company, err := s.companyRepo.CreateCompany(r.Context(), req.Name, req.Description)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrInvalidArgument):
			httpapi.WriteError(w, http.StatusBadRequest, "invalid_request", "Название компании обязательно")
		case errors.Is(err, db.ErrConflict):
			httpapi.WriteError(w, http.StatusConflict, "company_already_exists", "Компания с таким названием уже существует")
		default:
			log.Printf("[server] CreateCompanyHandler: %v", err)
			httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		}
		return
	}

	httpapi.WriteJSON(w, http.StatusCreated, company)
}

func (s *Server) ListLocationsHandler(w http.ResponseWriter, r *http.Request) {
	companyID, ok := optionalIntQuery(w, r, "company_id", "invalid_company_id", "Некорректный company_id")
	if !ok {
		return
	}

	city := optionalStringQuery(r, "city")

	locations, err := s.locationRepo.ListLocations(r.Context(), companyID, city)
	if err != nil {
		log.Printf("[server] ListLocationsHandler: %v", err)
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"items": locations,
	})
}

func (s *Server) CreateLocationHandler(w http.ResponseWriter, r *http.Request) {
	var req createLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_request", "Некорректное тело запроса")
		return
	}

	exists, err := s.companyRepo.ExistsByID(r.Context(), req.CompanyID)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrInvalidID):
			httpapi.WriteError(w, http.StatusBadRequest, "invalid_request", "Некорректный company_id")
		default:
			log.Printf("[server] CreateLocationHandler: ExistsByID(%d): %v", req.CompanyID, err)
			httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		}
		return
	}

	if !exists {
		httpapi.WriteError(w, http.StatusNotFound, "company_not_found", "Компания не найдена")
		return
	}

	location, err := s.locationRepo.CreateLocation(
		r.Context(),
		req.CompanyID,
		req.City,
		req.Address,
		req.Lat,
		req.Lng,
		req.Timezone,
	)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrInvalidID), errors.Is(err, db.ErrInvalidArgument):
			httpapi.WriteError(w, http.StatusBadRequest, "invalid_request", "Некорректные данные локации")
		default:
			log.Printf("[server] CreateLocationHandler: %v", err)
			httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		}
		return
	}

	httpapi.WriteJSON(w, http.StatusCreated, location)
}

func (s *Server) ListAdminsHandler(w http.ResponseWriter, r *http.Request) {
	admins, err := s.userRepo.ListAdmins(r.Context())
	if err != nil {
		log.Printf("[server] ListAdminsHandler: %v", err)
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"items": admins,
	})
}

func (s *Server) CreateAdminHandler(w http.ResponseWriter, r *http.Request) {
	var req createAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_request", "Некорректное тело запроса")
		return
	}

	adminID, err := s.auth.RegisterWithRole(
		r.Context(),
		req.Username,
		req.Email,
		req.Password,
		users.RoleAdmin,
	)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrEmailExists):
			httpapi.WriteError(w, http.StatusConflict, "email_already_exists", "Пользователь с таким email уже существует")
		case errors.Is(err, auth.ErrInvalidEmail):
			httpapi.WriteError(w, http.StatusBadRequest, "invalid_email", "Некорректный email")
		case errors.Is(err, auth.ErrEmptyEmail), errors.Is(err, auth.ErrEmptyUsername):
			httpapi.WriteError(w, http.StatusBadRequest, "invalid_request", "Заполните имя и email")
		case errors.Is(err, auth.ErrPasswordTooShort), errors.Is(err, auth.ErrPasswordTooLong), errors.Is(err, auth.ErrPasswordInvalidChars):
			httpapi.WriteError(w, http.StatusBadRequest, "invalid_password", "Некорректный пароль")
		default:
			log.Printf("[server] CreateAdminHandler: %v", err)
			httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		}
		return
	}

	u, err := s.auth.GetUserByID(r.Context(), adminID)
	if err != nil {
		log.Printf("[server] CreateAdminHandler: GetUserByID(%d): %v", adminID, err)
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		return
	}

	httpapi.WriteJSON(w, http.StatusCreated, map[string]any{
		"id":        u.ID,
		"username":  u.Username,
		"email":     u.Email,
		"role":      u.Role,
		"locations": []any{},
	})
}

func (s *Server) AssignAdminToLocationHandler(w http.ResponseWriter, r *http.Request) {
	adminID, ok := parsePathID(w, r, "admin_id", "invalid_admin_id", "Некорректный id администратора")
	if !ok {
		return
	}

	var req assignAdminLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_request", "Некорректное тело запроса")
		return
	}

	if req.LocationID <= 0 {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_request", "Некорректный location_id")
		return
	}

	if err := s.userRepo.AssignAdminToLocation(r.Context(), adminID, req.LocationID); err != nil {
		switch {
		case errors.Is(err, db.ErrConflict):
			httpapi.WriteError(w, http.StatusConflict, "assignment_already_exists", "Администратор уже назначен на эту локацию")
		case errors.Is(err, db.ErrNotFound):
			httpapi.WriteError(w, http.StatusNotFound, "admin_not_found", "Администратор или локация не найдены")
		case errors.Is(err, db.ErrInvalidID):
			httpapi.WriteError(w, http.StatusBadRequest, "invalid_request", "Некорректный id")
		default:
			log.Printf("[server] AssignAdminToLocationHandler: %v", err)
			httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		}
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"admin_id":    adminID,
		"location_id": req.LocationID,
	})
}

func (s *Server) DeleteAdminLocationAssignmentHandler(w http.ResponseWriter, r *http.Request) {
	adminID, ok := parsePathID(w, r, "admin_id", "invalid_admin_id", "Некорректный id администратора")
	if !ok {
		return
	}

	locationID, ok := parsePathID(w, r, "location_id", "invalid_location_id", "Некорректный id локации")
	if !ok {
		return
	}

	if err := s.userRepo.DeleteAdminLocationAssignment(r.Context(), adminID, locationID); err != nil {
		switch {
		case errors.Is(err, db.ErrNotFound):
			httpapi.WriteError(w, http.StatusNotFound, "assignment_not_found", "Назначение не найдено")
		case errors.Is(err, db.ErrInvalidID):
			httpapi.WriteError(w, http.StatusBadRequest, "invalid_request", "Некорректный id")
		default:
			log.Printf("[server] DeleteAdminLocationAssignmentHandler: %v", err)
			httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		}
		return
	}

	httpapi.WriteJSON(w, http.StatusNoContent, nil)
}

func (s *Server) ModerationRoomsHandler(w http.ResponseWriter, r *http.Request) {
	items, err := s.roomRepo.ListModerationRooms(r.Context())
	if err != nil {
		log.Printf("[server] ModerationRoomsHandler: %v", err)
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"items": items,
	})
}

func (s *Server) ApproveRoomHandler(w http.ResponseWriter, r *http.Request) {
	roomID, ok := parsePathID(w, r, "room_id", "invalid_room_id", "Некорректный id помещения")
	if !ok {
		return
	}

	err := s.roomRepo.ApproveRoom(r.Context(), roomID)
	if err != nil {
		writeModerationRoomError(w, err, "cannot_approve_room", "Нельзя одобрить помещение в текущем статусе")
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"id":     roomID,
		"status": rooms.StatusPublished,
	})
}

func (s *Server) RejectRoomHandler(w http.ResponseWriter, r *http.Request) {
	roomID, ok := parsePathID(w, r, "room_id", "invalid_room_id", "Некорректный id помещения")
	if !ok {
		return
	}

	var req rejectRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_request", "Некорректное тело запроса")
		return
	}

	err := s.roomRepo.RejectRoom(r.Context(), roomID, req.Reason)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrInvalidArgument):
			httpapi.WriteError(w, http.StatusBadRequest, "invalid_request", "Укажите причину отклонения")
		default:
			writeModerationRoomError(w, err, "cannot_reject_room", "Нельзя отклонить помещение в текущем статусе")
		}
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"id":               roomID,
		"status":           rooms.StatusRejected,
		"rejection_reason": req.Reason,
	})
}

func (s *Server) ArchiveRoomHandler(w http.ResponseWriter, r *http.Request) {
	roomID, ok := parsePathID(w, r, "room_id", "invalid_room_id", "Некорректный id помещения")
	if !ok {
		return
	}

	err := s.roomRepo.ArchiveRoom(r.Context(), roomID)
	if err != nil {
		writeModerationRoomError(w, err, "cannot_archive_room", "Нельзя архивировать помещение в текущем статусе")
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"id":     roomID,
		"status": rooms.StatusArchived,
	})
}

func writeModerationRoomError(w http.ResponseWriter, err error, conflictCode, conflictMessage string) {
	switch {
	case errors.Is(err, db.ErrInvalidID):
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_room_id", "Некорректный id помещения")
	case errors.Is(err, db.ErrNotFound):
		httpapi.WriteError(w, http.StatusNotFound, "room_not_found", "Помещение не найдено")
	case errors.Is(err, db.ErrConflict):
		httpapi.WriteError(w, http.StatusConflict, conflictCode, conflictMessage)
	default:
		log.Printf("[server] moderation room error: %v", err)
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
	}
}
