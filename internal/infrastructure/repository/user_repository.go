package repository

import (
	"errors"

	"github.com/Amandasilvbr/products-crud/internal/domain/model"
	"github.com/Amandasilvbr/products-crud/internal/domain/repository"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// UserRepository implements the repository interface for user operations
type UserRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewUserRepository initializes a new UserRepository with the provided database and logger
func NewUserRepository(db *gorm.DB, logger *zap.Logger) repository.UserRepositoryInterface {
	return &UserRepository{
		db:     db,
		logger: logger,
	}
}

// FindByEmail retrieves a user by their email address
func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		// If the error is a 'record not found' error, it's not a system failure
		if err == gorm.ErrRecordNotFound {
			r.logger.Debug("User not found")
			return nil, nil
		}
		// For any other error, log it and return the error
		r.logger.Error("Error fetching user by email", zap.Error(err))
		return nil, err
	}
	return &user, nil
}

// Create adds a new user to the database
func (r *UserRepository) Create(user *model.User) error {
	// Check for an existing user with the same email before creating a new one
	existingUser, err := r.FindByEmail(user.Email)
	if err != nil {
		r.logger.Error("Error checking existing user", zap.Error(err))
		return err
	}
	// If a user is found, return an error to prevent duplicates
	if existingUser != nil {
		r.logger.Warn("User already exists")
		return errors.New("user with this email already exists")
	}

	// Create the new user record in the database
	if err := r.db.Create(user).Error; err != nil {
		r.logger.Error("Error creating user", zap.Error(err))
		return err
	}
	r.logger.Info("User created successfully")
	return nil
}
