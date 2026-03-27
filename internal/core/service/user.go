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
		return nil, response.User(false, http.StatusInternalServerError, "failed to load roles", nil, &[]string{err.Error()})
	}
	if len(roles) != len(codes) {
		return nil, response.User(false, http.StatusBadRequest, "one or more role codes are invalid", nil, nil)
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
		return response.User(false, http.StatusInternalServerError, "failed to serialize user", nil, &[]string{err.Error()})
	}
	errChan := make(chan error, 2)
	go func() { errChan <- u.cache.Set(ctx, key, serialized, u.config.Redis.TTL) }()
	go func() { errChan <- u.cache.DeleteByPrefix(ctx, "users:*") }()
	for i := 0; i < 2; i++ {
		if err := <-errChan; err != nil {
			return response.User(false, http.StatusInternalServerError, "failed to update cache", nil, &[]string{err.Error()})
		}
	}
	return nil
}

func (u *userServ) CreateUser(ctx *gin.Context, req request.CreateUserRequest) *response.UserResult {
	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		return response.User(false, http.StatusInternalServerError, "failed to hash password", nil, &[]string{err.Error()})
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
			return response.User(false, http.StatusConflict, pgErr.Detail, nil, &[]string{pgErr.Detail})
		}
		return response.User(false, http.StatusInternalServerError, err.Error(), nil, &[]string{err.Error()})
	}

	if errResp := u.setCache(ctx, created); errResp != nil {
		return errResp
	}
	return response.User(true, http.StatusCreated, "user created successfully", created, nil)
}

func (u *userServ) GetUser(ctx *gin.Context, username string) *response.UserResult {
	key := util.GenerateCacheKey("user", username)

	if cached, err := u.cache.Get(ctx, key); err == nil {
		var user domain.User
		if err := util.Deserialize(cached, &user); err != nil {
			return response.User(false, http.StatusInternalServerError, "failed to deserialize user", nil, &[]string{err.Error()})
		}
		log.Println("cache hit:", key)
		// return &response.UserResult{Success: true, StatusCode: http.StatusOK, Message: "user fetched successfully", Data: &user}
		return response.User(true, http.StatusOK, "user fetched successfully", &user, nil)
	}

	user, err := u.userRepo.Get(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.User(false, http.StatusNotFound, gorm.ErrRecordNotFound.Error(), nil, &[]string{err.Error()})
		}
		return response.User(false, http.StatusInternalServerError, err.Error(), nil, &[]string{err.Error()})
	}

	serialized, err := util.Serialize(user)
	if err != nil {
		return response.User(false, http.StatusInternalServerError, "failed to serialize user", nil, &[]string{err.Error()})
	}
	if err := u.cache.Set(ctx, key, serialized, u.config.Redis.TTL); err != nil {
		return response.User(false, http.StatusInternalServerError, "failed to set cache", nil, &[]string{err.Error()})
	}

	return response.User(true, http.StatusOK, "user fetched successfully", user, nil)
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
				result = response.User(false, http.StatusNotFound, gorm.ErrRecordNotFound.Error(), nil, &[]string{err.Error()})
			} else {
				result = response.User(false, http.StatusInternalServerError, err.Error(), nil, &[]string{err.Error()})
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
		return response.User(false, http.StatusInternalServerError, "failed to update user", nil, &[]string{err.Error()})
	}

	if errResp := u.setCache(ctx, updated); errResp != nil {
		return errResp
	}
	return response.User(true, http.StatusOK, "user updated successfully", updated, nil)
}

func (u *userServ) DeleteUser(ctx *gin.Context, username string) *response.UserResult {
	user, err := u.userRepo.Get(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.User(false, http.StatusNotFound, gorm.ErrRecordNotFound.Error(), nil, &[]string{err.Error()})
		}
		return response.User(false, http.StatusInternalServerError, err.Error(), nil, &[]string{err.Error()})
	}

	u.invalidateCache(ctx, user.Username)

	if err := u.userRepo.Delete(user); err != nil {
		return response.User(false, http.StatusInternalServerError, "failed to delete user", nil, &[]string{err.Error()})
	}
	return response.User(true, http.StatusNoContent, "user deleted successfully", nil, nil)
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
