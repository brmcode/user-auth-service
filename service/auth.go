package service

import (
	"errors"
	"fmt"
	"log"

	"github.com/brmcode/user-auth-service/domain"
	"github.com/brmcode/user-auth-service/dto"
	"github.com/brmcode/user-auth-service/dto/response"
	"github.com/brmcode/user-auth-service/pkg/auth"
	"github.com/brmcode/user-auth-service/pkg/config"
	"github.com/brmcode/user-auth-service/repository"
	"github.com/brmcode/user-auth-service/util"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type AuthenticationService interface {
	Login(cred dto.LoginModel) (*dto.LoginUserResponse, *response.Error)
	Register(req dto.CreateUserRequest) (*domain.User, *response.Error)
}

type authService struct {
	config       *config.Auth
	userRepo     repository.UserRepository
	tokenService auth.TokenService
}

// Login implements AuthenticationService.
func (a *authService) Login(cred dto.LoginModel) (*dto.LoginUserResponse, *response.Error) {
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

	// Generate token
	token, err := a.tokenService.GenerateToken(user.Username, user.Role, a.config.TokenDuration)
	if err != nil {
		log.Printf("Error: %s", err.Error())
		return nil, response.NewError(500, fmt.Sprintf("could not generate token: %s", err.Error()))
	}

	return &dto.LoginUserResponse{AccessToken: token, User: user}, nil
}

// Register implements AuthenticationService.
func (a *authService) Register(req dto.CreateUserRequest) (*domain.User, *response.Error) {
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

	createdUser, err := a.userRepo.Create(user)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, response.NewError(409, pgErr.Detail)
		}

		return nil, response.NewError(500, err.Error())
	}

	return createdUser, nil
}

func NewAuthenticationService(config *config.Auth, userRepo repository.UserRepository, tokenService auth.TokenService) AuthenticationService {
	return &authService{
		config:       config,
		userRepo:     userRepo,
		tokenService: tokenService,
	}
}
