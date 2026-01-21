package dto

import (
	"time"

	"github.com/brmcode/user-auth-service/internal/core/domain"
	"github.com/google/uuid"
)

type LoginModel struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Role     string `json:"role" validate:"required,oneof=ADMIN USER"`
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
	RefreshToken          string    `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
}

type RegisterUserRequest struct {
	FirstName string `json:"first_name" validate:"required,min=3,max=20"`
	LastName  string `json:"last_name" validate:"required,min=3,max=20"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=6"`
}

type OAuthRegisterUserRequest struct {
	Provider       string `json:"provider" validate:"required,max=20"`
	ProviderUserID string `json:"provider_user_id" validate:"required"`
	FirstName      string `json:"first_name" validate:"required,min=3,max=20"`
	LastName       string `json:"last_name" validate:"required,min=3,max=20"`
	Email          string `json:"email" validate:"required,email"`
	Password       string `json:"password" validate:"required,min=6"`
}
