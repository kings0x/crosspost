package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	client *redis.Client
}

func NewRedis(ctx context.Context, url string) (*Redis, error) {
	opts, err := redis.ParseURL(url)

	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url err: %w", err)
	}

	opts.PoolSize = 10
	opts.MinIdleConns = 2
	opts.ConnMaxLifetime = 30 * time.Minute
	opts.ConnMaxIdleTime = 5 * time.Minute

	client := redis.NewClient(opts)

	redis := &Redis{
		client,
	}

	if err := redis.Ping(ctx); err != nil {
		return nil, err
	}

	return redis, nil

}

func (redis *Redis) Get(ctx context.Context, key string) ([]byte, error) {
	return nil, nil
}
func (redis *Redis) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return nil
}
func (redis *Redis) Delete(ctx context.Context, key string) error {
	return nil
}

func (redis *Redis) Ping(ctx context.Context) error {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	if err := redis.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis.client.Ping(): %w", err)
	}

	return nil
}

func (redis *Redis) Close() error {
	if err := redis.client.Close(); err != nil {
		return fmt.Errorf("redis: failed to close redis connection with err %w", err)
	}
	return nil
}
