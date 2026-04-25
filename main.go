package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"gitlab.com/5130904-20104-teams/software-design-project/internal/auth"
	"gitlab.com/5130904-20104-teams/software-design-project/internal/db"
	"gitlab.com/5130904-20104-teams/software-design-project/internal/server"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	userRepo := db.NewUserRepo(pool)
	roomRepo := db.NewRoomRepo(pool)
	bookingRepo := db.NewBookingRepo(pool)

	authService := auth.NewAuthService(userRepo, "session_id", 30*time.Minute, 5*time.Minute)

	staticPath := os.Getenv("STATIC_PATH")

	tmplHTML := template.Must(template.ParseFiles(
		staticPath+"/home/home.html",
		staticPath+"/room/room.html",
		staticPath+"/auth/auth.html",
		staticPath+"/me/me.html",
	))

	srv := server.NewServer(authService, roomRepo, bookingRepo, tmplHTML)

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	mux.Handle("/auth", authService.AuthMiddleware(http.HandlerFunc(srv.AuthHandler)))
	mux.HandleFunc("POST /api/register", srv.RegisterHandler)
	mux.HandleFunc("POST /api/login", srv.LoginHandler)
	mux.HandleFunc("POST /api/logout", srv.LogoutHandler)

	mux.Handle("/", http.RedirectHandler("/", http.StatusMovedPermanently))
	mux.Handle("GET /{$}", authService.AuthMiddleware(http.HandlerFunc(srv.HomeHandler)))
	mux.HandleFunc("/ws/home", srv.HomeWSHandler)

	mux.Handle("/room/", authService.AuthMiddleware(http.HandlerFunc(srv.RoomHandler)))
	mux.HandleFunc("/ws/booking", srv.BookingWSHandler)
	mux.Handle("POST /api/booking/new", authService.AuthMiddleware(http.HandlerFunc(srv.BookingHandler)))

	mux.Handle("/me", authService.AuthMiddleware(http.HandlerFunc(srv.MeHandler)))
	mux.Handle("/ws/me", authService.AuthMiddleware(http.HandlerFunc(srv.MeWSHandler)))
	mux.Handle("POST /api/booking/stop", authService.AuthMiddleware(http.HandlerFunc(srv.CancelBookingHandler)))

	log.Println("Server started on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
