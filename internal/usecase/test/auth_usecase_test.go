package usecase_test

import (
    "errors"
    "os"
    "testing"

    "github.com/Amandasilvbr/products-crud/internal/domain/model"
    "github.com/Amandasilvbr/products-crud/internal/usecase"
    "github.com/stretchr/testify/assert"
    "go.uber.org/zap"
    "golang.org/x/crypto/bcrypt"
)

// mockUserRepo is a mock implementation of the user repository for testing purposes
type mockUserRepo struct {
    user *model.User
    err  error
}

// FindByEmail mocks the repository's method to find a user by email
func (m *mockUserRepo) FindByEmail(email string) (*model.User, error) {
    if m.err != nil {
        return nil, m.err
    }
    return m.user, nil
}

// Create mocks the repository's method to create a user
func (m *mockUserRepo) Create(user *model.User) error {
    if m.err != nil {
        return m.err
    }
    m.user = user
    return nil
}

// mockEnv sets up environment variables required for the tests
func mockEnv(t *testing.T) {
    // Set environment variables to mock the configuration needed for the application
    envVars := map[string]string{
        "DB_HOST":        "localhost",
        "DB_PORT":        "5432",
        "DB_DATABASE":    "testdb",
        "DB_USERNAME":    "testuser",
        "DB_PASSWORD":    "testpass",
        "APP_ENV":        "test",
        "JWT_SECRET_KEY": "test-secret",
        "RABBITMQ_URL":   "amqp://guest:guest@localhost:5672/",
        "SMTP_FROM":      "noreply@test.com",
        "SMTP_USER":      "smtpuser",
        "SMTP_PASSWORD":  "smtppass",
        "SMTP_HOST":      "localhost",
        "SMTP_PORT":      "587",
    }

    // Set environment variables
    for key, value := range envVars {
        os.Setenv(key, value)
    }

    // Clean up environment variables after the test
    t.Cleanup(func() {
        for key := range envVars {
            os.Unsetenv(key)
        }
    })
}

// TestLogin tests the Login functionality of the AuthUsecase
func TestLogin(t *testing.T) {
    // Initialize a no-op logger to suppress logging during tests
    logger := zap.NewNop()

    // Subtest: Successful login scenario
    t.Run("Success", func(t *testing.T) {
        // Set up environment variables
        mockEnv(t)

        // Generate a hashed password for the test user
        hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)

        // Set up the mock repository with a valid user
        repo := &mockUserRepo{
            user: &model.User{
                Name:     "Amanda",
                Email:    "amanda@test.com",
                Password: string(hashedPassword),
            },
        }

        // Initialize the AuthUsecase with the mock repository and logger
        authUC := usecase.NewAuthUsecase(repo, logger)

        // Attempt to log in with correct email and password
        token, err := authUC.Login("amanda@test.com", "123456")

        // Assert that no error occurred during login
        assert.NoError(t, err)
        // Assert that a non-empty token was returned
        assert.NotEmpty(t, token)
    })

    // Subtest: Login with incorrect password
    t.Run("WrongPassword", func(t *testing.T) {
        // Set up environment variables
        mockEnv(t)

        // Generate a hashed password for the test user
        hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)

        // Set up the mock repository with a valid user
        repo := &mockUserRepo{
            user: &model.User{
                Name:     "Amanda",
                Email:    "amanda@test.com",
                Password: string(hashedPassword),
            },
        }

        // Initialize the AuthUsecase with the mock repository and logger
        authUC := usecase.NewAuthUsecase(repo, logger)

        // Attempt to log in with correct email but incorrect password
        token, err := authUC.Login("amanda@test.com", "1234")

        // Assert that an error occurred during login
        assert.Error(t, err)
        // Assert that the error message is "incorrect password"
        assert.Equal(t, "incorrect password", err.Error())
        // Assert that no token was returned
        assert.Empty(t, token)
    })

    // Subtest: Login with non-existent user
    t.Run("UserNotFound", func(t *testing.T) {
        // Set up environment variables
        mockEnv(t)

        // Set up the mock repository to return a "not found" error
        repo := &mockUserRepo{err: errors.New("not found")}

        // Initialize the AuthUsecase with the mock repository and logger
        authUC := usecase.NewAuthUsecase(repo, logger)

        // Attempt to log in with a non-existent email
        token, err := authUC.Login("test@test.com", "123456")

        // Assert that an error occurred during login
        assert.Error(t, err)
        // Assert that the error message is "user not found"
        assert.Equal(t, "user not found", err.Error())
        // Assert that no token was returned
        assert.Empty(t, token)
    })
}

