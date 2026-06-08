package main

import (
	"log"
	"os"
	"strconv"

	"github.com/kings0x/crossPost/internal/config"
	"github.com/kings0x/crossPost/internal/db"
)

func main() {
	cfg, err := config.Load()

	if err != nil {
		log.Fatalf("config.Load: %v", err)
	}

	dbURL := cfg.DATABASE_URL

	// cmd/migrate/main.go
	if len(os.Args) < 2 {
		log.Fatal("usage: migrate [up|down]")
	}

	direction := os.Args[1]

	mm, err := db.NewMigrationManager(dbURL, "./migrations")

	if err != nil {
		log.Fatalf("db.NewMigrationManager: %v", err)
	}

	switch direction {
	case "up":
		if err := mm.Up(); err != nil {
			log.Fatalf("mm.Up: %v", err)
		}
	case "down":
		if err := mm.Down(); err != nil {
			log.Fatalf("mm.Down: %v", err)
		}

	case "force":
		if len(os.Args) < 3 {
			log.Fatal("usage: migrate force <version>")
		}
		version, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatalf("invalid version: %v", err)
		}
		if err := mm.Force(version); err != nil {
			log.Fatalf("mm.Force: %v", err)
		}
	default:
		log.Fatalf("unknown direction: %s", direction)
	}
}
