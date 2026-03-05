package service

import (
	"errors"
	"log"
	"net/http"

	"github.com/brmcode/user-auth-service/internal/core/domain"
	"github.com/gin-gonic/gin"

	"github.com/brmcode/user-auth-service/internal/adapter/http/handler/dto/request"
	"github.com/brmcode/user-auth-service/internal/adapter/http/handler/dto/response"
	"github.com/brmcode/user-auth-service/internal/core/port"
	"github.com/brmcode/user-auth-service/pkg/config"
	"github.com/brmcode/user-auth-service/pkg/util"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type userServ struct {
	userRepo port.UserRepository
	cache    port.CacheRepository
	config   *config.Configuration
}

// CreateUser implements UserService.
func (u *userServ) CreateUser(ctx *gin.Context, req request.CreateUserRequest) *response.User {

	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		return response.NewUser(false, http.StatusInternalServerError, "failed to hash password", nil, &[]string{err.Error()})
	}

	user := &domain.User{
		Username:       util.RandomUsername(),
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		Email:          req.Email,
		HashedPassword: hashedPassword,
		Role:           req.Role,
	}

	createdUser, err := u.userRepo.Create(user)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return response.NewUser(false, http.StatusConflict, pgErr.Detail, nil, &[]string{pgErr.Detail})
		}

		return response.NewUser(false, http.StatusInternalServerError, err.Error(), nil, &[]string{err.Error()})
	}

	cacheKey := util.GenerateCacheKey("user", createdUser.Username)
	userSerialized, err := util.Serialize(createdUser)
	if err != nil {
		return response.NewUser(false, http.StatusInternalServerError, "failed to serialize user", nil, &[]string{err.Error()})
	}

	// Parallel cache operations: set new cache and delete prefix cache concurrently
	errChan := make(chan error, 2)
	go func() {
		errChan <- u.cache.Set(ctx, cacheKey, userSerialized, u.config.Redis.TTL)
	}()
	go func() {
		errChan <- u.cache.DeleteByPrefix(ctx, "users:*")
	}()

	// Wait for both operations
	for i := 0; i < 2; i++ {
		if err := <-errChan; err != nil {
			return response.NewUser(false, http.StatusInternalServerError, "failed to set cache", nil, &[]string{err.Error()})
		}
	}

	return response.NewUser(true, http.StatusCreated, "user created successfully", createdUser, nil)
}

// DeleteUser implements UserService.
func (u *userServ) DeleteUser(ctx *gin.Context, username string) *response.User {
	user, err := u.userRepo.Get(username)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NewUser(false, http.StatusNotFound, gorm.ErrRecordNotFound.Error(), nil, &[]string{gorm.ErrRecordNotFound.Error()})
		}

		return response.NewUser(false, http.StatusInternalServerError, err.Error(), nil, &[]string{err.Error()})
	}

	cacheKey := util.GenerateCacheKey("user", user.Username)

	// Parallel cache operations: delete specific key and prefix cache concurrently
	errChan := make(chan error, 2)
	go func() {
		errChan <- u.cache.Delete(ctx, cacheKey)
	}()
	go func() {
		errChan <- u.cache.DeleteByPrefix(ctx, "users:*")
	}()

	// Wait for both operations
	for i := 0; i < 2; i++ {
		if err := <-errChan; err != nil {
			return response.NewUser(false, http.StatusInternalServerError, "failed to delete cache", nil, &[]string{err.Error()})
		}
	}

	err = u.userRepo.Delete(user)
	if err != nil {
		return response.NewUser(false, http.StatusInternalServerError, "failed to delete user", nil, &[]string{err.Error()})
	}

	return response.NewUser(true, http.StatusNoContent, "user deleted successfully", nil, nil)
}

// GetUser implements UserService.
func (u *userServ) GetUser(ctx *gin.Context, username string) *response.User {

	var user *domain.User
	cacheKey := util.GenerateCacheKey("user", username)
	cacheUser, err := u.cache.Get(ctx, cacheKey)
	if err == nil {
		err = util.Deserialize(cacheUser, &user)
		if err != nil {
			return response.NewUser(false, http.StatusInternalServerError, "failed to deserialize user", nil, &[]string{err.Error()})
		}
		log.Println("cache hit:", cacheKey)
		return &response.User{Success: true, StatusCode: http.StatusOK, Message: "user fetched successfully", Data: *user}
	}

	user, err = u.userRepo.Get(username)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NewUser(false, http.StatusNotFound, gorm.ErrRecordNotFound.Error(), nil, &[]string{gorm.ErrRecordNotFound.Error()})
		}

		return response.NewUser(false, http.StatusInternalServerError, err.Error(), nil, &[]string{err.Error()})
	}

	userSerialized, err := util.Serialize(user)
	if err != nil {
		return response.NewUser(false, http.StatusInternalServerError, "failed to serialize user", nil, &[]string{err.Error()})
	}
	err = u.cache.Set(ctx, cacheKey, userSerialized, u.config.Redis.TTL)
	if err != nil {
		return response.NewUser(false, http.StatusInternalServerError, "failed to set cache", nil, &[]string{err.Error()})
	}

	return response.NewUser(true, http.StatusOK, "user fetched successfully", user, nil)
}

// UpdateUser implements UserService.
func (u *userServ) UpdateUser(ctx *gin.Context, req request.UpdateUserRequest) *response.User {
	user, err := u.userRepo.Get(req.Username)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NewUser(false, http.StatusNotFound, gorm.ErrRecordNotFound.Error(), nil, &[]string{gorm.ErrRecordNotFound.Error()})
		}

		return response.NewUser(false, http.StatusInternalServerError, err.Error(), nil, &[]string{err.Error()})
	}

	user.FirstName = req.FirstName
	user.LastName = req.LastName
	user.Role = req.Role

	updatedUser, err := u.userRepo.Update(user)
	if err != nil {
		return response.NewUser(false, http.StatusInternalServerError, "failed to update user", nil, &[]string{err.Error()})
	}

	cacheKey := util.GenerateCacheKey("user", updatedUser.Username)

	userSerialized, err := util.Serialize(updatedUser)
	if err != nil {
		return response.NewUser(false, http.StatusInternalServerError, "failed to serialize user", nil, &[]string{err.Error()})
	}

	// Parallel cache operations: set new cache and delete prefix cache concurrently
	// SET will overwrite existing key, so no need to delete first
	errChan := make(chan error, 2)
	go func() {
		errChan <- u.cache.Set(ctx, cacheKey, userSerialized, u.config.Redis.TTL)
	}()
	go func() {
		errChan <- u.cache.DeleteByPrefix(ctx, "users:*")
	}()

	// Wait for both operations
	for i := 0; i < 2; i++ {
		if err := <-errChan; err != nil {
			return response.NewUser(false, http.StatusInternalServerError, "failed to set cache", nil, &[]string{err.Error()})
		}
	}

	return response.NewUser(true, http.StatusOK, "user updated successfully", updatedUser, nil)
}

func NewUserService(userRepo port.UserRepository, cache port.CacheRepository, config *config.Configuration) port.UserService {
	return &userServ{userRepo, cache, config}
}
