package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewRedis(ctx context.Context, url string) (*redis.Client, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("NewRedis: %w", err)
	}

	opts.PoolSize = 10
	opts.MinIdleConns = 2
	opts.ConnMaxLifetime = 30 * time.Minute
	opts.ConnMaxIdleTime = 5 * time.Minute

	client := redis.NewClient(opts)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		return nil, fmt.Errorf("NewRedis: %w", err)
	}

	return client, nil
}
