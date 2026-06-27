// pkg/errs/app_error.go
package errs

import (
	"investment_monitoring/pkg/logger"
	"net/http"
)

// AppError adalah custom error struct kita
type AppError struct {
	Code             int               `json:"code"`
	Message          string            `json:"message"`
	Err              error             `json:"-"` // error asli (untuk logging, tidak dikirim ke json)
	ValidationErrors map[string]string `json:"validationErrors"`
}

// Implement interface error bawaan Go
func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Code >= 500 {
		logger.L().Error("REQUEST_FAILED", "status", e.Code, "err", e.Err)
	}
	if e.Code >= http.StatusInternalServerError {
		// TODO: send to sentry
		return "INTERNAL_SERVER_ERROR"
	}
	// logger.L().Error(strconv.Itoa(e.Code), "error", e.Err.Error())
	return e.Message
}

// --- Factory Functions (Mirip NestJS Exceptions) ---

func NewBadRequest(message string) *AppError {
	return &AppError{
		Code:    http.StatusBadRequest,
		Message: message,
	}
}

func NewNotFound(message string) *AppError {
	defaultMessage := "DATA_NOT_FOUND"
	if message == "" {
		message = defaultMessage
	}
	return &AppError{
		Code:    http.StatusNotFound,
		Message: message,
	}
}

func NewUnauthorized(message string) *AppError {
	if message == "" {
		message = "UNAUTHORIZED"
	}
	return &AppError{
		Code:    http.StatusUnauthorized,
		Message: message,
	}
}

func NewForbidden(message string) *AppError {
	if message == "" {
		message = "FORBIDDEN"
	}
	return &AppError{
		Code:    http.StatusForbidden,
		Message: message,
	}
}

func NewTooManyRequests(message string) *AppError {
	if message == "" {
		message = "TOO_MANY_REQUESTS"
	}
	return &AppError{
		Code:    http.StatusTooManyRequests,
		Message: message,
	}
}

func NewInternalServerError(err error) *AppError {
	return &AppError{
		Code:    http.StatusInternalServerError,
		Message: "INTERNAL_SERVER_ERROR",
		Err:     err,
	}
}

func NewValidationFailed(validationErrors map[string]string) *AppError {
	return &AppError{
		Code:             http.StatusBadRequest,
		Message:          "VALIDATION_FAILED",
		ValidationErrors: validationErrors,
	}
}
