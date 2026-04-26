package main

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/nikolaarden2000/software-design-project/backend/app"
)

func main() {
	cfg := app.LoadConfig()

	application, err := app.New(context.Background(), cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer application.Close()

	log.Printf("Backend API server started on %s", cfg.HTTPAddr)

	if err := application.HTTPServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
