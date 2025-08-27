package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rachitnimje/trackle-web/models"
	"github.com/rachitnimje/trackle-web/utils"
	"gorm.io/gorm"
)

// GetWorkoutStats returns statistics about workout frequency
func GetWorkoutStats(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from token
		userID, exists := c.Get("userID")
		if !exists {
			utils.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated", nil)
			return
		}

		// Get time range from query params (default to month)
		timeRange := c.DefaultQuery("timeRange", "month")

		// Determine the start date based on time range
		var startDate time.Time
		now := time.Now()
		
		switch timeRange {
		case "week":
			startDate = now.AddDate(0, 0, -7)
		case "year":
			startDate = now.AddDate(-1, 0, 0)
		default: // month
			startDate = now.AddDate(0, -1, 0)
		}

		// Query to get workout counts by date
		var results []struct {
			Date  time.Time
			Count int
		}

		// Format date according to time range
		var dateFormat string
		var labels []string
		
		switch timeRange {
		case "week":
			dateFormat = "Mon" // Day of week
		case "year":
			dateFormat = "Jan" // Month
		default: // month
			dateFormat = "Jan 02" // Month and day
		}

		// Get workouts grouped by date
		query := db.Model(&models.Workout{}).
			Select("DATE(created_at) as date, COUNT(*) as count").
			Where("user_id = ? AND created_at >= ?", userID, startDate).
			Group("DATE(created_at)").
			Order("date")
		
		if err := query.Find(&results).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch workout statistics", err)
			return
		}

		// Prepare data for response
		var data []int
		dateMap := make(map[string]int)

		// Process results
		for _, r := range results {
			formattedDate := r.Date.Format(dateFormat)
			dateMap[formattedDate] = r.Count
		}

		// Generate complete date range based on time range
		var allDates []time.Time
		
		switch timeRange {
		case "week":
			// Last 7 days
			for i := 6; i >= 0; i-- {
				allDates = append(allDates, now.AddDate(0, 0, -i))
			}
		case "year":
			// Last 12 months
			for i := 11; i >= 0; i-- {
				allDates = append(allDates, now.AddDate(0, -i, 0))
			}
		default: // month
			// Last 30 days
			for i := 29; i >= 0; i-- {
				allDates = append(allDates, now.AddDate(0, 0, -i))
			}
		}

		// Fill in the labels and data
		for _, date := range allDates {
			formattedDate := date.Format(dateFormat)
			labels = append(labels, formattedDate)
			count, exists := dateMap[formattedDate]
			if exists {
				data = append(data, count)
			} else {
				data = append(data, 0)
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"labels": labels,
			"data":   data,
		})
	}
}

// GetExerciseProgress returns the progress of a specific exercise over time
func GetExerciseProgress(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from token
		userID, exists := c.Get("userID")
		if !exists {
			utils.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated", nil)
			return
		}

		// Get exercise ID from URL
		exerciseID := c.Param("id")
		if exerciseID == "" {
			utils.ErrorResponse(c, http.StatusBadRequest, "Exercise ID is required", nil)
			return
		}

		// Get time range from query params (default to month)
		timeRange := c.DefaultQuery("timeRange", "month")

		// Determine the start date based on time range
		var startDate time.Time
		now := time.Now()
		
		switch timeRange {
		case "week":
			startDate = now.AddDate(0, 0, -7)
		case "year":
			startDate = now.AddDate(-1, 0, 0)
		default: // month
			startDate = now.AddDate(0, -1, 0)
		}

		// Query to get exercise data
		type ExerciseData struct {
			Date   time.Time
			Weight float64
		}
		var results []ExerciseData

		// Query to get the max weight for each date for the specified exercise
		query := db.Table("workout_exercises").
			Select("DATE(workout_exercises.created_at) as date, MAX(workout_exercises.weight) as weight").
			Joins("JOIN workouts ON workout_exercises.workout_id = workouts.id").
			Where("workouts.user_id = ? AND workout_exercises.exercise_id = ? AND workout_exercises.created_at >= ?", 
				userID, exerciseID, startDate).
			Group("DATE(workout_exercises.created_at)").
			Order("date")
		
		if err := query.Find(&results).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch exercise progress", err)
			return
		}

		// Prepare data for response
		var dates []string
		var weights []float64

		for _, r := range results {
			dates = append(dates, r.Date.Format("2006-01-02"))
			weights = append(weights, r.Weight)
		}

		c.JSON(http.StatusOK, gin.H{
			"dates":   dates,
			"weights": weights,
		})
	}
}

// GetAggregateStats returns aggregate statistics about workouts and exercises
func GetAggregateStats(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from token
		userID, exists := c.Get("userID")
		if !exists {
			utils.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated", nil)
			return
		}

		// Total workouts
		var totalWorkouts int64
		if err := db.Model(&models.Workout{}).Where("user_id = ?", userID).Count(&totalWorkouts).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch workout statistics", err)
			return
		}

		// Total exercises performed
		var totalExercises int64
		if err := db.Model(&models.Exercise{}).
			Joins("JOIN workout_exercises ON exercises.id = workout_exercises.exercise_id").
			Joins("JOIN workouts ON workout_exercises.workout_id = workouts.id").
			Where("workouts.user_id = ?", userID).
			Count(&totalExercises).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch exercise statistics", err)
			return
		}

		// Average workout duration
		type AvgDuration struct {
			AvgDuration float64
		}
		var avgDuration AvgDuration
		if err := db.Model(&models.Workout{}).
			Select("COALESCE(AVG(duration_minutes), 0) as avg_duration").
			Where("user_id = ?", userID).
			Scan(&avgDuration).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch duration statistics", err)
			return
		}

		// Get list of exercises used
		type Exercise struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		var exercises []Exercise
		if err := db.Model(&models.Exercise{}).
			Select("DISTINCT exercises.id, exercises.name").
			Joins("JOIN workout_exercises ON exercises.id = workout_exercises.exercise_id").
			Joins("JOIN workouts ON workout_exercises.workout_id = workouts.id").
			Where("workouts.user_id = ?", userID).
			Scan(&exercises).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch exercise list", err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"totalWorkouts": totalWorkouts,
			"totalExercises": totalExercises,
			"avgDuration":   avgDuration.AvgDuration,
			"exercises":     exercises,
		})
	}
}
