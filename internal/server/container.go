package server

import (
	"github.com/kings0x/crossPost/internal/auth"
	"github.com/kings0x/crossPost/internal/cache"
	"github.com/kings0x/crossPost/internal/db"
)

type Container struct {
	auth *auth.AuthHandler
}

func newContainer(db *db.Database, redis cache.Cache) *Container {
	//repositories
	authRepo := auth.NewAuthRepository(db, redis)

	//services
	authService := auth.NewAuthService(authRepo)

	//handlers
	auth := auth.NewAuthHandler(authService)

	return &Container{
		auth,
	}
}
