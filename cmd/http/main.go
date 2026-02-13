package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/brmcode/user-auth-service/internal/adapter/controller"
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
	userCtrl := controller.NewUserController(validator, c.UserService)
	authCtrl := controller.NewAuthController(validator, c.UserService, c.AuthService)
	oauthCtrl := controller.NewOAuthController(c.AuthService)

	router, err := controller.NewRouter(
		c.Cfg,
		c.TokenService,
		userCtrl,
		authCtrl,
		oauthCtrl,
	)
	if err != nil {
		log.Fatal(err)
	}

	router.Serve(":" + c.Cfg.HTTP.Port)
}
