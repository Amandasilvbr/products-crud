package server

import (
	"github.com/Amandasilvbr/products-crud/internal/handler"
	"github.com/Amandasilvbr/products-crud/internal/handler/middleware"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

// SetupRoutes configures the API routes
func SetupRoutes(r *gin.Engine, authHandler *handler.AuthHandler, productHandler *handler.ProductHandler, logger *zap.Logger) {
	// Configure Swagger route
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Group all API routes under the "/api" prefix
	api := r.Group("/api")

	// Public routes for authentication and user registration
	api.POST("/login", authHandler.Login)
	api.POST("/register", authHandler.CreateUser)

	// Protected routes with JWT middleware
	api.Use(middleware.JWTMiddleware(logger))
	api.POST("/products", productHandler.Create)
	api.GET("/products", productHandler.GetAll)
	api.GET("/products/:sku", productHandler.GetBySKU)
	api.PUT("/products", productHandler.Update)
	api.DELETE("/products", productHandler.Delete)
}