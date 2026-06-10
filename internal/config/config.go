package config

import (
	"fmt"
	"os"
)

type Config struct {
	PORT                 string
	DATABASE_URL         string
	REDIS_URL            string
	APP_ENV              string
	BACKEND_URL          string
	COOKIE_DOMAIN        string
	FRONTEND_URL         string
	GOOGLE_CLIENT_ID     string
	GOOGLE_CLIENT_SECRET string
	JWT_SECRET           string
	SESSION_SECRET       string
	SESSION_ENCRYPT_KEY  string
	EMAIL_FROM           string
	RESEND_API_KEY       string
}

func Load() (Config, error) {

	cfg := Config{
		PORT:                 getenv("PORT", "8000"),
		DATABASE_URL:         getenv("DATABASE_URL", ""),
		REDIS_URL:            getenv("REDIS_URL", ""),
		APP_ENV:              getenv("APP_ENV", "development"),
		BACKEND_URL:          getenv("BACKEND_URL", ""),
		COOKIE_DOMAIN:        getenv("COOKIE_DOMAIN", ""),
		FRONTEND_URL:         getenv("FRONTEND_URL", ""),
		GOOGLE_CLIENT_ID:     getenv("GOOGLE_CLIENT_ID", ""),
		GOOGLE_CLIENT_SECRET: getenv("GOOGLE_CLIENT_SECRET", ""),
		JWT_SECRET:           getenv("JWT_SECRET", ""),
		SESSION_SECRET:       getenv("SESSION_SECRET", ""),
		SESSION_ENCRYPT_KEY:  getenv("SESSION_ENCRYPT_KEY", ""),
		EMAIL_FROM:           getenv("EMAIL_FROM", ""),
		RESEND_API_KEY:       getenv("RESEND_API_KEY", ""),
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil

}

func (cfg *Config) Validate() error {
	if cfg.DATABASE_URL == "" {
		return fmt.Errorf("DATABASE_URL NOT FOUND, it is required")
	}

	if cfg.REDIS_URL == "" {
		return fmt.Errorf("REDIS_URL NOT FOUND, it is required")
	}

	if cfg.BACKEND_URL == "" {
		return fmt.Errorf("BACKEND_URL NOT FOUND")
	}

	if cfg.GOOGLE_CLIENT_ID == "" {
		return fmt.Errorf("GOOGLE_CLIENT_ID NOT FOUND")
	}

	if cfg.GOOGLE_CLIENT_SECRET == "" {
		return fmt.Errorf("GOOGLE_CLIENT_SECRET NOT FOUND")
	}

	if cfg.EMAIL_FROM == "" {
		return fmt.Errorf("EMAIL_FROM NOT FOUND")
	}

	if cfg.RESEND_API_KEY == "" {
		return fmt.Errorf("RESEND_API_KEY NOT FOUND")
	}

	if cfg.SESSION_SECRET == "" {
		return fmt.Errorf("SESSION_SECRET NOT FOUND")
	}

	if cfg.SESSION_ENCRYPT_KEY == "" {
		return fmt.Errorf("SESSION_ENCRYPT_KEY NOT FOUND")
	}

	if cfg.JWT_SECRET == "" {
		return fmt.Errorf("JWT_SECRET NOT FOUND")
	}

	return nil
}

func getenv(value, fallback string) string {
	if result := os.Getenv(value); result != "" {
		return result
	}
	return fallback
}
