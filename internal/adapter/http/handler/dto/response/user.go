package response

import "github.com/brmcode/user-auth-service/internal/core/domain"

type UserResult struct {
	Success    bool         `json:"success"`
	StatusCode int          `json:"status_code"`
	Message    string       `json:"message"`
	Data       *domain.User `json:"data,omitempty"`
	Errors     []string     `json:"errors,omitempty"`
}

type ListUserResult struct {
	Success    bool          `json:"success"`
	StatusCode int           `json:"status_code"`
	Message    string        `json:"message"`
	Data       []domain.User `json:"data,omitempty"`
	Errors     []string      `json:"errors,omitempty"`
}

func User(success bool, statusCode int, message string, data *domain.User, errors *[]string) *UserResult {
	if success {
		return &UserResult{Success: true, StatusCode: statusCode, Message: message, Data: data}
	}
	var errs []string
	if errors != nil {
		errs = *errors
	}
	return &UserResult{Success: false, StatusCode: statusCode, Message: message, Errors: errs}
}

func ListUser(success bool, statusCode int, message string, data *[]domain.User, errors *[]string) *ListUserResult {
	if success {
		return &ListUserResult{Success: true, StatusCode: statusCode, Message: message, Data: *data}
	}
	var errs []string
	if errors != nil {
		errs = *errors
	}
	return &ListUserResult{Success: false, StatusCode: statusCode, Message: message, Errors: errs}
}
