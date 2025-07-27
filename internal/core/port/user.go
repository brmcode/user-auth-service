package port

import (
	"github.com/brmcode/user-auth-service/internal/core/domain"
	"github.com/brmcode/user-auth-service/internal/core/dto/request"
	"github.com/brmcode/user-auth-service/internal/core/dto/response"
)

type UserRepository interface {
	Create(user *domain.User) (*domain.User, error)
	Get(username string) (*domain.User, error)
	GetByEmailAndRole(email string, role string) (*domain.User, error)
	Update(user *domain.User) (*domain.User, error)
	Delete(user *domain.User) error
}

type UserService interface {
	CreateUser(req request.CreateUserRequest) (*domain.User, *response.Error)
	GetUser(username string) (*domain.User, *response.Error)
	UpdateUser(req request.UpdateUserRequest) (*domain.User, *response.Error)
	DeleteUser(username string) *response.Error
}
