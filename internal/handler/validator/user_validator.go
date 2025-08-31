package validator

import (
	"fmt"

	"github.com/Amandasilvbr/products-crud/internal/dtos"

	"github.com/go-playground/validator/v10"
)

// UserValidator wraps the go-playground/validator instance
// It provides methods for validating user-related data transfer objects (DTOs)
type UserValidator struct {
	validate *validator.Validate
}

// NewUserValidator creates and returns a new instance of UserValidator
func NewUserValidator() *UserValidator {
	v := validator.New()
	return &UserValidator{validate: v}
}

// ValidateUserLogin checks a UserLoginDTO against a set of validation rules
// It returns a map of validation errors for the login fields
func (v *UserValidator) ValidateUserLogin(dto *dtos.UserLoginDTO) map[string]string {
	err := v.validate.Struct(dto)
	if err == nil {
		return nil // Return nil if validation passes
	}

	// Create a map to hold custom error messages
	errors := make(map[string]string)
	// Iterate over the validation errors
	for _, err := range err.(validator.ValidationErrors) {
		field := err.Field()
		tag := err.Tag()
		value := err.Value()

		// Generate user-friendly error messages based on the field and validation rule
		switch field + "|" + tag {
		case "Email|required":
			errors[field] = "The email field is required and cannot be empty"
		case "Email|email":
			errors[field] = fmt.Sprintf("The email must be a valid email address, got '%v'", value)
		case "Password|required":
			errors[field] = "The password field is required and cannot be empty"
		case "Password|min":
			errors[field] = fmt.Sprintf("The password must be at least 6 characters long, got %d characters", len(value.(string)))
		default:
			errors[field] = fmt.Sprintf("Validation failed for field %s with rule %s, got value '%v'", field, tag, value)
		}
	}
	return errors
}

// ValidateUserCreate checks a UserCreateDTO against a set of validation rules
// It returns a map of validation errors for the user creation fields
func (v *UserValidator) ValidateUserCreate(dto *dtos.UserCreateDTO) map[string]string {
	err := v.validate.Struct(dto)
	if err == nil {
		return nil // Return nil if validation passes
	}

	// Create a map to hold custom error messages
	errors := make(map[string]string)
	// Iterate over the validation errors
	for _, err := range err.(validator.ValidationErrors) {
		field := err.Field()
		tag := err.Tag()
		value := err.Value()

		// Generate user-friendly error messages based on the field and validation rule
		switch field + "|" + tag {
		case "Name|required":
			errors[field] = "The name field is required and cannot be empty"
		case "Name|min":
			errors[field] = fmt.Sprintf("The name must be at least 3 characters long, got %d characters", len(value.(string)))
		case "Name|max":
			errors[field] = fmt.Sprintf("The name cannot exceed 100 characters, got %d characters", len(value.(string)))
		case "Email|required":
			errors[field] = "The email field is required and cannot be empty"
		case "Email|email":
			errors[field] = fmt.Sprintf("The email must be a valid email address, got '%v'", value)
		case "Password|required":
			errors[field] = "The password field is required and cannot be empty"
		case "Password|min":
			errors[field] = fmt.Sprintf("The password must be at least 6 characters long, got %d characters", len(value.(string)))
		default:
			errors[field] = fmt.Sprintf("Validation failed for field %s with rule %s, got value '%v'", field, tag, value)
		}
	}
	return errors
}
