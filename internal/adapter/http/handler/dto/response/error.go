package response

type Error struct {
	Success    bool   `json:"success"`
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
}

func NewError(statusCode int, message string) *Error {
	return &Error{
		Success:    false,
		StatusCode: statusCode,
		Message:    message,
	}
}
