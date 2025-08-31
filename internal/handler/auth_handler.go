package handler

import (
	"net/http"

	"github.com/Amandasilvbr/products-crud/internal/domain/usecase"
	"github.com/Amandasilvbr/products-crud/internal/dtos"
	"github.com/Amandasilvbr/products-crud/internal/handler/validator"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	authUsecase usecase.AuthUsecaseInterface
	validator   *validator.UserValidator
	logger      *zap.Logger
}

// NewAuthHandler creates and returns a new instance of AuthHandler
func NewAuthHandler(authUsecase usecase.AuthUsecaseInterface, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		authUsecase: authUsecase,
		validator:   validator.NewUserValidator(),
		logger:      logger,
	}
}

// Login godoc
//
//	@Summary		Authenticate a user
//	@Description	Autentica um usuário com base em e-mail e senha, retornando um token JWT válido para endpoints protegidos.
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			credentials	body		dtos.UserLoginDTO	true	"User credentials (email and password)"
//	@Success		200			{object}	dtos.LoginResponse	"Successful authentication with JWT token"
//	@Router			/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var input dtos.UserLoginDTO
	// Bind the incoming JSON payload to the UserLoginDTO struct
	if err := c.ShouldBindJSON(&input); err != nil {
		h.logger.Debug("Invalid login request body", zap.Error(err), zap.String("operation", "login"))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body format",
		})
		return
	}

	// Validate the input data using the user validator
	if errors := h.validator.ValidateUserLogin(&input); len(errors) > 0 {
		h.logger.Warn("Validation failed for login", zap.Any("errors", errors), zap.String("operation", "login"))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": errors,
		})
		return
	}

	// Call the use case to perform the login logic
	token, err := h.authUsecase.Login(input.Email, input.Password)
	if err != nil {
		h.logger.Warn("Login failed", zap.String("email", input.Email), zap.Error(err), zap.String("operation", "login"))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Return the JWT token on successful authentication
	h.logger.Info("User authenticated", zap.String("email", input.Email), zap.String("operation", "login"))
	c.JSON(http.StatusOK, gin.H{"token": token})
}

// CreateUser godoc
//
//	@Summary		Create a new user
//	@Description	Registra um novo usuário com nome, e-mail e senha. O e-mail deve ser único e a senha deve atender aos critérios de validação.
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			user	body		dtos.UserCreateDTO		true	"User data for registration (name, email, password)"
//	@Success		201		{object}	dtos.CreateUserResponse	"User created successfully"
//	@Router			/register [post]
func (h *AuthHandler) CreateUser(c *gin.Context) {
	var input dtos.UserCreateDTO
	// Bind the incoming JSON payload to the UserCreateDTO struct
	if err := c.ShouldBindJSON(&input); err != nil {
		h.logger.Debug("Invalid user creation request body", zap.Error(err), zap.String("operation", "create_user"))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body format",
		})
		return
	}
	// Validate the input data using the user validator
	if errors := h.validator.ValidateUserCreate(&input); len(errors) > 0 {
		h.logger.Warn("Validation failed for user creation",
			zap.Any("errors", errors),
			zap.String("operation", "create_user"))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": errors,
		})
		return
	}

	// Call the use case to create the new user
	err := h.authUsecase.CreateUser(input.Name, input.Email, input.Password)
	if err != nil {
		// Handle specific error for existing user
		if err.Error() == "user with this email already exists" {
			h.logger.Warn("Email already in use", zap.String("email", input.Email), zap.String("operation", "create_user"))
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Email already in use",
			})
			return
		}
		// Handle generic server errors
		h.logger.Error("Failed to create user", zap.Error(err), zap.String("operation", "create_user"))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create user",
		})
		return
	}

	// Return a success message upon user creation
	h.logger.Info("User created", zap.String("email", input.Email), zap.String("operation", "create_user"))
	c.JSON(http.StatusCreated, gin.H{
		"message": "User created successfully",
	})
}
