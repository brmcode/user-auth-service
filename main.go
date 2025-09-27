package main

import (
	"context"
	"fmt"
	"log"

	"github.com/brmcode/user-auth-service/internal/adapter/auth/jwt"
	"github.com/brmcode/user-auth-service/internal/adapter/auth/paseto"
	"github.com/brmcode/user-auth-service/internal/adapter/controller"
	"github.com/brmcode/user-auth-service/internal/adapter/middleware"
	"github.com/brmcode/user-auth-service/internal/adapter/storage/database"
	"github.com/brmcode/user-auth-service/internal/adapter/storage/database/repository"
	"github.com/brmcode/user-auth-service/internal/adapter/storage/redis"
	"github.com/brmcode/user-auth-service/internal/adapter/validator"
	"github.com/brmcode/user-auth-service/internal/core/port"
	"github.com/brmcode/user-auth-service/internal/core/service"
	"github.com/brmcode/user-auth-service/pkg/config"
)

func main() {
	fmt.Println("Hello, World!")

	config, err := config.New(".")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := database.New(config.DB)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	defer db.Close()

	err = db.Migrate()
	if err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	ctx := context.Background()
	cache, err := redis.New(ctx, config.Redis)
	if err != nil {
		log.Fatalf("failed to connect to cache: %v", err)
	}

	defer cache.Close()

	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)

	userServ := service.NewUserService(userRepo, cache, config)
	if config.Auth.TokenType == "paseto" {

	}
	tokenServ, err := newTokenService(config.Auth)
	if err != nil {
		log.Fatalf("failed to initialize token service: %v", err)
	}
	authServ := service.NewAuthenticationService(config.Auth, userRepo, sessionRepo, tokenServ, cache)

	validator := validator.NewValidator()
	userCtrl := controller.NewUserController(validator, userServ)
	authCtrl := controller.NewAuthController(validator, userServ, authServ)

	middleware.Set(tokenServ, db)

	router, err := controller.NewRouter(
		config,
		tokenServ,
		userCtrl,
		authCtrl,
	)
	if err != nil {
		log.Fatal(err)
	}

	router.Serve(":" + config.HTTP.Port)
}

func newTokenService(config *config.Auth) (port.TokenService, error) {
	switch config.TokenType {
	case "paseto", "PASETO":
		return paseto.New(config.SecretKey)
	case "jwt", "JWT":
		return jwt.New(config.SecretKey)
	default:
		return nil, fmt.Errorf("unsupported token type "+"\"%s\". Only \"paseto\" and \"jwt\" are supported", config.TokenType)
	}
}
