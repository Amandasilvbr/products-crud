package model

import "time"

// Product represents the data model for a product in the database
type Product struct {
	SKU int `gorm:"primaryKey" json:"sku" validate:"required,gt=0"`
	Name string `json:"name" validate:"required,min=3,max=100"`
	Description string `json:"description"`
	Price float64 `json:"price" validate:"required,gt=0"`
	Category string `json:"category" validate:"required,min=3,max=100"`
	Link string `json:"link" validate:"omitempty,url"`
	ImageLink string `json:"imageLink" validate:"omitempty,url"`
	Availability string `json:"availability" validate:"required,oneof='in stock' 'out of stock'"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	CreatedBy string `json:"createdBy"`
}
