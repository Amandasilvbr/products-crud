package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Init initializes a global zap logger with environment-specific configurations
func Init(isDevelopment bool) error {
	var logger *zap.Logger

	// Configure the logger for the development environment
	if isDevelopment {
		// Development configuration provides a human-readable, colored output
		cfg := zap.NewDevelopmentEncoderConfig()
		// Enables colored log levels
		cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder 
		 // Sets a standard time format
		cfg.EncodeTime = zapcore.ISO8601TimeEncoder       
		core := zapcore.NewCore(
			zapcore.NewConsoleEncoder(cfg),
			zapcore.Lock(os.Stdout),
			zapcore.DebugLevel,
		)
		// Include caller information and stack traces for error-level logs
		logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
	} else {
		// Configure the logger for the production environment
		cfg := zap.NewProductionEncoderConfig()
		cfg.EncodeTime = zapcore.ISO8601TimeEncoder
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(cfg),
			zapcore.Lock(os.Stdout),
			zapcore.InfoLevel,
		)
		logger = zap.New(core, zap.AddCaller())
	}

	// Set the configured logger as the global instance for the application
	zap.ReplaceGlobals(logger)
	zap.L().Info("Zap logger initialized successfully", zap.Bool("development", isDevelopment))
	return nil
}
