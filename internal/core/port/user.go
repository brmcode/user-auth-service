package port

import (
	"github.com/brmcode/user-auth-service/internal/adapter/http/handler/dto/request"
	"github.com/brmcode/user-auth-service/internal/adapter/http/handler/dto/response"
	"github.com/brmcode/user-auth-service/internal/core/domain"
	"github.com/gin-gonic/gin"
)

type UserRepository interface {
	Create(user *domain.User) (*domain.User, error)
	Get(username string) (*domain.User, error)
	GetByEmail(email string) (*domain.User, error)
	GetByEmailAndRole(email string, role string) (*domain.User, error)
	GetByEmailUnscoped(email string) (*domain.User, error)
	Update(user *domain.User) (*domain.User, error)
	Delete(user *domain.User) error
}

type UserService interface {
	CreateUser(ctx *gin.Context, req request.CreateUserRequest) *response.User
	GetUser(ctx *gin.Context, username string) *response.User
	UpdateUser(ctx *gin.Context, req request.UpdateUserRequest) *response.User
	DeleteUser(ctx *gin.Context, username string) *response.User
}
