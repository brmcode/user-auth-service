package response

import dto "github.com/brmcode/user-auth-service/internal/adapter/http/handler/dto/common"

type LoginResult struct {
	Success    bool                   `json:"success"`
	StatusCode int                    `json:"status_code"`
	Message    string                 `json:"message"`
	NewUser    bool                   `json:"new_user,omitempty"`
	Data       *dto.LoginUserResponse `json:"data,omitempty"`
}

type RefreshTokenResult struct {
	Success    bool                          `json:"success"`
	StatusCode int                           `json:"status_code"`
	Message    string                        `json:"message"`
	Data       *dto.ReNewAccessTokenResponse `json:"data,omitempty"`
}

type LogoutResult struct {
	Success    bool   `json:"success"`
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
}

func Login(success bool, statusCode int, message string, newUser bool, data *dto.LoginUserResponse) *LoginResult {
	if success {
		return &LoginResult{Success: true, StatusCode: statusCode, Message: message, NewUser: newUser, Data: data}
	}

	return &LoginResult{Success: false, StatusCode: statusCode, Message: message}
}

func RefreshToken(success bool, statusCode int, message string, data *dto.ReNewAccessTokenResponse) *RefreshTokenResult {
	if success {
		return &RefreshTokenResult{Success: true, StatusCode: statusCode, Message: message, Data: data}
	}

	return &RefreshTokenResult{Success: false, StatusCode: statusCode, Message: message}
}

func Logout(success bool, statusCode int, message string, errors *[]string) *LogoutResult {
	if success {
		return &LogoutResult{Success: true, StatusCode: statusCode, Message: message}
	}

	return &LogoutResult{Success: false, StatusCode: statusCode, Message: message}
}
