package config

import (
	"fmt"
	"os"
)

type Config struct {
	PORT         string
	DATABASE_URL string
	REDIS_URL    string
	APP_ENV      string
}

func Load() (Config, error) {

	cfg := Config{
		PORT:         getenv("PORT", "8000"),
		DATABASE_URL: getenv("DATABASE_URL", ""),
		REDIS_URL:    getenv("REDIS_URL", ""),
		APP_ENV:      getenv("APP_ENV", "developement"),
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

	return nil
}

func getenv(value, fallback string) string {
	if result := os.Getenv(value); result != "" {
		return result
	}
	return fallback
}
