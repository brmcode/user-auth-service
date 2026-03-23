package service

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/brmcode/user-auth-service/internal/adapter/auth/google"
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
	roleRepo         port.RoleRepository
	sessionRepo      port.SessionRepository
	oauthAccountRepo port.OauthAccountRepository
	tokenService     port.TokenService
	cache            port.CacheRepository
}

func (a *authService) defaultUserRole() ([]domain.Role, error) {
	r, err := a.roleRepo.GetByCode(domain.USER_ROLE)
	if err != nil {
		return nil, fmt.Errorf("could not load default role: %w", err)
	}
	return []domain.Role{*r}, nil
}

func (a *authService) cacheUser(ctx *gin.Context, user *domain.User) error {
	key := util.GenerateCacheKey("user", user.Username)
	data, err := util.Serialize(user)
	if err != nil {
		return err
	}
	if err := a.cache.Set(ctx, key, data, a.config.Redis.TTL); err != nil {
		return err
	}
	return a.cache.DeleteByPrefix(ctx, "users:*")
}

func (a *authService) loginSuccess(ctx *gin.Context, user *domain.User) *response.LoginResult {
	token, err := util.IssueSessionAndTokens(ctx, user, a.config, a.tokenService, a.sessionRepo)
	if err != nil {
		return response.Login(false, 500, err.Error(), false, nil, &[]string{err.Error()})
	}
	return response.Login(true, 200, "login successful", false, token, nil)
}

func (a *authService) linkOAuthAccount(username, provider, providerUserID, email string) error {
	_, err := a.oauthAccountRepo.Create(&domain.OauthAccount{
		ID:             uuid.New(),
		Username:       username,
		Provider:       provider,
		ProviderUserID: providerUserID,
		Email:          email,
	})
	return err
}

func roleCodesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	ac, bc := make([]string, len(a)), make([]string, len(b))
	copy(ac, a)
	copy(bc, b)
	sort.Strings(ac)
	sort.Strings(bc)
	for i := range ac {
		if ac[i] != bc[i] {
			return false
		}
	}
	return true
}

func (a *authService) Login(ctx *gin.Context, cred dto.LoginModel) *response.LoginResult {
	var (
		user *domain.User
		err  error
	)

	if cred.Role != "" {
		user, err = a.userRepo.GetByEmailAndRole(cred.Email, cred.Role)
	} else {
		user, err = a.userRepo.GetByEmail(cred.Email)
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
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

func (a *authService) Register(ctx *gin.Context, req dto.RegisterUserRequest) *response.User {
	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		return response.NewUser(false, 500, "failed to hash password", nil, &[]string{err.Error()})
	}

	defaultRoles, err := a.defaultUserRole()
	if err != nil {
		return response.NewUser(false, 500, err.Error(), nil, &[]string{err.Error()})
	}

	user := &domain.User{
		Username:       util.RandomUsername(),
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		Email:          req.Email,
		ImageURL:       req.ImageURL,
		HashedPassword: hashedPassword,
		Roles:          defaultRoles,
	}

	createdUser, err := a.userRepo.Create(user)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return response.NewUser(false, 409, pgErr.Detail, nil, &[]string{pgErr.Detail})
		}
		return response.NewUser(false, 500, err.Error(), nil, &[]string{err.Error()})
	}

	key := util.GenerateCacheKey("user", createdUser.Username)
	if serialized, serErr := util.Serialize(createdUser); serErr == nil {
		errChan := make(chan error, 2)
		go func() { errChan <- a.cache.Set(ctx, key, serialized, a.config.Redis.TTL) }()
		go func() { errChan <- a.cache.DeleteByPrefix(ctx, "users:*") }()
		for i := 0; i < 2; i++ {
			if cacheErr := <-errChan; cacheErr != nil {
				log.Printf("[Register] cache error for %s: %v", createdUser.Username, cacheErr)
			}
		}
	}

	return response.NewUser(true, 201, "user registered successfully", createdUser, nil)
}

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
			return response.RefreshToken(false, 500, fmt.Sprintf("failed to block sessions: %s", err), nil, &[]string{err.Error()})
		}
		return response.RefreshToken(false, 401, "refresh token reuse detected: all sessions blocked", nil, nil)
	}
	if time.Now().After(session.ExpiresAt) {
		return response.RefreshToken(false, 401, "expired session", nil, nil)
	}

	userOfSession, err := a.userRepo.Get(session.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.RefreshToken(false, 404, "invalid session: user data missing", nil, &[]string{err.Error()})
		}
		return response.RefreshToken(false, 500, err.Error(), nil, &[]string{err.Error()})
	}

	if !roleCodesEqual(userOfSession.RoleCodes(), refreshPayload.Roles) {
		if err := a.sessionRepo.BlockAllSessions(session.Username); err != nil {
			return response.RefreshToken(false, 500, fmt.Sprintf("failed to block sessions: %s", err), nil, &[]string{err.Error()})
		}
		return response.RefreshToken(false, 401, "user roles changed: please log in again", nil, nil)
	}

	accessToken, accessPayload, err := a.tokenService.GenerateToken(
		uuid.Nil, refreshPayload.Username, refreshPayload.Roles, a.config.Auth.TokenDuration,
	)
	if err != nil {
		return response.RefreshToken(false, 500, fmt.Sprintf("could not generate access token: %s", err), nil, &[]string{err.Error()})
	}

	newRefreshToken, newRefreshPayload, err := a.tokenService.GenerateToken(
		refreshPayload.ID, refreshPayload.Username, refreshPayload.Roles, a.config.Auth.RefreshTokenDuration,
	)
	if err != nil {
		return response.RefreshToken(false, 500, fmt.Sprintf("could not generate refresh token: %s", err), nil, &[]string{err.Error()})
	}

	session.RefreshToken = newRefreshToken
	session.ExpiresAt = newRefreshPayload.ExpiresAt
	if _, err := a.sessionRepo.Update(session); err != nil {
		return response.RefreshToken(false, 500, err.Error(), nil, &[]string{err.Error()})
	}

	return response.RefreshToken(true, 200, "token renewed successfully", &dto.ReNewAccessTokenResponse{
		AccessToken:           accessToken,
		AccessTokenExpriresAt: accessPayload.ExpiresAt,
		RefreshToken:          newRefreshToken,
		RefreshTokenExpiresAt: newRefreshPayload.ExpiresAt,
	}, nil)
}

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

	key := util.GenerateCacheKey("user", refreshPayload.Username)
	if err := a.cache.Delete(ctx, key); err != nil {
		log.Printf("[Logout] cache delete failed for %s: %v", refreshPayload.Username, err)
	}
	return response.Logout(true, 200, "logged out successfully", nil)
}

func (a *authService) OAuthLogin(ctx *gin.Context, provider string, gUser goth.User) *response.LoginResult {
	// Case 1: existing OAuth account.
	account, err := a.oauthAccountRepo.GetByProvider(provider, gUser.UserID)
	if err == nil {
		user, err := a.userRepo.Get(account.Username)
		if err != nil {
			return response.Login(false, 500, "failed to get user", false, nil, &[]string{err.Error()})
		}
		return a.loginSuccess(ctx, user)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return response.Login(false, 500, "failed to look up oauth account", false, nil, &[]string{err.Error()})
	}

	// Case 2: email matches an existing user → link the new OAuth account.
	user, err := a.userRepo.GetByEmailUnscoped(gUser.Email)
	if err == nil {
		if user.DeletedAt.Valid {
			user.DeletedAt = gorm.DeletedAt{}
			if _, err := a.userRepo.Update(user); err != nil {
				return response.Login(false, 500, "failed to restore user", false, nil, &[]string{err.Error()})
			}
		}
		if err := a.linkOAuthAccount(user.Username, provider, gUser.UserID, gUser.Email); err != nil {
			return response.Login(false, 500, "failed to link oauth account", false, nil, &[]string{err.Error()})
		}
		if err := a.cacheUser(ctx, user); err != nil {
			log.Printf("[OAuthLogin] cache error for %s: %v", user.Username, err)
		}
		return a.loginSuccess(ctx, user)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return response.Login(false, 500, "failed to look up user", false, nil, &[]string{err.Error()})
	}

	// Case 3: brand new user.
	defaultRoles, err := a.defaultUserRole()
	if err != nil {
		return response.Login(false, 500, err.Error(), false, nil, &[]string{err.Error()})
	}

	newUser := &domain.User{
		Username:  util.RandomUsername(),
		FirstName: gUser.FirstName,
		LastName:  gUser.LastName,
		Email:     gUser.Email,
		ImageURL:  gUser.AvatarURL,
		Roles:     defaultRoles,
	}
	created, err := a.userRepo.Create(newUser)
	if err != nil {
		return response.Login(false, 500, "failed to create user", false, nil, &[]string{err.Error()})
	}
	if err := a.linkOAuthAccount(created.Username, provider, gUser.UserID, gUser.Email); err != nil {
		return response.Login(false, 500, "failed to link oauth account", false, nil, &[]string{err.Error()})
	}
	if err := a.cacheUser(ctx, created); err != nil {
		log.Printf("[OAuthLogin] cache error for %s: %v", created.Username, err)
	}

	result := a.loginSuccess(ctx, created)
	result.NewUser = true
	return result
}

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
		return response.Login(false, 500, "failed to look up oauth account", false, nil, &[]string{err.Error()})
	}

	user, err := a.userRepo.GetByEmailUnscoped(claims.Email)
	if err == nil {
		if user.DeletedAt.Valid {
			user.DeletedAt = gorm.DeletedAt{}
			if _, err := a.userRepo.Update(user); err != nil {
				return response.Login(false, 500, "failed to restore user", false, nil, &[]string{err.Error()})
			}
		}
		if err := a.linkOAuthAccount(user.Username, "google", claims.Subject, claims.Email); err != nil {
			return response.Login(false, 500, "failed to link google account", false, nil, &[]string{err.Error()})
		}
		if err := a.cacheUser(ctx, user); err != nil {
			log.Printf("[GoogleAuthMobile] cache error for %s: %v", user.Username, err)
		}
		return a.loginSuccess(ctx, user)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return response.Login(false, 500, "failed to look up user", false, nil, &[]string{err.Error()})
	}

	defaultRoles, err := a.defaultUserRole()
	if err != nil {
		return response.Login(false, 500, err.Error(), false, nil, &[]string{err.Error()})
	}

	newUser := &domain.User{
		Username:  util.RandomUsername(),
		FirstName: claims.FirstName,
		LastName:  claims.LastName,
		Email:     claims.Email,
		ImageURL:  claims.AvatarURL,
		Roles:     defaultRoles,
	}
	created, err := a.userRepo.Create(newUser)
	if err != nil {
		return response.Login(false, 500, "failed to create user", false, nil, &[]string{err.Error()})
	}
	if err := a.linkOAuthAccount(created.Username, "google", claims.Subject, claims.Email); err != nil {
		return response.Login(false, 500, "failed to link google account", false, nil, &[]string{err.Error()})
	}
	if err := a.cacheUser(ctx, created); err != nil {
		return response.Login(false, 500, err.Error(), false, nil, &[]string{err.Error()})
	}

	result := a.loginSuccess(ctx, created)
	result.NewUser = true
	return result
}

func NewAuthenticationService(
	cfg *config.Configuration,
	userRepo port.UserRepository,
	roleRepo port.RoleRepository,
	sessionRepo port.SessionRepository,
	oauthAccountRepo port.OauthAccountRepository,
	tokenService port.TokenService,
	cache port.CacheRepository,
) port.AuthenticationService {
	return &authService{
		config:           cfg,
		userRepo:         userRepo,
		roleRepo:         roleRepo,
		sessionRepo:      sessionRepo,
		oauthAccountRepo: oauthAccountRepo,
		tokenService:     tokenService,
		cache:            cache,
	}
}
