package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/rachitnimje/trackle-web/models"
	"github.com/rachitnimje/trackle-web/utils"
)

type CreateTemplateRequest struct {
	Name        string                          `json:"name" binding:"required"`
	Description string                          `json:"description"`
	Exercises   []CreateTemplateExerciseRequest `json:"exercises" binding:"required,min=1"`
}

type CreateTemplateExerciseRequest struct {
	ExerciseID uint `json:"exercise_id" binding:"required"`
	Sets       int  `json:"sets" binding:"required,min=1"`
}

type GetAllTemplatesResponse struct {
	ID          string `json:"id"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type TemplateResponse struct {
	ID          string                     `json:"id"`
	CreatedAt   string                     `json:"created_at"`
	UpdatedAt   string                     `json:"updated_at"`
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	UserID      uint                       `json:"user_id"`
	Exercises   []TemplateExerciseResponse `json:"exercises"`
}

type TemplateExerciseResponse struct {
	ExerciseID  uint   `json:"exercise_id"`
	Sets        int    `json:"sets"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

func CreateUserTemplate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// extract the user_id from context
		userID, exists := c.Get("user_id")
		if !exists {
			appErr := utils.NewAuthenticationError("User not authenticated", nil)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// extract the request body from context
		var req CreateTemplateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			appErr := utils.NewValidationError("Invalid request data", err)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// TODO: check for duplicate names in the existing table

		// get the non-duplicate exercise ids from request
		exerciseIDMap := make(map[uint]bool)
		var exerciseIDs []uint
		for _, e := range req.Exercises {
			if exerciseIDMap[e.ExerciseID] {
				utils.ErrorResponse(c, http.StatusBadRequest, "Duplicate exercise IDs not allowed", nil)
				return
			}
			exerciseIDMap[e.ExerciseID] = true
			exerciseIDs = append(exerciseIDs, e.ExerciseID)
		}

		// verify all exercises exist
		var exerciseCount int64
		if err := db.Model(&models.Exercise{}).Where("id IN ?", exerciseIDs).Count(&exerciseCount).Error; err != nil {
			appErr := utils.NewDatabaseError("Failed to verify exercises", err)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		if int(exerciseCount) != len(exerciseIDs) {
			appErr := utils.NewInvalidInputError("One or more exercise IDs are invalid", nil)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// transaction manager for atomic operation
		var createdTemplateID uint
		utils.TransactionManager(db, c, func(tx *gorm.DB) error {
			// create template
			template := models.Template{
				Name:        req.Name,
				UserID:      userID.(uint),
				Description: req.Description,
			}

			if err := tx.Create(&template).Error; err != nil {
				return utils.NewDatabaseError("Failed to create template", err)
			}

			createdTemplateID = template.ID

			// create template exercises for batch insert
			var templateExercises []models.TemplateExercise
			for _, e := range req.Exercises {
				templateExercise := models.TemplateExercise{
					TemplateID: template.ID,
					ExerciseID: e.ExerciseID,
					Sets:       e.Sets,
				}
				templateExercises = append(templateExercises, templateExercise)
			}

			// batch create template exercises
			if err := tx.Create(&templateExercises).Error; err != nil {
				return utils.NewDatabaseError("Failed to create template exercise", err)
			}

			return nil
		})

		// Check if we need to reload template data for response
		if createdTemplateID > 0 {
			var template models.Template
			if err := db.Preload("Exercises.Exercise").First(&template, createdTemplateID).Error; err != nil {
				appErr := utils.NewDatabaseError("Failed to load template", err)
				utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
				return
			}

			utils.CreatedResponse(c, "Template created successfully", template)
		} else {
			utils.CreatedResponse(c, "Template created successfully", nil)
		}
	}
}

func GetAllUserTemplates(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		//extract the user_id from context
		userID, exists := c.Get("user_id")
		if !exists {
			appErr := utils.NewAuthenticationError("User not authenticated", nil)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// Parse pagination parameters
		page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
		if err != nil || page < 1 {
			page = 1
		}

		limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
		if err != nil || limit < 1 || limit > 100 {
			limit = 10
		}

		offset := (page - 1) * limit

		var templates []models.Template
		var total int64

		// Build query with user_id filter
		query := db.Model(&models.Template{}).Where("user_id = ?", userID)

		// Get search parameter
		search := c.Query("search")

		if search != "" {
			query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
		}

		// Count total templates for this user with applied filters
		if err := query.Count(&total).Error; err != nil {
			appErr := utils.NewDatabaseError("Failed to count templates", err)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// Return early if no template found
		if total == 0 {
			utils.PaginatedResponse(c, "Templates retrieved successfully", []models.Template{}, page, limit, total)
			return
		}

		if err := query.Preload("Exercises.Exercise").
			Order("created_at DESC").
			Offset(offset).
			Limit(limit).
			Find(&templates).Error; err != nil {
			appErr := utils.NewDatabaseError("Failed to fetch templates", err)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		var response []GetAllTemplatesResponse

		for _, template := range templates {
			response = append(response, GetAllTemplatesResponse{
				ID:          strconv.Itoa(int(template.ID)),
				CreatedAt:   template.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				UpdatedAt:   template.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
				Name:        template.Name,
				Description: template.Description,
			})
		}

		utils.PaginatedResponse(c, "Templates retrieved successfully", response, page, limit, total)
	}
}

func GetUserTemplate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// extract the user_id from context
		userID, exists := c.Get("user_id")
		if !exists {
			appErr := utils.NewAuthenticationError("User not authenticated", nil)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// parse the template ID with validation
		templateIDStr := c.Param("id")
		if templateIDStr == "" {
			appErr := utils.NewInvalidInputError("Template ID is required", nil)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		templateID, err := strconv.ParseUint(templateIDStr, 10, 32)
		if err != nil || templateID == 0 {
			appErr := utils.NewInvalidInputError("Invalid template ID", err)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		var template models.Template

		if err := db.Where("id = ? AND user_id = ?", templateIDStr, userID).
			Preload("Exercises.Exercise").
			First(&template).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				appErr := utils.NewNotFoundError("Template not found", nil)
				utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			} else {
				appErr := utils.NewDatabaseError("Failed to fetch template", err)
				utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			}
			return
		}

		response := TemplateResponse{
			ID:          strconv.Itoa(int(template.ID)),
			CreatedAt:   template.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   template.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Name:        template.Name,
			Description: template.Description,
			UserID:      template.UserID,
			Exercises:   make([]TemplateExerciseResponse, len(template.Exercises)),
		}

		for i, templateExercise := range template.Exercises {
			response.Exercises[i] = TemplateExerciseResponse{
				ExerciseID:  templateExercise.ExerciseID,
				Sets:        templateExercise.Sets,
				Name:        templateExercise.Exercise.Name,
				Description: templateExercise.Exercise.Description,
				Category:    templateExercise.Exercise.Category,
			}
		}

		utils.SuccessResponse(c, "Template retrieved successfully", response)
	}
}

func DeleteUserTemplate(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// extract the user_id from context
		userID, exists := c.Get("user_id")
		if !exists {
			appErr := utils.NewAuthenticationError("User not authenticated", nil)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		templateIDStr := c.Param("id")
		if templateIDStr == "" {
			appErr := utils.NewInvalidInputError("Template ID is required", nil)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		templateID, err := strconv.ParseUint(templateIDStr, 10, 32)
		if err != nil {
			appErr := utils.NewInvalidInputError("Invalid template ID", err)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// First check if template exists and belongs to user
		var template models.Template
		if err := db.Where("id = ? AND user_id = ?", templateID, userID).First(&template).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				appErr := utils.NewNotFoundError("Template not found", nil)
				utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			} else {
				appErr := utils.NewDatabaseError("Failed to find template", err)
				utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			}
			return
		}

		// transaction manager to handle the deletion atomically
		utils.TransactionManager(db, c, func(tx *gorm.DB) error {
			// First delete template exercises
			if err := tx.Where("template_id = ?", templateID).Delete(&models.TemplateExercise{}).Error; err != nil {
				return utils.NewDatabaseError("Failed to delete template exercises", err)
			}

			// Then delete the template
			if err := tx.Delete(&template).Error; err != nil {
				return utils.NewDatabaseError("Failed to delete template", err)
			}

			return nil
		})

		utils.SuccessResponse(c, "Template deleted successfully", nil)
	}
}
