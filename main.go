package main

import (
	"context"
	"log"
	"net"

	"github.com/brmcode/user-auth-service/internal/adapter/auth/google"
	"github.com/brmcode/user-auth-service/internal/adapter/grpc"
	"github.com/brmcode/user-auth-service/internal/adapter/http/handler"
	"github.com/brmcode/user-auth-service/internal/adapter/middleware"
	"github.com/brmcode/user-auth-service/internal/adapter/storage/database"
	"github.com/brmcode/user-auth-service/internal/adapter/storage/database/repository"
	"github.com/brmcode/user-auth-service/internal/adapter/storage/redis"
	"github.com/brmcode/user-auth-service/internal/adapter/validator"
	"github.com/brmcode/user-auth-service/internal/core/service"
	"github.com/brmcode/user-auth-service/pkg/config"
	"github.com/brmcode/user-auth-service/pkg/oauth"
	"github.com/brmcode/user-auth-service/pkg/util"
)

var cfg *config.Configuration

func init() {
	var err error
	cfg, err = config.New(".")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	oauth.Init(cfg.OAuth)
}

func main() {
	db, err := database.New(cfg.DB)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	defer db.Close()

	err = db.Migrate()
	if err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	ctx := context.Background()
	cache, err := redis.New(ctx, cfg.Redis)
	if err != nil {
		log.Fatalf("failed to connect to cache: %v", err)
	}

	defer cache.Close()

	userRepo := repository.NewUserRepository(db.DB)
	roleRepo := repository.NewRoleRepository(db.DB)
	sessionRepo := repository.NewSessionRepository(db.DB)
	oauthAccountRepo := repository.NewOauthAccountRepository(db.DB)
	uow := database.NewUnitOfWork(db.DB)

	userServ := service.NewUserService(userRepo, roleRepo, uow, cache, cfg)
	if cfg.Auth.TokenType == "paseto" {

	}
	tokenServ, err := util.NewTokenService(cfg.Auth)
	if err != nil {
		log.Fatalf("failed to initialize token service: %v", err)
	}
	authServ := service.NewAuthenticationService(cfg, uow, userRepo, roleRepo, sessionRepo, oauthAccountRepo, tokenServ, cache)
	idTokenVerifier := google.NewIDTokenVerifier(cfg.OAuth.GoogleClientID)

	validator := validator.NewValidator()
	userCtrl := handler.NewUserHandler(validator, userServ)
	authCtrl := handler.NewAuthHandler(validator, userServ, authServ)
	oauthCtrl := handler.NewOAuthHandler(authServ, idTokenVerifier)

	middleware.Set(tokenServ, db)

	userServer := grpc.NewUserServer(userRepo, roleRepo, cache, cfg)
	authServer := grpc.NewAuthServer(cfg, userRepo, sessionRepo, tokenServ)

	listener, err := net.Listen("tcp", ":"+cfg.Grpc.Port)
	if err != nil {
		log.Fatal(err)
	}

	grpcServer, err := grpc.NewServer(cfg, userServer, authServer)
	go grpcServer.Serve(listener)

	router, err := handler.NewRouter(
		cfg,
		tokenServ,
		userCtrl,
		authCtrl,
		oauthCtrl,
	)
	if err != nil {
		log.Fatal(err)
	}

	router.Serve(":" + cfg.HTTP.Port)
}
