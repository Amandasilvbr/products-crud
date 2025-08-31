package usecase_test

import (
	"context"
	"testing"

	"github.com/Amandasilvbr/products-crud/internal/domain/model"
	ucdomain "github.com/Amandasilvbr/products-crud/internal/domain/usecase"
	"github.com/Amandasilvbr/products-crud/internal/usecase"

	"github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockProductRepository simulates the behavior of the product repository.
type MockProductRepository struct {
	mock.Mock
}

// Create mocks the repository's Create method, which creates products and returns a map of SKUs to error messages.
func (m *MockProductRepository) Create(ctx context.Context, products []*model.Product) map[int]string {
	// Call the mock with the provided context and products, returning the configured result.
	args := m.Called(ctx, products)
	// Handle the case where the mock is configured to return nil (indicating success).
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(map[int]string)
}

// GetAll mocks the repository's GetAll method, returning a slice of products and an error.
func (m *MockProductRepository) GetAll(ctx context.Context) ([]*model.Product, error) {
	// Call the mock with the provided context, returning the configured products and error.
	args := m.Called(ctx)
	return args.Get(0).([]*model.Product), args.Error(1)
}

// GetBySKU mocks the repository's GetBySKU method, returning a single product by SKU and an error.
func (m *MockProductRepository) GetBySKU(ctx context.Context, sku int) (*model.Product, error) {
	// Call the mock with the provided context and SKU, returning the configured product and error.
	args := m.Called(ctx, sku)
	// Handle the case where the product is nil (e.g., not found).
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Product), args.Error(1)
}

// Update mocks the repository's Update method, updating products and returning a map of SKUs to error messages.
func (m *MockProductRepository) Update(ctx context.Context, products []*model.Product) map[int]string {
	// Call the mock with the provided context and products, returning the configured result.
	args := m.Called(ctx, products)
	// Handle the case where the mock is configured to return nil (indicating success).
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(map[int]string)
}

// Delete mocks the repository's Delete method, deleting products by SKUs and returning a map of SKUs to error messages.
func (m *MockProductRepository) Delete(ctx context.Context, skus []int) map[int]string {
	// Call the mock with the provided context and SKUs, returning the configured result.
	args := m.Called(ctx, skus)
	// Handle the case where the mock is configured to return nil (indicating success).
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(map[int]string)
}

// MockRabbitMQClient simulates the behavior of the RabbitMQ client.
type MockRabbitMQClient struct {
	mock.Mock
}

// Publish mocks the RabbitMQ client's Publish method, which publishes a message to a queue.
func (m *MockRabbitMQClient) Publish(ctx context.Context, queueName, body string) error {
	// Call the mock with the provided context, queue name, and message body, returning the configured error.
	args := m.Called(ctx, queueName, body)
	return args.Error(0)
}

// Consume mocks the RabbitMQ client's Consume method, which consumes messages from a queue.
func (m *MockRabbitMQClient) Consume(queueName, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp091.Table) (<-chan amqp091.Delivery, error) {
	// Call the mock with the provided parameters, returning the configured channel and error.
	ret := m.Called(queueName, consumer, autoAck, exclusive, noLocal, noWait, args)
	return ret.Get(0).(<-chan amqp091.Delivery), ret.Error(1)
}

// Close mocks the RabbitMQ client's Close method, closing the connection.
func (m *MockRabbitMQClient) Close() {
	// Call the mock to record the invocation (no return value needed).
	m.Called()
}

// setupTest creates a test environment with mocks and context.
func setupTest(t *testing.T) (ucdomain.ProductUseCaseInterface, *MockProductRepository, *MockRabbitMQClient, context.Context) {
	// Initialize a background context for the test.
	ctx := context.Background()
	// Create a no-op logger for testing (no logs are written).
	logger := zap.NewNop()
	// Create a mock product repository.
	repo := &MockProductRepository{}
	// Create a mock RabbitMQ client.
	rabbitMQ := &MockRabbitMQClient{}
	// Initialize the product use case with the mock repository, logger, and RabbitMQ client.
	uc := usecase.NewProductUseCase(repo, logger, rabbitMQ)
	return uc, repo, rabbitMQ, ctx
}

// product is a default test product used across test cases.
var product = &model.Product{SKU: 1, Name: "Produto Teste"}
var products = []*model.Product{product}
var userEmail = "teste@exemplo.com"

// TestProductUseCase runs all test cases for the ProductUseCase.
func TestProductUseCase(t *testing.T) {
	// Define a slice of test cases, each with a name, setup function, execution function, expected result, and error flag.
	tests := []struct {
		name     string
		setup    func(*MockProductRepository, *MockRabbitMQClient)
		execute  func(ucdomain.ProductUseCaseInterface, context.Context) interface{}
		expected interface{}
		hasError bool
	}{
		// Test case for successful product creation.
		{
			name: "Create_Success",
			setup: func(repo *MockProductRepository, rabbitMQ *MockRabbitMQClient) {
				// Mock the repository's Create method to return nil (success).
				repo.On("Create", mock.Anything, products).Return(nil).Once()
				// Mock the RabbitMQ Publish method to return nil (success).
				rabbitMQ.On("Publish", mock.Anything, "product_events", mock.Anything).Return(nil).Once()
			},
			execute: func(uc ucdomain.ProductUseCaseInterface, ctx context.Context) interface{} {
				// Execute the Create method of the use case.
				return uc.Create(ctx, products, userEmail)
			},
			expected: nil, // Expect nil for successful creation.
			hasError: false,
		},
		// Test case for product creation with errors.
		{
			name: "Create_WithErrors",
			setup: func(repo *MockProductRepository, rabbitMQ *MockRabbitMQClient) {
				// Mock the repository's Create method to return an error map.
				repo.On("Create", mock.Anything, products).Return(map[int]string{1: "Erro ao criar"}).Once()
			},
			execute: func(uc ucdomain.ProductUseCaseInterface, ctx context.Context) interface{} {
				// Execute the Create method of the use case.
				return uc.Create(ctx, products, userEmail)
			},
			expected: map[int]string{1: "Erro ao criar"}, // Expect an error map.
			hasError: true,
		},
		// Test case for successfully retrieving all products.
		{
			name: "GetAll_Success",
			setup: func(repo *MockProductRepository, rabbitMQ *MockRabbitMQClient) {
				// Mock the repository's GetAll method to return the test products and no error.
				repo.On("GetAll", mock.Anything).Return(products, nil).Once()
			},
			execute: func(uc ucdomain.ProductUseCaseInterface, ctx context.Context) interface{} {
				// Execute the GetAll method and return both result and error.
				result, err := uc.GetAll(ctx)
				return []interface{}{result, err}
			},
			expected: []interface{}{products, nil}, // Expect the products and no error.
			hasError: false,
		},
		// Test case for successfully retrieving a product by SKU.
		{
			name: "GetBySKU_Success",
			setup: func(repo *MockProductRepository, rabbitMQ *MockRabbitMQClient) {
				// Mock the repository's GetBySKU method to return the test product and no error.
				repo.On("GetBySKU", mock.Anything, 1).Return(product, nil).Once()
			},
			execute: func(uc ucdomain.ProductUseCaseInterface, ctx context.Context) interface{} {
				// Execute the GetBySKU method and return both result and error.
				result, err := uc.GetBySKU(ctx, 1)
				return []interface{}{result, err}
			},
			expected: []interface{}{product, nil}, // Expect the product and no error.
			hasError: false,
		},
		// Test case for retrieving a product by SKU when it is not found.
		{
			name: "GetBySKU_NotFound",
			setup: func(repo *MockProductRepository, rabbitMQ *MockRabbitMQClient) {
				// Mock the repository's GetBySKU method to return nil and a not-found error.
				repo.On("GetBySKU", mock.Anything, 1).Return(nil, usecase.ErrProductNotFound).Once()
			},
			execute: func(uc ucdomain.ProductUseCaseInterface, ctx context.Context) interface{} {
				// Execute the GetBySKU method and return both result and error.
				result, err := uc.GetBySKU(ctx, 1)
				return []interface{}{result, err}
			},
			expected: []interface{}{(*model.Product)(nil), usecase.ErrProductNotFound}, // Expect nil product and not-found error.
			hasError: true,
		},
		// Test case for successful product update.
		{
			name: "Update_Success",
			setup: func(repo *MockProductRepository, rabbitMQ *MockRabbitMQClient) {
				// Mock the repository's GetBySKU to verify the product exists.
				repo.On("GetBySKU", mock.Anything, 1).Return(product, nil).Once()
				// Mock the repository's Update method to return nil (success).
				repo.On("Update", mock.Anything, products).Return(nil).Once()
				// Mock the RabbitMQ Publish method to return nil (success).
				rabbitMQ.On("Publish", mock.Anything, "product_events", mock.Anything).Return(nil).Once()
			},
			execute: func(uc ucdomain.ProductUseCaseInterface, ctx context.Context) interface{} {
				// Execute the Update method of the use case.
				return uc.Update(ctx, products, userEmail)
			},
			expected: nil, // Expect nil for successful update.
			hasError: false,
		},
		// Test case for updating a product that is not found.
		{
			name: "Update_NotFound",
			setup: func(repo *MockProductRepository, rabbitMQ *MockRabbitMQClient) {
				// Mock the repository's GetBySKU to return a not-found error.
				repo.On("GetBySKU", mock.Anything, 1).Return(nil, usecase.ErrProductNotFound).Once()
			},
			execute: func(uc ucdomain.ProductUseCaseInterface, ctx context.Context) interface{} {
				// Execute the Update method of the use case.
				return uc.Update(ctx, products, userEmail)
			},
			expected: map[int]string{1: "Product with SKU 1 not found"}, // Expect an error map.
			hasError: true,
		},
		// Test case for successful product deletion.
		{
			name: "Delete_Success",
			setup: func(repo *MockProductRepository, rabbitMQ *MockRabbitMQClient) {
				// Mock the repository's GetBySKU to verify the product exists.
				repo.On("GetBySKU", mock.Anything, 1).Return(product, nil).Once()
				// Mock the repository's Delete method to return nil (success).
				repo.On("Delete", mock.Anything, []int{1}).Return(nil).Once()
				// Mock the RabbitMQ Publish method to return nil (success).
				rabbitMQ.On("Publish", mock.Anything, "product_events", mock.Anything).Return(nil).Once()
			},
			execute: func(uc ucdomain.ProductUseCaseInterface, ctx context.Context) interface{} {
				// Execute the Delete method of the use case.
				return uc.Delete(ctx, []int{1}, userEmail)
			},
			expected: nil, // Expect nil for successful deletion.
			hasError: false,
		},
		// Test case for deleting a product that is not found.
		{
			name: "Delete_NotFound",
			setup: func(repo *MockProductRepository, rabbitMQ *MockRabbitMQClient) {
				// Mock the repository's GetBySKU to return a not-found error.
				repo.On("GetBySKU", mock.Anything, 1).Return(nil, usecase.ErrProductNotFound).Once()
			},
			execute: func(uc ucdomain.ProductUseCaseInterface, ctx context.Context) interface{} {
				// Execute the Delete method of the use case.
				return uc.Delete(ctx, []int{1}, userEmail)
			},
			expected: map[int]string{1: "Product with SKU 1 not found"}, // Expect an error map.
			hasError: true,
		},
	}

	// Iterate over each test case and run it as a subtest.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up the test environment with mocks and context.
			uc, repo, rabbitMQ, ctx := setupTest(t)
			// Apply the test case's setup function to configure the mocks.
			tt.setup(repo, rabbitMQ)
			// Execute the test case's logic.
			result := tt.execute(uc, ctx)

			// Handle assertions based on the type of result.
			if !tt.hasError && tt.name != "GetAll_Success" && tt.name != "GetBySKU_Success" {
				// For success cases that return a map, assert that the result is nil (indicating success).
				assert.Nil(t, result, "Unexpected result for %s", tt.name)
			} else if slice, ok := result.([]interface{}); ok {
				// For methods returning a (value, error) pair, assert both the value and error match the expected values.
				assert.Equal(t, tt.expected.([]interface{})[0], slice[0], "Unexpected result for %s", tt.name)
				assert.Equal(t, tt.expected.([]interface{})[1], slice[1], "Unexpected error for %s", tt.name)
			} else {
				// For methods returning an error map, assert the result matches the expected map.
				assert.Equal(t, tt.expected, result, "Unexpected result for %s", tt.name)
			}

			// Verify that all expected mock interactions occurred.
			repo.AssertExpectations(t)
			rabbitMQ.AssertExpectations(t)
		})
	}
}
