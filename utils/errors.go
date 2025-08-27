package utils

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"runtime"
)

// Error types
var (
	ErrValidation       = errors.New("validation error")
	ErrAuthentication   = errors.New("authentication error")
	ErrAuthorization    = errors.New("authorization error")
	ErrNotFound         = errors.New("resource not found")
	ErrDuplicateEntry   = errors.New("duplicate entry")
	ErrDatabase         = errors.New("database error")
	ErrInternal         = errors.New("internal server error")
	ErrInvalidInput     = errors.New("invalid input")
	ErrExternalService  = errors.New("external service error")
)

// AppError represents an application error with context
type AppError struct {
	Type       error  // The type of error
	Message    string // User-friendly message
	StatusCode int    // HTTP status code
	Err        error  // Original error
	Stack      string // Stack trace for debugging
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s", e.Type.Error(), e.Err.Error())
	}
	return e.Type.Error()
}

// NewError creates a new AppError
func NewError(errType error, statusCode int, message string, err error) *AppError {
	pc, file, line, _ := runtime.Caller(1)
	funcName := runtime.FuncForPC(pc).Name()
	stack := fmt.Sprintf("%s:%d %s", file, line, funcName)

	// Log the error with stack trace
	logError(errType, message, err, stack)

	return &AppError{
		Type:       errType,
		Message:    message,
		StatusCode: statusCode,
		Err:        err,
		Stack:      stack,
	}
}

// Helper functions for common errors
func NewValidationError(message string, err error) *AppError {
	return NewError(ErrValidation, http.StatusBadRequest, message, err)
}

func NewAuthenticationError(message string, err error) *AppError {
	return NewError(ErrAuthentication, http.StatusUnauthorized, message, err)
}

func NewAuthorizationError(message string, err error) *AppError {
	return NewError(ErrAuthorization, http.StatusForbidden, message, err)
}

func NewNotFoundError(message string, err error) *AppError {
	return NewError(ErrNotFound, http.StatusNotFound, message, err)
}

func NewDuplicateEntryError(message string, err error) *AppError {
	return NewError(ErrDuplicateEntry, http.StatusConflict, message, err)
}

func NewDatabaseError(message string, err error) *AppError {
	return NewError(ErrDatabase, http.StatusInternalServerError, message, err)
}

func NewInternalError(message string, err error) *AppError {
	return NewError(ErrInternal, http.StatusInternalServerError, message, err)
}

func NewInvalidInputError(message string, err error) *AppError {
	return NewError(ErrInvalidInput, http.StatusBadRequest, message, err)
}

func NewExternalServiceError(message string, err error) *AppError {
	return NewError(ErrExternalService, http.StatusInternalServerError, message, err)
}

// logError logs an error with context
func logError(errType error, message string, err error, stack string) {
	// In production, this should use a proper logging framework
	log.Printf("[ERROR] Type: %s, Message: %s, Original Error: %v\nStack: %s\n", 
		errType.Error(), 
		message, 
		err, 
		stack,
	)
}
