package utils

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TransactionManager handles database transactions with proper error handling
func TransactionManager(db *gorm.DB, c *gin.Context, fn func(*gorm.DB) error) {
	tx := db.Begin()
	
	// Ensure rollback on panic
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			// Convert panic to error response
			appErr := NewInternalError("Internal server error", nil)
			ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
		}
	}()

	// Execute the transaction function
	err := fn(tx)
	if err != nil {
		tx.Rollback()
		
		// The error is already handled in the function
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		appErr := NewDatabaseError("Failed to commit transaction", err)
		ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
	}
}
