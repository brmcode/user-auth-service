package response

import dto "github.com/brmcode/user-auth-service/internal/adapter/http/handler/dto/common"

type LoginResult struct {
	Success    bool                   `json:"success"`
	StatusCode int                    `json:"status_code"`
	Message    string                 `json:"message"`
	Data       *dto.LoginUserResponse `json:"data,omitempty"`
	Errors     []string               `json:"errors,omitempty"`
}

type RefreshTokenResult struct {
	Success    bool                          `json:"success"`
	StatusCode int                           `json:"status_code"`
	Message    string                        `json:"message"`
	Data       *dto.ReNewAccessTokenResponse `json:"data,omitempty"`
	Errors     []string                      `json:"errors,omitempty"`
}

type LogoutResult struct {
	Success    bool     `json:"success"`
	StatusCode int      `json:"status_code"`
	Message    string   `json:"message"`
	Errors     []string `json:"errors,omitempty"`
}

func Login(success bool, statusCode int, message string, data *dto.LoginUserResponse, errors *[]string) *LoginResult {
	if success {
		return &LoginResult{Success: true, StatusCode: statusCode, Message: message, Data: data}
	}
	var errs []string
	if errors != nil {
		errs = *errors
	}
	return &LoginResult{Success: false, StatusCode: statusCode, Message: message, Errors: errs}
}

func RefreshToken(success bool, statusCode int, message string, data *dto.ReNewAccessTokenResponse, errors *[]string) *RefreshTokenResult {
	if success {
		return &RefreshTokenResult{Success: true, StatusCode: statusCode, Message: message, Data: data}
	}
	var errs []string
	if errors != nil {
		errs = *errors
	}
	return &RefreshTokenResult{Success: false, StatusCode: statusCode, Message: message, Errors: errs}
}

func Logout(success bool, statusCode int, message string, errors *[]string) *LogoutResult {
	if success {
		return &LogoutResult{Success: true, StatusCode: statusCode, Message: message}
	}
	var errs []string
	if errors != nil {
		errs = *errors
	}
	return &LogoutResult{Success: false, StatusCode: statusCode, Message: message, Errors: errs}
}
