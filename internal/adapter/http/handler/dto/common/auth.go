package dto

import (
	"time"

	"github.com/brmcode/user-auth-service/internal/core/domain"
	"github.com/google/uuid"
)

type LoginModel struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Role     string `json:"role" validate:"omitempty,role"`
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
	RefreshToken string `json:"refresh_token" validate:"required"`
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
	ImageURL  string `json:"image_url" validate:"optional_url"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=6"`
}

type GoogleAuthRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}
