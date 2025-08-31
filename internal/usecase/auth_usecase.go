package usecase

import (
	"errors"
	"time"

	"github.com/Amandasilvbr/products-crud/internal/config"
	"github.com/Amandasilvbr/products-crud/internal/domain/model"
	"github.com/Amandasilvbr/products-crud/internal/domain/repository"
	"github.com/Amandasilvbr/products-crud/internal/domain/usecase"

	"github.com/dgrijalva/jwt-go"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// AuthUsecase implements the business logic for authentication operations
type AuthUsecase struct {
	userRepo repository.UserRepositoryInterface
	logger   *zap.Logger
}

// NewAuthUsecase creates a new instance of AuthUsecase
func NewAuthUsecase(userRepo repository.UserRepositoryInterface, logger *zap.Logger) usecase.AuthUsecaseInterface {
	return &AuthUsecase{
		userRepo: userRepo,
		logger:   logger,
	}
}

// Login handles the user authentication process
func (u *AuthUsecase) Login(email, password string) (string, error) {
	// Find the user in the repository by their email address
	user, err := u.userRepo.FindByEmail(email)
	if err != nil {
		u.logger.Warn("User not found", zap.String("email", email), zap.Error(err), zap.String("operation", "login"))
		return "", errors.New("user not found")
	}

	// Compare the provided password with the stored hashed password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		u.logger.Warn("Invalid password", zap.String("email", email), zap.String("operation", "login"))
		return "", errors.New("incorrect password")
	}

	// Create JWT claims, including user details and an expiration time
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":    user.ID,
		"name":  user.Name,
		"email": user.Email,
		"exp":   time.Now().Add(time.Hour * 48).Unix(),
	})

	// Load configuration to retrieve the JWT secret key
	cfg, err := config.New()
	if err != nil {
		u.logger.Error("Failed to load config", zap.Error(err), zap.String("operation", "login"))
		return "", errors.New("failed to load config")
	}
	secret := cfg.JWTSecret

	// Ensure the JWT secret key is configured
	u.logger.Info("JWT_SECRET value", zap.String("secret", secret))
	if secret == "" {
		u.logger.Error("JWT_SECRET not set", zap.String("operation", "login"))
		return "", errors.New("JWT secret key not configured")
	}

	// Sign the token with the secret key to generate the final token string
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		u.logger.Error("Failed to generate JWT", zap.Error(err), zap.String("operation", "login"))
		return "", err
	}

	u.logger.Info("User logged in", zap.String("email", user.Email), zap.String("operation", "login"))
	return tokenString, nil
}

// CreateUser handles the registration of a new user
func (u *AuthUsecase) CreateUser(name, email, password string) error {
	// Generate a secure hash of the user's password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		u.logger.Error("Failed to hash password", zap.Error(err), zap.String("operation", "create_user"))
		return errors.New("failed to hash password")
	}

	// Create a new user model with the provided data
	user := &model.User{
		Name:     name,
		Email:    email,
		Password: string(hashedPassword),
	}

	// Call the repository to create the user, handling potential errors like duplicates
	if err := u.userRepo.Create(user); err != nil {
		if err.Error() == "user with this email already exists" {
			u.logger.Warn("Email already in use", zap.String("email", email), zap.String("operation", "create_user"))
			return err
		}
		u.logger.Error("Failed to create user", zap.Error(err), zap.String("operation", "create_user"))
		return err
	}

	u.logger.Info("Created user", zap.String("email", email), zap.String("operation", "create_user"))
	return nil
}
