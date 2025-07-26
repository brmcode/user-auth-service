package dto

import (
	"time"

	"github.com/brmcode/user-auth-service/domain"
	"github.com/google/uuid"
)

type LoginModel struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type LoginUserResponse struct {
	SessionID             uuid.UUID    `json:"session_id"`
	AccessToken           string       `json:"access_token"`
	AccessTokenExpriresAt time.Time    `json:"access_token_expires_at"`
	RefreshToken          string       `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time    `json:"refresh_token_expires_at"`
	User                  *domain.User `json:"user"`
}

type ReNewAccessTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type ReNewAccessTokenResponse struct {
	AccessToken           string    `json:"access_token"`
	AccessTokenExpriresAt time.Time `json:"access_token_expires_at"`
}

type RegisterUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}
