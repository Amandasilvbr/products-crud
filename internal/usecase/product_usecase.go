package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Amandasilvbr/products-crud/internal/domain/messaging"
	"github.com/Amandasilvbr/products-crud/internal/domain/model"
	"github.com/Amandasilvbr/products-crud/internal/domain/repository"
	"github.com/Amandasilvbr/products-crud/internal/domain/usecase"

	"go.uber.org/zap"
)

// ErrProductNotFound is a standard error for when a product is not found
var (
	ErrProductNotFound = errors.New("product not found")
)

// ProductUseCase implements the business logic for product-related operations
type ProductUseCase struct {
	productRepo repository.ProductRepositoryInterface
	logger      *zap.Logger
	rabbitMQ    messaging.Publisher
}

// NewProductUseCase creates a new instance of ProductUseCase
func NewProductUseCase(repo repository.ProductRepositoryInterface, logger *zap.Logger, rabbitMQ messaging.Publisher) usecase.ProductUseCaseInterface {
	return &ProductUseCase{
		productRepo: repo,
		logger:      logger,
		rabbitMQ:    rabbitMQ,
	}
}

// Create handles the logic for creating new products
func (uc *ProductUseCase) Create(ctx context.Context, products []*model.Product, userEmail string) map[int]string {
	errors := uc.productRepo.Create(ctx, products)
	if len(errors) > 0 {
		uc.logger.Warn("Failed to create some products", zap.Any("errors", errors), zap.Int("count", len(errors)))
	}

	// Publish messages for the products that were created successfully
	for _, product := range products {
		if _, exists := errors[product.SKU]; !exists {
			uc.publishToRabbitMQ(ctx, "product_created", product, userEmail)
			uc.logger.Info("Published product creation event", zap.Int("sku", product.SKU), zap.String("user_email", userEmail))
		}
	}

	if len(errors) == 0 {
		uc.logger.Info("Created all products successfully", zap.Int("count", len(products)), zap.String("operation", "create"))
		return nil
	}

	return errors
}

// GetAll retrieves all products by calling the repository
func (uc *ProductUseCase) GetAll(ctx context.Context) ([]*model.Product, error) {
	products, err := uc.productRepo.GetAll(ctx)
	if err != nil {
		uc.logger.Error("Failed to fetch products", zap.Error(err), zap.String("operation", "get_all"))
		return nil, err
	}
	uc.logger.Info("Fetched all products", zap.Int("count", len(products)), zap.String("operation", "get_all"))
	return products, nil
}

// GetBySKU retrieves a single product by its SKU
func (uc *ProductUseCase) GetBySKU(ctx context.Context, sku int) (*model.Product, error) {
	product, err := uc.productRepo.GetBySKU(ctx, sku)
	if err != nil {
		uc.logger.Error("Failed to fetch product", zap.Int("sku", sku), zap.Error(err), zap.String("operation", "get_by_sku"))
		return nil, err
	}
	if product == nil {
		uc.logger.Warn("Product not found", zap.Int("sku", sku), zap.String("operation", "get_by_sku"))
		return nil, ErrProductNotFound
	}
	uc.logger.Info("Fetched product", zap.Int("sku", sku), zap.String("operation", "get_by_sku"))
	return product, nil
}

// Update handles the logic for updating existing products
func (uc *ProductUseCase) Update(ctx context.Context, products []*model.Product, userEmail string) map[int]string {
	// Verify the existence of all products before attempting to update them
	for _, product := range products {
		_, err := uc.GetBySKU(ctx, product.SKU)
		if err != nil {
			uc.logger.Warn("Cannot update non-existent product", zap.Int("sku", product.SKU), zap.Error(err), zap.String("operation", "update"))
			return map[int]string{product.SKU: fmt.Sprintf("Product with SKU %d not found", product.SKU)}
		}
	}

	errors := uc.productRepo.Update(ctx, products)
	if len(errors) > 0 {
		uc.logger.Warn("Failed to update some products", zap.Any("errors", errors), zap.Int("count", len(errors)))
	}

	// Publish messages for successfully updated products
	for _, product := range products {
		if _, exists := errors[product.SKU]; !exists {
			uc.publishToRabbitMQ(ctx, "product_updated", product, userEmail)
			uc.logger.Info("Published product update event", zap.Int("sku", product.SKU), zap.String("user_email", userEmail))
		}
	}

	if len(errors) == 0 {
		uc.logger.Info("Updated all products successfully", zap.Int("count", len(products)), zap.String("operation", "update"))
		return nil
	}

	return errors
}

// Delete handles the logic for deleting products
func (uc *ProductUseCase) Delete(ctx context.Context, skus []int, userEmail string) map[int]string {
	// Store products to publish deletion events after successful deletion
	productsToDelete := make(map[int]*model.Product)
	errors := make(map[int]string)

	// Verifica todos os SKUs e coleta produtos válidos
	for _, sku := range skus {
		product, err := uc.GetBySKU(ctx, sku)
		if err != nil {
			uc.logger.Warn("Cannot delete non-existent product", zap.Int("sku", sku), zap.Error(err))
			errors[sku] = fmt.Sprintf("Product with SKU %d not found", sku)
			continue // Continua processando os outros SKUs
		}
		productsToDelete[sku] = product
	}

	// Deleta apenas os produtos válidos
	if len(productsToDelete) > 0 {
		validSKUs := make([]int, 0, len(productsToDelete))
		for sku := range productsToDelete {
			validSKUs = append(validSKUs, sku)
		}

		// Chama o repositório para deletar os produtos válidos
		deleteErrors := uc.productRepo.Delete(ctx, validSKUs)
		for sku, errMsg := range deleteErrors {
			errors[sku] = errMsg
			uc.logger.Warn("Failed to delete product", zap.Int("sku", sku), zap.String("error", errMsg))
		}

		// Publica eventos para produtos deletados com sucesso
		for _, sku := range validSKUs {
			if _, exists := deleteErrors[sku]; !exists {
				uc.publishToRabbitMQ(ctx, "product_deleted", productsToDelete[sku], userEmail)
				uc.logger.Info("Published product deletion event", zap.Int("sku", sku), zap.String("user_email", userEmail))
			}
		}
	}

	if len(errors) == 0 {
		uc.logger.Info("Deleted all products successfully", zap.Int("count", len(skus)), zap.String("operation", "delete"))
		return nil
	}

	uc.logger.Warn("Some products could not be deleted", zap.Any("errors", errors), zap.Int("count", len(errors)))
	return errors
}

// publishToRabbitMQ is a helper function to marshal and send product event messages
func (uc *ProductUseCase) publishToRabbitMQ(ctx context.Context, event string, product *model.Product, userEmail string) {
	msg, err := json.Marshal(map[string]interface{}{
		"event":             event,
		"sku":               product.SKU,
		"name":              product.Name,
		"responsible_email": userEmail,
	})
	if err != nil {
		uc.logger.Error("Failed to marshal RabbitMQ message", zap.Int("sku", product.SKU), zap.String("event", event), zap.Error(err))
		return
	}

	err = uc.rabbitMQ.Publish(ctx, "product_events", string(msg))
	if err != nil {
		uc.logger.Error("Failed to publish to RabbitMQ", zap.Int("sku", product.SKU), zap.String("event", event), zap.String("message", string(msg)), zap.Error(err))
		return
	}

	uc.logger.Info("Successfully published to RabbitMQ", zap.Int("sku", product.SKU), zap.String("event", event), zap.String("message", string(msg)))
}
