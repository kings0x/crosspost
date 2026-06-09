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
	GOOGLE_CLIENT_ID     string
	GOOGLE_CLIENT_SECRET string
	SESSION_SECRET       string
	SMTP_HOST            string
	SMTP_PORT            string
	SMTP_USER            string
	SMTP_PASS            string
	FROM_EMAIL           string
}

func Load() (Config, error) {

	cfg := Config{
		PORT:                 getenv("PORT", "8000"),
		DATABASE_URL:         getenv("DATABASE_URL", ""),
		REDIS_URL:            getenv("REDIS_URL", ""),
		APP_ENV:              getenv("APP_ENV", "developement"),
		BACKEND_URL:          getenv("BACKEND_URL", ""),
		GOOGLE_CLIENT_ID:     getenv("GOOGLE_CLIENT_ID", ""),
		GOOGLE_CLIENT_SECRET: getenv("GOOGLE_CLIENT_SECRET", ""),
		SESSION_SECRET:       getenv("SESSION_SECRET", ""),
		SMTP_HOST:            getenv("SMTP_HOST", ""),
		SMTP_PORT:            getenv("SMTP_PORT", ""),
		SMTP_USER:            getenv("SMTP_USER", ""),
		SMTP_PASS:            getenv("SMTP_PASS", ""),
		FROM_EMAIL:           getenv("FROM_EMAIL", ""),
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

	return nil
}

func getenv(value, fallback string) string {
	if result := os.Getenv(value); result != "" {
		return result
	}
	return fallback
}
