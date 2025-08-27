package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/rachitnimje/trackle-web/models"
	"github.com/rachitnimje/trackle-web/utils"
)

type RegisterRequest struct {
	Username string `json:"username" binding:"required,username"`
	FullName string `json:"full_name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,strongpassword"`
	Role     string `json:"role"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RegisterResponse struct {
	ProfileUser models.ProfileUser `json:"user"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type UserResponse struct {
	ID string `json:"id" binding:"required"`
	CreatedAt string `json:"created_at" binding:"required"`
	Email string `json:"email" binding:"required"`
	Username string `json:"username" binding:"required"`
}

type GoogleOAuthRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required"`
	GoogleID string `json:"googleId" binding:"required"`
	Name     string `json:"name"`
	Picture  string `json:"picture"`
}

func Register(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			errorMsg := utils.ValidationErrorToText(err)
			appErr := utils.NewValidationError(errorMsg, err)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// Email already validated by the binding tag
		// Username already validated by the binding tag
		// Password already validated by the binding tag

		role := req.Role
		if role == "" {
			role = "user"
		}

		// Check if user already exists
		var existingProfile models.ProfileUser
		if err := db.Where("email = ? OR username = ?", utils.TrimAndLower(req.Email), req.Username).First(&existingProfile).Error; err == nil {
			// Check which field is duplicated for a more specific message
			var conflictField string
			if existingProfile.Email == utils.TrimAndLower(req.Email) {
				conflictField = "email"
			} else if existingProfile.Username == req.Username {
				conflictField = "username"
			} else {
				conflictField = "account"
			}
			var userMsg string
			switch conflictField {
			case "email":
				userMsg = "This email is already registered. Please use a different email."
			case "username":
				userMsg = "This username is already taken. Please choose another username."
			default:
				userMsg = "An account with these details already exists."
			}
			appErr := utils.NewDuplicateEntryError(userMsg, nil)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// Hash password
	    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	    if err != nil {
		    appErr := utils.NewInternalError("Failed to process password", err)
		    utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
		    return
	    }

	       profileUser := models.ProfileUser{
		       Username: req.Username,
		       Email: utils.TrimAndLower(req.Email),
		       Name: req.FullName,
		       Role: role,
		       PasswordLess: false,
	       }

	       var createdProfile models.ProfileUser
	       utils.TransactionManager(db, c, func(tx *gorm.DB) error {
		       if err := tx.Create(&profileUser).Error; err != nil {
			       return utils.NewDatabaseError("Failed to create profile user", err)
		       }
		       authUser := models.AuthUser{
			       UserID: profileUser.ID,
			       Password: string(hashedPassword),
		       }
		       if err := tx.Create(&authUser).Error; err != nil {
			       return utils.NewDatabaseError("Failed to create auth user", err)
		       }
		       createdProfile = profileUser
		       return nil
	       })

	    if createdProfile.ID > 0 {
				response := RegisterResponse{
			    ProfileUser: createdProfile,
		    }
		    utils.CreatedResponse(c, "User created successfully", response)
	    }
	}
}

func Login(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			errorMsg := utils.ValidationErrorToText(err)
			appErr := utils.NewValidationError(errorMsg, err)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// Find user by email
	       var profileUser models.ProfileUser
	       if err := db.Where("email = ?", utils.TrimAndLower(req.Email)).First(&profileUser).Error; err != nil {
		       appErr := utils.NewAuthenticationError("Invalid credentials", nil)
		       utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
		       return
	       }
	       var authUser models.AuthUser
	       if err := db.Where("user_id = ?", profileUser.ID).First(&authUser).Error; err != nil {
		       appErr := utils.NewAuthenticationError("Invalid credentials", nil)
		       utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
		       return
	       }
	       // Verify password
	       if err := bcrypt.CompareHashAndPassword([]byte(authUser.Password), []byte(req.Password)); err != nil {
		       appErr := utils.NewAuthenticationError("Invalid credentials", nil)
		       utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
		       return
	       }
	       // Generate JWT token
	       token, err := utils.GenerateJWT(profileUser.ID)
	       if err != nil {
		       appErr := utils.NewInternalError("Failed to generate token", err)
		       utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
		       return
	       }
	       // Set cookie
	       c.SetSameSite(http.SameSiteLaxMode)
	       c.SetCookie("auth_token", token, 24*60*60, "/", "", false, true) // 24 hours
	       response := LoginResponse{
		       Token: token,
	       }
	       utils.SuccessResponse(c, "Login successful", response)
	}
}

func Logout() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Clear the auth cookie
		c.SetCookie("auth_token", "", -1, "/", "", false, true)
		utils.SuccessResponse(c, "Logged out successfully", nil)
	}
}

func Me(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			appErr := utils.NewAuthenticationError("User not authenticated", nil)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

	       var profileUser models.ProfileUser
	       if err := db.First(&profileUser, userID).Error; err != nil {
		       appErr := utils.NewNotFoundError("User not found", err)
		       utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
		       return
	       }

	       userResponse := &UserResponse{
		       ID:          strconv.Itoa(int(profileUser.ID)),
		       CreatedAt:   profileUser.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		       Username: profileUser.Username,
		       Email: profileUser.Email,
	       }

		utils.SuccessResponse(c, "User retrieved successfully", userResponse)
	}
}

func GoogleOAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req GoogleOAuthRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			errorMsg := utils.ValidationErrorToText(err)
			appErr := utils.NewValidationError(errorMsg, err)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

	       // Check if user already exists by email or Google ID
	       var existingProfile models.ProfileUser
	       result := db.Where("email = ? OR google_id = ?", req.Email, req.GoogleID).First(&existingProfile)
	       if result.Error == nil {
		       // User exists, generate token and return
		       token, err := utils.GenerateJWT(existingProfile.ID)
		       if err != nil {
			       utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate token", err)
			       return
		       }
		       utils.SuccessResponse(c, "Google OAuth login successful", map[string]interface{}{
			       "token": token,
			       "user":  existingProfile,
		       })
		       return
	       }
	       // Check if user with this username already exists
	       var usernameCheck models.ProfileUser
	       if db.Where("username = ?", req.Username).First(&usernameCheck).Error == nil {
		       // Username exists, modify it to make it unique
		       req.Username = req.Username + "_" + req.GoogleID[:8]
	       }
	       // Create new profile user
	       profileUser := models.ProfileUser{
		       Username: req.Username,
		       Email: req.Email,
		       Name: req.Name,
		       GoogleID: &req.GoogleID,
		       Role: "user",
		       PasswordLess: true,
	       }
	       if err := db.Create(&profileUser).Error; err != nil {
		       utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to create user", err)
		       return
	       }
	       // Generate JWT token
	       token, err := utils.GenerateJWT(profileUser.ID)
	       if err != nil {
		       utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate token", err)
		       return
	       }
	       utils.SuccessResponse(c, "Google OAuth registration successful", map[string]interface{}{
		       "token": token,
		       "user":  profileUser,
	       })
	}
}
