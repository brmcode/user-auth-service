package port

import (
	"time"

	"github.com/brmcode/user-auth-service/internal/adapter/auth"
	"github.com/brmcode/user-auth-service/internal/adapter/auth/google"
	dto "github.com/brmcode/user-auth-service/internal/adapter/http/handler/dto/common"
	"github.com/brmcode/user-auth-service/internal/adapter/http/handler/dto/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/markbates/goth"
)

type AuthenticationService interface {
	Login(ctx *gin.Context, cred dto.LoginModel) *response.LoginResult
	Register(ctx *gin.Context, req dto.RegisterUserRequest) *response.User
	ReNewAccessToken(ctx *gin.Context, req dto.ReNewAccessTokenRequest) *response.RefreshTokenResult
	Logout(ctx *gin.Context, req dto.ReNewAccessTokenRequest) *response.LogoutResult
	OAuthLogin(ctx *gin.Context, provider string, gUser goth.User) *response.LoginResult
	GoogleAuthMobile(ctx *gin.Context, payload *google.Payload) *response.LoginResult
}

type TokenService interface {
	GenerateToken(tokenID uuid.UUID, username string, roles []string, duration time.Duration) (string, *auth.Payload, error)
	VerifyToken(tokenString string) (*auth.Payload, error)
}
