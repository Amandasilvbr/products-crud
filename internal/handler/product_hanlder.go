package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Amandasilvbr/products-crud/internal/domain/model"
	"github.com/Amandasilvbr/products-crud/internal/domain/usecase"
	"github.com/Amandasilvbr/products-crud/internal/dtos"
	"github.com/Amandasilvbr/products-crud/internal/handler/validator"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ProductHandler handles HTTP requests related to products
type ProductHandler struct {
	productUseCase usecase.ProductUseCaseInterface
	validator      *validator.ProductValidator
	logger         *zap.Logger
}

// NewProductHandler creates a new instance of ProductHandler
func NewProductHandler(useCase usecase.ProductUseCaseInterface, logger *zap.Logger) *ProductHandler {
	return &ProductHandler{
		productUseCase: useCase,
		validator:      validator.NewProductValidator(),
		logger:         logger,
	}
}

// Create godoc
//
//	@Summary		Create one or more products
//	@Description	Cria novos produtos com base em um único objeto ou em uma matriz de objetos no corpo da solicitação
//	@Tags			Products
//	@Accept			json
//	@Produce		json
//	@Param			products	body		[]dtos.CreateProductDTO		true	"Product data to create"
//	@Success		201			{object}	dtos.CreateProductResponse	"Product(s) created successfully"
//	@Security		bearerAuth
//	@Router			/products [post]
func (h *ProductHandler) Create(c *gin.Context) {
	// Ler o corpo da requisição
	body, err := c.GetRawData()
	if err != nil {
		h.logger.Error("Failed to read request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	var inputs []dtos.CreateProductDTO

	// Tentar deserializar como um array de produtos
	err = json.Unmarshal(body, &inputs)
	if err != nil {
		// Se falhar, tentar deserializar como um único produto
		var singleInput dtos.CreateProductDTO
		if err_single := json.Unmarshal(body, &singleInput); err_single != nil {
			h.logger.Error("Invalid request body format", zap.Error(err_single))
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request body format. Must be a product object or an array of product objects.",
				"details": err_single.Error(),
			})
			return
		}
		inputs = []dtos.CreateProductDTO{singleInput}
	}

	// Estrutura para armazenar o resultado de cada item no lote
	type batchResult struct {
		Index  int               `json:"index"`
		Status string            `json:"status"`
		Errors map[string]string `json:"errors,omitempty"`
	}
	var results []batchResult
	var products []*model.Product

	// Obter informações do usuário do contexto (definidas pelo middleware JWT)
	user, exists := c.Get("userName")
	if !exists {
		h.logger.Error("User not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userName, ok := user.(string)
	if !ok {
		h.logger.Error("Invalid user format in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data"})
		return
	}

	userEmail, exists := c.Get("userEmail")
	if !exists {
		h.logger.Error("User email not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User email not found"})
		return
	}

	userEmailStr, ok := userEmail.(string)
	if !ok {
		h.logger.Error("Invalid user email format in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user email data"})
		return
	}

	// Iterar sobre cada entrada, validá-la e convertê-la para o modelo de domínio
	for i, input := range inputs {
		product := &model.Product{
			SKU:          input.SKU,
			Name:         input.Name,
			Description:  input.Description,
			Price:        input.Price,
			Category:     input.Category,
			Link:         input.Link,
			ImageLink:    input.ImageLink,
			Availability: input.Availability,
			CreatedBy:    userName,
		}

		// Validar o modelo do produto
		if errors := h.validator.ValidateProduct(product); errors != nil {
			h.logger.Warn("Validation errors for product", zap.Int("index", i), zap.Any("errors", errors))
			results = append(results, batchResult{
				Index:  i,
				Status: "error",
				Errors: errors,
			})
		} else {
			results = append(results, batchResult{
				Index:  i,
				Status: "ok", 
			})
			products = append(products, product)
		}
	}

	// Chamar o caso de uso para criar os produtos válidos
	var createErrors, publishErrors map[int]string
	if len(products) > 0 {
		createErrors, publishErrors = h.productUseCase.Create(c.Request.Context(), products, userEmailStr)
	}

	// Atualizar os resultados com base nos erros de criação e publicação
	for i, product := range products {
		resultIndex := -1
		for j, result := range results {
			if result.Index == i && result.Status == "pending" {
				resultIndex = j
				break
			}
		}
		if resultIndex == -1 {
			continue // Produto já inválido na validação inicial
		}

		if errMsg, exists := createErrors[product.SKU]; exists {
			results[resultIndex].Status = "error"
			if results[resultIndex].Errors == nil {
				results[resultIndex].Errors = make(map[string]string)
			}
			results[resultIndex].Errors["creation_error"] = errMsg
			h.logger.Error("Failed to create product", zap.Int("sku", product.SKU), zap.String("error", errMsg))
		} else if errMsg, exists := publishErrors[product.SKU]; exists {
			results[resultIndex].Status = "error"
			if results[resultIndex].Errors == nil {
				results[resultIndex].Errors = make(map[string]string)
			}
			results[resultIndex].Errors["publish_error"] = errMsg
			h.logger.Error("Failed to publish product event", zap.Int("sku", product.SKU), zap.String("error", errMsg))
		} else {
			results[resultIndex].Status = "ok"
		}
	}

	// Verificar se houve algum erro (validação, criação ou publicação)
	hasErrors := false
	for _, result := range results {
		if result.Status == "error" {
			hasErrors = true
			break
		}
	}

	// Retornar resposta apropriada
	if hasErrors {
		h.logger.Warn("Errors found in processing one or more products")
		c.JSON(http.StatusMultiStatus, gin.H{
			"message": "Some products were processed with errors",
			"results": results,
		})
		return
	}

	// Resposta de sucesso se todos os produtos foram criados e publicados
	h.logger.Info("Product(s) created successfully", zap.Int("count", len(products)))
	c.JSON(http.StatusCreated, gin.H{
		"message": "Product(s) created successfully",
		"results": results,
	})
}

// GetAll godoc
//
//	@Summary		Get all products
//	@Description	Recupera uma lista de todos os produtos do banco de dados
//	@Tags			Products
//	@Produce		json
//	@Success		200	{array}	dtos.ProductResponseDTO	"Products retrieved successfully"
//	@Security		bearerAuth
//	@Router			/products [get]
func (h *ProductHandler) GetAll(c *gin.Context) {
	// Call the use case to retrieve all products
	products, err := h.productUseCase.GetAll(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to retrieve products", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve products"})
		return
	}

	// Map the domain models to response DTOs
	var responseDTOs []dtos.ProductResponseDTO
	for _, p := range products {
		responseDTOs = append(responseDTOs, dtos.ProductResponseDTO{
			SKU:          p.SKU,
			Name:         p.Name,
			Description:  p.Description,
			Price:        p.Price,
			Category:     p.Category,
			Link:         p.Link,
			ImageLink:    p.ImageLink,
			Availability: p.Availability,
			CreatedBy:    p.CreatedBy,
			CreatedAt:    p.CreatedAt,
			UpdatedAt:    p.UpdatedAt,
		})
	}

	// Return the list of products
	h.logger.Info("Products retrieved successfully", zap.Int("count", len(products)))
	c.JSON(http.StatusOK, responseDTOs)
}

// GetBySKU godoc
//
//	@Summary		Get a product by SKU
//	@Description	Recupera os detalhes de um único produto usando seu SKU
//	@Tags			Products
//	@Produce		json
//	@Param			sku	path		int						true	"Product SKU"
//	@Success		200	{object}	dtos.ProductResponseDTO	"Product retrieved successfully"
//	@Security		bearerAuth
//	@Router			/products/{sku} [get]
func (h *ProductHandler) GetBySKU(c *gin.Context) {
	// Parse the SKU from the URL parameter
	sku, err := strconv.Atoi(c.Param("sku"))
	if err != nil {
		h.logger.Error("Invalid SKU format", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid SKU format"})
		return
	}

	// Call the use case to retrieve the product by its SKU
	product, err := h.productUseCase.GetBySKU(c.Request.Context(), sku)
	if err != nil {
		h.logger.Warn("Product not found", zap.Int("sku", sku), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// Map the domain model to a response DTO
	responseDTO := dtos.ProductResponseDTO{
		SKU:          product.SKU,
		Name:         product.Name,
		Description:  product.Description,
		Price:        product.Price,
		Category:     product.Category,
		Link:         product.Link,
		ImageLink:    product.ImageLink,
		Availability: product.Availability,
		CreatedAt:    product.CreatedAt,
		CreatedBy:    product.CreatedBy,
		UpdatedAt:    product.UpdatedAt,
	}

	// Return the product details
	h.logger.Info("Product retrieved successfully", zap.Int("sku", sku))
	c.JSON(http.StatusOK, responseDTO)
}

// Update godoc
//
//	@Summary		Update one or more products
//	@Description	Atualiza produtos existentes com base em um único objeto ou em uma matriz de objetos no corpo da solicitação
//	@Tags			Products
//	@Accept			json
//	@Produce		json
//	@Param			products	body		[]dtos.UpdateProductDTO		true	"Product data to update"
//	@Success		200			{object}	dtos.UpdateProductResponse	"Product(s) updated successfully"
//	@Security		bearerAuth
//	@Router			/products [put]
func (h *ProductHandler) Update(c *gin.Context) {
	// Read the raw request body to handle both single and multiple updates
	body, err := c.GetRawData()
	if err != nil {
		h.logger.Error("Failed to read request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Try to unmarshal as an array of update objects
	var inputs []dtos.UpdateProductDTO
	err = json.Unmarshal(body, &inputs)
	if err != nil {
		// Try to unmarshal as a single update object
		var singleInput dtos.UpdateProductDTO
		if err_single := json.Unmarshal(body, &singleInput); err_single != nil {
			h.logger.Error("Invalid request body format", zap.Error(err_single))
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request body format. Must be a product object or an array of product objects.",
				"details": err_single.Error(),
			})
			return
		}
		// Wrap the single object in a slice
		inputs = []dtos.UpdateProductDTO{singleInput}
	}

	// Define a struct for batch operation results
	type batchResult struct {
		Index  int               `json:"index"`
		SKU    int               `json:"sku"`
		Status string            `json:"status"`
		Errors map[string]string `json:"errors,omitempty"`
	}
	var results []batchResult
	var products []*model.Product

	// Retrieve the user email from the context
	userEmail, exists := c.Get("userEmail")
	if !exists {
		h.logger.Error("User email not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User email not found"})
		return
	}

	userEmailStr, ok := userEmail.(string)
	if !ok {
		h.logger.Error("Invalid user email format in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user email data"})
		return
	}

	// Validate and process each item in the update request
	for i, input := range inputs {
		product := &model.Product{
			SKU:          input.Sku,
			Name:         input.Name,
			Description:  input.Description,
			Price:        input.Price,
			Category:     input.Category,
			Link:         input.Link,
			ImageLink:    input.ImageLink,
			Availability: input.Availability,
		}

		if errors := h.validator.ValidateProduct(product); errors != nil {
			h.logger.Warn("Validation errors for product", zap.Int("index", i), zap.Any("errors", errors))
			results = append(results, batchResult{
				Index:  i,
				SKU:    product.SKU,
				Status: "error",
				Errors: errors,
			})
		} else {
			results = append(results, batchResult{
				Index:  i,
				SKU:    product.SKU,
				Status: "ok",
			})
			products = append(products, product)
		}
	}

	// Return validation errors if exists
	hasErrors := false
	for _, result := range results {
		if result.Status == "error" {
			hasErrors = true
			break
		}
	}
	if hasErrors {
		h.logger.Warn("Validation errors found in one or more products")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation errors found in one or more products",
			"results": results,
		})
		return
	}

	// Call the use case to perform the update
	updateErrors := h.productUseCase.Update(c.Request.Context(), products, userEmailStr)
	if updateErrors != nil {
		// If the use case returns errors, update the results
		for i, product := range products {
			if errMsg, exists := updateErrors[product.SKU]; exists {
				results[i].Status = "error"
				if results[i].Errors == nil {
					results[i].Errors = make(map[string]string)
				}
				results[i].Errors["update_error"] = errMsg
				h.logger.Error("Failed to update product", zap.Int("sku", product.SKU), zap.String("error", errMsg))
			}
		}
		h.logger.Error("Failed to update one or more products")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to update one or more products",
			"results": results,
		})
		return
	}

	// Return a success response
	h.logger.Info("Product(s) updated successfully", zap.Int("count", len(products)))
	c.JSON(http.StatusOK, gin.H{
		"message": "Product(s) updated successfully",
		"results": results,
	})
}

// Delete godoc
//
//	@Summary		Delete one or more products
//	@Description	Exclui produtos do banco de dados com base em um único SKU ou em uma matriz de SKUs no corpo da solicitação
//	@Tags			Products
//	@Accept			json
//	@Produce		json
//	@Param			skus	body		[]int						true	"SKUs of products to delete"
//	@Success		200		{object}	dtos.DeleteProductResponse	"Product(s) deleted successfully"
//	@Security		bearerAuth
//	@Router			/products [delete]
func (h *ProductHandler) Delete(c *gin.Context) {
	// Read the raw request body to handle both single and multiple SKUs
	body, err := c.GetRawData()
	if err != nil {
		h.logger.Error("Failed to read request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Try to unmarshal as an array of SKUs
	var skus []int
	err = json.Unmarshal(body, &skus)
	if err != nil {
		// Try to unmarshal as a single SKU
		var singleSKU int
		if err_single := json.Unmarshal(body, &singleSKU); err_single != nil {
			h.logger.Error("Invalid request body format", zap.Error(err_single))
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request body format. Must be a SKU or an array of SKUs.",
				"details": err_single.Error(),
			})
			return
		}
		// Wrap the single SKU in a slice
		skus = []int{singleSKU}
	}

	// Define a struct for batch results
	type batchResult struct {
		Index  int               `json:"index"`
		SKU    int               `json:"sku"`
		Status string            `json:"status"`
		Errors map[string]string `json:"errors,omitempty"`
	}
	var results []batchResult

	// Get user email from context
	userEmail, exists := c.Get("userEmail")
	if !exists {
		h.logger.Error("User email not found in context", zap.String("operation", "delete"))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User email not found"})
		return
	}

	userEmailStr, ok := userEmail.(string)
	if !ok {
		h.logger.Error("Invalid user email format in context", zap.String("operation", "delete"))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user email data"})
		return
	}

	// Initialize results for each SKU to be deleted
	for i, sku := range skus {
		results = append(results, batchResult{
			Index:  i,
			SKU:    sku,
			Status: "ok",
		})
	}

	// Call the use case to perform the deletion
	deleteErrors := h.productUseCase.Delete(c.Request.Context(), skus, userEmailStr)
	if deleteErrors != nil {
		// If the use case returns errors, update the results
		for i, sku := range skus {
			if errMsg, exists := deleteErrors[sku]; exists {
				results[i].Status = "error"
				if results[i].Errors == nil {
					results[i].Errors = make(map[string]string)
				}
				results[i].Errors["delete_error"] = errMsg
				h.logger.Error("Failed to delete product", zap.Int("sku", sku), zap.String("error", errMsg))
			}
		}
		h.logger.Error("Failed to delete one or more products")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to delete one or more products",
			"results": results,
		})
		return
	}

	// Return a success response
	h.logger.Info("Product(s) deleted successfully", zap.Int("count", len(skus)))
	c.JSON(http.StatusOK, gin.H{
		"message": "Product(s) deleted successfully",
		"results": results,
	})
}
