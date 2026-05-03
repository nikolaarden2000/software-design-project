package app

import (
	"net/http"

	"gitlab.com/5130904-20104-teams/software-design-project/backend/auth"
	"gitlab.com/5130904-20104-teams/software-design-project/backend/httpapi"
	"gitlab.com/5130904-20104-teams/software-design-project/backend/server"
	"gitlab.com/5130904-20104-teams/software-design-project/backend/users"
)

func RegisterRoutes(mux *http.ServeMux, authService *auth.AuthService, srv *server.Server) {
	requireAuth := authService.RequireAuth
	requireAdmin := authService.RequireRole(users.RoleAdmin, users.RoleSuperuser)
	requireSuperuser := authService.RequireRole(users.RoleSuperuser)

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		httpapi.WriteJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	mux.Handle(
		"GET /api/me",
		authService.OptionalUserMiddleware(http.HandlerFunc(srv.MeHandler)),
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
		requireAuth(http.HandlerFunc(srv.BookingHandler)),
	)

	mux.Handle(
		"GET /api/me/bookings",
		requireAuth(http.HandlerFunc(srv.MyBookingsHandler)),
	)

	mux.Handle(
		"POST /api/bookings/{id}/cancel",
		requireAuth(http.HandlerFunc(srv.CancelBookingHandler)),
	)

	mux.Handle(
		"GET /api/admin/locations",
		requireAdmin(http.HandlerFunc(srv.AdminLocationsHandler)),
	)

	mux.Handle(
		"GET /api/admin/rooms",
		requireAdmin(http.HandlerFunc(srv.AdminRoomsHandler)),
	)

	mux.Handle(
		"GET /api/admin/rooms/{room_id}",
		requireAdmin(http.HandlerFunc(srv.AdminRoomDetailsHandler)),
	)

	mux.Handle(
		"POST /api/admin/rooms",
		requireAdmin(http.HandlerFunc(srv.CreateAdminRoomHandler)),
	)

	mux.Handle(
		"PATCH /api/admin/rooms/{room_id}",
		requireAdmin(http.HandlerFunc(srv.UpdateAdminRoomHandler)),
	)

	mux.Handle(
		"POST /api/admin/rooms/{room_id}/submit",
		requireAdmin(http.HandlerFunc(srv.SubmitAdminRoomHandler)),
	)

	mux.Handle(
		"POST /api/admin/rooms/{room_id}/archive",
		requireAdmin(http.HandlerFunc(srv.ArchiveAdminRoomHandler)),
	)

	mux.Handle(
		"GET /api/admin/bookings",
		requireAdmin(http.HandlerFunc(srv.AdminBookingsHandler)),
	)

	mux.Handle(
		"POST /api/admin/bookings/{booking_id}/cancel",
		requireAdmin(http.HandlerFunc(srv.CancelAdminBookingHandler)),
	)

	mux.Handle(
		"GET /api/superuser/companies",
		requireSuperuser(http.HandlerFunc(srv.ListCompaniesHandler)),
	)

	mux.Handle(
		"POST /api/superuser/companies",
		requireSuperuser(http.HandlerFunc(srv.CreateCompanyHandler)),
	)

	mux.Handle(
		"GET /api/superuser/locations",
		requireSuperuser(http.HandlerFunc(srv.ListLocationsHandler)),
	)

	mux.Handle(
		"POST /api/superuser/locations",
		requireSuperuser(http.HandlerFunc(srv.CreateLocationHandler)),
	)

	mux.Handle(
		"GET /api/superuser/admins",
		requireSuperuser(http.HandlerFunc(srv.ListAdminsHandler)),
	)

	mux.Handle(
		"POST /api/superuser/admins",
		requireSuperuser(http.HandlerFunc(srv.CreateAdminHandler)),
	)

	mux.Handle(
		"POST /api/superuser/admins/{admin_id}/locations",
		requireSuperuser(http.HandlerFunc(srv.AssignAdminToLocationHandler)),
	)

	mux.Handle(
		"DELETE /api/superuser/admins/{admin_id}/locations/{location_id}",
		requireSuperuser(http.HandlerFunc(srv.DeleteAdminLocationAssignmentHandler)),
	)

	mux.Handle(
		"GET /api/superuser/rooms/moderation",
		requireSuperuser(http.HandlerFunc(srv.ModerationRoomsHandler)),
	)

	mux.Handle(
		"POST /api/superuser/rooms/{room_id}/approve",
		requireSuperuser(http.HandlerFunc(srv.ApproveRoomHandler)),
	)

	mux.Handle(
		"POST /api/superuser/rooms/{room_id}/reject",
		requireSuperuser(http.HandlerFunc(srv.RejectRoomHandler)),
	)

	mux.Handle(
		"POST /api/superuser/rooms/{room_id}/archive",
		requireSuperuser(http.HandlerFunc(srv.ArchiveRoomHandler)),
	)
}
