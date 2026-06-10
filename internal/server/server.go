package server

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kings0x/crossPost/internal/auth"
	"github.com/kings0x/crossPost/internal/config"
	"github.com/kings0x/crossPost/internal/db"
	"github.com/kings0x/crossPost/internal/middleware"
	rds "github.com/redis/go-redis/v9"
)

type Server struct {
	http  *http.Server
	db    *db.Database
	redis *rds.Client
	cfg   *config.Config
}

func New(db *db.Database, redisClient *rds.Client, cfg *config.Config) *Server {

	router := gin.New()

	router.Use(gin.Recovery())

	router.Use(middleware.RequestLogger())

	http := &http.Server{
		Addr:         ":" + cfg.PORT,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	srv := &Server{
		http,
		db,
		redisClient,
		cfg,
	}

	// Initialize OAuth providers and session store once at server startup
	// so handlers don't re-register providers on each construction.
	auth.SetupOAuth(cfg)

	srv.RegisterRoutes(router)

	return srv

}

func (srv *Server) Start() error {
	slog.Info("starting server", "addr", srv.http.Addr)
	addr := "http://localhost" + srv.http.Addr

	log.Printf("server listening on %v\n", addr)

	if err := srv.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (srv *Server) Shutdown(shutdownCtx context.Context) error {
	slog.Info("shutdown signal received")

	if err := srv.http.Shutdown(shutdownCtx); err != nil {
		return err
	}

	slog.Info("server stopped")

	return nil

}

func (srv *Server) RegisterRoutes(r *gin.Engine) {

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "running",
		})
	})

	r.GET("/health", srv.CheckHealth)

	c := newContainer(srv.db, srv.redis, srv.cfg)

	v1 := r.Group("v1")
	{
		auth := v1.Group("auth")
		{
			auth.POST("/forgot-password", c.auth.ForgotPassword)
			auth.POST("/reset-password", c.auth.ResetPassword)
			auth.POST("/signup", c.auth.SignUp)
			auth.POST("/login", c.auth.Login)
			auth.GET("/verify", c.auth.VerifyEmail)
			auth.POST("/refresh", c.auth.Refresh)
			auth.POST("/revoke", c.auth.Revoke)
			auth.POST("/logout", c.auth.Logout)

			oauth := auth.Group("/oauth")
			{
				oauth.GET("/:provider", c.auth.OauthBegin)
				oauth.GET("/:provider/callback", c.auth.OauthCallback)
			}
		}
	}
}

func (srv *Server) CheckHealth(c *gin.Context) {

	ctx := c.Request.Context()

	if err := srv.db.HealthCheck(ctx); err != nil {
		slog.ErrorContext(ctx, "srv.db.HealthCheck", "err", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"error":  "database unavailable",
		})

		return
	}

	if err := srv.redis.Ping(ctx).Err(); err != nil {
		slog.ErrorContext(ctx, "srv.redis.Ping", "err", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"error":  "redis unavailable",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
