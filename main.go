package main

import (
	"fmt"
	"log"

	"github.com/brmcode/user-auth-service/controller"
	"github.com/brmcode/user-auth-service/database"
	"github.com/brmcode/user-auth-service/pkg/auth"
	"github.com/brmcode/user-auth-service/pkg/auth/jwt"
	"github.com/brmcode/user-auth-service/pkg/auth/paseto"
	"github.com/brmcode/user-auth-service/pkg/config"
	"github.com/brmcode/user-auth-service/repository"
	"github.com/brmcode/user-auth-service/service"
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

	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)

	userServ := service.NewUserService(userRepo)
	if config.Auth.TokenType == "paseto" {

	}
	tokenServ, err := newTokenService(config.Auth)
	if err != nil {
		log.Fatalf("failed to initialize token service: %v", err)
	}
	authServ := service.NewAuthenticationService(config.Auth, userRepo, sessionRepo, tokenServ)

	userCtrl := controller.NewUserController(userServ)
	authCtrl := controller.NewAuthController(userServ, authServ)

	controller.SetTokenService(tokenServ, db)

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

func newTokenService(config *config.Auth) (auth.TokenService, error) {
	switch config.TokenType {
	case "paseto", "PASETO":
		return paseto.New(config.SecretKey)
	case "jwt", "JWT":
		return jwt.New(config.SecretKey)
	default:
		return nil, fmt.Errorf("unsupported token type "+"\"%s\". Only \"paseto\" and \"jwt\" are supported", config.TokenType)
	}
}
