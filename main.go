package main

import (
	"fmt"
	"log"

	"github.com/brmcode/user-auth-service/controller"
	"github.com/brmcode/user-auth-service/database"
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

	userServ := service.NewUserService(userRepo)
	tokenServ, err := paseto.New(config.Auth.SecretKey)
	if err != nil {
		log.Fatalf("failed to initialize token service: %v", err)
	}
	authServ := service.NewAuthenticationService(config.Auth, userRepo, tokenServ)

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
