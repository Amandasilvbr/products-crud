package middleware

import (
	"strings"

	"github.com/Amandasilvbr/products-crud/internal/config"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// JWTMiddleware creates a Gin middleware for handling JWT authentication
// It extracts, parses, and validates the token from the Authorization header
func JWTMiddleware(zapLogger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Retrieve the Authorization header from the request
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			zapLogger.Debug("Missing Authorization header")
			c.JSON(401, gin.H{"error": "Token not provided"})
			c.Abort()
			return
		}

		// Check if the token is in the "Bearer <token>" format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			zapLogger.Debug("Invalid token format", zap.String("header", authHeader))
			c.JSON(401, gin.H{"error": "Invalid token format"})
			c.Abort()
			return
		}

		// Load application configuration to get the JWT secret key
		cfg, err := config.New()
		if err != nil {
			zapLogger.Error("Failed to load config", zap.Error(err))
			c.JSON(500, gin.H{"error": "Internal server error"})
			c.Abort()
			return
		}

		// Ensure the JWT secret key is configured
		secretKey := cfg.JWTSecret
		if secretKey == "" {
			zapLogger.Error("JWT secret key not configured")
			c.JSON(500, gin.H{"error": "Internal server error"})
			c.Abort()
			return
		}

		// Parse and validate the JWT token using the secret key
		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
			// Verify that the signing method is HMAC
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secretKey), nil
		})
		if err != nil || token == nil || !token.Valid {
			zapLogger.Warn("Invalid JWT token", zap.Error(err))
			c.JSON(401, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Extract claims from the validated token
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			zapLogger.Warn("Invalid JWT claims")
			c.JSON(401, gin.H{"error": "Invalid claims"})
			c.Abort()
			return
		}

		// Validate and extract user's name from claims
		userName, ok := claims["name"].(string)
		if !ok || userName == "" {
			zapLogger.Warn("Missing or invalid name in JWT claims")
			c.JSON(401, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		// Validate and extract user's ID from claims
		userID, ok := claims["id"].(float64)
		if !ok {
			zapLogger.Warn("Missing or invalid id in JWT claims")
			c.JSON(401, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		// Validate and extract user's email from claims
		userEmail, ok := claims["email"].(string)
		if !ok || userEmail == "" {
			zapLogger.Warn("Missing or invalid email in JWT claims")
			c.JSON(401, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		// Set user information in the Gin context
		c.Set("userName", userName)
		c.Set("userID", uint(userID))
		c.Set("userEmail", userEmail)
		c.Next()
	}
}
