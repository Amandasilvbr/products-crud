package dtos

import "time"

// CreateProductDTO represents the data transfer object for creating a new product
type CreateProductDTO struct {
	SKU          int     `json:"sku" validate:"required"`
	Name         string  `json:"name" validate:"required,min=3,max=100"`
	Description  string  `json:"description" validate:"max=500"`
	Price        float64 `json:"price" validate:"required,gt=0"`
	Category     string  `json:"category" validate:"required,min=3,max=100"`
	Link         string  `json:"link" validate:"omitempty,url"`
	ImageLink    string  `json:"image_link" validate:"omitempty,url"`
	Availability string  `json:"availability" validate:"required,oneof='in stock' 'out of stock'"`
}

// UpdateProductDTO represents the data transfer object for updating an existing product
type UpdateProductDTO struct {
	Sku          int     `json:"sku" validate:"required"`
	Name         string  `json:"name" validate:"omitempty,min=3,max=100"`
	Description  string  `json:"description" validate:"omitempty,max=500"`
	Price        float64 `json:"price" validate:"omitempty,gt=0"`
	Category     string  `json:"category" validate:"omitempty,min=3,max=100"`
	Link         string  `json:"link" validate:"omitempty,url"`
	ImageLink    string  `json:"image_link" validate:"omitempty,url"`
	Availability string  `json:"availability" validate:"omitempty,oneof='in stock' 'out of stock'"`
}

// ProductResponseDTO represents the data transfer object for returning product information
type ProductResponseDTO struct {
	SKU          int       `json:"sku"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Price        float64   `json:"price"`
	Category     string    `json:"category"`
	Link         string    `json:"link,omitempty"`
	ImageLink    string    `json:"image_link,omitempty"`
	Availability string    `json:"availability"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	CreatedBy    string    `json:"createdBy"`
}
