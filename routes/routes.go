package routes

import (
	"github.com/rachitnimje/trackle-web/controllers"
	"github.com/rachitnimje/trackle-web/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRoutes(r *gin.Engine, db *gorm.DB) {
	// Public routes
	r.POST("/register", controllers.Register(db))
	r.POST("/login", controllers.Login(db))
	r.POST("/auth/google-oauth", controllers.GoogleOAuth(db))

	// Protected routes
	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddleware())
	{
		// User profile routes
		api.GET("/me", controllers.Me(db))
		api.POST("/logout", controllers.Logout())

		// User templates routes
		api.POST("/me/templates", controllers.CreateUserTemplate(db))
		api.GET("/me/templates", controllers.GetAllUserTemplates(db))
		api.GET("/me/templates/:id", controllers.GetUserTemplate(db))
		api.DELETE("/me/templates/:id", controllers.DeleteUserTemplate(db))

		// User workouts routes
		api.POST("/me/workouts", controllers.CreateUserWorkout(db))
		api.GET("/me/workouts", controllers.GetAllUserWorkouts(db))
		api.GET("/me/workouts/:id", controllers.GetUserWorkout(db))
		api.PUT("/me/workouts/:id", controllers.UpdateUserWorkout(db))
		api.DELETE("/me/workouts/:id", controllers.DeleteUserWorkout(db))

		// Exercise routes (general resources)
		api.GET("/exercises", controllers.GetAllExercises(db))
		api.POST("/exercises", controllers.CreateExercise(db))
		api.GET("/exercises/:id", controllers.GetExercise(db))
		api.PUT("/exercises/:id", controllers.UpdateExercise(db))
		api.DELETE("/exercises/:id", controllers.DeleteExercise(db))
		
		// Exercise metadata routes
		api.GET("/exercises/categories", controllers.GetExerciseCategories)
		api.GET("/exercises/muscles", controllers.GetPrimaryMuscles)
		api.GET("/exercises/equipment", controllers.GetEquipmentTypes)
		
		// Statistics routes
		api.GET("/stats/workouts", controllers.GetWorkoutStats(db))
		api.GET("/stats/exercises/:id", controllers.GetExerciseProgress(db))
		api.GET("/stats/aggregate", controllers.GetAggregateStats(db))
	}
}
