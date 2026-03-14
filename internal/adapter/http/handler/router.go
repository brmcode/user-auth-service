package handler

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	mw "github.com/brmcode/user-auth-service/internal/adapter/middleware"
	"github.com/brmcode/user-auth-service/internal/core/domain"
	"github.com/brmcode/user-auth-service/internal/core/port"
	"github.com/brmcode/user-auth-service/pkg/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Router struct {
	*gin.Engine
}

func NewRouter(
	config *config.Configuration,
	tokenServ port.TokenService,
	userHandler *UserHandler,
	authHandler *AuthHandler,
	oauthHandler *OAuthHandler,

) (*Router, error) {
	router := gin.Default()
	router.Use(mw.RateLimitMiddleware())
	router.Use(gin.Recovery())
	router.Use(cors.Default())

	router.GET("", serverRunning)

	api := router.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/register", authHandler.Register)
			auth.POST("/refresh_token", authHandler.RefreshToken)
			auth.POST("/logout", authHandler.Logout)
		}
		oauth := api.Group("/oauth")
		{
			oauth.GET("/:provider", oauthHandler.Begin)
			oauth.GET("/:provider/callback", oauthHandler.Callback)
		}
		user := api.Group("/users")
		{
			user.POST("", mw.Authorized(domain.ADMIN_ROLE), userHandler.CreateUser)
			user.GET("/:username", mw.Authorized(), userHandler.GetUser)
			user.PUT("/:username", mw.Authorized(), userHandler.UpdateUser)
			user.DELETE("/:username", mw.Authorized(), userHandler.DeleteUser)
		}
	}

	return &Router{router}, nil
}

func (r *Router) Serve(listenAddr string) {
	srv := &http.Server{
		Addr:    listenAddr,
		Handler: r,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s", err)
		}
	}()
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown failed: %v", err)
	}

	log.Printf("\nserver gracefully stopped")
}

func serverRunning(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"message": "server is running",
	})
}
