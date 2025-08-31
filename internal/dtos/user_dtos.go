package dtos

// UserLoginDTO represents the data transfer object for user login
type UserLoginDTO struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" binding:"required,min=6"`
}

// UserCreateDTO represents the data transfer object for creating a new user
type UserCreateDTO struct {
    Name     string `json:"name" validate:"required,min=3,max=100"`
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=6"`
}