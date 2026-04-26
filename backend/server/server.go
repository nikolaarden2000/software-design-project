package server

import (
	"context"
	"net/http"
	"time"

	"github.com/nikolaarden2000/software-design-project/backend/bookings"
	"github.com/nikolaarden2000/software-design-project/backend/rooms"
	"github.com/nikolaarden2000/software-design-project/backend/users"
)

type AuthService interface {
	Register(ctx context.Context, username, email, password string) (int, error)
	Login(ctx context.Context, email, password string, w http.ResponseWriter) (*users.User, error)
	Logout(w http.ResponseWriter, r *http.Request)
	GetUserByID(ctx context.Context, id int) (*users.User, error)
}

type RoomRepo interface {
	GetRoomsBatchByCity(ctx context.Context, lastID, limit int, city string) ([]rooms.Room, error)
	GetCompaniesByCity(ctx context.Context, city string) ([]string, error)
	GetRoomPageData(ctx context.Context, roomID int) (*rooms.RoomPageData, error)
}

type BookingRepo interface {
	GetRoomAvailability(ctx context.Context, roomID, days int, now time.Time) ([]rooms.DateAvailability, error)
	CreateBooking(ctx context.Context, userID, roomID int, date string, slots []string, now time.Time) (int, error)
	GetUserBookings(ctx context.Context, userID int, now time.Time) ([]bookings.BookingHistoryItem, error)
	CancelBooking(ctx context.Context, bookingID, userID int, now time.Time) error
}

type Server struct {
	auth        AuthService
	roomRepo    RoomRepo
	bookingRepo BookingRepo
}

func NewServer(auth AuthService, roomRepo RoomRepo, bookingRepo BookingRepo) *Server {
	return &Server{
		auth:        auth,
		roomRepo:    roomRepo,
		bookingRepo: bookingRepo,
	}
}
