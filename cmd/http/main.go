package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/brmcode/user-auth-service/internal/adapter/http/handler"
	"github.com/brmcode/user-auth-service/internal/adapter/middleware"
	"github.com/brmcode/user-auth-service/internal/adapter/validator"
	"github.com/brmcode/user-auth-service/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	c, err := app.Bootstrap(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	middleware.Set(c.TokenService, c.DB)

	validator := validator.NewValidator()
	userCtrl := handler.NewUserHandler(validator, c.UserService)
	authCtrl := handler.NewAuthHandler(validator, c.UserService, c.AuthService)
	oauthCtrl := handler.NewOAuthHandler(c.AuthService, c.IDTokenVerifier)
	mediaCtrl := handler.NewMediaHandler("uploads/avatars")

	router, err := handler.NewRouter(
		c.Cfg,
		c.TokenService,
		userCtrl,
		authCtrl,
		oauthCtrl,
		mediaCtrl,
	)
	if err != nil {
		log.Fatal(err)
	}

	router.Serve(":" + c.Cfg.HTTP.Port)
}
