package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Amandasilvbr/products-crud/internal/domain/model"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ConnectDB establishes a connection to the PostgreSQL database using GORM
func ConnectDB(host, port, user, dbname, password string, zapLogger *zap.Logger) (*gorm.DB, error) {
	// Construct the Data Source Name (DSN) for the PostgreSQL connection
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user,
		password,
		host,
		port, 
		dbname,
	)

	// Open a connection to the database
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold: time.Second,
				Colorful:      true,
			},
		),
	})
	// Handle connection errors by logging and terminating the application
	if err != nil {
		zapLogger.Error("Failed to connect to database", zap.Error(err))
		panic("failed to connect to database: " + err.Error())
	}

	// Log successful connection and return the database instance
	zapLogger.Info("Successfully connected to database")
	return db, nil
}

// RunMigrations applies auto-migrations for the specified GORM models
func RunMigrations(db *gorm.DB, zapLogger *zap.Logger) error {
	// AutoMigrate will create or update tables for the Product and User models
	err := db.AutoMigrate(
		&model.Product{},
		&model.User{},
	)
	// Handle migration errors by logging and terminating the application
	if err != nil {
		zapLogger.Error("Failed to run migrations", zap.Error(err))
		panic("failed to run migrations: " + err.Error())
	}

	// Log successful migration
	zapLogger.Info("Migration completed successfully")
	return nil
}
