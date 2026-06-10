package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

func (r *AuthRepository) queryUpsertUser(ctx context.Context, gothUser *goth.User, role string) (UpsertUserRow, error) {
	query := `
		INSERT INTO users (email, avatar_url, email_verified, email_verified_at, role)
		VALUES ($1, $2, $3, $4, $5)
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
		role,
	).Scan(&user.ID, &user.Email, &user.AvatarURL, &user.CreatedAt)

	if err != nil {
		return UpsertUserRow{}, fmt.Errorf("queryUpsertUser: %w", err)
	}

	return user, nil
}

func (r *AuthRepository) repoInsertToken(ctx context.Context, user_id, token_hash, user_agent string, ip_address string, expires_at time.Time) error {
	// Store session data in Redis. Key will be "session:<token_hash>".
	key := "session:" + token_hash
	payload := SessionPayload{UserID: user_id, UserAgent: user_agent, IPAddress: ip_address}
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("repoInsertToken: %w", err)
	}

	ttl := time.Until(expires_at)
	if err := r.redis.Set(ctx, key, b, ttl).Err(); err != nil {
		return fmt.Errorf("repoInsertToken: %w", err)
	}
	// keep quick lookup of sessions per user
	if err := r.redis.SAdd(ctx, "user_sessions:"+user_id, key).Err(); err != nil {
		return fmt.Errorf("repoInsertToken: %w", err)
	}
	// ensure the set expires no later than the session TTL
	_ = r.redis.Expire(ctx, "user_sessions:"+user_id, ttl)

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

func (r *AuthRepository) GetUserByEmail(ctx context.Context, email string) (*UserRow, error) {
	query := `
		SELECT id, email, password_hash, avatar_url, email_verified, created_at
		FROM users WHERE email = $1
	`

	var u UserRow
	var password sql.NullString
	var avatar sql.NullString
	err := r.db.Pool.QueryRow(ctx, query, email).Scan(&u.ID, &u.Email, &password, &avatar, &u.EmailVerified, &u.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("GetUserByEmail: %w", err)
		}
		return nil, fmt.Errorf("GetUserByEmail: %w", err)
	}
	u.PasswordHash = password
	u.AvatarURL = avatar
	return &u, nil
}

func (r *AuthRepository) CreateUser(ctx context.Context, email string, passwordHash, role string) (uuid.UUID, error) {
	query := `
		INSERT INTO users (email, password_hash, role) VALUES ($1, $2, $3) RETURNING id
	`
	var id uuid.UUID
	err := r.db.Pool.QueryRow(ctx, query, email, passwordHash, role).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("CreateUser: %w", err)
	}
	return id, nil
}

func (r *AuthRepository) MarkUserVerified(ctx context.Context, userID string) error {
	query := `UPDATE users SET email_verified = true, email_verified_at = now(), updated_at = now() WHERE id = $1`
	if _, err := r.db.Pool.Exec(ctx, query, userID); err != nil {
		return fmt.Errorf("MarkUserVerified: %w", err)
	}
	return nil
}

func (r *AuthRepository) UpdatePassword(ctx context.Context, userID string, passwordHash string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = now() WHERE id = $2`
	if _, err := r.db.Pool.Exec(ctx, query, passwordHash, userID); err != nil {
		return fmt.Errorf("UpdatePassword: %w", err)
	}
	return nil
}

// User token helpers (used for magic links etc.)
func (r *AuthRepository) repoSetUserToken(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := r.redis.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("repoSetUserToken: %w", err)
	}
	return nil
}

func (r *AuthRepository) repoConsumeUserToken(ctx context.Context, key string) (string, error) {
	script := rds.NewScript(`
		local val = redis.call("GET", KEYS[1])
		if val then
			redis.call("DEL", KEYS[1])
		end
		return val
	`)

	res, err := script.Run(ctx, r.redis, []string{key}).Result()
	if err != nil && err != rds.Nil {
		return "", fmt.Errorf("repoConsumeUserToken: %w", err)
	}
	if res == nil {
		return "", fmt.Errorf("repoConsumeUserToken: %w", fmt.Errorf("not found"))
	}
	s, ok := res.(string)
	if !ok {
		return "", fmt.Errorf("repoConsumeUserToken: %w", fmt.Errorf("invalid value type"))
	}
	return s, nil
}

// Session helpers
func (r *AuthRepository) repoGetSession(ctx context.Context, tokenHash string) (string, error) {
	key := "session:" + tokenHash
	val, err := r.redis.Get(ctx, key).Result()
	if err != nil {
		if err == rds.Nil {
			return "", fmt.Errorf("repoGetSession: %w", fmt.Errorf("not found"))
		}
		return "", fmt.Errorf("repoGetSession: %w", err)
	}
	return val, nil
}

func (r *AuthRepository) repoRotateSession(ctx context.Context, oldTokenHash, newTokenHash string, payload []byte, ttl time.Duration) error {
	// Atomic rotate: GET old, DEL old, SET new, update user_sessions set membership
	script := rds.NewScript(`
		local val = redis.call("GET", KEYS[1])
		if not val then return nil end
		local obj = cjson.decode(val)
		local uid = obj.user_id
		redis.call("DEL", KEYS[1])
		redis.call("SET", KEYS[2], ARGV[1], "PX", ARGV[2])
		redis.call("SREM", "user_sessions:"..uid, KEYS[1])
		redis.call("SADD", "user_sessions:"..uid, KEYS[2])
		redis.call("PEXPIRE", "user_sessions:"..uid, ARGV[2])
		return 1
	`)

	ttlMs := int64(ttl / time.Millisecond)
	res, err := script.Run(ctx, r.redis, []string{"session:" + oldTokenHash, "session:" + newTokenHash}, string(payload), fmt.Sprintf("%d", ttlMs)).Result()
	if err != nil && err != rds.Nil {
		return fmt.Errorf("repoRotateSession: %w", err)
	}
	if res == nil {
		return fmt.Errorf("repoRotateSession: %w", fmt.Errorf("not found"))
	}
	return nil
}

func (r *AuthRepository) repoDeleteSession(ctx context.Context, tokenHash string) error {
	key := "session:" + tokenHash
	// attempt to remove session and cleanup user_sessions membership
	val, err := r.redis.Get(ctx, key).Result()
	if err != nil {
		if err == rds.Nil {
			return nil
		}
		return fmt.Errorf("repoDeleteSession: %w", err)
	}
	var sp SessionPayload
	if err := json.Unmarshal([]byte(val), &sp); err == nil {
		_ = r.redis.SRem(ctx, "user_sessions:"+sp.UserID, key)
	}
	if err := r.redis.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("repoDeleteSession: %w", err)
	}
	return nil
}

// Delete all sessions for a user
func (r *AuthRepository) repoDeleteSessionsByUser(ctx context.Context, userID string) error {
	setKey := "user_sessions:" + userID
	members, err := r.redis.SMembers(ctx, setKey).Result()
	if err != nil && err != rds.Nil {
		return fmt.Errorf("repoDeleteSessionsByUser: %w", err)
	}
	if len(members) > 0 {
		if err := r.redis.Del(ctx, members...).Err(); err != nil {
			return fmt.Errorf("repoDeleteSessionsByUser: %w", err)
		}
	}
	if err := r.redis.Del(ctx, setKey).Err(); err != nil && err != rds.Nil {
		return fmt.Errorf("repoDeleteSessionsByUser: %w", err)
	}
	return nil
}
