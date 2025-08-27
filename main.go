package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/rachitnimje/trackle-web/config"
	"github.com/rachitnimje/trackle-web/middleware"
	"github.com/rachitnimje/trackle-web/routes"
	"github.com/rachitnimje/trackle-web/utils"
)

func main() {
	// err := godotenv.Load()
	// if err != nil {
	// 	log.Fatal("Error loading .env file")
	// }

	// Connect to database
	db := config.ConnectDB()

	// Run migrations
	config.MigrateDB(db)

	// Initialize validator
	utils.InitValidator()

	// Initialize Gin router
	r := gin.New() // Use New() instead of Default() to customize middleware

	// Add recovery middleware to handle panics
	r.Use(gin.Recovery())
	
	// Add custom recovery middleware for better error handling
	r.Use(middleware.ErrorRecoveryMiddleware())

	// Add middleware
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.LoggerMiddleware())

	// Setup routes
	routes.SetupRoutes(r, db)

	// Get port from environment or default to 8080
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
