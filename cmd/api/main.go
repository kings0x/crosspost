package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kings0x/crossPost/internal/cache"
	"github.com/kings0x/crossPost/internal/config"
	"github.com/kings0x/crossPost/internal/db"
	"github.com/kings0x/crossPost/internal/server"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("run: %w", err)
	}

	setupLogger(cfg.APP_ENV)

	pgDb, err := db.NewDatabase(ctx, cfg.DATABASE_URL)
	if err != nil {
		return fmt.Errorf("run: %w", err)
	}
	defer pgDb.Close()

	redis, err := cache.NewRedis(ctx, cfg.REDIS_URL)
	if err != nil {
		return fmt.Errorf("run: %w", err)
	}
	defer redis.Close()

	srv := server.New(pgDb, redis, &cfg)

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("srv.Start: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("srv.Shutdown", "err", err)
	}

	slog.Info("server stopped")
	return nil
}

func setupLogger(mode string) {
	var handler slog.Handler

	if mode == "production" || mode == "staging" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	}

	slog.SetDefault(slog.New(handler))
}
