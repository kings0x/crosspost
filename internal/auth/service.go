package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v3"
	"github.com/go-jose/go-jose/v3/jwt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/argon2"

	"github.com/kings0x/crossPost/internal/config"
	"github.com/kings0x/crossPost/internal/email"
	"github.com/markbates/goth"
)

type AuthService struct {
	repo *AuthRepository
}

func NewAuthService(repo *AuthRepository) *AuthService {
	return &AuthService{repo}
}

func (s *AuthService) ServiceOauthCallback(ctx context.Context, cfg *config.Config, gothUser *goth.User, user_agent, ip_address string) (*LoginUserResponse, error) {

	user, err := s.repo.queryUpsertUser(ctx, gothUser, "user")
	if err != nil {
		slog.Info("queryUpsertUser", "err", err)
		return nil, fmt.Errorf("ServiceOauthCallback: %w", err)
	}

	// persist oauth account details
	if err := s.repo.repoInsertOauthAccount(ctx, user.ID.String(), *gothUser); err != nil {
		slog.Info("InsertOauthAcc", "err", err)
		return nil, fmt.Errorf("ServiceOauthCallback: %w", err)
	}

	accessToken, err := createJwtToken(cfg.JWT_SECRET, user.ID.String())
	if err != nil {
		slog.Info("createJWT", "err", err)
		return nil, fmt.Errorf("ServiceOauthCallback: %w", err)
	}

	refreshToken, err := createRefreshToken()
	if err != nil {
		slog.Info("createRefreshToken", "err", err)
		return nil, fmt.Errorf("ServiceOauthCallback: %w", err)
	}

	token_hash := hashToken(refreshToken)

	expire_at := time.Now().Add(720 * time.Hour)

	if err := s.repo.repoInsertToken(ctx, user.ID.String(), token_hash, user_agent, ip_address, expire_at); err != nil {
		slog.Info("repoInsertToken", "err", err)
		return nil, fmt.Errorf("ServiceOauthCallback: %w", err)
	}

	response := LoginUserResponse{
		UserId:       user.ID.String(),
		Email:        user.Email,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		AvatarURL:    user.AvatarURL.String,
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

// SignUp: create user record if missing (do not require verification), send magic link
func (s *AuthService) ServiceSignUp(ctx context.Context, cfg *config.Config, emailAddr, password string) error {
	// check for existing user
	user, err := s.repo.GetUserByEmail(ctx, emailAddr)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("ServiceSignUp: %w", err)
		}
	}

	var userID uuid.UUID
	if err == nil {
		// user exists
		if user.EmailVerified {
			return fmt.Errorf("ServiceSignUp: %w", fmt.Errorf("email already verified"))
		}
		userID = user.ID
		if password != "" {
			hash, err := HashPassword(password)
			if err != nil {
				return fmt.Errorf("ServiceSignUp: %w", err)
			}
			if err := s.repo.UpdatePassword(ctx, userID.String(), hash); err != nil {
				return fmt.Errorf("ServiceSignUp: %w", err)
			}
		}
	} else {
		// create user
		var passHash string
		if password != "" {
			h, err := HashPassword(password)
			if err != nil {
				return fmt.Errorf("ServiceSignUp: %w", err)
			}
			passHash = h
		}
		id, err := s.repo.CreateUser(ctx, emailAddr, passHash, "user")
		if err != nil {
			return fmt.Errorf("ServiceSignUp: %w", err)
		}
		userID = id
	}

	// generate verification token and store in redis
	token, err := createRefreshToken()
	if err != nil {
		return fmt.Errorf("ServiceSignUp: %w", err)
	}
	tokenHash := hashToken(token)
	payload := TokenPayload{UserID: userID.String(), Email: emailAddr}
	b, _ := json.Marshal(payload)

	if err := s.repo.repoSetUserToken(ctx, "token:"+tokenHash, b, 5*time.Minute); err != nil {

		return fmt.Errorf("ServiceSignUp: %w", err)
	}
	slog.Info("repoSetUserToken", "token", tokenHash)

	// send verification email
	link := cfg.BACKEND_URL + "/v1/auth/verify?token=" + token
	if err := email.SendVerificationEmail(cfg, emailAddr, link); err != nil {
		return fmt.Errorf("ServiceSignUp: %w", err)
	}

	return nil
}

// ForgotPassword: generate reset token and email frontend reset link
func (s *AuthService) ServiceForgotPassword(ctx context.Context, cfg *config.Config, emailAddr string) error {
	// find user; if not found, return success to avoid account enumeration
	user, err := s.repo.GetUserByEmail(ctx, emailAddr)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("ServiceForgotPassword: %w", err)
	}

	token, err := createRefreshToken()
	if err != nil {
		return fmt.Errorf("ServiceForgotPassword: %w", err)
	}
	tokenHash := hashToken(token)
	payload := TokenPayload{UserID: user.ID.String(), Email: emailAddr}
	b, _ := json.Marshal(payload)

	if err := s.repo.repoSetUserToken(ctx, "token:"+tokenHash, b, 1*time.Hour); err != nil {
		return fmt.Errorf("ServiceForgotPassword: %w", err)
	}

	link := cfg.FRONTEND_URL + "/reset-password?token=" + token
	if err := email.SendPasswordResetEmail(cfg, emailAddr, link); err != nil {
		return fmt.Errorf("ServiceForgotPassword: %w", err)
	}
	return nil
}

// ResetPassword: consume reset token, update password, and revoke existing sessions.
func (s *AuthService) ServiceResetPassword(ctx context.Context, cfg *config.Config, token, newPassword, userAgent, ipAddr string) error {
	if token == "" {
		return fmt.Errorf("ServiceResetPassword: %w", fmt.Errorf("missing token"))
	}
	tokenHash := hashToken(token)
	res, err := s.repo.repoConsumeUserToken(ctx, "token:"+tokenHash)
	if err != nil {
		return fmt.Errorf("ServiceResetPassword: %w", err)
	}
	var payload TokenPayload
	if err := json.Unmarshal([]byte(res), &payload); err != nil {
		return fmt.Errorf("ServiceResetPassword: %w", err)
	}
	uid := payload.UserID
	if uid == "" {
		return fmt.Errorf("ServiceResetPassword: %w", fmt.Errorf("invalid token payload"))
	}

	hashed, err := HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("ServiceResetPassword: %w", err)
	}
	if err := s.repo.UpdatePassword(ctx, uid, hashed); err != nil {
		return fmt.Errorf("ServiceResetPassword: %w", err)
	}

	// revoke existing sessions for security (do not issue new tokens)
	if err := s.repo.repoDeleteSessionsByUser(ctx, uid); err != nil {
		return fmt.Errorf("ServiceResetPassword: %w", err)
	}

	return nil
}

// VerifyEmail: consume verification token, mark user verified, and issue tokens
func (s *AuthService) ServiceVerifyEmail(ctx context.Context, cfg *config.Config, token string, userAgent, ipAddr string) (*LoginUserResponse, error) {
	if token == "" {
		return nil, fmt.Errorf("ServiceVerifyEmail: %w", fmt.Errorf("missing token"))
	}

	tokenHash := hashToken(token)
	slog.Info("ServiceVerifyEmail", "token", tokenHash)
	res, err := s.repo.repoConsumeUserToken(ctx, "token:"+tokenHash)
	if err != nil {
		return nil, fmt.Errorf("ServiceVerifyEmail: %w", err)
	}
	var payload TokenPayload
	if err := json.Unmarshal([]byte(res), &payload); err != nil {
		return nil, fmt.Errorf("ServiceVerifyEmail: %w", err)
	}
	uid := payload.UserID
	if uid == "" {
		return nil, fmt.Errorf("ServiceVerifyEmail: %w", fmt.Errorf("invalid token payload"))
	}

	if err := s.repo.MarkUserVerified(ctx, uid); err != nil {
		return nil, fmt.Errorf("ServiceVerifyEmail: %w", err)
	}

	// issue tokens
	accessToken, err := createJwtToken(cfg.JWT_SECRET, uid)
	if err != nil {
		return nil, fmt.Errorf("ServiceVerifyEmail: %w", err)
	}

	refreshToken, err := createRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("ServiceVerifyEmail: %w", err)
	}

	tokenHash2 := hashToken(refreshToken)
	expireAt := time.Now().Add(30 * 24 * time.Hour)
	if err := s.repo.repoInsertToken(ctx, uid, tokenHash2, userAgent, ipAddr, expireAt); err != nil {
		return nil, fmt.Errorf("ServiceVerifyEmail: %w", err)
	}

	resp := LoginUserResponse{
		UserId:       uid,
		Email:        payload.Email,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		AvatarURL:    "",
		CreatedAt:    time.Now().String(),
	}

	return &resp, nil
}

// Login: authenticate user by password and issue tokens if verified
func (s *AuthService) ServiceLogin(ctx context.Context, cfg *config.Config, email, password, userAgent, ipAddr string) (*LoginUserResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("ServiceLogin: %w", err)
	}
	if !user.EmailVerified {
		return nil, fmt.Errorf("ServiceLogin: %w", fmt.Errorf("email not verified"))
	}
	if !user.PasswordHash.Valid || password == "" {
		return nil, fmt.Errorf("ServiceLogin: %w", fmt.Errorf("password required"))
	}
	if err := ComparePassword(user.PasswordHash.String, password); err != nil {
		return nil, fmt.Errorf("ServiceLogin: %w", err)
	}

	accessToken, err := createJwtToken(cfg.JWT_SECRET, user.ID.String())
	if err != nil {
		return nil, fmt.Errorf("ServiceLogin: %w", err)
	}

	refreshToken, err := createRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("ServiceLogin: %w", err)
	}

	tokenHash := hashToken(refreshToken)
	expireAt := time.Now().Add(30 * 24 * time.Hour)
	if err := s.repo.repoInsertToken(ctx, user.ID.String(), tokenHash, userAgent, ipAddr, expireAt); err != nil {
		return nil, fmt.Errorf("ServiceLogin: %w", err)
	}

	resp := LoginUserResponse{
		UserId:       user.ID.String(),
		Email:        user.Email,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		AvatarURL:    user.AvatarURL.String,
		CreatedAt:    user.CreatedAt.String(),
	}
	return &resp, nil
}

// Refresh: rotate refresh token and return new tokens
func (s *AuthService) ServiceRefresh(ctx context.Context, cfg *config.Config, rawRefreshToken, userAgent, ipAddr string) (*LoginUserResponse, error) {
	if rawRefreshToken == "" {
		return nil, fmt.Errorf("ServiceRefresh: %w", fmt.Errorf("missing token"))
	}
	oldHash := hashToken(rawRefreshToken)
	sessionStr, err := s.repo.repoGetSession(ctx, oldHash)
	if err != nil {
		return nil, fmt.Errorf("ServiceRefresh: %w", err)
	}
	var sp SessionPayload
	if err := json.Unmarshal([]byte(sessionStr), &sp); err != nil {
		return nil, fmt.Errorf("ServiceRefresh: %w", err)
	}
	uid := sp.UserID

	newRefresh, err := createRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("ServiceRefresh: %w", err)
	}
	newHash := hashToken(newRefresh)
	newPayload := SessionPayload{UserID: uid, UserAgent: userAgent, IPAddress: ipAddr}
	b, _ := json.Marshal(newPayload)
	ttl := 30 * 24 * time.Hour
	if err := s.repo.repoRotateSession(ctx, oldHash, newHash, b, ttl); err != nil {
		return nil, fmt.Errorf("ServiceRefresh: %w", err)
	}

	accessToken, err := createJwtToken(cfg.JWT_SECRET, uid)
	if err != nil {
		return nil, fmt.Errorf("ServiceRefresh: %w", err)
	}

	resp := LoginUserResponse{
		UserId:       uid,
		Email:        "",
		AccessToken:  accessToken,
		RefreshToken: newRefresh,
		AvatarURL:    "",
		CreatedAt:    time.Now().String(),
	}
	return &resp, nil
}

// Revoke: remove a refresh session
func (s *AuthService) ServiceRevoke(ctx context.Context, rawRefreshToken string) error {
	if rawRefreshToken == "" {
		return fmt.Errorf("ServiceRevoke: %w", fmt.Errorf("missing token"))
	}
	h := hashToken(rawRefreshToken)
	if err := s.repo.repoDeleteSession(ctx, h); err != nil {
		return fmt.Errorf("ServiceRevoke: %w", err)
	}
	return nil
}

// Logout: remove all sessions for a user (used when logging out via access token)
func (s *AuthService) ServiceLogout(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("ServiceLogout: %w", fmt.Errorf("missing user id"))
	}
	if err := s.repo.repoDeleteSessionsByUser(ctx, userID); err != nil {
		return fmt.Errorf("ServiceLogout: %w", err)
	}
	return nil
}

// Argon2id password helpers (inlined here to keep auth package as handler->service->repo)
const (
	argonTime    uint32 = 2
	argonMemory  uint32 = 64 * 1024
	argonThreads uint8  = 4
	argonKeyLen  uint32 = 32
	argonSaltLen int    = 16
)

func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("HashPassword: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", argonMemory, argonTime, argonThreads, b64Salt, b64Hash)
	return encoded, nil
}

func ComparePassword(encoded, password string) error {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return fmt.Errorf("ComparePassword: invalid hash format")
	}

	params := parts[3]
	var m uint64
	var t uint64
	var p uint64
	for _, kv := range strings.Split(params, ",") {
		pair := strings.SplitN(kv, "=", 2)
		if len(pair) != 2 {
			continue
		}
		var err error
		switch pair[0] {
		case "m":
			m, err = strconv.ParseUint(pair[1], 10, 32)
			if err != nil {
				return fmt.Errorf("ComparePassword: invalid m parameter")
			}
		case "t":
			t, err = strconv.ParseUint(pair[1], 10, 32)
			if err != nil {
				return fmt.Errorf("ComparePassword: invalid t parameter")
			}
		case "p":
			p, err = strconv.ParseUint(pair[1], 10, 8)
			if err != nil {
				return fmt.Errorf("ComparePassword: invalid p parameter")
			}
		}
	}

	if m == 0 || t == 0 || p == 0 {
		return fmt.Errorf("ComparePassword: invalid argon2 parameters")
	}

	saltB64 := parts[4]
	hashB64 := parts[5]

	salt, err := base64.RawStdEncoding.DecodeString(saltB64)
	if err != nil {
		return fmt.Errorf("ComparePassword: %w", err)
	}
	expectedHash, err := base64.RawStdEncoding.DecodeString(hashB64)
	if err != nil {
		return fmt.Errorf("ComparePassword: %w", err)
	}

	derived := argon2.IDKey([]byte(password), salt, uint32(t), uint32(m), uint8(p), uint32(len(expectedHash)))

	if subtle.ConstantTimeCompare(derived, expectedHash) != 1 {
		return fmt.Errorf("ComparePassword: password mismatch")
	}

	return nil
}
