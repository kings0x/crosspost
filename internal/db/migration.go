package db

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

type MigrationManager struct {
	migrate *migrate.Migrate
}

func NewMigrationManager(dbURL, migrationsPath string) (*MigrationManager, error) {
	mm, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsPath),
		dbURL,
	)

	if err != nil {
		return nil, fmt.Errorf("NewMigrationManager: %w", err)
	}

	return &MigrationManager{migrate: mm}, nil
}

func (mm *MigrationManager) Up() error {
	if err := mm.migrate.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("MigrationManager.Up: %w", err)
	}
	return nil
}

func (mm *MigrationManager) Down() error {
	if err := mm.migrate.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("MigrationManager.Down: %w", err)
	}
	return nil
}

func RunMigrations(dbURL, migrationsPath string) error {
	mm, err := NewMigrationManager(dbURL, migrationsPath)
	if err != nil {
		return fmt.Errorf("RunMigrations: %w", err)
	}

	if err := mm.Up(); err != nil {
		return fmt.Errorf("RunMigrations: %w", err)
	}

	return nil
}
