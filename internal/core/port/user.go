package port

import (
	"github.com/brmcode/user-auth-service/internal/core/domain"
	"github.com/brmcode/user-auth-service/internal/core/dto/request"
	"github.com/brmcode/user-auth-service/internal/core/dto/response"
	"github.com/gin-gonic/gin"
)

type UserRepository interface {
	Create(user *domain.User) (*domain.User, error)
	Get(username string) (*domain.User, error)
	GetByEmailAndRole(email string, role string) (*domain.User, error)
	Update(user *domain.User) (*domain.User, error)
	Delete(user *domain.User) error
}

type UserService interface {
	CreateUser(ctx *gin.Context, req request.CreateUserRequest) (*domain.User, *response.Error)
	GetUser(ctx *gin.Context, username string) (*domain.User, *response.Error)
	UpdateUser(ctx *gin.Context, req request.UpdateUserRequest) (*domain.User, *response.Error)
	DeleteUser(ctx *gin.Context, username string) *response.Error
}
