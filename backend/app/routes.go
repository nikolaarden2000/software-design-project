package app

import (
	"net/http"

	"github.com/nikolaarden2000/software-design-project/backend/auth"
	"github.com/nikolaarden2000/software-design-project/backend/httpapi"
	"github.com/nikolaarden2000/software-design-project/backend/server"
)

func RegisterRoutes(mux *http.ServeMux, authService *auth.AuthService, srv *server.Server) {
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		httpapi.WriteJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	mux.Handle(
		"GET /api/me",
		authService.AuthMiddleware(http.HandlerFunc(srv.MeHandler)),
	)

	mux.HandleFunc("POST /api/register", srv.RegisterHandler)
	mux.HandleFunc("POST /api/login", srv.LoginHandler)
	mux.HandleFunc("POST /api/logout", srv.LogoutHandler)

	mux.HandleFunc("GET /api/rooms", srv.RoomsHandler)
	mux.HandleFunc("GET /api/rooms/filters", srv.RoomFiltersHandler)
	mux.HandleFunc("GET /api/rooms/{id}", srv.RoomDetailsHandler)
	mux.HandleFunc("GET /api/rooms/{id}/availability", srv.RoomAvailabilityHandler)

	mux.Handle(
		"POST /api/bookings",
		authService.AuthMiddleware(http.HandlerFunc(srv.BookingHandler)),
	)

	mux.Handle(
		"GET /api/me/bookings",
		authService.AuthMiddleware(http.HandlerFunc(srv.MyBookingsHandler)),
	)

	mux.Handle(
		"POST /api/bookings/{id}/cancel",
		authService.AuthMiddleware(http.HandlerFunc(srv.CancelBookingHandler)),
	)
}
