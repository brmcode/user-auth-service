package port

import (
	"time"

	"github.com/brmcode/user-auth-service/internal/adapter/auth"
	dto "github.com/brmcode/user-auth-service/internal/adapter/http/handler/dto/common"
	"github.com/brmcode/user-auth-service/internal/adapter/http/handler/dto/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/markbates/goth"
)

type AuthenticationService interface {
	Login(ctx *gin.Context, cred dto.LoginModel) *response.Login
	Register(ctx *gin.Context, req dto.RegisterUserRequest) *response.User
	ReNewAccessToken(ctx *gin.Context, req dto.ReNewAccessTokenRequest) *response.RefreshToken
	Logout(ctx *gin.Context, req dto.ReNewAccessTokenRequest) *response.Logout
	OAuthLogin(ctx *gin.Context, provider string, gUser goth.User) *response.Login
}

type TokenService interface {
	GenerateToken(tokenID uuid.UUID, username string, role string, duration time.Duration) (string, *auth.Payload, error)
	VerifyToken(tokenString string) (*auth.Payload, error)
}
