package service

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/brmcode/user-auth-service/internal/adapter/google"
	dto "github.com/brmcode/user-auth-service/internal/adapter/http/handler/dto/common"
	"github.com/brmcode/user-auth-service/internal/adapter/http/handler/dto/response"
	"github.com/brmcode/user-auth-service/internal/core/domain"

	"github.com/brmcode/user-auth-service/internal/core/port"
	"github.com/brmcode/user-auth-service/pkg/config"
	"github.com/brmcode/user-auth-service/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/markbates/goth"
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

// GoogleAuthMobile implements [port.AuthenticationService].
func (a *authService) GoogleAuthMobile(ctx *gin.Context, payload *google.Payload) *response.LoginResult {
	account, err := a.oauthAccountRepo.GetByProvider("google", payload.Subject)
	if err == nil {
		user, err := a.userRepo.Get(account.Username)
		if err != nil {
			return response.Login(false, 500, "failed to get user", nil, &[]string{err.Error()})
		}
		return a.loginSuccess(ctx, user)
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return response.Login(false, 500, "failed to get oauth account", nil, &[]string{err.Error()})
	}
	var user *domain.User
	user, err = a.userRepo.GetByEmailUnscoped(payload.Email)
	if err == nil {
		if user.DeletedAt.Valid {
			user.DeletedAt = gorm.DeletedAt{}
			if _, err := a.userRepo.Update(user); err != nil {
				return response.Login(false, 500, "failed to restore user", nil, &[]string{err.Error()})
			}
		}
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		newUser := &domain.User{
			Username:  util.RandomUsername(),
			FirstName: payload.FirstName,
			LastName:  payload.LastName,
			Email:     payload.Email,
			ImageURL:  payload.AvatarURL,
			Role:      domain.USER_ROLE,
		}
		user, err = a.userRepo.Create(newUser)
		if err != nil {
			return response.Login(false, 500, "failed to create user", nil, &[]string{err.Error()})
		}
	} else {
		return response.Login(false, 500, "failed to lookup user", nil, &[]string{err.Error()})
	}

	_, err = a.oauthAccountRepo.Create(&domain.OauthAccount{
		ID:             uuid.New(),
		Username:       user.Username,
		Provider:       "google",
		ProviderUserID: payload.Subject,
		Email:          payload.Email,
	})
	if err != nil {
		return response.Login(false, 500, "failed to create oauth account", nil, &[]string{err.Error()})
	}

	// cache user session
	if err := a.cacheUser(ctx, user); err != nil {
		return response.Login(false, 500, err.Error(), nil, &[]string{err.Error()})
	}

	return a.loginSuccess(ctx, user)
}

// OAuthLogin implements port.AuthenticationService.
// func (a *authService) OAuthLogin(ctx *gin.Context, provider string, gUser goth.User) *response.Login {
// 	account, err := a.oauthAccountRepo.GetByProvider(provider, gUser.UserID)
// 	if err != nil {
// 		if errors.Is(err, gorm.ErrRecordNotFound) {
// 			// 1. Try to find any user (including soft-deleted) with the same email.
// 			existingUser, errUser := a.userRepo.GetByEmailUnscoped(gUser.Email)
// 			if errUser == nil {
// 				// User exists (maybe soft-deleted) – restore if necessary and link OAuth account.
// 				if existingUser.DeletedAt.Valid {
// 					existingUser.DeletedAt = gorm.DeletedAt{}
// 					if _, err := a.userRepo.Update(existingUser); err != nil {
// 						return response.Login(false, 500, "failed to restore user", nil, &[]string{err.Error()})
// 					}
// 				}

// 				// Ensure there's an OAuth account for this user.
// 				_, err := a.oauthAccountRepo.Create(&domain.OauthAccount{
// 					ID:             uuid.New(),
// 					Username:       existingUser.Username,
// 					Provider:       provider,
// 					ProviderUserID: gUser.UserID,
// 					Email:          gUser.Email,
// 				})

// 				if err != nil {
// 					return response.Login(false, 500, "failed to create oauth account", nil, &[]string{err.Error()})
// 				}

// 				// Update cache for the restored/existing user.
// 				cacheKey := util.GenerateCacheKey("user", existingUser.Username)
// 				userSerialized, err := util.Serialize(existingUser)
// 				if err != nil {
// 					return response.Login(false, 500, err.Error(), nil, &[]string{err.Error()})
// 				}

// 				errChan := make(chan error, 2)
// 				go func() {
// 					errChan <- a.cache.Set(ctx, cacheKey, userSerialized, a.config.Redis.TTL)
// 				}()
// 				go func() {
// 					errChan <- a.cache.DeleteByPrefix(ctx, "users:*")
// 				}()

// 				for i := 0; i < 2; i++ {
// 					if err := <-errChan; err != nil {
// 						return response.Login(false, 500, err.Error(), nil, &[]string{err.Error()})
// 					}
// 				}
// 				token, err := util.IssueSessionAndTokens(ctx, existingUser, a.config, a.tokenService, a.sessionRepo)
// 				if err != nil {
// 					return response.Login(false, 500, err.Error(), nil, &[]string{err.Error()})
// 				}
// 				return response.Login(true, 200, "oauth login successful", token, nil)
// 			}

// 			// 2. User with this email does not exist at all – create new user and OAuth account.
// 			if !errors.Is(errUser, gorm.ErrRecordNotFound) {
// 				return response.Login(false, 500, "failed to lookup user by email", nil, &[]string{errUser.Error()})
// 			}

// 			user := &domain.User{
// 				Username:  util.RandomUsername(),
// 				FirstName: gUser.FirstName,
// 				LastName:  gUser.LastName,
// 				Email:     gUser.Email,
// 				ImageURL:  gUser.AvatarURL,
// 				Role:      domain.USER_ROLE,
// 			}

// 			createdUser, err := a.userRepo.Create(user)
// 			if err != nil {
// 				return response.Login(false, 500, "failed to create user", nil, &[]string{err.Error()})
// 			}

// 			_, err = a.oauthAccountRepo.Create(&domain.OauthAccount{
// 				ID:             uuid.New(),
// 				Username:       createdUser.Username,
// 				Provider:       provider,
// 				ProviderUserID: gUser.UserID,
// 				Email:          gUser.Email,
// 			})
// 			if err != nil {
// 				return response.Login(false, 500, "failed to create oauth account", nil, &[]string{err.Error()})
// 			}

// 			token, err := util.IssueSessionAndTokens(ctx, createdUser, a.config, a.tokenService, a.sessionRepo)
// 			if err != nil {
// 				return response.Login(false, 500, err.Error(), nil, &[]string{err.Error()})
// 			}
// 			//set cache for the new user and invalidate users list cache concurrently
// 			{
// 				cacheKey := util.GenerateCacheKey("user", createdUser.Username)
// 				userSerialized, err := util.Serialize(createdUser)
// 				if err != nil {
// 					return response.Login(false, 500, err.Error(), nil, &[]string{err.Error()})
// 				}

// 				errChan := make(chan error, 2)
// 				go func() {
// 					errChan <- a.cache.Set(ctx, cacheKey, userSerialized, a.config.Redis.TTL)
// 				}()
// 				go func() {
// 					errChan <- a.cache.DeleteByPrefix(ctx, "users:*")
// 				}()

// 				for i := 0; i < 2; i++ {
// 					if err := <-errChan; err != nil {
// 						return response.Login(false, 500, err.Error(), nil, &[]string{err.Error()})
// 					}
// 				}
// 			}
// 			return response.Login(true, 200, "oauth login successful", token, nil)
// 		}
// 		return response.Login(false, 500, "failed to get oauth account", nil, &[]string{err.Error()})
// 	}

// 	// OAuth account exists, get the associated user
// 	user, err := a.userRepo.Get(account.Username)
// 	if err != nil {
// 		return response.Login(false, 500, "failed to get user", nil, &[]string{err.Error()})
// 	}

// 	token, err := util.IssueSessionAndTokens(ctx, user, a.config, a.tokenService, a.sessionRepo)
// 	if err != nil {
// 		return response.Login(false, 500, err.Error(), nil, &[]string{err.Error()})
// 	}
// 	return response.Login(true, 200, "oauth login successful", token, nil)
// }

func (a *authService) OAuthLogin(ctx *gin.Context, provider string, gUser goth.User) *response.LoginResult {
	account, err := a.oauthAccountRepo.GetByProvider(provider, gUser.UserID)
	if err == nil {
		user, err := a.userRepo.Get(account.Username)
		if err != nil {
			return response.Login(false, 500, "failed to get user", nil, &[]string{err.Error()})
		}
		return a.loginSuccess(ctx, user)
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return response.Login(false, 500, "failed to get oauth account", nil, &[]string{err.Error()})
	}

	user, err := a.userRepo.GetByEmailUnscoped(gUser.Email)
	if err == nil {
		if user.DeletedAt.Valid {
			user.DeletedAt = gorm.DeletedAt{}
			if _, err := a.userRepo.Update(user); err != nil {
				return response.Login(false, 500, "failed to restore user", nil, &[]string{err.Error()})
			}
		}
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		newUser := &domain.User{
			Username:  util.RandomUsername(),
			FirstName: gUser.FirstName,
			LastName:  gUser.LastName,
			Email:     gUser.Email,
			ImageURL:  gUser.AvatarURL,
			Role:      domain.USER_ROLE,
		}

		user, err = a.userRepo.Create(newUser)
		if err != nil {
			return response.Login(false, 500, "failed to create user", nil, &[]string{err.Error()})
		}
	} else {
		return response.Login(false, 500, "failed to lookup user", nil, &[]string{err.Error()})
	}

	_, err = a.oauthAccountRepo.Create(&domain.OauthAccount{
		ID:             uuid.New(),
		Username:       user.Username,
		Provider:       provider,
		ProviderUserID: gUser.UserID,
		Email:          gUser.Email,
	})
	if err != nil {
		return response.Login(false, 500, "failed to create oauth account", nil, &[]string{err.Error()})
	}

	// cache
	if err := a.cacheUser(ctx, user); err != nil {
		return response.Login(false, 500, err.Error(), nil, &[]string{err.Error()})
	}

	return a.loginSuccess(ctx, user)
}

// ReNewAccessToken implements AuthenticationService.
func (a *authService) ReNewAccessToken(ctx *gin.Context, req dto.ReNewAccessTokenRequest) *response.RefreshTokenResult {
	// Step 1: Verify the refresh token
	refreshPayload, err := a.tokenService.VerifyToken(req.RefreshToken)
	if err != nil {
		return response.RefreshToken(false, 401, err.Error(), nil, &[]string{err.Error()})
	}

	// Step 2: Retrieve the session by token ID
	session, err := a.sessionRepo.Get(refreshPayload.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.RefreshToken(false, 404, "session not found", nil, &[]string{err.Error()})
		}
		return response.RefreshToken(false, 500, err.Error(), nil, &[]string{err.Error()})
	}

	// Step 3: Validate session state and ownership
	if session.IsBlocked {
		return response.RefreshToken(false, 401, "blocked session", nil, nil)
	}
	if session.Username != refreshPayload.Username {
		return response.RefreshToken(false, 401, "incorrect session user", nil, nil)
	}
	if session.RefreshToken != req.RefreshToken {
		if err := a.sessionRepo.BlockAllSessions(session.Username); err != nil {
			return response.RefreshToken(false, 500, fmt.Sprintf("failed to block sessions: %s", err.Error()), nil, &[]string{err.Error()})
		}
		return response.RefreshToken(false, 401, "refresh token reuse detected: sessions have been blocked", nil, nil)
	}
	if time.Now().After(session.ExpiresAt) {
		return response.RefreshToken(false, 401, "expired session", nil, nil)
	}

	// Step 4: Ensure user still exists and their role matches the token
	userOfsession, err := a.userRepo.Get(session.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.RefreshToken(false, 404, "invalid session: user data missing", nil, &[]string{err.Error()})
		}
		return response.RefreshToken(false, 500, err.Error(), nil, &[]string{err.Error()})
	}
	if userOfsession.Role != refreshPayload.Role {
		if err := a.sessionRepo.BlockAllSessions(session.Username); err != nil {
			return response.RefreshToken(false, 500, fmt.Sprintf("failed to block sessions: %s", err.Error()), nil, &[]string{err.Error()})
		}
		return response.RefreshToken(false, 401, "user role has changed: sessions have been invalidated, please login again", nil, nil)
	}

	// Step 5: Generate new access token
	accessToken, accessPayload, err := a.tokenService.GenerateToken(
		uuid.Nil,
		refreshPayload.Username,
		refreshPayload.Role,
		a.config.Auth.TokenDuration,
	)
	if err != nil {
		return response.RefreshToken(false, 500, fmt.Sprintf("could not generate new access token: %s", err.Error()), nil, &[]string{err.Error()})
	}

	// Step 6: Generate new refresh token
	RefreshToken, newRefreshPayload, err := a.tokenService.GenerateToken(
		refreshPayload.ID,
		refreshPayload.Username,
		refreshPayload.Role,
		a.config.Auth.RefreshTokenDuration,
	)
	if err != nil {
		return response.RefreshToken(false, 500, fmt.Sprintf("could not generate new refresh token: %s", err.Error()), nil, &[]string{err.Error()})
	}

	// Step 7: Update session with new refresh token and expiry
	session.RefreshToken = RefreshToken
	session.ExpiresAt = newRefreshPayload.ExpiresAt

	if _, err := a.sessionRepo.Update(session); err != nil {
		return response.RefreshToken(false, 500, err.Error(), nil, &[]string{err.Error()})
	}

	// Step 8: Return response
	return response.RefreshToken(true, 200, "refresh token renewed successfully", &dto.ReNewAccessTokenResponse{
		AccessToken:           accessToken,
		AccessTokenExpriresAt: accessPayload.ExpiresAt,
		RefreshToken:          RefreshToken,
		RefreshTokenExpiresAt: newRefreshPayload.ExpiresAt,
	}, nil)
}

// Logout implements AuthenticationService.
func (a *authService) Logout(ctx *gin.Context, req dto.ReNewAccessTokenRequest) *response.LogoutResult {
	refreshPayload, err := a.tokenService.VerifyToken(req.RefreshToken)
	if err != nil {
		return response.Logout(false, 401, err.Error(), &[]string{err.Error()})
	}
	session, err := a.sessionRepo.Get(refreshPayload.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.Logout(false, 404, "session not found", &[]string{err.Error()})
		}
		return response.Logout(false, 500, err.Error(), &[]string{err.Error()})
	}
	if refreshPayload.Username != session.Username {
		return response.Logout(false, 403, "session does not belong to this user", nil)
	}
	if session.IsBlocked {
		return response.Logout(true, 409, "session already invalidated", nil)
	}

	if err := a.sessionRepo.BlockSession(session.ID); err != nil {
		return response.Logout(false, 500, err.Error(), &[]string{err.Error()})
	}

	cacheKey := util.GenerateCacheKey("user", refreshPayload.Username)
	if err := a.cache.Delete(ctx, cacheKey); err != nil {
		log.Printf("[Logout] failed to delete cache for %s: %v", refreshPayload.Username, err)
	}
	return response.Logout(true, 200, "logged out successfully", nil)
}

// Login implements AuthenticationService.
func (a *authService) Login(ctx *gin.Context, cred dto.LoginModel) *response.LoginResult {
	user, err := a.userRepo.GetByEmailAndRole(cred.Email, cred.Role)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Login(false, 404, "user not found", nil, &[]string{err.Error()})
		}
		return response.Login(false, 500, err.Error(), nil, &[]string{err.Error()})
	}

	if err := util.ComparePassword(cred.Password, user.HashedPassword); err != nil {
		return response.Login(false, 400, "invalid credentials", nil, &[]string{err.Error()})
	}

	token, err := util.IssueSessionAndTokens(ctx, user, a.config, a.tokenService, a.sessionRepo)
	if err != nil {
		return response.Login(false, 500, err.Error(), nil, &[]string{err.Error()})
	}
	return response.Login(true, 200, "login successful", token, nil)
}

// Register implements AuthenticationService.
func (a *authService) Register(ctx *gin.Context, req dto.RegisterUserRequest) *response.User {
	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		return response.NewUser(false, 500, "failed to hash password", nil, &[]string{err.Error()})
	}

	user := &domain.User{
		Username:       util.RandomUsername(),
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		Email:          req.Email,
		ImageURL:       req.ImageURL,
		HashedPassword: hashedPassword,
		Role:           domain.USER_ROLE,
	}

	createdUser, err := a.userRepo.Create(user)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return response.NewUser(false, 409, pgErr.Detail, nil, &[]string{pgErr.Detail})
		}

		return response.NewUser(false, 500, err.Error(), nil, &[]string{err.Error()})
	}

	cacheKey := util.GenerateCacheKey("user", createdUser.Username)
	userSerialized, err := util.Serialize(createdUser)
	if err != nil {
		return response.NewUser(false, 500, err.Error(), nil, &[]string{err.Error()})
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
			return response.NewUser(false, 500, err.Error(), nil, &[]string{err.Error()})
		}
	}

	return response.NewUser(true, 201, "user registered successfully", createdUser, nil)
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

func (a *authService) cacheUser(ctx *gin.Context, user *domain.User) error {
	cacheKey := util.GenerateCacheKey("user", user.Username)

	data, err := util.Serialize(user)
	if err != nil {
		return err
	}

	if err := a.cache.Set(ctx, cacheKey, data, a.config.Redis.TTL); err != nil {
		return err
	}

	return a.cache.DeleteByPrefix(ctx, "users:*")
}

func (a *authService) loginSuccess(ctx *gin.Context, user *domain.User) *response.LoginResult {
	token, err := util.IssueSessionAndTokens(ctx, user, a.config, a.tokenService, a.sessionRepo)
	if err != nil {
		return response.Login(false, 500, err.Error(), nil, &[]string{err.Error()})
	}

	return response.Login(true, 200, "oauth login successful", token, nil)
}
