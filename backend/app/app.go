package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"gitlab.com/5130904-20104-teams/software-design-project/backend/auth"
	"gitlab.com/5130904-20104-teams/software-design-project/backend/bookings"
	"gitlab.com/5130904-20104-teams/software-design-project/backend/companies"
	"gitlab.com/5130904-20104-teams/software-design-project/backend/locations"
	"gitlab.com/5130904-20104-teams/software-design-project/backend/rooms"
	"gitlab.com/5130904-20104-teams/software-design-project/backend/server"
	"gitlab.com/5130904-20104-teams/software-design-project/backend/users"
)

type App struct {
	Config     Config
	DB         *pgxpool.Pool
	HTTPServer *http.Server
	Auth       *auth.AuthService
}

func New(ctx context.Context, cfg Config) (*App, error) {
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is not set")
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	userRepo := users.NewRepository(pool)
	companyRepo := companies.NewRepository(pool)
	locationRepo := locations.NewRepository(pool)
	roomRepo := rooms.NewRepository(pool)
	bookingRepo := bookings.NewRepository(pool)

	authService := auth.NewAuthService(
		userRepo,
		cfg.CookieName,
		30*time.Minute,
		5*time.Minute,
	)

	apiServer := server.NewServer(
		authService,
		userRepo,
		companyRepo,
		locationRepo,
		roomRepo,
		bookingRepo,
	)

	mux := http.NewServeMux()
	RegisterRoutes(mux, authService, apiServer)

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &App{
		Config:     cfg,
		DB:         pool,
		HTTPServer: httpServer,
		Auth:       authService,
	}, nil
}

func (a *App) Close() {
	if a.Auth != nil {
		a.Auth.Shutdown()
	}

	if a.DB != nil {
		a.DB.Close()
	}
}
