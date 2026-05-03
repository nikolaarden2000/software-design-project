package server

import (
	"net/http"
	"strconv"
	"strings"

	"gitlab.com/5130904-20104-teams/software-design-project/backend/auth"
	"gitlab.com/5130904-20104-teams/software-design-project/backend/httpapi"
	"gitlab.com/5130904-20104-teams/software-design-project/backend/users"
)

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

func currentUserFromRequest(w http.ResponseWriter, r *http.Request) (*users.User, bool) {
	currentUser, ok := auth.UserFromContext(r.Context())
	if !ok || currentUser == nil {
		httpapi.WriteError(w, http.StatusUnauthorized, "unauthorized", "Необходимо войти в аккаунт")
		return nil, false
	}

	return currentUser, true
}
