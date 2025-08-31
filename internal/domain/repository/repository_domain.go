package repository

import (
	"github.com/Amandasilvbr/products-crud/internal/domain/model"
)

// UserRepository defines the interface for user data access operations
type UserRepositoryInterface interface {
	FindByEmail(email string) (*model.User, error)
	Create(user *model.User) error
}
