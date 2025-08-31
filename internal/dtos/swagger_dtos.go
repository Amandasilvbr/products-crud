package dtos

// LoginResponse defines the structure for a successful login response.
type LoginResponse struct {
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// CreateUserResponse defines the structure for a successful user creation response.
type CreateUserResponse struct {
	Message string `json:"message" example:"User created successfully"`
}

// BatchResult defines the structure for a single item in a batch operation response.
type BatchResult struct {
	Index  int               `json:"index" example:"0"`
	SKU    int               `json:"sku,omitempty" example:"12345"`
	Status string            `json:"status" example:"ok"`
	Errors map[string]string `json:"errors,omitempty"`
}

// CreateProductResponse defines the structure for a successful product creation response.
type CreateProductResponse struct {
	Message string        `json:"message" example:"Product(s) created successfully"`
	Results []BatchResult `json:"results"`
}

// UpdateProductResponse defines the structure for a successful product update response.
type UpdateProductResponse struct {
	Message string        `json:"message" example:"Product(s) updated successfully"`
	Results []BatchResult `json:"results"`
}

// DeleteProductResponse defines the structure for a successful product deletion response.
type DeleteProductResponse struct {
	Message string        `json:"message" example:"Product(s) deleted successfully"`
	Results []BatchResult `json:"results"`
}