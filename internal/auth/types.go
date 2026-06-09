package auth

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type LoginUserResponse struct {
	UserId       string
	Email        string
	AccessToken  string
	RefreshToken string
	AvatarURL    string
	CreatedAt    string
}

type OAuthState struct {
	UserID   string `json:"user_id,omitempty"` // empty if new signup
	Platform string `json:"platform"`          // "google", "github"
	Intent   string `json:"intent"`            // "login", "signup", "connect"
}

// TokenPayload represents one-time tokens stored in Redis (magic links, etc.)
type TokenPayload struct {
	UserID string `json:"user_id"`
	Kind   string `json:"kind,omitempty"`
	Email  string `json:"email,omitempty"`
}

// SessionPayload represents a refresh session stored in Redis
type SessionPayload struct {
	UserID    string `json:"user_id"`
	UserAgent string `json:"user_agent,omitempty"`
	IPAddress string `json:"ip_address,omitempty"`
}

type UserRow struct {
	ID            uuid.UUID
	Email         string
	PasswordHash  sql.NullString
	AvatarURL     sql.NullString
	EmailVerified bool
	CreatedAt     time.Time
}

type UpsertUserRow struct {
	ID        uuid.UUID
	Email     string
	AvatarURL string
	CreatedAt time.Time
}

type signupRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password,omitempty"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}
