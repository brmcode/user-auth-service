package response

import dto "github.com/brmcode/user-auth-service/internal/adapter/http/handler/dto/common"

type Login struct {
	Success    bool                  `json:"success"`
	StatusCode int                   `json:"status_code"`
	Message    string                `json:"message"`
	Data       dto.LoginUserResponse `json:"data,omitempty"`
	Errors     []string              `json:"errors,omitempty"`
}

type RefreshToken struct {
	Success    bool                         `json:"success"`
	StatusCode int                          `json:"status_code"`
	Message    string                       `json:"message"`
	Data       dto.ReNewAccessTokenResponse `json:"data,omitempty"`
	Errors     []string                     `json:"errors,omitempty"`
}

func NewLogin(success bool, statusCode int, message string, data *dto.LoginUserResponse, errors *[]string) *Login {
	if success {
		return &Login{Success: true, StatusCode: statusCode, Message: message, Data: *data}
	}
	return &Login{Success: false, StatusCode: statusCode, Message: message, Errors: *errors}
}

func NewRefreshToken(success bool, statusCode int, message string, data *dto.ReNewAccessTokenResponse, errors *[]string) *RefreshToken {
	if success {
		return &RefreshToken{Success: true, StatusCode: statusCode, Message: message, Data: *data}
	}
	return &RefreshToken{Success: false, StatusCode: statusCode, Message: message, Errors: *errors}
}
