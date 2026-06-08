package server

import (
	"github.com/kings0x/crossPost/internal/auth"
	"github.com/kings0x/crossPost/internal/config"
	"github.com/kings0x/crossPost/internal/db"
	rds "github.com/redis/go-redis/v9"
)

type Container struct {
	auth *auth.AuthHandler
}

func newContainer(db *db.Database, redisClient *rds.Client, cfg *config.Config) *Container {
	// repositories
	authRepo := auth.NewAuthRepository(db, redisClient)

	// services
	authService := auth.NewAuthService(authRepo)

	// handlers
	authHandler := auth.NewAuthHandler(authService, cfg)

	return &Container{
		authHandler,
	}
}
