package repository

import (
	"context"

	"github.com/Amandasilvbr/products-crud/internal/domain/model"
)

// UserRepository defines the interface for user data access operations
type ProductRepositoryInterface interface {
	Create(ctx context.Context, products []*model.Product) map[int]string
	GetAll(ctx context.Context) ([]*model.Product, error)
	GetBySKU(ctx context.Context, sku int) (*model.Product, error)
	Update(ctx context.Context, products []*model.Product) map[int]string
	Delete(ctx context.Context, skus []int) map[int]string
}
