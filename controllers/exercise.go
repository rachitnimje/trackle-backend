package controllers

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/rachitnimje/trackle-web/models"
	"github.com/rachitnimje/trackle-web/utils"
)

// Predefined categories, muscles, and equipment
var exerciseCategories = []string{
	"Strength",
	"Cardio",
	"Flexibility",
	"Balance",
	"Plyometric",
	"Powerlifting",
	"Olympic Weightlifting",
	"Calisthenics",
	"Functional",
}

var primaryMuscles = []string{
	"Chest",
	"Back",
	"Shoulders",
	"Biceps",
	"Triceps",
	"Forearms",
	"Quadriceps",
	"Hamstrings",
	"Calves",
	"Glutes",
	"Abdominals",
	"Obliques",
	"Trapezius",
	"Lats",
	"Rhomboids",
	"Deltoids",
	"Lower Back",
}

var equipmentTypes = []string{
	"Barbell",
	"Dumbbell",
	"Kettlebell",
	"Cable Machine",
	"Smith Machine",
	"Resistance Band",
	"Bodyweight",
	"Machine",
	"TRX/Suspension",
	"Medicine Ball",
	"Stability Ball",
	"Bench",
	"Pull-up Bar",
	"Treadmill",
	"Stationary Bike",
}

type CreateExerciseRequest struct {
	Name          string `json:"name" binding:"required"`
	Description   string `json:"description"`
	Category      string `json:"category"`
	PrimaryMuscle string `json:"primary_muscle"`
	Equipment     string `json:"equipment"`
}

type ExerciseResponse struct {
	ID            string `json:"id" binding:"required"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	Name          string `json:"name" binding:"required"`
	Description   string `json:"description"`
	Category      string `json:"category"`
	PrimaryMuscle string `json:"primary_muscle"`
	Equipment     string `json:"equipment"`
}

type UpdateExerciseRequest struct {
	Name          string `json:"name" binding:"required"`
	Description   string `json:"description"`
	Category      string `json:"category"`
	PrimaryMuscle string `json:"primary_muscle"`
	Equipment     string `json:"equipment"`
}

func CreateExercise(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var createExerciseRequest CreateExerciseRequest
		if err := c.ShouldBindJSON(&createExerciseRequest); err != nil {
			appErr := utils.NewValidationError("Invalid request data", err)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		if createExerciseRequest.Name == "" {
			appErr := utils.NewInvalidInputError("Exercise name is required", nil)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// check if exercise with the given name already exists
		var count int64
		if err := db.Model(&models.Exercise{}).Where("name = ?", createExerciseRequest.Name).Count(&count).Error; err != nil {
			appErr := utils.NewDatabaseError("Failed to retrieve exercise with given name", err)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		if count != 0 {
			appErr := utils.NewDuplicateEntryError("Exercise with the given name already exists", nil)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// create exercise model
		exercise := models.Exercise{
			Name:          createExerciseRequest.Name,
			Description:   createExerciseRequest.Description,
			Category:      createExerciseRequest.Category,
			PrimaryMuscle: createExerciseRequest.PrimaryMuscle,
			Equipment:     createExerciseRequest.Equipment,
		}

		// Use transaction manager for atomic operation
		var createdExercise models.Exercise
		utils.TransactionManager(db, c, func(tx *gorm.DB) error {
			// save the exercise to db
			if err := tx.Create(&exercise).Error; err != nil {
				return utils.NewDatabaseError("Failed to create exercise", err)
			}

			// Store the created exercise for response
			createdExercise = exercise
			return nil
		})

		if createdExercise.ID > 0 {
			utils.CreatedResponse(c, "Exercise created successfully", createdExercise)
		}
	}
}

func GetAllExercises(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// parse pagination parameters
		page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
		if err != nil || page < 1 {
			page = 1
		}

		limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
		if err != nil || limit < 1 || limit > 100 {
			limit = 10
		}

		offset := (page - 1) * limit

		category := c.Query("category")
		search := c.Query("search")
		muscle := c.Query("muscle")
		sortBy := c.DefaultQuery("sort", "name") // Default to name sorting

		query := db.Model(&models.Exercise{})

		// apply search, category, and muscle filters
		if category != "" {
			query = query.Where("category = ?", category)
		}
		if search != "" {
			query = query.Where("name ILIKE ?", "%"+search+"%")
		}
		if muscle != "" {
			query = query.Where("primary_muscle = ?", muscle)
		}

		var totalExercises int64
		if err := query.Count(&totalExercises).Error; err != nil {
			appErr := utils.NewDatabaseError("Failed to count exercises", err)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// Apply sorting
		switch sortBy {
		case "name":
			query = query.Order("name ASC")
		case "name_desc":
			query = query.Order("name DESC")
		case "category":
			query = query.Order("category ASC, name ASC")
		case "muscle":
			query = query.Order("primary_muscle ASC, name ASC")
		default:
			query = query.Order("name ASC") // Default fallback
		}

		var exercises []models.Exercise
		if err := query.Offset(offset).Limit(limit).Find(&exercises).Error; err != nil {
			appErr := utils.NewDatabaseError("Failed to fetch exercises", err)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		var getExercisesResponse []ExerciseResponse
		for _, exercise := range exercises {
			exerciseResponse := ExerciseResponse{
				ID:            strconv.Itoa(int(exercise.ID)),
				CreatedAt:     exercise.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				UpdatedAt:     exercise.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
				Name:          exercise.Name,
				Description:   exercise.Description,
				Category:      exercise.Category,
				PrimaryMuscle: exercise.PrimaryMuscle,
				Equipment:     exercise.Equipment,
			}
			getExercisesResponse = append(getExercisesResponse, exerciseResponse)
		}

		utils.PaginatedResponse(c, "Exercises retrieved successfully", getExercisesResponse, page, limit, totalExercises)
	}
}

func GetExercise(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		exerciseID := c.Param("id")
		if exerciseID == "" {
			appErr := utils.NewInvalidInputError("Exercise ID is required", nil)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		var exercise models.Exercise

		if err := db.Where("id = ?", exerciseID).First(&exercise).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				appErr := utils.NewNotFoundError("Exercise not found", nil)
				utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			} else {
				appErr := utils.NewDatabaseError("Failed to fetch exercise", err)
				utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			}
			return
		}

		// Map exercise model to ExerciseResponse DTO
		exerciseResponse := ExerciseResponse{
			ID:            strconv.Itoa(int(exercise.ID)),
			CreatedAt:     exercise.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:     exercise.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Name:          exercise.Name,
			Description:   exercise.Description,
			Category:      exercise.Category,
			PrimaryMuscle: exercise.PrimaryMuscle,
			Equipment:     exercise.Equipment,
		}

		utils.SuccessResponse(c, "Exercise retrieved successfully", exerciseResponse)
	}
}

func UpdateExercise(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		exerciseIDStr := c.Param("id")
		if exerciseIDStr == "" {
			appErr := utils.NewInvalidInputError("Exercise ID is required", nil)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		exerciseID, err := strconv.ParseUint(exerciseIDStr, 10, 32)
		if err != nil {
			appErr := utils.NewInvalidInputError("Invalid exercise ID", err)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// First check if the exercise exists
		var exercise models.Exercise
		if err := db.First(&exercise, exerciseID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				appErr := utils.NewNotFoundError("Exercise not found", nil)
				utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
				return
			}
			appErr := utils.NewDatabaseError("Failed to fetch exercise", err)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// Parse and validate request body
		var updateExerciseRequest UpdateExerciseRequest
		if err := c.ShouldBindJSON(&updateExerciseRequest); err != nil {
			appErr := utils.NewValidationError("Invalid request data", err)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// Check if the new name is already taken by another exercise
		if updateExerciseRequest.Name != exercise.Name {
			var count int64
			if err := db.Model(&models.Exercise{}).Where("name = ? AND id != ?", updateExerciseRequest.Name, exerciseID).Count(&count).Error; err != nil {
				appErr := utils.NewDatabaseError("Failed to check for exercise name uniqueness", err)
				utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
				return
			}

			if count > 0 {
				appErr := utils.NewDuplicateEntryError("Exercise with the given name already exists", nil)
				utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
				return
			}
		}

		// Update exercise with new values
		exercise.Name = updateExerciseRequest.Name
		exercise.Description = updateExerciseRequest.Description
		exercise.Category = updateExerciseRequest.Category
		exercise.PrimaryMuscle = updateExerciseRequest.PrimaryMuscle
		exercise.Equipment = updateExerciseRequest.Equipment

		// Use transaction manager for atomic operation
		var updatedExercise models.Exercise
		utils.TransactionManager(db, c, func(tx *gorm.DB) error {
			if err := tx.Save(&exercise).Error; err != nil {
				return utils.NewDatabaseError("Failed to update exercise", err)
			}

			updatedExercise = exercise
			return nil
		})

		if updatedExercise.ID > 0 {
			utils.SuccessResponse(c, "Exercise updated successfully", updatedExercise)
		}
	}
}

func DeleteExercise(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		exerciseIDStr := c.Param("id")
		if exerciseIDStr == "" {
			appErr := utils.NewInvalidInputError("Exercise ID is required", nil)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		exerciseID, err := strconv.ParseUint(exerciseIDStr, 10, 32)
		if err != nil {
			appErr := utils.NewInvalidInputError("Invalid exercise ID", err)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// First check if the exercise exists
		var exercise models.Exercise
		if err := db.First(&exercise, exerciseID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				appErr := utils.NewNotFoundError("Exercise not found", nil)
				utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
				return
			}
			appErr := utils.NewDatabaseError("Failed to fetch exercise", err)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// Use transaction manager for atomic operation
		utils.TransactionManager(db, c, func(tx *gorm.DB) error {
			// Delete the exercise
			if err := tx.Delete(&exercise).Error; err != nil {
				return utils.NewDatabaseError("Failed to delete exercise", err)
			}
			return nil
		})

		utils.SuccessResponse(c, "Exercise deleted successfully", nil)
	}
}

func GetExerciseCategories(c *gin.Context) {
	utils.SuccessResponse(c, "Exercise categories retrieved successfully", exerciseCategories)
}

func GetPrimaryMuscles(c *gin.Context) {
	utils.SuccessResponse(c, "Primary muscles retrieved successfully", primaryMuscles)
}

func GetEquipmentTypes(c *gin.Context) {
	utils.SuccessResponse(c, "Equipment types retrieved successfully", equipmentTypes)
}
