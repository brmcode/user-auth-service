package response

import "github.com/brmcode/user-auth-service/internal/core/domain"

type User struct {
	Success    bool        `json:"success"`
	StatusCode int         `json:"status_code"`
	Message    string      `json:"message"`
	Data       domain.User `json:"data,omitempty"`
	Errors     []string    `json:"errors,omitempty"`
}

type ListUser struct {
	Success    bool          `json:"success"`
	StatusCode int           `json:"status_code"`
	Message    string        `json:"message"`
	Data       []domain.User `json:"data,omitempty"`
	Errors     []string      `json:"errors,omitempty"`
}

func NewUser(success bool, statusCode int, message string, data *domain.User, errors *[]string) *User {
	if success {
		return &User{Success: true, StatusCode: statusCode, Message: message, Data: *data}
	}
	return &User{Success: false, StatusCode: statusCode, Message: message, Errors: *errors}
}

func NewListUser(success bool, statusCode int, message string, data *[]domain.User, errors *[]string) *ListUser {
	if success {
		return &ListUser{Success: true, StatusCode: statusCode, Message: message, Data: *data}
	}
	return &ListUser{Success: false, StatusCode: statusCode, Message: message, Errors: *errors}
}
