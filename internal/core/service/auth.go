package service

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/brmcode/user-auth-service/internal/core/domain"
	dto "github.com/brmcode/user-auth-service/internal/core/dto/common"
	"github.com/brmcode/user-auth-service/internal/core/dto/response"
	"github.com/brmcode/user-auth-service/internal/core/port"
	"github.com/brmcode/user-auth-service/pkg/config"
	"github.com/brmcode/user-auth-service/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type authService struct {
	config       *config.Auth
	userRepo     port.UserRepository
	sessionRepo  port.SessionRepository
	tokenService port.TokenService
	cache        port.CacheRepository
}

// ReNewAccessToken implements AuthenticationService.
func (a *authService) ReNewAccessToken(ctx *gin.Context, req dto.ReNewAccessTokenRequest) (*dto.ReNewAccessTokenResponse, *response.Error) {
	refreshPayload, err := a.tokenService.VerifyRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, response.NewError(401, err.Error())
	}

	var session *domain.Session
	cacheKey := util.GenerateCacheKey("session", refreshPayload.ID)
	cacheSession, err := a.cache.Get(ctx, cacheKey)
	cacheHit := false
	if err == nil && len(cacheSession) > 0 {
		if err := util.Deserialize(cacheSession, &session); err == nil {
			cacheHit = true
			log.Println("session cache hit")
		}
	}

	if !cacheHit {
		log.Println("session cache miss")
		session, err = a.sessionRepo.Get(refreshPayload.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, response.NewError(404, "session not found")
			}
			return nil, response.NewError(500, err.Error())
		}
	}

	if session.IsBlocked {
		return nil, response.NewError(401, "blocked session")
	}

	if session.Username != refreshPayload.Username {
		return nil, response.NewError(401, "incorrect session user")
	}

	if session.RefreshToken != req.RefreshToken {
		return nil, response.NewError(401, "mismatched session token")
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, response.NewError(401, "expired session")
	}

	accessToken, accessPayload, err := a.tokenService.GenerateToken(refreshPayload.Username, refreshPayload.Role, a.config.TokenDuration)
	if err != nil {
		return nil, response.NewError(500, fmt.Sprintf("could not generate token: %s", err.Error()))
	}

	session.ExpiresAt = time.Now().Add(a.config.RefreshTokenDuration)

	updatedSession, err := a.sessionRepo.Update(session)
	if err != nil {
		return nil, response.NewError(500, err.Error())
	}

	_ = a.cache.Delete(ctx, cacheKey)

	sessionSerialized, err := util.Serialize(updatedSession)
	if err != nil {
		return nil, response.NewError(500, err.Error())
	}

	err = a.cache.Set(ctx, cacheKey, sessionSerialized, time.Until(updatedSession.ExpiresAt))
	if err != nil {
		return nil, response.NewError(500, err.Error())
	}

	return &dto.ReNewAccessTokenResponse{
		AccessToken:           accessToken,
		AccessTokenExpriresAt: accessPayload.ExpiresAt,
	}, nil
}

// Login implements AuthenticationService.
func (a *authService) Login(ctx *gin.Context, cred dto.LoginModel) (*dto.LoginUserResponse, *response.Error) {
	user, err := a.userRepo.GetByEmailAndRole(cred.Email, cred.Role)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, response.NewError(404, "user not found")
		}
		return nil, response.NewError(500, err.Error())
	}

	if err := util.ComparePassword(cred.Password, user.HashedPassword); err != nil {
		return nil, response.NewError(400, "invalid credentials")
	}

	// Generate token
	accessToken, accessPayload, err := a.tokenService.GenerateToken(user.Username, user.Role, a.config.TokenDuration)
	if err != nil {
		log.Printf("Error: %s", err.Error())
		return nil, response.NewError(500, fmt.Sprintf("could not generate token: %s", err.Error()))
	}

	refresh_token, refreshPayload, err := a.tokenService.GenerateRefreshToken(user.Username, user.Role, a.config.RefreshTokenDuration)
	if err != nil {
		log.Printf("Error: %s", err.Error())
		return nil, response.NewError(500, fmt.Sprintf("could not generate refresh token: %s", err.Error()))
	}

	fmt.Println(a.config.RefreshTokenDuration)

	session, err := a.sessionRepo.Create(&domain.Session{
		ID:           refreshPayload.ID,
		Username:     user.Username,
		RefreshToken: refresh_token,
		UserAgent:    ctx.Request.UserAgent(),
		ClientIp:     ctx.ClientIP(),
		IsBlocked:    false,
		ExpiresAt:    refreshPayload.ExpiresAt,
	})

	if err != nil {
		return nil, response.NewError(500, err.Error())
	}

	cacheKey := util.GenerateCacheKey("session", refreshPayload.ID)
	sessionSerialized, err := util.Serialize(session)

	if err != nil {
		return nil, response.NewError(500, err.Error())
	}

	err = a.cache.Set(ctx, cacheKey, sessionSerialized, time.Until(refreshPayload.ExpiresAt))
	if err != nil {
		return nil, response.NewError(500, err.Error())
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

// Register implements AuthenticationService.
func (a *authService) Register(req dto.RegisterUserRequest) (*domain.User, *response.Error) {
	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		return nil, response.NewError(500, "failed to hash password")
	}

	user := &domain.User{
		Username:       util.RandomUsername(),
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		Email:          req.Email,
		HashedPassword: hashedPassword,
		Role:           domain.USER_ROLE,
	}

	createdUser, err := a.userRepo.Create(user)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, response.NewError(409, pgErr.Detail)
		}

		return nil, response.NewError(500, err.Error())
	}

	return createdUser, nil
}

func NewAuthenticationService(
	config *config.Auth,
	userRepo port.UserRepository,
	sessionRepo port.SessionRepository,
	tokenService port.TokenService,
	cache port.CacheRepository,
) port.AuthenticationService {
	return &authService{
		config:       config,
		userRepo:     userRepo,
		sessionRepo:  sessionRepo,
		tokenService: tokenService,
		cache:        cache,
	}
}
