package server

import (
	"context"
	"net/http"
	"time"

	"gitlab.com/5130904-20104-teams/software-design-project/backend/bookings"
	"gitlab.com/5130904-20104-teams/software-design-project/backend/companies"
	"gitlab.com/5130904-20104-teams/software-design-project/backend/locations"
	"gitlab.com/5130904-20104-teams/software-design-project/backend/rooms"
	"gitlab.com/5130904-20104-teams/software-design-project/backend/users"
)

type AuthService interface {
	Register(ctx context.Context, username, email, password string) (int, error)
	RegisterWithRole(ctx context.Context, username, email, password, role string) (int, error)
	Login(ctx context.Context, email, password string, w http.ResponseWriter) (*users.User, error)
	Logout(w http.ResponseWriter, r *http.Request)
	GetUserByID(ctx context.Context, id int) (*users.User, error)
}

type UserRepo interface {
	ListAdmins(ctx context.Context) ([]users.Admin, error)
	AssignAdminToLocation(ctx context.Context, adminID, locationID int) error
	DeleteAdminLocationAssignment(ctx context.Context, adminID, locationID int) error
}

type CompanyRepo interface {
	ListCompanies(ctx context.Context) ([]companies.Company, error)
	CreateCompany(ctx context.Context, name, description string) (*companies.Company, error)
	ExistsByID(ctx context.Context, id int) (bool, error)
}

type LocationRepo interface {
	ListLocations(ctx context.Context, companyID *int, city *string) ([]locations.Location, error)
	ListAdminLocations(ctx context.Context, adminID int, includeAll bool) ([]locations.AdminLocation, error)
	CreateLocation(ctx context.Context, companyID int, city string, address string, lat float64, lng float64, timezone string) (*locations.Location, error)
	GetLocationByID(ctx context.Context, id int) (*locations.Location, error)
	ExistsByID(ctx context.Context, id int) (bool, error)
}

type RoomRepo interface {
	GetRoomsBatchByCity(ctx context.Context, lastID, limit int, city string) ([]rooms.Room, error)
	GetCompaniesByCity(ctx context.Context, city string) ([]string, error)
	GetRoomPageData(ctx context.Context, roomID int) (*rooms.RoomPageData, error)

	ListAdminRooms(ctx context.Context, adminID int, includeAll bool, locationID *int, status *string) ([]rooms.AdminRoomListItem, error)
	GetAdminRoom(ctx context.Context, adminID int, includeAll bool, roomID int) (*rooms.AdminRoomDetails, error)
	CreateAdminRoom(ctx context.Context, creatorID int, includeAll bool, input rooms.AdminRoomInput) (*rooms.AdminRoomListItem, error)
	UpdateAdminRoom(ctx context.Context, adminID int, includeAll bool, roomID int, input rooms.AdminRoomInput) error
	SubmitAdminRoom(ctx context.Context, adminID int, includeAll bool, roomID int) error
	ArchiveAdminRoom(ctx context.Context, adminID int, includeAll bool, roomID int, mode string, now time.Time) (*rooms.AdminRoomArchiveResult, error)

	ListModerationRooms(ctx context.Context) ([]rooms.ModerationRoom, error)
	ApproveRoom(ctx context.Context, roomID int) error
	RejectRoom(ctx context.Context, roomID int, reason string) error
	ArchiveRoom(ctx context.Context, roomID int) error
}

type BookingRepo interface {
	GetRoomAvailability(ctx context.Context, roomID, days int, now time.Time) ([]rooms.DateAvailability, error)
	CreateBooking(ctx context.Context, userID, roomID int, date string, slots []string, now time.Time) (int, error)
	GetUserBookings(ctx context.Context, userID int, now time.Time) ([]bookings.BookingHistoryItem, error)
	CancelBooking(ctx context.Context, bookingID, userID int, now time.Time) error

	ListAdminBookings(ctx context.Context, adminID int, includeAll bool, locationID *int, roomID *int, status *string, now time.Time) ([]bookings.AdminBookingItem, error)
	CancelAdminBooking(ctx context.Context, adminID int, includeAll bool, bookingID int, now time.Time) error
}

type Server struct {
	auth         AuthService
	userRepo     UserRepo
	companyRepo  CompanyRepo
	locationRepo LocationRepo
	roomRepo     RoomRepo
	bookingRepo  BookingRepo
}

func NewServer(
	auth AuthService,
	userRepo UserRepo,
	companyRepo CompanyRepo,
	locationRepo LocationRepo,
	roomRepo RoomRepo,
	bookingRepo BookingRepo,
) *Server {
	return &Server{
		auth:         auth,
		userRepo:     userRepo,
		companyRepo:  companyRepo,
		locationRepo: locationRepo,
		roomRepo:     roomRepo,
		bookingRepo:  bookingRepo,
	}
}
