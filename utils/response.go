package utils

import (
	"errors"
	"log"
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/rachitnimje/trackle-web/models"
)

func SuccessResponse(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func CreatedResponse(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusCreated, models.APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func ErrorResponse(c *gin.Context, statusCode int, message string, err error) {
	// For debugging only - log the full error
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		log.Printf("[DEBUG] Error at %s:%d: %v", file, line, err)
	}
	
	// Check if it's our custom AppError
	var appError *AppError
	if errors.As(err, &appError) {
		// Use the status code and message from AppError
		c.JSON(appError.StatusCode, models.APIResponse{
			Success: false,
			Message: appError.Message,
			Error:   appError.Type.Error(), // Only return the error type, not the full error details
		})
		return
	}
	
	// For regular errors, don't expose details to clients in production
	errorMsg := ""
	if err != nil && gin.Mode() == gin.DebugMode {
		// Only in debug mode, return error details
		errorMsg = err.Error()
	}

	c.JSON(statusCode, models.APIResponse{
		Success: false,
		Message: message,
		Error:   errorMsg,
	})
}

func PaginatedResponse(c *gin.Context, message string, data interface{}, page, limit int, totalItems int64) {
	var totalPages int64

	if totalItems > 0 {
		totalPages = (totalItems + int64(limit) - 1) / int64(limit)
	} else {
		totalPages = 0
	}

	hasNext := int64(page) < totalPages
	hasPrev := page > 1

	c.JSON(http.StatusOK, models.PaginatedResponse{
		Success:    true,
		Message:    message,
		Data:       data,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
		Total:      totalItems,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
	})
}
