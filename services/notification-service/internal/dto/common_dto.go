package dto

// BaseResponse represents the standard API response structure
type BaseResponse[T any] struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    T      `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// APIResponse is a non-generic alias for BaseResponse[any]
type APIResponse = BaseResponse[any]

// QueuedResponse is the data payload returned for asynchronously queued jobs
type QueuedResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status,omitempty"`
}

// SuccessResponse creates a successful BaseResponse
func SuccessResponse[T any](data T, message ...string) BaseResponse[T] {
	msg := ""
	if len(message) > 0 {
		msg = message[0]
	}
	return BaseResponse[T]{
		Success: true,
		Message: msg,
		Data:    data,
	}
}

// ErrorResponse creates an error BaseResponse
func ErrorResponse[T any](errMsg string, message ...string) BaseResponse[T] {
	msg := ""
	if len(message) > 0 {
		msg = message[0]
	}
	return BaseResponse[T]{
		Success: false,
		Message: msg,
		Error:   errMsg,
	}
}
