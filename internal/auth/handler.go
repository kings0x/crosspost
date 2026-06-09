package auth

import (
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

	platform := c.Param("provider") // "google"
	intent := c.Query("intent")

	url, err := h.service.ServiceOauthBegin(c.Request.Context(), h.cfg, platform, intent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, url)
}

func (h *AuthHandler) OauthCallback(c *gin.Context) {
	setProvider(c)

	state := c.Query("state")

	user_agent := c.Request.UserAgent()
	ip_addr := c.Request.RemoteAddr

	gothUser, err := gothic.CompleteUserAuth(c.Writer, c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	res, err := h.service.ServiceOauthCallback(c.Request.Context(), h.cfg, &gothUser, state, user_agent, ip_addr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "something went wrong"})
		return
	}

	secure := h.cfg.APP_ENV != "development"
	c.SetCookie("refresh_token", res.RefreshToken, 60*60*24*30, "/", h.cfg.BACKEND_URL, secure, true)

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

	secure := h.cfg.APP_ENV != "development"
	c.SetCookie("refresh_token", res.RefreshToken, 60*60*24*30, "/", h.cfg.BACKEND_URL, secure, true)
	c.JSON(http.StatusOK, gin.H{"access_token": res.AccessToken, "user_id": res.UserId})
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	res, err := h.service.ServiceVerifyEmail(c.Request.Context(), h.cfg, token, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	secure := h.cfg.APP_ENV != "development"
	c.SetCookie("refresh_token", res.RefreshToken, 60*60*24*30, "/", h.cfg.BACKEND_URL, secure, true)
	c.JSON(http.StatusOK, gin.H{"access_token": res.AccessToken, "user_id": res.UserId})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	//we are meant to pass this from a middleware but for now we will just read it from the cookie
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

	secure := h.cfg.APP_ENV != "development"
	c.SetCookie("refresh_token", res.RefreshToken, 60*60*24*30, "/", h.cfg.BACKEND_URL, secure, true)
	c.JSON(http.StatusOK, gin.H{"access_token": res.AccessToken})
}

func (h *AuthHandler) Revoke(c *gin.Context) {
	//same here refresh token should come from middleware but we will read it from the cookie for now
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
	c.SetCookie("refresh_token", "", -1, "/", h.cfg.BACKEND_URL, false, true)
	c.JSON(http.StatusOK, gin.H{"status": "revoked"})
}

func setProvider(c *gin.Context) {
	provider := c.Param("provider")
	q := c.Request.URL.Query()
	q.Set("provider", provider)
	c.Request.URL.RawQuery = q.Encode()
}

//db would be
//users
//user_sessions
//user_tokens(one time stuff)
//i think a seperate place for oauth

//6 functions
//first is google oauth
//second is to handle the callback
//third is the signup function
//fourth is the login function
//fifth is the token refresh function
//sixth is the logout function

//then we would have two pure functions
//issue_token
//setupProvider

func SetupOAuth(cfg *config.Config) {
	goth.UseProviders(
		google.New(
			cfg.GOOGLE_CLIENT_ID,
			cfg.GOOGLE_CLIENT_SECRET,
			cfg.BACKEND_URL+"/v1/auth/google/callback",
			"email", "profile",
		),
	)

	store := sessions.NewCookieStore([]byte(cfg.SESSION_SECRET))
	store.MaxAge(300)
	store.Options.HttpOnly = true
	store.Options.Secure = cfg.APP_ENV != "development"
	gothic.Store = store
}
