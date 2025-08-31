package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Amandasilvbr/products-crud/cmd/consumer"
	"github.com/Amandasilvbr/products-crud/internal/config"
	"github.com/Amandasilvbr/products-crud/internal/handler"
	"github.com/Amandasilvbr/products-crud/internal/infrastructure/database"
	"github.com/Amandasilvbr/products-crud/internal/infrastructure/logger"
	"github.com/Amandasilvbr/products-crud/internal/infrastructure/messaging"
	"github.com/Amandasilvbr/products-crud/internal/infrastructure/repository"
	"github.com/Amandasilvbr/products-crud/internal/server"
	"github.com/Amandasilvbr/products-crud/internal/usecase"

	"go.uber.org/zap"
)

// @title           Products CRUD API
// @version         1.0
// @description     API desenvolvida para oferecer funcionalidades de criação, consulta, atualização e exclusão de produtos, com alertas via e-mail utilizando RabbitMQ.
// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html
// @BasePath  /api
// @securityDefinitions.apikey bearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and a JWT token.
func main() {
	// Load application configurations from environment variables or a config file
	cfg, err := config.New()
	if err != nil {
		panic("failed to load configs: " + err.Error())
	}

	// Initialize the Zap logger
	isDevelopment := (cfg.AppEnv == "development")
	if err := logger.Init(isDevelopment); err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	zapLogger := zap.L()
	zapLogger.Info("Logger initialized successfully", zap.String("environment", cfg.AppEnv))

	// Set up a context that can be canceled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Establish a connection to the PostgreSQL database
	db, err := database.ConnectDB(
		cfg,
		zapLogger,
	)
	if err != nil {
		zapLogger.Fatal("Failed to connect to database", zap.Error(err))
	}
	zapLogger.Info("Successfully connected to database")

	// Run database migrations
	if err := database.RunMigrations(db, zapLogger); err != nil {
		zapLogger.Fatal("Failed to run migrations", zap.Error(err))
	}
	zapLogger.Info("Database migrations executed successfully")

	// Get the RabbitMQ URL from an environment variable
	amqpURL := os.Getenv("RABBITMQ_URL")
	if amqpURL == "" {
		zapLogger.Fatal("RABBITMQ_URL environment variable is not set")
	}

	// Initialize RabbitMQ client for publishing messages
	rabbitMQ, err := messaging.NewRabbitMQClient(amqpURL)
	if err != nil {
		zapLogger.Fatal("Failed to connect to RabbitMQ for API", zap.Error(err))
	}
	defer rabbitMQ.Close()
	zapLogger.Info("RabbitMQ connection for API established")

	// Declare the queue for publishing events
	err = rabbitMQ.DeclareQueue("product_events")
	if err != nil {
		zapLogger.Fatal("Failed to declare RabbitMQ queue for API", zap.Error(err))
	}
	zapLogger.Info("RabbitMQ queue declared successfully for API")

	// Initialize RabbitMQ consumer
	consumer, err := consumer.NewConsumer(zapLogger, amqpURL, "product_events")
	if err != nil {
		zapLogger.Fatal("Failed to initialize RabbitMQ consumer", zap.Error(err))
	}
	defer consumer.Close()
	zapLogger.Info("RabbitMQ consumer initialized successfully")

	// Start RabbitMQ consumer in a goroutine
	go func() {
		zapLogger.Info("Starting RabbitMQ consumer")
		if err := consumer.Start(ctx); err != nil {
			zapLogger.Error("Consumer stopped with error", zap.Error(err))
		}
	}()

	// Dependency Injection
	userRepo := repository.NewUserRepository(db, zapLogger)
	productRepo := repository.NewProductRepository(db, zapLogger)
	authUsecase := usecase.NewAuthUsecase(userRepo, zapLogger)
	productUsecase := usecase.NewProductUseCase(productRepo, zapLogger, rabbitMQ)
	authHandler := handler.NewAuthHandler(authUsecase, zapLogger)
	productHandler := handler.NewProductHandler(productUsecase, zapLogger)

	// Initialize and start the HTTP server
	go func() {
		if err := server.Start(ctx, authHandler, productHandler, zapLogger); err != nil {
			zapLogger.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	// Controlled Shutdown Handling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	zapLogger.Info("Shutting down application...")
	cancel()
}
