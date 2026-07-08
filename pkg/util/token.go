package util

import (
	"fmt"

	"github.com/brmcode/user-auth-service/internal/adapter/auth"
	"github.com/brmcode/user-auth-service/internal/adapter/auth/jwt"
	"github.com/brmcode/user-auth-service/internal/adapter/auth/paseto"
	dto "github.com/brmcode/user-auth-service/internal/adapter/http/handler/dto/common"
	"github.com/brmcode/user-auth-service/internal/core/domain"
	"github.com/brmcode/user-auth-service/internal/core/port"
	"github.com/brmcode/user-auth-service/pkg/config"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func NewTokenService(cfg *config.Auth) (port.TokenService, error) {
	switch cfg.TokenType {
	case "paseto", "PASETO":
		return paseto.New(cfg.SecretKey)
	case "jwt", "JWT":
		return jwt.New(cfg.SecretKey)
	default:
		return nil, fmt.Errorf("unsupported token type %q. Only \"paseto\" and \"jwt\" are supported", cfg.TokenType)
	}
}

func IssueSessionAndTokens(
	ctx *gin.Context,
	user *domain.User,
	cfg *config.Configuration,
	tokenService port.TokenService,
	sessionRepo port.SessionRepository,
) (*dto.LoginUserResponse, error) {
	accessToken, accessPayload, err := generateAccessToken(user, cfg, tokenService)
	if err != nil {
		return nil, fmt.Errorf("could not generate access token: %v", err)
	}

	refreshToken, refreshPayload, err := generateRefreshToken(user, cfg, tokenService)
	if err != nil {
		return nil, fmt.Errorf("could not generate refresh token: %v", err)
	}

	session, err := sessionRepo.Create(&domain.Session{
		ID:           refreshPayload.ID,
		Username:     user.Username,
		RefreshToken: refreshToken,
		UserAgent:    ctx.Request.UserAgent(),
		ClientIP:     ctx.ClientIP(),
		ExpiresAt:    refreshPayload.ExpiresAt,
	})
	if err != nil {
		return nil, err
	}

	return &dto.LoginUserResponse{
		SessionID:             session.ID,
		AccessToken:           accessToken,
		AccessTokenExpriresAt: accessPayload.ExpiresAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshPayload.ExpiresAt,
		User:                  user,
	}, nil
}

func generateAccessToken(user *domain.User, cfg *config.Configuration, ts port.TokenService) (string, *auth.Payload, error) {
	return ts.GenerateToken(uuid.Nil, user.Username, user.RoleCodes(), cfg.Auth.TokenDuration)
}

func generateRefreshToken(user *domain.User, cfg *config.Configuration, ts port.TokenService) (string, *auth.Payload, error) {
	return ts.GenerateToken(uuid.Nil, user.Username, user.RoleCodes(), cfg.Auth.RefreshTokenDuration)
}
