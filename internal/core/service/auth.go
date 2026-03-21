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
func (a *authService) GoogleAuthMobile(ctx *gin.Context, claims *google.Payload) *response.LoginResult {
	account, err := a.oauthAccountRepo.GetByProvider("google", claims.Subject)
	if err == nil {
		user, err := a.userRepo.Get(account.Username)
		if err != nil {
			return response.Login(false, 500, "failed to get user", false, nil, &[]string{err.Error()})
		}
		return a.loginSuccess(ctx, user)
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return response.Login(false, 500, "failed to get oauth account", false, nil, &[]string{err.Error()})
	}

	user, err := a.userRepo.GetByEmailUnscoped(claims.Email)
	if err == nil {
		if user.DeletedAt.Valid {
			user.DeletedAt = gorm.DeletedAt{}
			if _, err := a.userRepo.Update(user); err != nil {
				return response.Login(false, 500, "failed to restore user", false, nil, &[]string{err.Error()})
			}
		}
		if err := a.linkGoogleAccount(user.Username, claims); err != nil {
			return response.Login(false, 500, "failed to link google account", false, nil, &[]string{err.Error()})
		}
		if err := a.cacheUser(ctx, user); err != nil {
			log.Printf("[GoogleAuthMobile] failed to cache user %s: %v", user.Username, err)
		}
		return a.loginSuccess(ctx, user)
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return response.Login(false, 500, "failed to lookup user", false, nil, &[]string{err.Error()})
	}

	newUser := &domain.User{
		Username:  util.RandomUsername(),
		FirstName: claims.FirstName,
		LastName:  claims.LastName,
		Email:     claims.Email,
		ImageURL:  claims.AvatarURL,
		Role:      domain.USER_ROLE,
	}
	created, err := a.userRepo.Create(newUser)
	if err != nil {
		return response.Login(false, 500, "failed to create user", false, nil, &[]string{err.Error()})
	}

	if err := a.linkGoogleAccount(created.Username, claims); err != nil {
		return response.Login(false, 500, "failed to link google account", false, nil, &[]string{err.Error()})
	}

	if err := a.cacheUser(ctx, created); err != nil {
		return response.Login(false, 500, err.Error(), false, nil, &[]string{err.Error()})
	}

	result := a.loginSuccess(ctx, created)
	result.NewUser = true
	return result
}

// OAuthLogin implements [port.AuthenticationService].
func (a *authService) OAuthLogin(ctx *gin.Context, provider string, gUser goth.User) *response.LoginResult {
	account, err := a.oauthAccountRepo.GetByProvider(provider, gUser.UserID)
	if err == nil {
		user, err := a.userRepo.Get(account.Username)
		if err != nil {
			return response.Login(false, 500, "failed to get user", false, nil, &[]string{err.Error()})
		}
		return a.loginSuccess(ctx, user)
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return response.Login(false, 500, "failed to get oauth account", false, nil, &[]string{err.Error()})
	}

	user, err := a.userRepo.GetByEmailUnscoped(gUser.Email)
	if err == nil {
		if user.DeletedAt.Valid {
			user.DeletedAt = gorm.DeletedAt{}
			if _, err := a.userRepo.Update(user); err != nil {
				return response.Login(false, 500, "failed to restore user", false, nil, &[]string{err.Error()})
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
			return response.Login(false, 500, "failed to create user", false, nil, &[]string{err.Error()})
		}
	} else {
		return response.Login(false, 500, "failed to lookup user", false, nil, &[]string{err.Error()})
	}

	_, err = a.oauthAccountRepo.Create(&domain.OauthAccount{
		ID:             uuid.New(),
		Username:       user.Username,
		Provider:       provider,
		ProviderUserID: gUser.UserID,
		Email:          gUser.Email,
	})
	if err != nil {
		return response.Login(false, 500, "failed to create oauth account", false, nil, &[]string{err.Error()})
	}

	if err := a.cacheUser(ctx, user); err != nil {
		log.Printf("[OAuthLogin] failed to cache user %s: %v", user.Username, err)
	}

	return a.loginSuccess(ctx, user)
}

// ReNewAccessToken implements [port.AuthenticationService].
func (a *authService) ReNewAccessToken(ctx *gin.Context, req dto.ReNewAccessTokenRequest) *response.RefreshTokenResult {
	refreshPayload, err := a.tokenService.VerifyToken(req.RefreshToken)
	if err != nil {
		return response.RefreshToken(false, 401, err.Error(), nil, &[]string{err.Error()})
	}

	session, err := a.sessionRepo.Get(refreshPayload.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.RefreshToken(false, 404, "session not found", nil, &[]string{err.Error()})
		}
		return response.RefreshToken(false, 500, err.Error(), nil, &[]string{err.Error()})
	}

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

	accessToken, accessPayload, err := a.tokenService.GenerateToken(
		uuid.Nil,
		refreshPayload.Username,
		refreshPayload.Role,
		a.config.Auth.TokenDuration,
	)
	if err != nil {
		return response.RefreshToken(false, 500, fmt.Sprintf("could not generate new access token: %s", err.Error()), nil, &[]string{err.Error()})
	}

	RefreshToken, newRefreshPayload, err := a.tokenService.GenerateToken(
		refreshPayload.ID,
		refreshPayload.Username,
		refreshPayload.Role,
		a.config.Auth.RefreshTokenDuration,
	)
	if err != nil {
		return response.RefreshToken(false, 500, fmt.Sprintf("could not generate new refresh token: %s", err.Error()), nil, &[]string{err.Error()})
	}

	session.RefreshToken = RefreshToken
	session.ExpiresAt = newRefreshPayload.ExpiresAt

	if _, err := a.sessionRepo.Update(session); err != nil {
		return response.RefreshToken(false, 500, err.Error(), nil, &[]string{err.Error()})
	}

	return response.RefreshToken(true, 200, "refresh token renewed successfully", &dto.ReNewAccessTokenResponse{
		AccessToken:           accessToken,
		AccessTokenExpriresAt: accessPayload.ExpiresAt,
		RefreshToken:          RefreshToken,
		RefreshTokenExpiresAt: newRefreshPayload.ExpiresAt,
	}, nil)
}

// Logout implements [port.AuthenticationService].
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

// Login implements [port.AuthenticationService].
func (a *authService) Login(ctx *gin.Context, cred dto.LoginModel) *response.LoginResult {
	user, err := a.userRepo.GetByEmailAndRole(cred.Email, cred.Role)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.Login(false, 404, "user not found", false, nil, &[]string{err.Error()})
		}
		return response.Login(false, 500, err.Error(), false, nil, &[]string{err.Error()})
	}

	if err := util.ComparePassword(cred.Password, user.HashedPassword); err != nil {
		return response.Login(false, 400, "invalid credentials", false, nil, &[]string{err.Error()})
	}

	token, err := util.IssueSessionAndTokens(ctx, user, a.config, a.tokenService, a.sessionRepo)
	if err != nil {
		return response.Login(false, 500, err.Error(), false, nil, &[]string{err.Error()})
	}
	return response.Login(true, 200, "login successful", false, token, nil)
}

// Register implements [port.AuthenticationService].
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
		log.Printf("[Register] failed to serialize user %s: %v", createdUser.Username, err)
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
			log.Printf("[Register] cache operation failed for user %s: %v", createdUser.Username, err)
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
		return response.Login(false, 500, err.Error(), false, nil, &[]string{err.Error()})
	}

	return response.Login(true, 200, "oauth login successful", false, token, nil)
}

func (a *authService) linkGoogleAccount(username string, payload *google.Payload) error {
	_, err := a.oauthAccountRepo.Create(&domain.OauthAccount{
		ID:             uuid.New(),
		Username:       username,
		Provider:       "google",
		ProviderUserID: payload.Subject,
		Email:          payload.Email,
	})
	return err
}
