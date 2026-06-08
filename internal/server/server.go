package server

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kings0x/crossPost/internal/cache"
	"github.com/kings0x/crossPost/internal/config"
	"github.com/kings0x/crossPost/internal/db"
	"github.com/kings0x/crossPost/internal/middleware"
)

type Server struct {
	http  *http.Server
	db    *db.Database
	redis cache.Cache
}

func New(db *db.Database, redis cache.Cache, cfg *config.Config) *Server {

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
		redis,
	}

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

	c := newContainer(srv.db, srv.redis)

	v1 := r.Group("v1")
	{
		auth := v1.Group("auth")
		{
			auth.POST("/do-something", c.auth.DoSomehting)
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

	if err := srv.redis.Ping(c.Request.Context()); err != nil {
		slog.ErrorContext(ctx, "srv.redis.Ping", "err", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"error":  "redis unavailable",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
