package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Configs holds all the configuration variables for the application
// These values are loaded from environment variables
type Configs struct {
	DbHost       string
	DbPort       string
	DbDatabase   string
	DbUsername   string
	DbPassword   string
	AppEnv       string
	JWTKey       string
	RabbitMQURL  string
	SMTPFrom     string
	SMTPUser     string
	SMTPPassword string
	SMTPHost     string
	SMTPPort     string
	JWTSecret    string
}

// New loads the environment variables from a .env file,
// validates that all required variables are present, and returns them in a Configs struct
// It aggregates all missing variable errors and returns them as a single error
func New() (*Configs, error) {
	if err := godotenv.Load(".env"); err == nil {
		fmt.Println("Arquivo .env carregado")
	} else {
		fmt.Println("Nenhum .env encontrado, usando variáveis do ambiente")
	}

	var errorList []error
	cfg := &Configs{}

	// Validate and assign each required environment variable
	cfg.DbHost, errorList = getRequiredEnv("DB_HOST", errorList)
	cfg.DbPort, errorList = getRequiredEnv("DB_PORT", errorList)
	cfg.DbDatabase, errorList = getRequiredEnv("DB_DATABASE", errorList)
	cfg.DbUsername, errorList = getRequiredEnv("DB_USERNAME", errorList)
	cfg.DbPassword, errorList = getRequiredEnv("DB_PASSWORD", errorList)
	cfg.AppEnv, errorList = getRequiredEnv("APP_ENV", errorList)
	cfg.JWTKey, errorList = getRequiredEnv("JWT_SECRET_KEY", errorList)
	cfg.RabbitMQURL, errorList = getRequiredEnv("RABBITMQ_URL", errorList)
	cfg.SMTPFrom, errorList = getRequiredEnv("SMTP_FROM", errorList)
	cfg.SMTPUser, errorList = getRequiredEnv("SMTP_USER", errorList)
	cfg.SMTPPassword, errorList = getRequiredEnv("SMTP_PASSWORD", errorList)
	cfg.SMTPHost, errorList = getRequiredEnv("SMTP_HOST", errorList)
	cfg.SMTPPort, errorList = getRequiredEnv("SMTP_PORT", errorList)
	cfg.JWTSecret, errorList = getRequiredEnv("JWT_SECRET_KEY", errorList)

	if len(errorList) > 0 {
		return nil, errors.Join(errorList...)
	}

	return cfg, nil
}

// getRequiredEnv is a helper function that retrieves a required environment variable
// If the variable is not found, it appends an error to the provided error slice
func getRequiredEnv(key string, errs []error) (string, []error) {
	value, exists := os.LookupEnv(key)
	if !exists {
		errs = append(errs, fmt.Errorf("environment variable \"%s\" not found", key))
	}
	return value, errs
}
