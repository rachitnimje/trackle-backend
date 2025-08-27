package utils

import (
	"regexp"
	"strings"
	
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// InitValidator initializes the validator with custom validations
func InitValidator() {
	// Get Gin's validator
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		// Register custom validation for strong password
		v.RegisterValidation("strongpassword", ValidateStrongPassword)
		v.RegisterValidation("username", ValidateUsername)
	}
}

// ValidateStrongPassword validates that the password meets strong password requirements
func ValidateStrongPassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	
	// Minimum length of 8 characters
	if len(password) < 8 {
		return false
	}

	// At least one uppercase letter
	uppercaseRegex := regexp.MustCompile(`[A-Z]`)
	if !uppercaseRegex.MatchString(password) {
		return false
	}

	// At least one lowercase letter
	lowercaseRegex := regexp.MustCompile(`[a-z]`)
	if !lowercaseRegex.MatchString(password) {
		return false
	}

	// At least one number
	numberRegex := regexp.MustCompile(`[0-9]`)
	if !numberRegex.MatchString(password) {
		return false
	}

	// At least one special character
	specialCharRegex := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`)
	return specialCharRegex.MatchString(password)
}

// ValidateUsername validates that the username meets requirements
func ValidateUsername(fl validator.FieldLevel) bool {
	username := fl.Field().String()
	
	// Username should be 3-30 characters, alphanumeric and underscore only
	if len(username) < 3 || len(username) > 30 {
		return false
	}

	usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	return usernameRegex.MatchString(username)
}

func IsValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

func TrimAndLower(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
