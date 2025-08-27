package models

import "gorm.io/gorm"

type AuthUser struct {
	gorm.Model
	UserID   uint `json:"user_id" gorm:"not null;unique;foreignKey:UserID;references:ID"`
	Password string `json:"-" gorm:"not null"`
}

type ProfileUser struct {
	gorm.Model
	Username    string  `json:"username" gorm:"unique;not null;uniqueIndex"`
	Email       string  `json:"email" gorm:"unique;not null"`
	Name        string  `json:"full_name" gorm:"not null"`
	Role        string  `json:"role" gorm:"not null;default:'user'"`
	GoogleID    *string `json:"google_id" gorm:"unique"`
	PasswordLess bool   `json:"passwordLess" gorm:"not null;default:false"`
}
