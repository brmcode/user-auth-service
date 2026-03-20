package app

import (
	"context"
	"fmt"
	"log"

	"github.com/brmcode/user-auth-service/internal/adapter/google"
	"github.com/brmcode/user-auth-service/internal/adapter/storage/database"
	"github.com/brmcode/user-auth-service/internal/adapter/storage/database/repository"
	"github.com/brmcode/user-auth-service/internal/adapter/storage/redis"
	"github.com/brmcode/user-auth-service/internal/core/service"
	"github.com/brmcode/user-auth-service/pkg/config"
	"github.com/brmcode/user-auth-service/pkg/oauth"
	"github.com/brmcode/user-auth-service/pkg/util"
)

func Bootstrap(ctx context.Context) (*Container, error) {
	cfg, err := config.New(".")
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %v", err)
	}

	oauth.Init(cfg.OAuth)

	db, err := database.New(cfg.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	if err := db.Migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %v", err)
	}

	cache, err := redis.New(ctx, cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %v", err)
	}

	// repositories
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	oauthAccountRepo := repository.NewOauthAccountRepository(db)

	// services
	tokenServ, err := util.NewTokenService(cfg.Auth)
	if err != nil {
		log.Fatalf("failed to init token service: %v", err)
	}
	userServ := service.NewUserService(userRepo, cache, cfg)
	authServ := service.NewAuthenticationService(cfg, userRepo, sessionRepo, oauthAccountRepo, tokenServ, cache)

	idTokenVerifier := google.NewIDTokenVerifier(cfg.OAuth.GoogleClientID)

	return &Container{
		Cfg:              cfg,
		DB:               db,
		Cache:            cache,
		UserRepo:         userRepo,
		SessionRepo:      sessionRepo,
		OauthAccountRepo: oauthAccountRepo,
		UserService:      userServ,
		AuthService:      authServ,
		TokenService:     tokenServ,
		IDTokenVerifier:  idTokenVerifier,
	}, nil
}
