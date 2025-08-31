package repository

import (
	"context"
	"fmt"

	"github.com/Amandasilvbr/products-crud/internal/domain/model"
	"github.com/Amandasilvbr/products-crud/internal/domain/repository"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ProductRepository implements the repository interface for product operations
type ProductRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewProductRepository creates a new instance of ProductRepository
func NewProductRepository(db *gorm.DB, logger *zap.Logger) repository.ProductRepositoryInterface {
	return &ProductRepository{
		db:     db,
		logger: logger,
	}
}

// Create handles the creation of one or more products in the database
// It performs validations for duplicates and existing SKUs, returning a map of any errors
func (r *ProductRepository) Create(ctx context.Context, products []*model.Product) map[int]string {
	if len(products) == 0 {
		r.logger.Warn("No products provided for creation")
		return nil
	}

	// Initialize maps for tracking errors and duplicate SKUs in the input
	errors := make(map[int]string)
	skuSet := make(map[int]struct{})
	for _, product := range products {
		if _, exists := skuSet[product.SKU]; exists {
			r.logger.Warn("Duplicate SKU in input list", zap.Int("sku", product.SKU))
			errors[product.SKU] = fmt.Sprintf("Duplicate SKU %d in input list", product.SKU)
		} else {
			skuSet[product.SKU] = struct{}{}
		}
	}

	// Iterate through products to perform database operations
	for _, product := range products {
		// Check if the context has been cancelled during the operation
		if ctx.Err() != nil {
			r.logger.Error("Context cancelled", zap.Error(ctx.Err()))
			for _, p := range products {
				if errors[p.SKU] == "" {
					errors[p.SKU] = fmt.Sprintf("Operation cancelled for SKU %d: %s", p.SKU, ctx.Err().Error())
				}
			}
			r.logger.Warn("Some products failed to create", zap.Any("errors", errors))
			return errors
		}

		// Skip products that already have an error
		if errors[product.SKU] != "" {
			continue
		}

		// Check if a product with the same SKU already exists in the database
		var existingProduct model.Product
		if err := r.db.WithContext(ctx).Where("sku = ?", product.SKU).First(&existingProduct).Error; err == nil {
			r.logger.Warn("Product with SKU already exists", zap.Int("sku", product.SKU))
			errors[product.SKU] = fmt.Sprintf("Product with SKU %d already exists", product.SKU)
			continue
		} else if err != gorm.ErrRecordNotFound {
			r.logger.Error("Error checking SKU existence", zap.Int("sku", product.SKU), zap.Error(err))
			errors[product.SKU] = fmt.Sprintf("Error checking SKU %d: %s", product.SKU, err.Error())
			continue
		}

		// Create the new product record
		if err := r.db.WithContext(ctx).Create(product).Error; err != nil {
			r.logger.Error("Error creating product", zap.Int("sku", product.SKU), zap.Error(err))
			errors[product.SKU] = fmt.Sprintf("Error creating product with SKU %d: %s", product.SKU, err.Error())
		}
	}

	// Return the map of errors if any occurred, otherwise return nil
	if len(errors) > 0 {
		r.logger.Warn("Some products failed to create", zap.Any("errors", errors))
		return errors
	}
	return nil
}

// GetAll retrieves all products from the database
func (r *ProductRepository) GetAll(ctx context.Context) ([]*model.Product, error) {
	var products []*model.Product
	if err := r.db.WithContext(ctx).Find(&products).Error; err != nil {
		r.logger.Error("Error fetching all products", zap.Error(err))
		return nil, err
	}
	return products, nil
}

// GetBySKU retrieves a single product by its SKU
func (r *ProductRepository) GetBySKU(ctx context.Context, sku int) (*model.Product, error) {
	var product model.Product
	result := r.db.WithContext(ctx).First(&product, "sku = ?", sku)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			r.logger.Warn("Product not found", zap.Int("sku", sku))
			return nil, gorm.ErrRecordNotFound
		}
		r.logger.Error("Error fetching product by SKU", zap.Int("sku", sku), zap.Error(result.Error))
		return nil, result.Error
	}
	return &product, nil
}

// Update modifies one or more existing products in the database
// It returns a map of errors for any products that failed to update
func (r *ProductRepository) Update(ctx context.Context, products []*model.Product) map[int]string {
	if len(products) == 0 {
		r.logger.Warn("No products provided for update")
		return nil
	}

	errors := make(map[int]string)
	for _, product := range products {
		result := r.db.WithContext(ctx).Where("sku = ?", product.SKU).Updates(product)
		if result.Error != nil {
			r.logger.Error("Error updating product", zap.Int("sku", product.SKU), zap.Error(result.Error))
			errors[product.SKU] = fmt.Sprintf("Error to update product with SKU %d: %s", product.SKU, result.Error.Error())
			continue
		}
		if result.RowsAffected == 0 {
			r.logger.Warn("No product found to update", zap.Int("sku", product.SKU))
			errors[product.SKU] = fmt.Sprintf("Product with SKU %d not found", product.SKU)
		}
	}

	if len(errors) > 0 {
		r.logger.Warn("Some products failed to update", zap.Any("errors", errors))
		return errors
	}
	return nil
}

// Delete removes a batch of products from the database by their SKUs
// It returns a map of errors for any SKUs that failed to delete
func (r *ProductRepository) Delete(ctx context.Context, skus []int) map[int]string {
	if len(skus) == 0 {
		r.logger.Warn("No SKUs provided for deletion")
		return nil
	}

	errors := make(map[int]string)
	for _, sku := range skus {
		result := r.db.WithContext(ctx).Where("sku = ?", sku).Delete(&model.Product{})
		if result.Error != nil {
			r.logger.Error("Error deleting product", zap.Int("sku", sku), zap.Error(result.Error))
			errors[sku] = fmt.Sprintf("Error to delete product with SKU %d: %s", sku, result.Error.Error())
			continue
		}
		if result.RowsAffected == 0 {
			r.logger.Warn("No product found to delete", zap.Int("sku", sku))
			errors[sku] = fmt.Sprintf("Product with SKU %d not found", sku)
		}
	}

	if len(errors) > 0 {
		r.logger.Warn("Some products failed to delete", zap.Any("errors", errors))
		return errors
	}
	return nil
}
