package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/nikolaarden2000/software-design-project/backend/db"
	"github.com/nikolaarden2000/software-design-project/backend/httpapi"
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

func optionalIntQuery(w http.ResponseWriter, r *http.Request, name, code, message string) (*int, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return nil, true
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		httpapi.WriteError(w, http.StatusBadRequest, code, message)
		return nil, false
	}

	return &value, true
}

func optionalStringQuery(r *http.Request, name string) *string {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return nil
	}

	return &raw
}
