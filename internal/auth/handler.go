package auth

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/kings0x/crossPost/internal/config"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
)

type AuthHandler struct {
	service *AuthService
	cfg     *config.Config
}

func NewAuthHandler(service *AuthService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{service, cfg}
}

func (h *AuthHandler) OauthBegin(c *gin.Context) {
	setProvider(c)

	session, err := gothic.Store.Get(c.Request, gothic.SessionName)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	intent := c.Query("intent")

	// TODO
	//would implement this later if we want to support multiple oauth flows (e.g. link account) and need to distinguish between them in the callback
	session.Values["intent"] = intent

	if err := gothic.Store.Save(c.Request, c.Writer, session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	gothic.BeginAuthHandler(c.Writer, c.Request)
}

func (h *AuthHandler) OauthCallback(c *gin.Context) {
	setProvider(c)

	gothUser, err := gothic.CompleteUserAuth(c.Writer, c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	session, err := gothic.Store.Get(c.Request, gothic.SessionName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// intent, _ := session.Values["intent"].(string)

	delete(session.Values, "intent")
	if err := session.Save(c.Request, c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user_agent := c.Request.UserAgent()
	ip_addr := c.ClientIP()

	res, err := h.service.ServiceOauthCallback(c.Request.Context(), h.cfg, &gothUser, user_agent, ip_addr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "something went wrong"})
		return
	}

	SetCookie(c, "refresh_token", res.RefreshToken, 60*60*24*30, h.cfg)

	c.JSON(http.StatusOK, gin.H{
		"userId":      res.UserId,
		"email":       res.Email,
		"avatarUrl":   res.AvatarURL,
		"accessToken": res.AccessToken,
		"createdAt":   res.CreatedAt,
	})

}

// SignUp and Login handlers implemented below

func (h *AuthHandler) SignUp(c *gin.Context) {
	var req signupRequest
	if err := c.BindJSON(&req); err != nil {
		slog.Error("invalid request", "err", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := h.service.ServiceSignUp(c.Request.Context(), h.cfg, req.Email, req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "verification_sent"})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	res, err := h.service.ServiceLogin(c.Request.Context(), h.cfg, req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	SetCookie(c, "refresh_token", res.RefreshToken, 60*60*24*30, h.cfg)
	c.JSON(http.StatusOK, gin.H{"access_token": res.AccessToken, "user_id": res.UserId})
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := h.service.ServiceForgotPassword(c.Request.Context(), h.cfg, req.Email); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "reset_sent"})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := h.service.ServiceResetPassword(c.Request.Context(), h.cfg, req.Token, req.Password, c.Request.UserAgent(), c.ClientIP()); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Clear any existing refresh cookie since sessions have been revoked
	SetCookie(c, "refresh_token", "", -1, h.cfg)

	c.JSON(http.StatusOK, gin.H{"status": "password_reset"})
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	res, err := h.service.ServiceVerifyEmail(c.Request.Context(), h.cfg, token, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	SetCookie(c, "refresh_token", res.RefreshToken, 60*60*24*30, h.cfg)
	c.JSON(http.StatusOK, gin.H{"access_token": res.AccessToken, "user_id": res.UserId})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	rt, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing refresh token"})
		return
	}

	res, err := h.service.ServiceRefresh(c.Request.Context(), h.cfg, rt, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	SetCookie(c, "refresh_token", res.RefreshToken, 60*60*24*30, h.cfg)
	c.JSON(http.StatusOK, gin.H{"access_token": res.AccessToken})
}

func (h *AuthHandler) Revoke(c *gin.Context) {
	rt, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing token"})
		return
	}
	if err := h.service.ServiceRevoke(c.Request.Context(), rt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// delete cookie
	SetCookie(c, "refresh_token", "", -1, h.cfg)
	c.JSON(http.StatusOK, gin.H{"status": "revoked"})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	// Prefer cookie-based session revocation
	if rt, err := c.Cookie("refresh_token"); err == nil {
		if err := h.service.ServiceRevoke(c.Request.Context(), rt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// clear cookie

		SetCookie(c, "refresh_token", "", -1, h.cfg)
		c.JSON(http.StatusOK, gin.H{"status": "logged_out"})
		return
	}

	// Fallback: if bearer token provided, use middleware-injected user_id to remove all sessions
	uid, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing credentials"})
		return
	}
	userID, ok := uid.(string)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	if err := h.service.ServiceLogout(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// clear cookie if present
	SetCookie(c, "refresh_token", "", -1, h.cfg)

	c.JSON(http.StatusOK, gin.H{"status": "logged_out"})
}

func setProvider(c *gin.Context) {
	provider := c.Param("provider")
	q := c.Request.URL.Query()
	q.Set("provider", provider)
	c.Request.URL.RawQuery = q.Encode()
}

func SetupOAuth(cfg *config.Config) {
	goth.UseProviders(
		google.New(
			cfg.GOOGLE_CLIENT_ID,
			cfg.GOOGLE_CLIENT_SECRET,
			cfg.BACKEND_URL+"/v1/auth/oauth/google/callback",
			"email", "profile",
		),
	)

	store := sessions.NewCookieStore([]byte(cfg.SESSION_SECRET), []byte(cfg.SESSION_ENCRYPT_KEY))
	secure := cfg.APP_ENV != "development"

	var domain string
	if secure {
		domain = cfg.COOKIE_DOMAIN
	} else {
		domain = ""
	}

	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		Secure:   secure,
		Domain:   domain,
		SameSite: http.SameSiteLaxMode,
	}
	gothic.Store = store
}

func SetCookie(c *gin.Context, name, value string, maxAge int, cfg *config.Config) {
	secure := cfg.APP_ENV != "development"
	var domain string
	if secure {
		domain = cfg.COOKIE_DOMAIN
	} else {
		domain = ""
	}
	c.SetCookie(name, value, maxAge, "/", domain, secure, true)

}
