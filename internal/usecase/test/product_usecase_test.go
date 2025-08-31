package usecase_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/Amandasilvbr/products-crud/internal/domain/model"
	ucdomain "github.com/Amandasilvbr/products-crud/internal/domain/usecase"
	"github.com/Amandasilvbr/products-crud/internal/usecase"

	"github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockProductRepository simula o comportamento do repositório de produtos.
type MockProductRepository struct {
	mock.Mock
}

func (m *MockProductRepository) Create(ctx context.Context, products []*model.Product) map[int]string {
	args := m.Called(ctx, products)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(map[int]string)
}

func (m *MockProductRepository) GetAll(ctx context.Context) ([]*model.Product, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*model.Product), args.Error(1)
}

func (m *MockProductRepository) GetBySKU(ctx context.Context, sku int) (*model.Product, error) {
	args := m.Called(ctx, sku)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Product), args.Error(1)
}

func (m *MockProductRepository) Update(ctx context.Context, products []*model.Product) map[int]string {
	args := m.Called(ctx, products)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(map[int]string)
}

func (m *MockProductRepository) Delete(ctx context.Context, skus []int) map[int]string {
	args := m.Called(ctx, skus)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(map[int]string)
}

// MockRabbitMQClient simula o comportamento do cliente RabbitMQ.
type MockRabbitMQClient struct {
	mock.Mock
}

func (m *MockRabbitMQClient) Publish(ctx context.Context, queueName, body string) error {
	args := m.Called(ctx, queueName, body)
	return args.Error(0)
}

func (m *MockRabbitMQClient) Consume(queueName, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp091.Table) (<-chan amqp091.Delivery, error) {
	ret := m.Called(queueName, consumer, autoAck, exclusive, noLocal, noWait, args)
	return ret.Get(0).(<-chan amqp091.Delivery), ret.Error(1)
}

func (m *MockRabbitMQClient) Close() {
	m.Called()
}

// setupTest cria um ambiente de teste com mocks e contexto.
func setupTest(t *testing.T) (ucdomain.ProductUseCaseInterface, *MockProductRepository, *MockRabbitMQClient, context.Context) {
	ctx := context.Background()
	logger := zap.NewNop()
	repo := &MockProductRepository{}
	rabbitMQ := &MockRabbitMQClient{}
	uc := usecase.NewProductUseCase(repo, logger, rabbitMQ)
	return uc, repo, rabbitMQ, ctx
}

// Dados de teste
var product1 = &model.Product{SKU: 1, Name: "Produto 1", Price: 10.0}
var product2 = &model.Product{SKU: 2, Name: "", Price: 20.0} // Inválido
var product3 = &model.Product{SKU: 3, Name: "Produto 3", Price: 30.0}
var products = []*model.Product{product1, product2, product3}
var userEmail = "teste@exemplo.com"

// TestProductUseCase executa todos os casos de teste para o ProductUseCase.
func TestProductUseCase(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*MockProductRepository, *MockRabbitMQClient)
		execute  func(ucdomain.ProductUseCaseInterface, context.Context) interface{}
		expected interface{}
		hasError bool
	}{
		// Teste para criação bem-sucedida de produtos
		{
			name: "Create_Success",
			setup: func(repo *MockProductRepository, rabbitMQ *MockRabbitMQClient) {
				repo.On("Create", mock.Anything, []*model.Product{product1}).Return(nil).Once()
				rabbitMQ.On("Publish", mock.Anything, "product_events", mock.Anything).Return(nil).Once()
			},
			execute: func(uc ucdomain.ProductUseCaseInterface, ctx context.Context) interface{} {
				createErrors, publishErrors := uc.Create(ctx, []*model.Product{product1}, userEmail)
				return []interface{}{createErrors, publishErrors}
			},
			expected: []interface{}{nil, nil},
			hasError: false,
		},
		// Teste para criação com erros de validação
		{
			name: "Create_WithErrors",
			setup: func(repo *MockProductRepository, rabbitMQ *MockRabbitMQClient) {
				repo.On("Create", mock.Anything, products).Return(map[int]string{
					2: "name cannot be empty",
				}).Once()
				rabbitMQ.On("Publish", mock.Anything, "product_events", mock.Anything).Return(nil).Twice() // SKU 1 e 3
			},
			execute: func(uc ucdomain.ProductUseCaseInterface, ctx context.Context) interface{} {
				createErrors, publishErrors := uc.Create(ctx, products, userEmail)
				return []interface{}{createErrors, publishErrors}
			},
			expected: []interface{}{
				map[int]string{2: "name cannot be empty"},
				nil,
			},
			hasError: true,
		},
		// Teste para criação com erro de publicação
		{
			name: "Create_WithPublishErrors",
			setup: func(repo *MockProductRepository, rabbitMQ *MockRabbitMQClient) {
				repo.On("Create", mock.Anything, products).Return(nil).Once()
				rabbitMQ.On("Publish", mock.Anything, "product_events", mock.Anything).Return(fmt.Errorf("connection timeout")).Times(3) // SKU 1, 2, 3
			},
			execute: func(uc ucdomain.ProductUseCaseInterface, ctx context.Context) interface{} {
				createErrors, publishErrors := uc.Create(ctx, products, userEmail)
				return []interface{}{createErrors, publishErrors}
			},
			expected: []interface{}{
				nil,
				map[int]string{
					1: "Failed to publish to RabbitMQ: failed to publish to RabbitMQ: connection timeout",
					2: "Failed to publish to RabbitMQ: failed to publish to RabbitMQ: connection timeout",
					3: "Failed to publish to RabbitMQ: failed to publish to RabbitMQ: connection timeout",
				},
			},
			hasError: true,
		},
		// Teste para criação com erros mistos (validação e publicação)
		{
			name: "Create_WithMixedErrors",
			setup: func(repo *MockProductRepository, rabbitMQ *MockRabbitMQClient) {
				repo.On("Create", mock.Anything, products).Return(map[int]string{
					2: "name cannot be empty",
				}).Once()
				rabbitMQ.On("Publish", mock.Anything, "product_events", mock.Anything).Return(nil).Once()                              // SKU 1
				rabbitMQ.On("Publish", mock.Anything, "product_events", mock.Anything).Return(fmt.Errorf("connection timeout")).Once() // SKU 3
			},
			execute: func(uc ucdomain.ProductUseCaseInterface, ctx context.Context) interface{} {
				createErrors, publishErrors := uc.Create(ctx, products, userEmail)
				return []interface{}{createErrors, publishErrors}
			},
			expected: []interface{}{
				map[int]string{2: "name cannot be empty"},
				map[int]string{3: "Failed to publish to RabbitMQ: failed to publish to RabbitMQ: connection timeout"},
			},
			hasError: true,
		},
		// Teste para recuperar todos os produtos com sucesso
		{
			name: "GetAll_Success",
			setup: func(repo *MockProductRepository, rabbitMQ *MockRabbitMQClient) {
				repo.On("GetAll", mock.Anything).Return([]*model.Product{product1}, nil).Once()
			},
			execute: func(uc ucdomain.ProductUseCaseInterface, ctx context.Context) interface{} {
				result, err := uc.GetAll(ctx)
				return []interface{}{result, err}
			},
			expected: []interface{}{[]*model.Product{product1}, nil},
			hasError: false,
		},
		// Teste para recuperar um produto por SKU com sucesso
		{
			name: "GetBySKU_Success",
			setup: func(repo *MockProductRepository, rabbitMQ *MockRabbitMQClient) {
				repo.On("GetBySKU", mock.Anything, 1).Return(product1, nil).Once()
			},
			execute: func(uc ucdomain.ProductUseCaseInterface, ctx context.Context) interface{} {
				result, err := uc.GetBySKU(ctx, 1)
				return []interface{}{result, err}
			},
			expected: []interface{}{product1, nil},
			hasError: false,
		},
		// Teste para recuperar um produto por SKU não encontrado
		{
			name: "GetBySKU_NotFound",
			setup: func(repo *MockProductRepository, rabbitMQ *MockRabbitMQClient) {
				repo.On("GetBySKU", mock.Anything, 1).Return(nil, usecase.ErrProductNotFound).Once()
			},
			execute: func(uc ucdomain.ProductUseCaseInterface, ctx context.Context) interface{} {
				result, err := uc.GetBySKU(ctx, 1)
				return []interface{}{result, err}
			},
			expected: []interface{}{(*model.Product)(nil), usecase.ErrProductNotFound},
			hasError: true,
		},
		// Teste para atualização bem-sucedida
		{
			name: "Update_Success",
			setup: func(repo *MockProductRepository, rabbitMQ *MockRabbitMQClient) {
				repo.On("GetBySKU", mock.Anything, 1).Return(product1, nil).Once()
				repo.On("Update", mock.Anything, []*model.Product{product1}).Return(nil).Once()
				rabbitMQ.On("Publish", mock.Anything, "product_events", mock.Anything).Return(nil).Once()
			},
			execute: func(uc ucdomain.ProductUseCaseInterface, ctx context.Context) interface{} {
				return uc.Update(ctx, []*model.Product{product1}, userEmail)
			},
			expected: nil,
			hasError: false,
		},
		// Teste para atualização de produto não encontrado
		{
			name: "Update_NotFound",
			setup: func(repo *MockProductRepository, rabbitMQ *MockRabbitMQClient) {
				repo.On("GetBySKU", mock.Anything, 1).Return(nil, usecase.ErrProductNotFound).Once()
			},
			execute: func(uc ucdomain.ProductUseCaseInterface, ctx context.Context) interface{} {
				return uc.Update(ctx, []*model.Product{product1}, userEmail)
			},
			expected: map[int]string{1: "Product with SKU 1 not found"},
			hasError: true,
		},
		// Teste para exclusão bem-sucedida
		{
			name: "Delete_Success",
			setup: func(repo *MockProductRepository, rabbitMQ *MockRabbitMQClient) {
				repo.On("GetBySKU", mock.Anything, 1).Return(product1, nil).Once()
				repo.On("Delete", mock.Anything, []int{1}).Return(nil).Once()
				rabbitMQ.On("Publish", mock.Anything, "product_events", mock.Anything).Return(nil).Once()
			},
			execute: func(uc ucdomain.ProductUseCaseInterface, ctx context.Context) interface{} {
				return uc.Delete(ctx, []int{1}, userEmail)
			},
			expected: nil,
			hasError: false,
		},
		// Teste para exclusão de produto não encontrado
		{
			name: "Delete_NotFound",
			setup: func(repo *MockProductRepository, rabbitMQ *MockRabbitMQClient) {
				repo.On("GetBySKU", mock.Anything, 1).Return(nil, usecase.ErrProductNotFound).Once()
			},
			execute: func(uc ucdomain.ProductUseCaseInterface, ctx context.Context) interface{} {
				return uc.Delete(ctx, []int{1}, userEmail)
			},
			expected: map[int]string{1: "Product with SKU 1 not found"},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, repo, rabbitMQ, ctx := setupTest(t)
			tt.setup(repo, rabbitMQ)
			result := tt.execute(uc, ctx)

			if !tt.hasError && tt.name != "GetAll_Success" && tt.name != "GetBySKU_Success" && tt.name != "Create_Success" {
				// Para casos de sucesso que retornam um mapa (Update, Delete), espera nil
				assert.Empty(t, result, "Unexpected result for %s", tt.name)
			} else if slice, ok := result.([]interface{}); ok {
				// Para métodos que retornam (valor, erro) ou (createErrors, publishErrors)
				expectedSlice := tt.expected.([]interface{})
				// Comparar createErrors
				if expectedSlice[0] == nil {
					assert.Empty(t, slice[0], "Unexpected first result for %s", tt.name)
				} else {
					assert.Equal(t, expectedSlice[0], slice[0], "Unexpected first result for %s", tt.name)
				}
				// Comparar publishErrors
				if expectedSlice[1] == nil {
					assert.Empty(t, slice[1], "Unexpected second result for %s", tt.name)
				} else {
					assert.Equal(t, expectedSlice[1], slice[1], "Unexpected second result for %s", tt.name)
				}
			} else {
				// Para métodos que retornam um único valor
				if tt.expected == nil {
					assert.Empty(t, result, "Unexpected result for %s", tt.name)
				} else {
					assert.Equal(t, tt.expected, result, "Unexpected result for %s", tt.name)
				}
			}

			repo.AssertExpectations(t)
			rabbitMQ.AssertExpectations(t)
		})
	}
}
