package response

import "github.com/brmcode/user-auth-service/internal/core/domain"

type UserResult struct {
	Success    bool         `json:"success"`
	StatusCode int          `json:"status_code"`
	Message    string       `json:"message"`
	Data       *domain.User `json:"data,omitempty"`
}

type ListUserResult struct {
	Success    bool          `json:"success"`
	StatusCode int           `json:"status_code"`
	Message    string        `json:"message"`
	Data       []domain.User `json:"data,omitempty"`
}

func User(success bool, statusCode int, message string, data *domain.User) *UserResult {
	if success {
		return &UserResult{Success: true, StatusCode: statusCode, Message: message, Data: data}
	}

	return &UserResult{Success: false, StatusCode: statusCode, Message: message}
}

func ListUser(success bool, statusCode int, message string, data *[]domain.User) *ListUserResult {
	if success {
		return &ListUserResult{Success: true, StatusCode: statusCode, Message: message, Data: *data}
	}

	return &ListUserResult{Success: false, StatusCode: statusCode, Message: message}
}
