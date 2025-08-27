package models

import "gorm.io/gorm"

type Exercise struct {
	gorm.Model
	Name          string `json:"name" gorm:"not null"`
	Description   string `json:"description"`
	Category      string `json:"category"`
	PrimaryMuscle string `json:"primary_muscle"`
	Equipment     string `json:"equipment"`
}

type Template struct {
	gorm.Model
	Name        string             `json:"name" gorm:"not null"`
	Description string             `json:"description"`
	UserID      uint               `json:"user_id" gorm:"not null"`
	User        ProfileUser        `json:"-" gorm:"foreignKey:UserID"`
	Exercises   []TemplateExercise `json:"exercises" gorm:"foreignKey:TemplateID"`
}

type TemplateExercise struct {
	gorm.Model
	TemplateID uint     `json:"template_id" gorm:"not null"`
	ExerciseID uint     `json:"exercise_id" gorm:"not null"`
	Sets       int      `json:"sets" gorm:"not null;check:sets > 0"`
	Template   Template `json:"-" gorm:"foreignKey:TemplateID"`
	Exercise   Exercise `json:"exercise" gorm:"foreignKey:ExerciseID"`
}
