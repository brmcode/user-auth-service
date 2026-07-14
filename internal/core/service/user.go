package service

import (
	"errors"
	"log"
	"net/http"

	"github.com/brmcode/user-auth-service/internal/adapter/http/handler/dto/request"
	"github.com/brmcode/user-auth-service/internal/adapter/http/handler/dto/response"
	"github.com/brmcode/user-auth-service/internal/core/domain"
	"github.com/brmcode/user-auth-service/internal/core/port"
	"github.com/brmcode/user-auth-service/pkg/config"
	"github.com/brmcode/user-auth-service/pkg/i18n"
	"github.com/brmcode/user-auth-service/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type userServ struct {
	userRepo port.UserRepository
	roleRepo port.RoleRepository
	uow      port.UnitOfWork
	cache    port.CacheRepository
	config   *config.Configuration
}

func (u *userServ) resolveRoles(codes []string) ([]domain.Role, *response.UserResult) {
	return u.resolveRolesWithRepo(u.roleRepo, codes)
}

func (u *userServ) resolveRolesWithRepo(roleRepo port.RoleRepository, codes []string) ([]domain.Role, *response.UserResult) {
	roles, err := roleRepo.GetByCodes(codes)
	if err != nil {
		return nil, response.User(false, http.StatusInternalServerError, i18n.Translate("user.roles.load_failed"), nil)
	}
	if len(roles) != len(codes) {
		return nil, response.User(false, http.StatusBadRequest, i18n.Translate("user.roles.invalid_codes"), nil)
	}
	return roles, nil
}

func (u *userServ) invalidateCache(ctx *gin.Context, username string) {
	key := util.GenerateCacheKey("user", username)
	errChan := make(chan error, 2)
	go func() { errChan <- u.cache.Delete(ctx, key) }()
	go func() { errChan <- u.cache.DeleteByPrefix(ctx, "users:*") }()
	for i := 0; i < 2; i++ {
		if err := <-errChan; err != nil {
			log.Printf("[cache] invalidate error for %s: %v", username, err)
		}
	}
}

func (u *userServ) setCache(ctx *gin.Context, user *domain.User) *response.UserResult {
	key := util.GenerateCacheKey("user", user.Username)
	serialized, err := util.Serialize(user)
	if err != nil {
		return response.User(false, http.StatusInternalServerError, i18n.Translate("user.serialize_failed"), nil)
	}
	errChan := make(chan error, 2)
	go func() { errChan <- u.cache.Set(ctx, key, serialized, u.config.Redis.TTL) }()
	go func() { errChan <- u.cache.DeleteByPrefix(ctx, "users:*") }()
	for i := 0; i < 2; i++ {
		if err := <-errChan; err != nil {
			return response.User(false, http.StatusInternalServerError, i18n.Translate("user.cache.update_failed"), nil)
		}
	}
	return nil
}

func (u *userServ) CreateUser(ctx *gin.Context, req request.CreateUserRequest) *response.UserResult {
	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		return response.User(false, http.StatusInternalServerError, i18n.Translate("auth.register.password_hash_failed"), nil)
	}

	var (
		created *domain.User
		result  *response.UserResult
	)
	abortErr := errors.New("abort create user transaction")

	err = u.uow.Do(func(uow port.UnitOfWork) error {
		roles, errResp := u.resolveRolesWithRepo(uow.RoleRepo(), req.Roles)
		if errResp != nil {
			result = errResp
			return abortErr
		}

		user := &domain.User{
			Username:       util.RandomUsername(),
			FirstName:      req.FirstName,
			LastName:       req.LastName,
			Email:          req.Email,
			ImageURL:       req.ImageURL,
			HashedPassword: hashedPassword,
			Roles:          roles,
		}

		created, err = uow.UserRepo().Create(user)
		return err
	})
	if err != nil {
		if result != nil {
			return result
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return response.User(false, http.StatusConflict, i18n.Translate("user.create.conflict"), nil)
		}
		return response.User(false, http.StatusInternalServerError, i18n.Translate("common.internal_error"), nil)
	}

	if errResp := u.setCache(ctx, created); errResp != nil {
		return errResp
	}
	return response.User(true, http.StatusCreated, i18n.Translate("user.create.success"), created)
}

func (u *userServ) GetUser(ctx *gin.Context, username string) *response.UserResult {
	key := util.GenerateCacheKey("user", username)

	if cached, err := u.cache.Get(ctx, key); err == nil {
		var user domain.User
		if err := util.Deserialize(cached, &user); err != nil {
			return response.User(false, http.StatusInternalServerError, i18n.Translate("user.deserialize_failed"), nil)
		}
		log.Println("cache hit:", key)
		// return &response.UserResult{Success: true, StatusCode: http.StatusOK, Message: "user fetched successfully", Data: &user}
		return response.User(true, http.StatusOK, i18n.Translate("user.fetch.success"), &user)
	}

	user, err := u.userRepo.Get(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.User(false, http.StatusNotFound, i18n.Translate("user.not_found"), nil)
		}
		return response.User(false, http.StatusInternalServerError, i18n.Translate("common.internal_error"), nil)
	}

	serialized, err := util.Serialize(user)
	if err != nil {
		return response.User(false, http.StatusInternalServerError, i18n.Translate("user.serialize_failed"), nil)
	}
	if err := u.cache.Set(ctx, key, serialized, u.config.Redis.TTL); err != nil {
		return response.User(false, http.StatusInternalServerError, i18n.Translate("user.cache.set_failed"), nil)
	}

	return response.User(true, http.StatusOK, i18n.Translate("user.fetch.success"), user)
}

func (u *userServ) UpdateUser(ctx *gin.Context, req request.UpdateUserRequest) *response.UserResult {
	var (
		updated *domain.User
		result  *response.UserResult
	)
	abortErr := errors.New("abort update user transaction")

	err := u.uow.Do(func(uow port.UnitOfWork) error {
		user, err := uow.UserRepo().Get(req.Username)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				result = response.User(false, http.StatusNotFound, i18n.Translate("user.not_found"), nil)
			} else {
				result = response.User(false, http.StatusInternalServerError, i18n.Translate("common.internal_error"), nil)
			}
			return err
		}

		roles, errResp := u.resolveRolesWithRepo(uow.RoleRepo(), req.Roles)
		if errResp != nil {
			result = errResp
			return abortErr
		}

		user.FirstName = req.FirstName
		user.LastName = req.LastName
		user.ImageURL = req.ImageURL
		user.Roles = roles

		updated, err = uow.UserRepo().Update(user)
		return err
	})
	if err != nil {
		if result != nil {
			return result
		}
		return response.User(false, http.StatusInternalServerError, i18n.Translate("user.update.failed"), nil)
	}

	if errResp := u.setCache(ctx, updated); errResp != nil {
		return errResp
	}
	return response.User(true, http.StatusOK, i18n.Translate("user.update.success"), updated)
}

func (u *userServ) DeleteUser(ctx *gin.Context, username string) *response.UserResult {
	user, err := u.userRepo.Get(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.User(false, http.StatusNotFound, i18n.Translate("user.not_found"), nil)
		}
		return response.User(false, http.StatusInternalServerError, i18n.Translate("common.internal_error"), nil)
	}

	u.invalidateCache(ctx, user.Username)

	if err := u.userRepo.Delete(user); err != nil {
		return response.User(false, http.StatusInternalServerError, i18n.Translate("user.delete.failed"), nil)
	}
	return response.User(true, http.StatusNoContent, i18n.Translate("user.delete.success"), nil)
}

func NewUserService(
	userRepo port.UserRepository,
	roleRepo port.RoleRepository,
	uow port.UnitOfWork,
	cache port.CacheRepository,
	cfg *config.Configuration,
) port.UserService {
	return &userServ{
		userRepo: userRepo,
		roleRepo: roleRepo,
		uow:      uow,
		cache:    cache,
		config:   cfg,
	}
}
