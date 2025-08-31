package usecase

import (
	"context"

	"github.com/Amandasilvbr/products-crud/internal/domain/model"
)

// ProductUseCaseInterface defines the interface for product-related use cases
type ProductUseCaseInterface interface {
	Create(context.Context, []*model.Product, string) (map[int]string, map[int]string)
	GetAll(ctx context.Context) ([]*model.Product, error)
	GetBySKU(ctx context.Context, sku int) (*model.Product, error)
	Update(ctx context.Context, products []*model.Product, userEmail string) map[int]string
	Delete(ctx context.Context, skus []int, userEmail string) map[int]string
}
