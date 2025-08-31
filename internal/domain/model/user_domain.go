package model

import (
	"gorm.io/gorm"
)

// User represents the data model for a user in the database
type User struct {
	gorm.Model
	Name string `gorm:"primaryKey" json:"name" validate:"required,min=3,max=100"`
	Email string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}
