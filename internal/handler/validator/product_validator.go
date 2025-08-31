package validator

import (
	"fmt"

	"github.com/Amandasilvbr/products-crud/internal/domain/model"

	"github.com/go-playground/validator/v10"
)

// ProductValidator wraps the go-playground/validator instance
type ProductValidator struct {
	validate *validator.Validate
}

// NewProductValidator creates and returns a new instance of ProductValidator
func NewProductValidator() *ProductValidator {
	v := validator.New()

	return &ProductValidator{validate: v}
}

// ValidateProduct checks a Product model against a set of validation rules
func (v *ProductValidator) ValidateProduct(product *model.Product) map[string]string {
	err := v.validate.Struct(product)
	if err == nil {
		return nil // Return nil if no validation errors are found
	}

	// Create a map to hold the custom error messages
	errors := make(map[string]string)
	// Iterate over the validation errors returned by the validator
	for _, err := range err.(validator.ValidationErrors) {
		field := err.Field()
		tag := err.Tag()
		value := err.Value()

		// Use a switch statement to generate custom error messages for specific field and rule combinations
		switch field + "|" + tag {
		case "SKU|required":
			errors[field] = "The SKU field is required and cannot be empty"
		case "SKU|gt":
			errors[field] = fmt.Sprintf("The SKU must be a positive integer, got %v", value)
		case "Name|required":
			errors[field] = "The name field is required and cannot be empty"
		case "Name|min":
			errors[field] = fmt.Sprintf("The name must be at least 3 characters long, got %d characters", len(value.(string)))
		case "Name|max":
			errors[field] = fmt.Sprintf("The name cannot exceed 100 characters, got %d characters", len(value.(string)))
		case "Price|required":
			errors[field] = "The price field is required and cannot be empty"
		case "Price|gt":
			errors[field] = fmt.Sprintf("The price must be greater than 0, got %v", value)
		case "Stock|gte":
			errors[field] = fmt.Sprintf("The stock must be 0 or greater, got %v", value)
		case "Category|required":
			errors[field] = "The category field is required and cannot be empty"
		case "Category|max":
			errors[field] = fmt.Sprintf("The category cannot exceed 100 characters, got %d characters", len(value.(string)))
		case "Availability|oneof":
			errors[field] = fmt.Sprintf("The availability must be one of 'in stock' or 'out of stock', got '%v'", value)
		case "Link|url":
			errors[field] = fmt.Sprintf("The link must be a valid URL, got '%v'", value)
		case "ImageLink|url":
			errors[field] = fmt.Sprintf("The image link must be a valid URL, got '%v'", value)
		default:
			errors[field] = fmt.Sprintf("Validation failed for field %s with rule %s, got value '%v'", field, tag, value)
		}
	}
	// Return the map of generated error messages
	return errors
}
