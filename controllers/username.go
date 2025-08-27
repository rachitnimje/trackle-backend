package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rachitnimje/trackle-web/models"
	"gorm.io/gorm"
)

// CheckUsernameAvailability checks if a username is available
func CheckUsernameAvailability(db *gorm.DB) gin.HandlerFunc {
       return func(c *gin.Context) {
               username := c.Query("username")
               if username == "" {
                       c.JSON(http.StatusBadRequest, gin.H{"available": false, "message": "Username is required"})
                       return
               }
               var user models.ProfileUser
               if err := db.Where("username = ?", username).First(&user).Error; err == nil {
                       c.JSON(http.StatusOK, gin.H{"available": false, "message": "Username is already taken"})
                       return
               }
               c.JSON(http.StatusOK, gin.H{"available": true, "message": "Username is available"})
       }
}