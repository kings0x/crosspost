package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
	pool *pgxpool.Pool
}

func NewDatabase(ctx context.Context, url string) (*Database, error) {

	config, err := pgxpool.ParseConfig(url)

	if err != nil {
		return nil, fmt.Errorf("failed to parse config with err: %w", err)
	}

	config.MaxConnIdleTime = 30 * time.Minute
	config.MaxConnLifetime = 1 * time.Hour
	config.MaxConns = 20
	config.MinConns = 5
	config.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to db with err: %w", err)
	}

	db := &Database{
		pool,
	}

	if err := db.HealthCheck(ctx); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *Database) HealthCheck(ctx context.Context) error {

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)

	defer cancel()

	if err := db.pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping to db failed with err: %w", err)
	}

	var result int

	if err := db.pool.QueryRow(ctx, "SELECT 1").Scan(&result); err != nil {
		return fmt.Errorf("database query failed with err: %w", err)
	}

	return nil

}

func (db *Database) Close() {
	db.pool.Close()
}
