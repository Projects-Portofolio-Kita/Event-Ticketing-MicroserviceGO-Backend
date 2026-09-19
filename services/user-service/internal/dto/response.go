package dto

type BaseResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

func SuccessResponse(message string, data any) BaseResponse {
	return BaseResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
}

func ErrorResponse(message string, errCode string) BaseResponse {
	return BaseResponse{
		Success: false,
		Message: message,
		Error:   errCode,
	}
}
