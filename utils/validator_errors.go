package utils

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ValidationErrorToText converts validation errors to user-friendly error messages
func ValidationErrorToText(err error) string {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		var errMessages []string
		for _, e := range validationErrors {
			errMessages = append(errMessages, fieldErrorToText(e))
		}
		return strings.Join(errMessages, "; ")
	}
	return err.Error()
}

// fieldErrorToText converts a single field validation error to a user-friendly message
func fieldErrorToText(e validator.FieldError) string {
	field := strings.ToLower(e.Field())
	
	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return "Please enter a valid email address"
	case "strongpassword":
		return "Password must be at least 8 characters and include uppercase, lowercase, number, and special character"
	case "username":
		return "Username must be 3-30 characters, alphanumeric and underscore only"
	default:
		return fmt.Sprintf("%s failed on the %s tag", field, e.Tag())
	}
}
