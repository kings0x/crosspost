package auth

import (
	"github.com/kings0x/crossPost/internal/cache"
	"github.com/kings0x/crossPost/internal/db"
)

type AuthRepository struct {
	db    *db.Database
	cache cache.Cache
}

func NewAuthRepository(db *db.Database, cache cache.Cache) *AuthRepository {
	return &AuthRepository{db, cache}
}
