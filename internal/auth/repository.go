package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kings0x/crossPost/internal/db"
	"github.com/markbates/goth"
	rds "github.com/redis/go-redis/v9"
)

type AuthRepository struct {
	db    *db.Database
	redis *rds.Client
}

func NewAuthRepository(db *db.Database, redisClient *rds.Client) *AuthRepository {
	return &AuthRepository{db, redisClient}
}

func (r *AuthRepository) queryUpsertUser(ctx context.Context, gothUser *goth.User) (UpsertUserRow, error) {
	query := `
		INSERT INTO users (email, avatar_url, email_verified, email_verified_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (email) DO UPDATE SET
			avatar_url = EXCLUDED.avatar_url,
			email_verified = EXCLUDED.email_verified,
			email_verified_at = EXCLUDED.email_verified_at
		RETURNING id, email, avatar_url, created_at
	`

	var user UpsertUserRow
	err := r.db.Pool.QueryRow(ctx, query,
		gothUser.Email,
		gothUser.AvatarURL,
		true,
		time.Now(),
	).Scan(&user.ID, &user.Email, &user.AvatarURL, &user.CreatedAt)

	if err != nil {
		return UpsertUserRow{}, fmt.Errorf("queryUpsertUser: %w", err)
	}

	return user, nil
}

func (r *AuthRepository) repoInsertToken(ctx context.Context, user_id, token_hash, user_agent string, ip_address string, expires_at time.Time) error {
	// Store session data in Redis. Key will be "session:<token_hash>".
	key := "session:" + token_hash
	payload := map[string]string{
		"user_id":    user_id,
		"user_agent": user_agent,
		"ip_address": ip_address,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("repoInsertToken: %w", err)
	}

	ttl := time.Until(expires_at)
	if err := r.redis.Set(ctx, key, b, ttl).Err(); err != nil {
		return fmt.Errorf("repoInsertToken: %w", err)
	}

	return nil
}

func (r *AuthRepository) repoInsertOauthAccount(ctx context.Context, user_id string, gothUser goth.User) error {
	query := `
		INSERT INTO user_oauth_accounts (user_id, provider, provider_id, access_token, refresh_token, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (provider, provider_id) DO UPDATE SET access_token = EXCLUDED.access_token, refresh_token = EXCLUDED.refresh_token, expires_at = EXCLUDED.expires_at
	`

	_, err := r.db.Pool.Exec(ctx, query,
		user_id,
		gothUser.Provider,
		gothUser.UserID,
		gothUser.AccessToken,
		gothUser.RefreshToken,
		gothUser.ExpiresAt,
	)

	if err != nil {
		return fmt.Errorf("repoInsertOauthAccount: %w", err)
	}

	return nil
}

func (r *AuthRepository) repoSetOauthState(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := r.redis.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("repoSetOauthState: %w", err)
	}
	return nil
}

func (r *AuthRepository) repoVerifyOauthState(ctx context.Context, key string) (string, error) {
	// Atomically GET + DEL using a Lua script.
	script := rds.NewScript(`
		local val = redis.call("GET", KEYS[1])
		if val then
			redis.call("DEL", KEYS[1])
		end
		return val
	`)

	res, err := script.Run(ctx, r.redis, []string{key}).Result()
	if err != nil && err != rds.Nil {
		return "", fmt.Errorf("repoVerifyOauthState: %w", err)
	}
	if res == nil {
		return "", fmt.Errorf("repoVerifyOauthState: %w", fmt.Errorf("not found"))
	}
	s, ok := res.(string)
	if !ok {
		return "", fmt.Errorf("repoVerifyOauthState: %w", fmt.Errorf("invalid value type"))
	}
	return s, nil
}
