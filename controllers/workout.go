package controllers

import (
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/rachitnimje/trackle-web/models"
	"github.com/rachitnimje/trackle-web/utils"
)

type CreateWorkoutRequest struct {
	Name       string                `json:"name" binding:"required"`
	TemplateID uint                  `json:"template_id" binding:"required"`
	Notes      string                `json:"notes"`
	Status     string                `json:"status"`
	Entries    []WorkoutEntryRequest `json:"entries" binding:"required,min=1"`
}

type WorkoutEntryRequest struct {
	ExerciseID uint    `json:"exercise_id" binding:"required"`
	SetNumber  int     `json:"set_number" binding:"required,min=1"`
	Reps       int     `json:"reps" binding:"required,min=1"`
	Weight     float64 `json:"weight" binding:"min=0"`
}

type UserWorkoutsResponse struct {
	 ID    uint      `json:"id" binding:"required"`
	 Name  string    `json:"name" binding:"required"`
	 TemplateID   uint      `json:"template_id" binding:"required"`
	 TemplateName string    `json:"template_name" binding:"required"`
	 CreatedAt     time.Time `json:"created_at" binding:"required"`
	 Notes        string    `json:"notes"`
	 Status       string    `json:"status"`
}

type UserWorkoutResponse struct {
	 ID           string                     `json:"id"`
	 Name  string                     `json:"name" binding:"required"`
	 CreatedAt time.Time `json:"created_at" binding:"required"`
	 UpdatedAt time.Time `json:"updated_at"`
	 TemplateID   uint                       `json:"template_id" binding:"required"`
	 TemplateName string                     `json:"template_name" binding:"required"`
	 Notes        string                     `json:"notes"`
	 Status       string                     `json:"status"`
	 Entries      []UserWorkoutEntryResponse `json:"entries" binding:"required"`
}

type UserWorkoutEntryResponse struct {
	ExerciseID   uint    `json:"exercise_id" binding:"required"`
	ExerciseName string  `json:"exercise_name" binding:"required"`
	SetNumber    int     `json:"set_number" binding:"required"`
	Reps         int     `json:"reps" binding:"required"`
	Weight       float64 `json:"weight" binding:"required"`
}

type UpdateWorkoutRequest struct {
	Name       string                `json:"name" binding:"required"`
	TemplateID uint                  `json:"template_id" binding:"required"`
	Notes      string                `json:"notes"`
	Entries    []WorkoutEntryRequest `json:"entries" binding:"required,min=1"`
	Status     string                `json:"status"`
}

func CreateUserWorkout(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// extract user_id from context
		userID, exists := c.Get("user_id")
		if !exists {
			appErr := utils.NewAuthenticationError("User not authenticated", nil)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// bind and validate request
		var req CreateWorkoutRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			appErr := utils.NewValidationError("Invalid request data", err)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// check if the template belongs to the user
		var template models.Template
		if err := db.Where("id = ? and user_id = ?", req.TemplateID, userID).
			First(&template).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				appErr := utils.NewNotFoundError("Template not found or access denied", nil)
				utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			} else {
				appErr := utils.NewDatabaseError("Failed to verify template", err)
				utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			}
			return
		}

		// Extract unique exercise IDs using map for verification
		exerciseIDMap := make(map[uint]bool)
		for _, entry := range req.Entries {
			exerciseIDMap[entry.ExerciseID] = true
		}

		var exerciseIDs []uint
		for id := range exerciseIDMap {
			exerciseIDs = append(exerciseIDs, id)
		}

		// Verify all exercises exist
		var exerciseCount int64
		if err := db.Model(&models.Exercise{}).
			Where("id IN ?", exerciseIDs).
			Count(&exerciseCount).Error; err != nil {
			appErr := utils.NewDatabaseError("Failed to verify exercises", err)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		if int(exerciseCount) != len(exerciseIDs) {
			appErr := utils.NewInvalidInputError("One or more exercise IDs are invalid", nil)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// Use our transaction manager for better error handling
		utils.TransactionManager(db, c, func(tx *gorm.DB) error {
			// save the workout to db
			workout := models.Workout{
				Name:       req.Name,
				UserID:     userID.(uint),
				TemplateID: req.TemplateID,
				Notes:      req.Notes,
				Status:     req.Status,
			}

			if err := tx.Create(&workout).Error; err != nil {
				return utils.NewDatabaseError("Failed to create workout", err)
			}

			// save the workout entries to db
			var workoutEntries []models.WorkoutEntry
			for _, r := range req.Entries {
				workoutEntries = append(workoutEntries, models.WorkoutEntry{
					WorkoutID:  workout.ID,
					ExerciseID: r.ExerciseID,
					SetNumber:  r.SetNumber,
					Reps:       r.Reps,
					Weight:     r.Weight,
				})
			}

			if err := tx.Create(&workoutEntries).Error; err != nil {
				return utils.NewDatabaseError("Failed to create workout entries", err)
			}

			// Load the created workout with entries for response
			var createdWorkout models.Workout
			if err := tx.Preload("Entries.Exercise").First(&createdWorkout, workout.ID).Error; err != nil {
				return utils.NewDatabaseError("Failed to load created workout", err)
			}

			// Prepare response data outside the transaction
			c.Set("created_workout", createdWorkout)
			return nil
		})

		// Get workout from context if available
		createdWorkout, exists := c.Get("created_workout")
		if exists {
			utils.CreatedResponse(c, "Workout created successfully", createdWorkout)
		} else {
			utils.CreatedResponse(c, "Workout created successfully", nil)
		}
	}
}

func GetAllUserWorkouts(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// extract user_id from context
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

		// Get search and filter parameters
		search := c.Query("search")
		templateID := c.Query("template_id")

		// Start building the query
		query := db.Model(&models.Workout{}).Where("user_id = ?", userID)

		// Apply filters
		if search != "" {
			query = query.Where("name ILIKE ? OR notes ILIKE ?", "%"+search+"%", "%"+search+"%")
		}

		if templateID != "" {
			query = query.Where("template_id = ?", templateID)
		}

		// Count total workouts with applied filters
		var total int64
		if err := query.Count(&total).Error; err != nil {
			appErr := utils.NewDatabaseError("Failed to count workouts", err)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// Return early if no workouts found
		if total == 0 {
			utils.PaginatedResponse(c, "Workouts retrieved successfully", []UserWorkoutsResponse{}, page, limit, total)
			return
		}

		// Fetch workouts with pagination and filters
		var workouts []models.Workout
		if err := query.Preload("Template").
			Order("created_at DESC").
			Offset(offset).
			Limit(limit).
			Find(&workouts).Error; err != nil {
			appErr := utils.NewDatabaseError("Failed to retrieve workouts", err)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// Create response directly from preloaded data
		var userWorkoutsResponse []UserWorkoutsResponse
		for _, workout := range workouts {
			userWorkoutsResponse = append(userWorkoutsResponse, UserWorkoutsResponse{
				ID:    workout.ID,
				Name:  workout.Name,
				TemplateID:   workout.TemplateID,
				TemplateName: workout.Template.Name, 
				CreatedAt:     workout.CreatedAt,
				Notes:        workout.Notes,
				Status:       workout.Status,
			})
		}

		utils.PaginatedResponse(c, "Workouts retrieved successfully", userWorkoutsResponse, page, limit, total)
	}
}

func GetUserWorkout(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// extract user_id from context
		userID, exists := c.Get("user_id")
		if !exists {
			appErr := utils.NewAuthenticationError("User not authenticated", nil)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// parse the workout ID with validation
		workoutIDStr := c.Param("id")
		if workoutIDStr == "" {
			appErr := utils.NewInvalidInputError("Workout ID is required", nil)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		workoutID, err := strconv.ParseUint(workoutIDStr, 10, 32)
		if err != nil || workoutID == 0 {
			appErr := utils.NewInvalidInputError("Invalid workout ID", err)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// Fetch the workout with template in a single query
		var workout models.Workout
		if err := db.Where("id = ? AND user_id = ?", workoutID, userID).
			Preload("Template").
			First(&workout).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				appErr := utils.NewNotFoundError("Workout not found or access denied", nil)
				utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			} else {
				appErr := utils.NewDatabaseError("Failed to retrieve workout", err)
				utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			}
			return
		}

		// Fetch all workout entries with their exercises in a single query
		var workoutEntries []models.WorkoutEntry
		if err := db.Where("workout_id = ?", workoutID).
			Preload("Exercise").
			Find(&workoutEntries).Error; err != nil {
			appErr := utils.NewDatabaseError("Failed to retrieve workout entries", err)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// Map the entries to the response structure
		var workoutEntriesResponse []UserWorkoutEntryResponse
		for _, entry := range workoutEntries {
			workoutEntriesResponse = append(workoutEntriesResponse, UserWorkoutEntryResponse{
				ExerciseID:   entry.ExerciseID,
				ExerciseName: entry.Exercise.Name, // Access the preloaded Exercise data
				SetNumber:    entry.SetNumber,
				Reps:         entry.Reps,
				Weight:       entry.Weight,
			})
		}

		// Build the final response
		response := UserWorkoutResponse{
			ID:           strconv.Itoa(int(workout.ID)),
			CreatedAt: workout.CreatedAt,
			UpdatedAt: workout.UpdatedAt,
			TemplateID:   workout.TemplateID,
			TemplateName: workout.Template.Name, // Access the preloaded Template data
			Name:  workout.Name,
			Notes:        workout.Notes,
			Status:       workout.Status,
			Entries:      workoutEntriesResponse,
		}

		utils.SuccessResponse(c, "Workout retrieved successfully", response)
	}
}



func UpdateUserWorkout(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// extract user_id from context
		userID, exists := c.Get("user_id")
		if !exists {
			appErr := utils.NewAuthenticationError("User not authenticated", nil)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// parse the workout ID with validation
		workoutIDStr := c.Param("id")
		if workoutIDStr == "" {
			appErr := utils.NewInvalidInputError("Workout ID is required", nil)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		workoutID, err := strconv.ParseUint(workoutIDStr, 10, 32)
		if err != nil || workoutID == 0 {
			appErr := utils.NewInvalidInputError("Invalid workout ID", err)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// Check if workout exists and belongs to user
		var workout models.Workout
		if err := db.Where("id = ? AND user_id = ?", workoutID, userID).First(&workout).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				appErr := utils.NewNotFoundError("Workout not found or access denied", nil)
				utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			} else {
				appErr := utils.NewDatabaseError("Failed to retrieve workout", err)
				utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			}
			return
		}

		// bind and validate request
		var req UpdateWorkoutRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			appErr := utils.NewValidationError("Invalid request data", err)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// check if the template belongs to the user
		var template models.Template
		if err := db.Where("id = ? and user_id = ?", req.TemplateID, userID).
			First(&template).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				appErr := utils.NewNotFoundError("Template not found or access denied", nil)
				utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			} else {
				appErr := utils.NewDatabaseError("Failed to verify template", err)
				utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			}
			return
		}

		// Extract unique exercise IDs using map for verification
		exerciseIDMap := make(map[uint]bool)
		for _, entry := range req.Entries {
			exerciseIDMap[entry.ExerciseID] = true
		}

		var exerciseIDs []uint
		for id := range exerciseIDMap {
			exerciseIDs = append(exerciseIDs, id)
		}

		// Verify all exercises exist
		var exerciseCount int64
		if err := db.Model(&models.Exercise{}).
			Where("id IN ?", exerciseIDs).
			Count(&exerciseCount).Error; err != nil {
			appErr := utils.NewDatabaseError("Failed to verify exercises", err)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		if int(exerciseCount) != len(exerciseIDs) {
			appErr := utils.NewInvalidInputError("One or more exercise IDs are invalid", nil)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// Use our transaction manager for better error handling
		utils.TransactionManager(db, c, func(tx *gorm.DB) error {
			// Update the workout
			workout.Name = req.Name
			workout.TemplateID = req.TemplateID
			workout.Notes = req.Notes
			if req.Status != "" {
				workout.Status = req.Status
			}
			if err := tx.Save(&workout).Error; err != nil {
				return utils.NewDatabaseError("Failed to update workout", err)
			}

			// Delete existing workout entries
			if err := tx.Where("workout_id = ?", workout.ID).Delete(&models.WorkoutEntry{}).Error; err != nil {
				return utils.NewDatabaseError("Failed to delete existing workout entries", err)
			}

			// Create new workout entries
			var workoutEntries []models.WorkoutEntry
			for _, r := range req.Entries {
				workoutEntries = append(workoutEntries, models.WorkoutEntry{
					WorkoutID:  workout.ID,
					ExerciseID: r.ExerciseID,
					SetNumber:  r.SetNumber,
					Reps:       r.Reps,
					Weight:     r.Weight,
				})
			}

			if err := tx.Create(&workoutEntries).Error; err != nil {
				return utils.NewDatabaseError("Failed to create workout entries", err)
			}

			// Load the updated workout with entries for response
			var updatedWorkout models.Workout
			if err := tx.Preload("Entries.Exercise").First(&updatedWorkout, workout.ID).Error; err != nil {
				return utils.NewDatabaseError("Failed to load updated workout", err)
			}

			// Prepare response data outside the transaction
			c.Set("updated_workout", updatedWorkout)
			return nil
		})

		// Get workout from context if available
		updatedWorkout, exists := c.Get("updated_workout")
		if exists {
			utils.SuccessResponse(c, "Workout updated successfully", updatedWorkout)
		} else {
			utils.SuccessResponse(c, "Workout updated successfully", nil)
		}
	}
}

func DeleteUserWorkout(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// retrieve the user id from context
		userID, exists := c.Get("user_id")
		if !exists {
			appErr := utils.NewAuthenticationError("User not authenticated", nil)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// retrieve the workout id from params and validate
		workoutIDStr := c.Param("id")
		if workoutIDStr == "" {
			appErr := utils.NewInvalidInputError("Workout ID is required", nil)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		workoutID, err := strconv.ParseUint(workoutIDStr, 10, 32)
		if err != nil {
			appErr := utils.NewInvalidInputError("Invalid workout ID", nil)
			utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			return
		}

		// Check if workout exists and belongs to user
		var workout models.Workout
		if err := db.Where("id = ? AND user_id = ?", workoutID, userID).First(&workout).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				appErr := utils.NewNotFoundError("Workout not found", nil)
				utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			} else {
				appErr := utils.NewDatabaseError("Failed to find workout", err)
				utils.ErrorResponse(c, appErr.StatusCode, appErr.Message, appErr)
			}
			return
		}

		utils.TransactionManager(db, c, func(tx *gorm.DB) error {
			// Delete workout entries first
			if err := tx.Where("workout_id = ?", workoutID).Delete(&models.WorkoutEntry{}).Error; err != nil {
				return utils.NewDatabaseError("Failed to delete workout entries", err)
			}

			// Delete workout
			if err := tx.Delete(&workout).Error; err != nil {
				return utils.NewDatabaseError("Failed to delete workout", err)
			}

			return nil
		})

		utils.SuccessResponse(c, "Workout deleted successfully", nil)
	}
}
