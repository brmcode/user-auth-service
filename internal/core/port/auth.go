package port

import (
	"time"

	"github.com/brmcode/user-auth-service/internal/adapter/auth"
	"github.com/brmcode/user-auth-service/internal/core/domain"
	dto "github.com/brmcode/user-auth-service/internal/core/dto/common"
	"github.com/brmcode/user-auth-service/internal/core/dto/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/markbates/goth"
)

type AuthenticationService interface {
	Login(ctx *gin.Context, cred dto.LoginModel) (*dto.LoginUserResponse, *response.Error)
	Register(ctx *gin.Context, req dto.RegisterUserRequest) (*domain.User, *response.Error)
	ReNewAccessToken(ctx *gin.Context, req dto.ReNewAccessTokenRequest) (*dto.ReNewAccessTokenResponse, *response.Error)
	OAuthLogin(ctx *gin.Context, provider string, gUser goth.User) (*dto.LoginUserResponse, *response.Error)
}

type TokenService interface {
	GenerateToken(tokenID uuid.UUID, username string, role string, duration time.Duration) (string, *auth.Payload, error)
	VerifyToken(tokenString string) (*auth.Payload, error)
}
