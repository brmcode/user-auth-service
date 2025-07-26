package service

import (
	"errors"

	"github.com/brmcode/user-auth-service/domain"
	"github.com/brmcode/user-auth-service/dto"
	"github.com/brmcode/user-auth-service/dto/response"
	"github.com/brmcode/user-auth-service/repository"
	"github.com/brmcode/user-auth-service/util"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type UserService interface {
	CreateUser(req dto.CreateUserRequest) (*domain.User, *response.Error)
	GetUser(username string) (*domain.User, *response.Error)
	UpdateUser(req dto.UpdateUserRequest) (*domain.User, *response.Error)
	DeleteUser(username string) *response.Error
}

type userServ struct {
	userRepo repository.UserRepository
}

// CreateUser implements UserService.
func (u *userServ) CreateUser(req dto.CreateUserRequest) (*domain.User, *response.Error) {

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

	return createdUser, nil
}

// DeleteUser implements UserService.
func (u *userServ) DeleteUser(username string) *response.Error {
	user, err := u.userRepo.Get(username)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NewError(404, gorm.ErrRecordNotFound.Error())
		}

		return response.NewError(500, err.Error())
	}

	err = u.userRepo.Delete(user)
	if err != nil {
		return response.NewError(500, err.Error())
	}

	return nil
}

// GetUser implements UserService.
func (u *userServ) GetUser(username string) (*domain.User, *response.Error) {
	user, err := u.userRepo.Get(username)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, response.NewError(404, gorm.ErrRecordNotFound.Error())
		}

		return nil, response.NewError(500, err.Error())
	}

	return user, nil
}

// UpdateUser implements UserService.
func (u *userServ) UpdateUser(req dto.UpdateUserRequest) (*domain.User, *response.Error) {
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

	return updatedUser, nil
}

func NewUserService(userRepo repository.UserRepository) UserService {
	return &userServ{userRepo: userRepo}
}
