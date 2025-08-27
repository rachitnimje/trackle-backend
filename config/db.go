package config

import (
	"fmt"
	"log"
	"os"

	"github.com/rachitnimje/trackle-web/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectDB() *gorm.DB {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("SSL_MODE"),
	)

	// dsn := os.Getenv(("DB_URL"))

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}
	
	return db
}

func MigrateDB(db *gorm.DB) *gorm.DB {
       db.AutoMigrate(
	       &models.AuthUser{},
	       &models.ProfileUser{},
	       &models.Exercise{},
	       &models.Template{},
	       &models.TemplateExercise{},
	       &models.Workout{},
	       &models.WorkoutEntry{},
       )
	return db
}
