package controller

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/brmcode/user-auth-service/pkg/auth"
	"github.com/brmcode/user-auth-service/pkg/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const (
	ROLE_ADMIN = "ADMIN"
	ROLE_USER  = "USER"
)

type Router struct {
	*gin.Engine
}

func NewRouter(
	config *config.Configuration,
	tokenServ auth.TokenService,
	userCtrl *UserController,
	authCtrl *AuthController,

) (*Router, error) {
	router := gin.Default()
	router.Use(gin.Recovery())
	router.Use(cors.Default())

	router.GET("", serverRunning)

	api := router.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/login", authCtrl.Login)
			auth.POST("/register", authCtrl.Register)
			auth.POST("/refresh_token", authCtrl.RefreshToken)
		}
		user := api.Group("/users")
		{
			user.POST("", Authorized(ROLE_ADMIN), userCtrl.CreateUser)
			user.GET("/:username", Authorized(), userCtrl.GetUser)
			user.PUT("/:username", Authorized(), userCtrl.UpdateUser)
			user.DELETE("/:username", Authorized(), userCtrl.DeleteUser)
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
