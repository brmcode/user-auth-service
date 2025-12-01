package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/brmcode/user-auth-service/internal/core/domain"
	dto "github.com/brmcode/user-auth-service/internal/core/dto/common"
	"github.com/brmcode/user-auth-service/internal/core/dto/response"
	"github.com/brmcode/user-auth-service/internal/core/port"
	"github.com/brmcode/user-auth-service/pkg/config"
	"github.com/brmcode/user-auth-service/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type authService struct {
	config       *config.Configuration
	userRepo     port.UserRepository
	sessionRepo  port.SessionRepository
	tokenService port.TokenService
	cache        port.CacheRepository
}

// ReNewAccessToken implements AuthenticationService.
func (a *authService) ReNewAccessToken(ctx *gin.Context, req dto.ReNewAccessTokenRequest) (*dto.ReNewAccessTokenResponse, *response.Error) {
	// Step 1: Verify the refresh token
	refreshPayload, err := a.tokenService.VerifyToken(req.RefreshToken)
	if err != nil {
		return nil, response.NewError(401, err.Error())
	}

	// Step 2: Retrieve the session by token ID
	session, err := a.sessionRepo.Get(refreshPayload.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewError(404, "session not found")
		}
		return nil, response.NewError(500, err.Error())
	}

	// Step 3: Validate session state and ownership
	if session.IsBlocked {
		return nil, response.NewError(401, "blocked session")
	}
	if session.Username != refreshPayload.Username {
		return nil, response.NewError(401, "incorrect session user")
	}
	if session.RefreshToken != req.RefreshToken {
		if err := a.sessionRepo.BlockAllSessions(session.Username); err != nil {
			return nil, response.NewError(500, fmt.Sprintf("failed to block sessions: %s", err.Error()))
		}
		return nil, response.NewError(401, "refresh token reuse detected: sessions have been blocked")
	}
	if time.Now().After(session.ExpiresAt) {
		return nil, response.NewError(401, "expired session")
	}

	// Step 4: Ensure user still exists and their role matches the token
	userOfsession, err := a.userRepo.Get(session.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewError(404, "invalid session: user data missing")
		}
		return nil, response.NewError(500, err.Error())
	}
	if userOfsession.Role != refreshPayload.Role {
		if err := a.sessionRepo.BlockAllSessions(session.Username); err != nil {
			return nil, response.NewError(500, fmt.Sprintf("failed to block sessions: %s", err.Error()))
		}
		return nil, response.NewError(401, "user role has changed: sessions have been invalidated, please login again")
	}

	// Step 5: Generate new access token
	accessToken, accessPayload, err := a.tokenService.GenerateToken(
		uuid.Nil,
		refreshPayload.Username,
		refreshPayload.Role,
		a.config.Auth.TokenDuration,
	)
	if err != nil {
		return nil, response.NewError(500, fmt.Sprintf("could not generate new access token: %s", err.Error()))
	}

	// Step 6: Generate new refresh token
	newRefreshToken, newRefreshPayload, err := a.tokenService.GenerateToken(
		refreshPayload.ID,
		refreshPayload.Username,
		refreshPayload.Role,
		a.config.Auth.RefreshTokenDuration,
	)
	if err != nil {
		return nil, response.NewError(500, fmt.Sprintf("could not generate new refresh token: %s", err.Error()))
	}

	// Step 7: Update session with new refresh token and expiry
	session.RefreshToken = newRefreshToken
	session.ExpiresAt = newRefreshPayload.ExpiresAt

	if _, err := a.sessionRepo.Update(session); err != nil {
		return nil, response.NewError(500, err.Error())
	}

	// Step 8: Return response
	return &dto.ReNewAccessTokenResponse{
		AccessToken:           accessToken,
		AccessTokenExpriresAt: accessPayload.ExpiresAt,
		RefreshToken:          newRefreshToken,
		RefreshTokenExpiresAt: newRefreshPayload.ExpiresAt,
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
	accessToken, accessPayload, err := a.tokenService.GenerateToken(uuid.Nil, user.Username, user.Role, a.config.Auth.TokenDuration)
	if err != nil {
		return nil, response.NewError(500, fmt.Sprintf("could not generate access token: %s", err.Error()))
	}

	refresh_token, refreshPayload, err := a.tokenService.GenerateToken(uuid.Nil, user.Username, user.Role, a.config.Auth.RefreshTokenDuration)
	if err != nil {
		return nil, response.NewError(500, fmt.Sprintf("could not generate refresh token: %s", err.Error()))
	}

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
func (a *authService) Register(ctx *gin.Context, req dto.RegisterUserRequest) (*domain.User, *response.Error) {
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

	cacheKey := util.GenerateCacheKey("user", createdUser.Username)
	userSerialized, err := util.Serialize(createdUser)
	if err != nil {
		return nil, response.NewError(500, err.Error())
	}

	err = a.cache.Set(ctx, cacheKey, userSerialized, a.config.Redis.TTL)
	if err != nil {
		return nil, response.NewError(500, err.Error())
	}

	err = a.cache.DeleteByPrefix(ctx, "users:*")
	if err != nil {
		return nil, response.NewError(500, err.Error())
	}

	return createdUser, nil
}

func NewAuthenticationService(
	config *config.Configuration,
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
