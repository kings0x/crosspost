package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-jose/go-jose/v3"
	"github.com/go-jose/go-jose/v3/jwt"
	"github.com/google/uuid"

	"github.com/kings0x/crossPost/internal/config"
	"github.com/markbates/goth"
)

type AuthService struct {
	repo *AuthRepository
}

func NewAuthService(repo *AuthRepository) *AuthService {
	return &AuthService{repo}
}

func (s *AuthService) ServiceOauthBegin(ctx context.Context, cfg *config.Config, platform, intent string) (string, error) {
	state := uuid.New().String()

	oauthState := OAuthState{
		UserID:   state,
		Platform: platform,
		Intent:   intent,
	}

	data, err := json.Marshal(oauthState)
	if err != nil {
		return "", fmt.Errorf("ServiceOauthBegin: %w", err)
	}

	key := "oauth_state:" + state
	if err := s.repo.repoSetOauthState(ctx, key, data, 5*time.Minute); err != nil {
		return "", fmt.Errorf("ServiceOauthBegin: %w", err)
	}

	// begin oauth with our state
	provider, err := goth.GetProvider(platform)
	if err != nil {
		return "", fmt.Errorf("ServiceOauthBegin: %w", err)
	}

	session, err := provider.BeginAuth(state)
	if err != nil {
		return "", fmt.Errorf("ServiceOauthBegin: %w", err)
	}

	url, err := session.GetAuthURL()
	if err != nil {
		return "", fmt.Errorf("ServiceOauthBegin: %w", err)
	}

	return url, nil

}

func (s *AuthService) ServiceOauthCallback(ctx context.Context, cfg *config.Config, gothUser *goth.User, state string, user_agent, ip_address string) (*LoginUserResponse, error) {

	// verify state from Redis (atomic GET+DEL)
	if state == "" {
		return nil, fmt.Errorf("ServiceOauthCallback: %w", fmt.Errorf("missing oauth state"))
	}

	val, err := s.repo.repoVerifyOauthState(ctx, "oauth_state:"+state)
	if err != nil {
		return nil, fmt.Errorf("ServiceOauthCallback: %w", err)
	}

	var oauthState OAuthState
	if err := json.Unmarshal([]byte(val), &oauthState); err != nil {
		return nil, fmt.Errorf("ServiceOauthCallback: %w", err)
	}

	user, err := s.repo.queryUpsertUser(ctx, gothUser)
	if err != nil {
		return nil, fmt.Errorf("ServiceOauthCallback: %w", err)
	}

	// persist oauth account details
	if err := s.repo.repoInsertOauthAccount(ctx, user.ID.String(), *gothUser); err != nil {
		return nil, fmt.Errorf("ServiceOauthCallback: %w", err)
	}

	accessToken, err := createJwtToken(cfg.SESSION_SECRET, user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("ServiceOauthCallback: %w", err)
	}

	refreshToken, err := createRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("ServiceOauthCallback: %w", err)
	}

	token_hash := hashToken(refreshToken)

	expire_at := time.Now().Add(720 * time.Hour)

	if err := s.repo.repoInsertToken(ctx, user.ID.String(), token_hash, user_agent, ip_address, expire_at); err != nil {
		return nil, fmt.Errorf("ServiceOauthCallback: %w", err)
	}

	response := LoginUserResponse{
		UserId:       user.ID.String(),
		Email:        user.Email,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		AvatarURL:    user.AvatarURL,
		CreatedAt:    user.CreatedAt.String(),
	}

	return &response, nil

}

func createJwtToken(secret, subject string) (string, error) {
	// create a token
	sig, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.HS256, Key: []byte(secret)},
		(&jose.SignerOptions{}).WithType("JWT"),
	)

	if err != nil {
		return "", err
	}

	claims := jwt.Claims{
		Subject:  subject,
		Expiry:   jwt.NewNumericDate(time.Now().Add(2 * time.Hour)),
		IssuedAt: jwt.NewNumericDate(time.Now()),
	}

	token, err := jwt.Signed(sig).Claims(claims).CompactSerialize()

	if err != nil {
		return "", err
	}

	return token, nil
}

func createRefreshToken() (string, error) {
	b := make([]byte, 32)

	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(token string) string {

	hash := sha256.Sum256([]byte(token))

	return hex.EncodeToString(hash[:])
}
