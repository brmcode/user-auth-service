package service

import (
	"errors"
	"log"

	"github.com/brmcode/user-auth-service/internal/core/domain"
	"github.com/gin-gonic/gin"

	"github.com/brmcode/user-auth-service/internal/core/dto/request"
	"github.com/brmcode/user-auth-service/internal/core/dto/response"
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
func (u *userServ) CreateUser(ctx *gin.Context, req request.CreateUserRequest) (*domain.User, *response.Error) {

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
		Role:           req.Role,
	}

	createdUser, err := u.userRepo.Create(user)
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
		errChan <- u.cache.Set(ctx, cacheKey, userSerialized, u.config.Redis.TTL)
	}()
	go func() {
		errChan <- u.cache.DeleteByPrefix(ctx, "users:*")
	}()

	// Wait for both operations
	for i := 0; i < 2; i++ {
		if err := <-errChan; err != nil {
			return nil, response.NewError(500, err.Error())
		}
	}

	return createdUser, nil
}

// DeleteUser implements UserService.
func (u *userServ) DeleteUser(ctx *gin.Context, username string) *response.Error {
	user, err := u.userRepo.Get(username)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NewError(404, gorm.ErrRecordNotFound.Error())
		}

		return response.NewError(500, err.Error())
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
			return response.NewError(500, err.Error())
		}
	}

	err = u.userRepo.Delete(user)
	if err != nil {
		return response.NewError(500, err.Error())
	}

	return nil
}

// GetUser implements UserService.
func (u *userServ) GetUser(ctx *gin.Context, username string) (*domain.User, *response.Error) {

	var user *domain.User
	cacheKey := util.GenerateCacheKey("user", username)
	cacheUser, err := u.cache.Get(ctx, cacheKey)
	if err == nil {
		err = util.Deserialize(cacheUser, &user)
		if err != nil {
			return nil, response.NewError(500, err.Error())
		}
		log.Println("cache hit:", cacheKey)
		return user, nil
	}

	user, err = u.userRepo.Get(username)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, response.NewError(404, gorm.ErrRecordNotFound.Error())
		}

		return nil, response.NewError(500, err.Error())
	}

	userSerialized, err := util.Serialize(user)
	if err != nil {
		return nil, response.NewError(500, err.Error())
	}
	err = u.cache.Set(ctx, cacheKey, userSerialized, u.config.Redis.TTL)
	if err != nil {
		return nil, response.NewError(500, err.Error())
	}

	return user, nil
}

// UpdateUser implements UserService.
func (u *userServ) UpdateUser(ctx *gin.Context, req request.UpdateUserRequest) (*domain.User, *response.Error) {
	user, err := u.userRepo.Get(req.Username)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, response.NewError(404, gorm.ErrRecordNotFound.Error())
		}

		return nil, response.NewError(500, err.Error())
	}

	user.FirstName = req.FirstName
	user.LastName = req.LastName
	user.Role = req.Role

	updatedUser, err := u.userRepo.Update(user)
	if err != nil {
		return nil, response.NewError(500, err.Error())
	}

	cacheKey := util.GenerateCacheKey("user", updatedUser.Username)

	userSerialized, err := util.Serialize(updatedUser)
	if err != nil {
		return nil, response.NewError(500, err.Error())
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
			return nil, response.NewError(500, err.Error())
		}
	}

	return updatedUser, nil
}

func NewUserService(userRepo port.UserRepository, cache port.CacheRepository, config *config.Configuration) port.UserService {
	return &userServ{userRepo, cache, config}
}
