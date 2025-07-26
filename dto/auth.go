package dto

import "github.com/brmcode/user-auth-service/domain"

type LoginModel struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type LoginUserResponse struct {
	AccessToken string       `json:"access_token"`
	User        *domain.User `json:"user"`
}
