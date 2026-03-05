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

func NewTokenService(config *config.Auth) (port.TokenService, error) {
	switch config.TokenType {
	case "paseto", "PASETO":
		return paseto.New(config.SecretKey)
	case "jwt", "JWT":
		return jwt.New(config.SecretKey)
	default:
		return nil, fmt.Errorf("unsupported token type "+"\"%s\". Only \"paseto\" and \"jwt\" are supported", config.TokenType)
	}
}

func IssueSessionAndTokens(ctx *gin.Context, user *domain.User, config *config.Configuration, tokenService port.TokenService, sessionRepo port.SessionRepository) (*dto.LoginUserResponse, error) {
	// Generate token
	accessToken, accessPayload, err := generateAccessToken(user, config, tokenService)
	if err != nil {
		return nil, fmt.Errorf("could not generate access token: %v", err)
	}

	refresh_token, refreshPayload, err := generateRefreshToken(user, config, tokenService)
	if err != nil {
		return nil, fmt.Errorf("could not generate refresh token: %v", err)

	}

	session, err := sessionRepo.Create(&domain.Session{
		ID:           refreshPayload.ID,
		Username:     user.Username,
		RefreshToken: refresh_token,
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
		RefreshToken:          refresh_token,
		RefreshTokenExpiresAt: refreshPayload.ExpiresAt,
		User:                  user,
	}, nil
}

func generateAccessToken(user *domain.User, config *config.Configuration, tokenService port.TokenService) (string, *auth.Payload, error) {
	return tokenService.GenerateToken(
		uuid.Nil,
		user.Username,
		user.Role,
		config.Auth.TokenDuration,
	)
}

func generateRefreshToken(user *domain.User, config *config.Configuration, tokenService port.TokenService) (string, *auth.Payload, error) {
	return tokenService.GenerateToken(
		uuid.Nil,
		user.Username,
		user.Role,
		config.Auth.RefreshTokenDuration,
	)
}
