package server

import (
	"context"

	"github.com/Amandasilvbr/products-crud/cmd/api/docs"
	"github.com/Amandasilvbr/products-crud/internal/handler"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Start initializes and runs the HTTP server
func Start(ctx context.Context, authHandler *handler.AuthHandler, productHandler *handler.ProductHandler, logger *zap.Logger) error {
	// Create a new Gin router with default middleware
	r := gin.Default()

	// Configure Swagger/OpenAPI documentation
	docs.SwaggerInfo.BasePath = "/api"

	// Set up routes
	SetupRoutes(r, authHandler, productHandler, logger)

	// Run the server
	logger.Info("Starting HTTP server on port :8988")
	return r.Run(":8988")
}
