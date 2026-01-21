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
	config           *config.Configuration
	userRepo         port.UserRepository
	sessionRepo      port.SessionRepository
	oauthAccountRepo port.OauthAccountRepository
	tokenService     port.TokenService
	cache            port.CacheRepository
}

// OAuthLogin implements port.AuthenticationService.
func (a *authService) OAuthLogin(ctx *gin.Context, req dto.OAuthRegisterUserRequest) (*dto.LoginUserResponse, *response.Error) {
	account, err := a.oauthAccountRepo.GetByProvider(req.Provider, req.ProviderUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 1. Try to find any user (including soft-deleted) with the same email.
			existingUser, errUser := a.userRepo.GetByEmailUnscoped(req.Email)
			if errUser == nil {
				// User exists (maybe soft-deleted) – restore if necessary and link OAuth account.
				if existingUser.DeletedAt.Valid {
					existingUser.DeletedAt = gorm.DeletedAt{}
					if _, err := a.userRepo.Update(existingUser); err != nil {
						return nil, response.NewError(500, "failed to restore user")
					}
				}

				// Ensure there's an OAuth account for this user.
				_, err := a.oauthAccountRepo.Create(&domain.OauthAccount{
					ID:             uuid.New(),
					Username:       existingUser.Username,
					Provider:       req.Provider,
					ProviderUserID: req.ProviderUserID,
					Email:          req.Email,
				})

				if err != nil {
					return nil, response.NewError(500, "failed to create oauth account")
				}

				// Update cache for the restored/existing user.
				cacheKey := util.GenerateCacheKey("user", existingUser.Username)
				userSerialized, err := util.Serialize(existingUser)
				if err != nil {
					return nil, response.NewError(500, err.Error())
				}

				errChan := make(chan error, 2)
				go func() {
					errChan <- a.cache.Set(ctx, cacheKey, userSerialized, a.config.Redis.TTL)
				}()
				go func() {
					errChan <- a.cache.DeleteByPrefix(ctx, "users:*")
				}()

				for i := 0; i < 2; i++ {
					if err := <-errChan; err != nil {
						return nil, response.NewError(500, err.Error())
					}
				}

				return a.issueSessionAndTokens(ctx, existingUser)
			}

			// 2. User with this email does not exist at all – create new user and OAuth account.
			if !errors.Is(errUser, gorm.ErrRecordNotFound) {
				return nil, response.NewError(500, "failed to lookup user by email")
			}

			user := &domain.User{
				Username:  util.RandomUsername(),
				FirstName: req.FirstName,
				LastName:  req.LastName,
				Email:     req.Email,
				Role:      domain.USER_ROLE,
			}

			createdUser, err := a.userRepo.Create(user)
			if err != nil {
				return nil, response.NewError(500, "failed to create user")
			}

			cacheKey := util.GenerateCacheKey("user", createdUser.Username)
			userSerialized, err := util.Serialize(createdUser)
			if err != nil {
				return nil, response.NewError(500, err.Error())
			}

			errChan := make(chan error, 2)
			go func() {
				errChan <- a.cache.Set(ctx, cacheKey, userSerialized, a.config.Redis.TTL)
			}()
			go func() {
				errChan <- a.cache.DeleteByPrefix(ctx, "users:*")
			}()

			for i := 0; i < 2; i++ {
				if err := <-errChan; err != nil {
					return nil, response.NewError(500, err.Error())
				}
			}

			_, err = a.oauthAccountRepo.Create(&domain.OauthAccount{
				ID:             uuid.New(),
				Username:       createdUser.Username,
				Provider:       req.Provider,
				ProviderUserID: req.ProviderUserID,
				Email:          req.Email,
			})
			if err != nil {
				return nil, response.NewError(500, "failed to create oauth account")
			}

			return a.issueSessionAndTokens(ctx, createdUser)
		}
		return nil, response.NewError(500, "failed to get oauth account")
	}

	// OAuth account exists, get the associated user
	user, err := a.userRepo.Get(account.Username)
	if err != nil {
		return nil, response.NewError(500, "failed to get user")
	}
	return a.issueSessionAndTokens(ctx, user)
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

	return a.issueSessionAndTokens(ctx, user)
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

	// Parallel cache operations: set new cache and delete prefix cache concurrently
	errChan := make(chan error, 2)
	go func() {
		errChan <- a.cache.Set(ctx, cacheKey, userSerialized, a.config.Redis.TTL)
	}()
	go func() {
		errChan <- a.cache.DeleteByPrefix(ctx, "users:*")
	}()

	// Wait for both operations
	for i := 0; i < 2; i++ {
		if err := <-errChan; err != nil {
			return nil, response.NewError(500, err.Error())
		}
	}

	return createdUser, nil
}

func (a *authService) issueSessionAndTokens(ctx *gin.Context, user *domain.User) (*dto.LoginUserResponse, *response.Error) {
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

func NewAuthenticationService(
	config *config.Configuration,
	userRepo port.UserRepository,
	sessionRepo port.SessionRepository,
	oauthAccountRepo port.OauthAccountRepository,
	tokenService port.TokenService,
	cache port.CacheRepository,
) port.AuthenticationService {
	return &authService{
		config:           config,
		userRepo:         userRepo,
		sessionRepo:      sessionRepo,
		oauthAccountRepo: oauthAccountRepo,
		tokenService:     tokenService,
		cache:            cache,
	}
}
