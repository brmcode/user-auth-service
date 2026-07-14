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
	"github.com/brmcode/user-auth-service/pkg/i18n"
	"github.com/brmcode/user-auth-service/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/markbates/goth"
	"gorm.io/gorm"
)

type authService struct {
	config           *config.Configuration
	uow              port.UnitOfWork
	userRepo         port.UserRepository
	roleRepo         port.RoleRepository
	sessionRepo      port.SessionRepository
	oauthAccountRepo port.OauthAccountRepository
	tokenService     port.TokenService
	cache            port.CacheRepository
}

func (a *authService) defaultUserRole() ([]domain.Role, error) {
	return a.defaultUserRoleWithRepo(a.roleRepo)
}

func (a *authService) defaultUserRoleWithRepo(roleRepo port.RoleRepository) ([]domain.Role, error) {
	r, err := roleRepo.GetByCode(domain.USER_ROLE)
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
	result, _ := a.loginSuccessWithSessionRepo(ctx, user, a.sessionRepo)
	return result
}

func (a *authService) loginSuccessWithSessionRepo(ctx *gin.Context, user *domain.User, sessionRepo port.SessionRepository) (*response.LoginResult, error) {
	token, err := util.IssueSessionAndTokens(ctx, user, a.config, a.tokenService, sessionRepo)
	if err != nil {
		return response.Login(false, 500, i18n.Translate("common.internal_error"), false, nil), err
	}
	return response.Login(true, 200, i18n.Translate("auth.login.success"), false, token), nil
}

func (a *authService) linkOAuthAccount(username, provider, providerUserID, email string) error {
	return a.linkOAuthAccountWithRepo(a.oauthAccountRepo, username, provider, providerUserID, email)
}

func (a *authService) linkOAuthAccountWithRepo(oauthRepo port.OauthAccountRepository, username, provider, providerUserID, email string) error {
	_, err := oauthRepo.Create(&domain.OauthAccount{
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
			return response.Login(false, 404, i18n.Translate("auth.login.user_not_found"), false, nil)
		}
		return response.Login(false, 500, i18n.Translate("common.internal_error"), false, nil)
	}

	if err := util.ComparePassword(cred.Password, user.HashedPassword); err != nil {
		return response.Login(false, 400, i18n.Translate("auth.login.invalid_credentials"), false, nil)
	}

	token, err := util.IssueSessionAndTokens(ctx, user, a.config, a.tokenService, a.sessionRepo)
	if err != nil {
		return response.Login(false, 500, i18n.Translate("common.internal_error"), false, nil)
	}
	return response.Login(true, 200, i18n.Translate("auth.login.success"), false, token)
}

func (a *authService) Register(ctx *gin.Context, req dto.RegisterUserRequest) *response.UserResult {
	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		return response.User(false, 500, i18n.Translate("auth.register.password_hash_failed"), nil)
	}

	var createdUser *domain.User

	err = a.uow.Do(func(uow port.UnitOfWork) error {
		defaultRoles, err := a.defaultUserRoleWithRepo(uow.RoleRepo())
		if err != nil {
			return err
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

		createdUser, err = uow.UserRepo().Create(user)
		return err
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return response.User(false, 409, i18n.Translate("auth.register.conflict"), nil)
		}
		return response.User(false, 500, i18n.Translate("common.internal_error"), nil)
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

	return response.User(true, 201, i18n.Translate("auth.register.success"), createdUser)
}

func (a *authService) ReNewAccessToken(ctx *gin.Context, req dto.ReNewAccessTokenRequest) *response.RefreshTokenResult {
	refreshPayload, err := a.tokenService.VerifyToken(req.RefreshToken)
	if err != nil {
		return response.RefreshToken(false, 401, i18n.Translate("common.internal_error"), nil)
	}

	session, err := a.sessionRepo.Get(refreshPayload.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.RefreshToken(false, 404, i18n.Translate("session.not_found"), nil)
		}
		return response.RefreshToken(false, 500, i18n.Translate("common.internal_error"), nil)
	}

	if session.IsBlocked {
		return response.RefreshToken(false, 401, i18n.Translate("session.blocked"), nil)
	}
	if session.Username != refreshPayload.Username {
		return response.RefreshToken(false, 401, i18n.Translate("session.incorrect_user"), nil)
	}
	if session.RefreshToken != req.RefreshToken {
		if err := a.sessionRepo.BlockAllSessions(session.Username); err != nil {
			return response.RefreshToken(false, 500, fmt.Sprintf(i18n.Translate("session.failed_to_block"), err), nil)
		}
		return response.RefreshToken(false, 401, i18n.Translate("session.refresh_reuse"), nil)
	}
	if time.Now().After(session.ExpiresAt) {
		return response.RefreshToken(false, 401, i18n.Translate("session.expired"), nil)
	}

	userOfSession, err := a.userRepo.Get(session.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.RefreshToken(false, 404, i18n.Translate("session.invalid_user_data"), nil)
		}
		return response.RefreshToken(false, 500, i18n.Translate("common.internal_error"), nil)
	}

	if !roleCodesEqual(userOfSession.RoleCodes(), refreshPayload.Roles) {
		if err := a.sessionRepo.BlockAllSessions(session.Username); err != nil {
			return response.RefreshToken(false, 500, fmt.Sprintf(i18n.Translate("session.failed_to_block"), err), nil)
		}
		return response.RefreshToken(false, 401, i18n.Translate("session.user_roles_changed"), nil)
	}

	accessToken, accessPayload, err := a.tokenService.GenerateToken(
		uuid.Nil, refreshPayload.Username, refreshPayload.Roles, a.config.Auth.TokenDuration,
	)
	if err != nil {
		return response.RefreshToken(false, 500, fmt.Sprintf(i18n.Translate("token.access_generation_failed"), err), nil)
	}

	newRefreshToken, newRefreshPayload, err := a.tokenService.GenerateToken(
		refreshPayload.ID, refreshPayload.Username, refreshPayload.Roles, a.config.Auth.RefreshTokenDuration,
	)
	if err != nil {
		return response.RefreshToken(false, 500, fmt.Sprintf(i18n.Translate("token.refresh_generation_failed"), err), nil)
	}

	session.RefreshToken = newRefreshToken
	session.ExpiresAt = newRefreshPayload.ExpiresAt
	if _, err := a.sessionRepo.Update(session); err != nil {
		return response.RefreshToken(false, 500, i18n.Translate("common.internal_error"), nil)
	}

	return response.RefreshToken(true, 200, i18n.Translate("token.renewed"), &dto.ReNewAccessTokenResponse{
		AccessToken:           accessToken,
		AccessTokenExpriresAt: accessPayload.ExpiresAt,
		RefreshToken:          newRefreshToken,
		RefreshTokenExpiresAt: newRefreshPayload.ExpiresAt,
	})
}

func (a *authService) Logout(ctx *gin.Context, req dto.ReNewAccessTokenRequest) *response.LogoutResult {
	refreshPayload, err := a.tokenService.VerifyToken(req.RefreshToken)
	if err != nil {
		return response.Logout(false, 401, i18n.Translate("common.internal_error"), &[]string{err.Error()})
	}

	session, err := a.sessionRepo.Get(refreshPayload.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.Logout(false, 404, i18n.Translate("session.not_found"), &[]string{err.Error()})
		}
		return response.Logout(false, 500, i18n.Translate("common.internal_error"), &[]string{err.Error()})
	}
	if refreshPayload.Username != session.Username {
		return response.Logout(false, 403, i18n.Translate("auth.logout.session_not_belong"), nil)
	}
	if session.IsBlocked {
		return response.Logout(true, 409, i18n.Translate("auth.logout.session_already_invalid"), nil)
	}

	if err := a.sessionRepo.BlockSession(session.ID); err != nil {
		return response.Logout(false, 500, i18n.Translate("common.internal_error"), &[]string{err.Error()})
	}

	key := util.GenerateCacheKey("user", refreshPayload.Username)
	if err := a.cache.Delete(ctx, key); err != nil {
		log.Printf("[Logout] cache delete failed for %s: %v", refreshPayload.Username, err)
	}
	return response.Logout(true, 200, i18n.Translate("auth.logout.success"), nil)
}

func (a *authService) OAuthLogin(ctx *gin.Context, provider string, gUser goth.User) *response.LoginResult {
	var (
		authUser    *domain.User
		isNew       bool
		shouldCache bool
		result      *response.LoginResult
	)

	err := a.uow.Do(func(uow port.UnitOfWork) error {
		account, err := uow.OauthAccountRepo().GetByProvider(provider, gUser.UserID)
		if err == nil {
			authUser, err = uow.UserRepo().Get(account.Username)
			if err != nil {
				result = response.Login(false, 500, i18n.Translate("oauth.failed_get_user"), false, nil)
				return err
			}
			result, err = a.loginSuccessWithSessionRepo(ctx, authUser, uow.SessionRepo())
			return err
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			result = response.Login(false, 500, i18n.Translate("oauth.failed_lookup_account"), false, nil)
			return err
		}

		authUser, err = uow.UserRepo().GetByEmailUnscoped(gUser.Email)
		if err == nil {
			if authUser.DeletedAt.Valid {
				authUser.DeletedAt = gorm.DeletedAt{}
				if _, err := uow.UserRepo().Update(authUser); err != nil {
					result = response.Login(false, 500, i18n.Translate("oauth.failed_restore_user"), false, nil)
					return err
				}
			}
			if err := a.linkOAuthAccountWithRepo(uow.OauthAccountRepo(), authUser.Username, provider, gUser.UserID, gUser.Email); err != nil {
				result = response.Login(false, 500, i18n.Translate("oauth.failed_link_oauth_account"), false, nil)
				return err
			}
			shouldCache = true
			result, err = a.loginSuccessWithSessionRepo(ctx, authUser, uow.SessionRepo())
			return err
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			result = response.Login(false, 500, i18n.Translate("oauth.failed_lookup_user"), false, nil)
			return err
		}

		defaultRoles, err := a.defaultUserRoleWithRepo(uow.RoleRepo())
		if err != nil {
			result = response.Login(false, 500, i18n.Translate("common.internal_error"), false, nil)
			return err
		}

		newUser := &domain.User{
			Username:  util.RandomUsername(),
			FirstName: gUser.FirstName,
			LastName:  gUser.LastName,
			Email:     gUser.Email,
			ImageURL:  gUser.AvatarURL,
			Roles:     defaultRoles,
		}
		authUser, err = uow.UserRepo().Create(newUser)
		if err != nil {
			result = response.Login(false, 500, i18n.Translate("oauth.failed_create_user"), false, nil)
			return err
		}
		if err := a.linkOAuthAccountWithRepo(uow.OauthAccountRepo(), authUser.Username, provider, gUser.UserID, gUser.Email); err != nil {
			result = response.Login(false, 500, i18n.Translate("oauth.failed_link_oauth_account"), false, nil)
			return err
		}

		isNew = true
		shouldCache = true
		result, err = a.loginSuccessWithSessionRepo(ctx, authUser, uow.SessionRepo())
		return err
	})
	if err != nil {
		if result != nil {
			return result
		}
		return response.Login(false, 500, i18n.Translate("oauth.login_failed"), false, nil)
	}

	if shouldCache {
		if err := a.cacheUser(ctx, authUser); err != nil {
			log.Printf("[OAuthLogin] cache error for %s: %v", authUser.Username, err)
		}
	}

	result.NewUser = isNew
	return result
}

func (a *authService) GoogleAuthMobile(ctx *gin.Context, payload *google.Payload) *response.LoginResult {
	var (
		authUser    *domain.User
		isNew       bool
		shouldCache bool
		result      *response.LoginResult
	)

	err := a.uow.Do(func(uow port.UnitOfWork) error {
		account, err := uow.OauthAccountRepo().GetByProvider("google", payload.Subject)
		if err == nil {
			authUser, err = uow.UserRepo().Get(account.Username)
			if err != nil {
				result = response.Login(false, 500, i18n.Translate("oauth.failed_get_user"), false, nil)
				return err
			}
			result, err = a.loginSuccessWithSessionRepo(ctx, authUser, uow.SessionRepo())
			return err
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			result = response.Login(false, 500, i18n.Translate("oauth.failed_lookup_account"), false, nil)
			return err
		}

		authUser, err = uow.UserRepo().GetByEmailUnscoped(payload.Email)
		if err == nil {
			if authUser.DeletedAt.Valid {
				authUser.DeletedAt = gorm.DeletedAt{}
				if _, err := uow.UserRepo().Update(authUser); err != nil {
					result = response.Login(false, 500, i18n.Translate("oauth.failed_restore_user"), false, nil)
					return err
				}
			}
			if err := a.linkOAuthAccountWithRepo(uow.OauthAccountRepo(), authUser.Username, "google", payload.Subject, payload.Email); err != nil {
				result = response.Login(false, 500, i18n.Translate("oauth.failed_link_google_account"), false, nil)
				return err
			}
			shouldCache = true
			result, err = a.loginSuccessWithSessionRepo(ctx, authUser, uow.SessionRepo())
			return err
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			result = response.Login(false, 500, i18n.Translate("oauth.failed_lookup_user"), false, nil)
			return err
		}

		defaultRoles, err := a.defaultUserRoleWithRepo(uow.RoleRepo())
		if err != nil {
			result = response.Login(false, 500, i18n.Translate("common.internal_error"), false, nil)
			return err
		}

		newUser := &domain.User{
			Username:  util.RandomUsername(),
			FirstName: payload.FirstName,
			LastName:  payload.LastName,
			Email:     payload.Email,
			ImageURL:  payload.AvatarURL,
			Roles:     defaultRoles,
		}
		authUser, err = uow.UserRepo().Create(newUser)
		if err != nil {
			result = response.Login(false, 500, i18n.Translate("oauth.failed_create_user"), false, nil)
			return err
		}
		if err := a.linkOAuthAccountWithRepo(uow.OauthAccountRepo(), authUser.Username, "google", payload.Subject, payload.Email); err != nil {
			result = response.Login(false, 500, i18n.Translate("oauth.failed_link_google_account"), false, nil)
			return err
		}

		isNew = true
		shouldCache = true
		result, err = a.loginSuccessWithSessionRepo(ctx, authUser, uow.SessionRepo())
		return err
	})
	if err != nil {
		if result != nil {
			return result
		}
		return response.Login(false, 500, i18n.Translate("oauth.google_auth_login_failed"), false, nil)
	}

	if shouldCache {
		if err := a.cacheUser(ctx, authUser); err != nil {
			log.Printf("[GoogleAuthMobile] cache error for %s: %v", authUser.Username, err)
		}
	}

	result.NewUser = isNew
	return result
}

func NewAuthenticationService(
	cfg *config.Configuration,
	uow port.UnitOfWork,
	userRepo port.UserRepository,
	roleRepo port.RoleRepository,
	sessionRepo port.SessionRepository,
	oauthAccountRepo port.OauthAccountRepository,
	tokenService port.TokenService,
	cache port.CacheRepository,
) port.AuthenticationService {
	return &authService{
		config:           cfg,
		uow:              uow,
		userRepo:         userRepo,
		roleRepo:         roleRepo,
		sessionRepo:      sessionRepo,
		oauthAccountRepo: oauthAccountRepo,
		tokenService:     tokenService,
		cache:            cache,
	}
}
