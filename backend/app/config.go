package app

import "os"

type Config struct {
	DatabaseURL string
	HTTPAddr    string
	CookieName  string
}

func LoadConfig() Config {
	cfg := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		HTTPAddr:    os.Getenv("HTTP_ADDR"),
		CookieName:  os.Getenv("COOKIE_NAME"),
	}

	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = ":8080"
	}

	if cfg.CookieName == "" {
		cfg.CookieName = "session_id"
	}

	return cfg
}
