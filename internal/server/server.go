package server

import (
	"context"
	"html/template"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"gitlab.com/5130904-20104-teams/software-design-project/internal/models"
)

type AuthService interface {
	Register(ctx context.Context, username, email, password string) (int, error)
	Login(ctx context.Context, email, password string, w http.ResponseWriter) (string, error)
	Logout(w http.ResponseWriter, r *http.Request)
	GetUserByID(ctx context.Context, id int) (*models.User, error)
}

type RoomRepo interface {
	GetRoomsBatchByCity(ctx context.Context, lastID, limit int, city string) ([]models.Room, error)
	GetCompaniesByCity(ctx context.Context, city string) ([]string, error)
	GetRoomPageData(ctx context.Context, roomID int) (*models.RoomPageData, error)
}

type BookingRepo interface {
	GetRoomAvailability(ctx context.Context, roomID, days int, now time.Time) ([]models.DateAvailability, error)
	CreateBooking(ctx context.Context, userID, roomID int, date string, slots []string, now time.Time) (int, error)
	GetUserBookings(ctx context.Context, userID int, now time.Time) ([]models.BookingHistoryItem, error)
	CancelBooking(ctx context.Context, bookingID, userID int, now time.Time) error
}

type Server struct {
	auth        AuthService
	roomRepo    RoomRepo
	bookingRepo BookingRepo
	tmplHTML    *template.Template
	upgrader    websocket.Upgrader
}

func NewServer(auth AuthService, roomRepo RoomRepo, bookingRepo BookingRepo, tmpl *template.Template) *Server {
	return &Server{
		auth:        auth,
		roomRepo:    roomRepo,
		bookingRepo: bookingRepo,
		tmplHTML:    tmpl,
		upgrader: websocket.Upgrader{
			CheckOrigin:     func(r *http.Request) bool { return true },
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}
