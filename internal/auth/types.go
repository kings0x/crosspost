package auth

import (
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

type UpsertUserRow struct {
	ID        uuid.UUID
	Email     string
	AvatarURL string
	CreatedAt time.Time
}
