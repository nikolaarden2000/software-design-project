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

func (s *Server) AdminLocationsHandler(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := currentUserFromRequest(w, r)
	if !ok {
		return
	}

	items, err := s.locationRepo.ListAdminLocations(
		r.Context(),
		currentUser.ID,
		currentUser.Role == users.RoleSuperuser,
	)
	if err != nil {
		log.Printf("[server] AdminLocationsHandler: %v", err)
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"items": items,
	})
}

func (s *Server) AdminRoomsHandler(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := currentUserFromRequest(w, r)
	if !ok {
		return
	}

	locationID, ok := optionalIntQuery(w, r, "location_id", "invalid_location_id", "Некорректный location_id")
	if !ok {
		return
	}

	status := optionalStringQuery(r, "status")

	items, err := s.roomRepo.ListAdminRooms(
		r.Context(),
		currentUser.ID,
		currentUser.Role == users.RoleSuperuser,
		locationID,
		status,
	)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrForbidden):
			httpapi.WriteError(w, http.StatusForbidden, "forbidden", "Недостаточно прав")
		case errors.Is(err, db.ErrNotFound):
			httpapi.WriteError(w, http.StatusNotFound, "location_not_found", "Локация не найдена")
		case errors.Is(err, db.ErrInvalidArgument):
			httpapi.WriteError(w, http.StatusBadRequest, "invalid_request", "Некорректные параметры запроса")
		default:
			log.Printf("[server] AdminRoomsHandler: %v", err)
			httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		}
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"items": items,
	})
}

func (s *Server) AdminRoomDetailsHandler(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := currentUserFromRequest(w, r)
	if !ok {
		return
	}

	roomID, ok := parsePathID(w, r, "room_id", "invalid_room_id", "Некорректный id помещения")
	if !ok {
		return
	}

	room, err := s.roomRepo.GetAdminRoom(
		r.Context(),
		currentUser.ID,
		currentUser.Role == users.RoleSuperuser,
		roomID,
	)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrForbidden):
			httpapi.WriteError(w, http.StatusForbidden, "forbidden", "Недостаточно прав")
		case errors.Is(err, db.ErrNotFound):
			httpapi.WriteError(w, http.StatusNotFound, "room_not_found", "Помещение не найдено")
		default:
			log.Printf("[server] AdminRoomDetailsHandler: %v", err)
			httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		}
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, room)
}

func (s *Server) CreateAdminRoomHandler(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := currentUserFromRequest(w, r)
	if !ok {
		return
	}

	var req rooms.AdminRoomInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_request", "Некорректное тело запроса")
		return
	}

	room, err := s.roomRepo.CreateAdminRoom(
		r.Context(),
		currentUser.ID,
		currentUser.Role == users.RoleSuperuser,
		req,
	)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrInvalidID), errors.Is(err, db.ErrInvalidArgument):
			httpapi.WriteError(w, http.StatusBadRequest, "invalid_request", "Некорректные данные помещения")
		case errors.Is(err, db.ErrForbidden):
			httpapi.WriteError(w, http.StatusForbidden, "forbidden", "Недостаточно прав")
		case errors.Is(err, db.ErrNotFound):
			httpapi.WriteError(w, http.StatusNotFound, "location_not_found", "Локация не найдена")
		default:
			log.Printf("[server] CreateAdminRoomHandler: %v", err)
			httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		}
		return
	}

	httpapi.WriteJSON(w, http.StatusCreated, room)
}

func (s *Server) UpdateAdminRoomHandler(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := currentUserFromRequest(w, r)
	if !ok {
		return
	}

	roomID, ok := parsePathID(w, r, "room_id", "invalid_room_id", "Некорректный id помещения")
	if !ok {
		return
	}

	var req rooms.AdminRoomInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid_request", "Некорректное тело запроса")
		return
	}

	err := s.roomRepo.UpdateAdminRoom(
		r.Context(),
		currentUser.ID,
		currentUser.Role == users.RoleSuperuser,
		roomID,
		req,
	)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrInvalidID), errors.Is(err, db.ErrInvalidArgument):
			httpapi.WriteError(w, http.StatusBadRequest, "invalid_request", "Некорректные данные помещения")
		case errors.Is(err, db.ErrForbidden):
			httpapi.WriteError(w, http.StatusForbidden, "forbidden", "Недостаточно прав")
		case errors.Is(err, db.ErrNotFound):
			httpapi.WriteError(w, http.StatusNotFound, "room_not_found", "Помещение не найдено")
		case errors.Is(err, db.ErrConflict):
			httpapi.WriteError(w, http.StatusConflict, "cannot_edit_room", "Нельзя редактировать помещение в текущем статусе")
		default:
			log.Printf("[server] UpdateAdminRoomHandler: %v", err)
			httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		}
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"id":     roomID,
		"status": rooms.StatusDraft,
	})
}

func (s *Server) SubmitAdminRoomHandler(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := currentUserFromRequest(w, r)
	if !ok {
		return
	}

	roomID, ok := parsePathID(w, r, "room_id", "invalid_room_id", "Некорректный id помещения")
	if !ok {
		return
	}

	err := s.roomRepo.SubmitAdminRoom(
		r.Context(),
		currentUser.ID,
		currentUser.Role == users.RoleSuperuser,
		roomID,
	)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrForbidden):
			httpapi.WriteError(w, http.StatusForbidden, "forbidden", "Недостаточно прав")
		case errors.Is(err, db.ErrNotFound):
			httpapi.WriteError(w, http.StatusNotFound, "room_not_found", "Помещение не найдено")
		case errors.Is(err, db.ErrConflict):
			httpapi.WriteError(w, http.StatusConflict, "cannot_submit_room", "Нельзя отправить помещение на модерацию в текущем статусе")
		default:
			log.Printf("[server] SubmitAdminRoomHandler: %v", err)
			httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		}
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, map[string]any{
		"id":     roomID,
		"status": rooms.StatusPending,
	})
}

func currentUserFromRequest(w http.ResponseWriter, r *http.Request) (*users.User, bool) {
	currentUser, ok := auth.UserFromContext(r.Context())
	if !ok || currentUser == nil {
		httpapi.WriteError(w, http.StatusUnauthorized, "unauthorized", "Необходимо войти в аккаунт")
		return nil, false
	}

	return currentUser, true
}
