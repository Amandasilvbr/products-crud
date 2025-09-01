package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

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

type batchResult struct {
	Index  int               `json:"index"`
	SKU    int               `json:"sku"`
	Status string            `json:"status"`
	Errors map[string]string `json:"errors,omitempty"`
}

// Create godoc
//
//	@Summary		Cria um ou mais produtos
//	@Description	Cria novos produtos com base em um único objeto ou em uma matriz de objetos no corpo da solicitação
//	@Tags			Products
//	@Accept			json
//	@Produce		json
//	@Param			products	body		[]dtos.CreateProductDTO		true	"Product data to create"
//	@Success		200			{object}	dtos.CreateProductResponse	"Product(s) created successfully"
//	@Security		bearerAuth
//	@Router			/products [post]
func (h *ProductHandler) Create(c *gin.Context) {
	// Lê o corpo da requisição
	body, err := h.readRequestBody(c)
	if err != nil {
		return
	}

	var inputs []dtos.CreateProductDTO
	if err := json.Unmarshal(body, &inputs); err != nil {
		var singleInput dtos.CreateProductDTO
		if errSingle := json.Unmarshal(body, &singleInput); errSingle != nil {
			h.logger.Error("Invalid request body format", zap.Error(errSingle))
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request body format. Must be a product object or an array of product objects.",
				"details": errSingle.Error(),
			})
			return
		}
		inputs = []dtos.CreateProductDTO{singleInput}
	}

	var results []batchResult
	var products []*model.Product

	// Obtém informações do usuário
	userNameVal, ok := c.Get("userName")
	if !ok {
		h.logger.Error("User not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userName, _ := userNameVal.(string)

	userEmailVal, ok := c.Get("userEmail")
	if !ok {
		h.logger.Error("User email not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User email not found"})
		return
	}
	userEmail, _ := userEmailVal.(string)

	// Validate and prepare products
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

		if errs := h.validator.ValidateProduct(product); errs != nil {
			h.logger.Warn("Validation errors for product", zap.Int("index", i), zap.Any("errors", errs))
			results = append(results, batchResult{
				Index:  i,
				Status: "error",
				Errors: errs,
			})
		} else {
			results = append(results, batchResult{
				Index:  i,
				Status: "pending",
			})
			products = append(products, product)
		}
	}

	// Create products
	var createErrors, publishErrors map[int]string
	if len(products) > 0 {
		createErrors, publishErrors = h.productUseCase.Create(c.Request.Context(), products, userEmail)
	}

	// Refresh results
	for i, product := range products {
		resultIndex := -1
		for j, r := range results {
			if r.Index == i && r.Status == "pending" {
				resultIndex = j
				break
			}
		}
		if resultIndex == -1 {
			continue
		}

		if errMsg, exists := createErrors[product.SKU]; exists {
			if strings.Contains(errMsg, "already exists") {
				results[resultIndex].Status = "conflict"
			} else {
				results[resultIndex].Status = "error"
			}
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

	// Calls the helper to determinate the final status HTTP
	status := determineHTTPStatus(results)

	h.logger.Info("Batch processed", zap.Int("http_status", status))
	c.JSON(status, gin.H{
		"message": "Batch processed",
		"results": results,
	})
}

// GetAll godoc
//
//	@Summary		Recupera todos os produtos
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
//	@Summary		Recupera um produto pelo SKU
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
//	@Summary		Atualiza um ou mais produtos
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
	body, err := h.readRequestBody(c)
	if err != nil {
		return
	}

	// Try to unmarshal the request body into an array of UpdateProductDTO
	var inputs []dtos.UpdateProductDTO
	if err := json.Unmarshal(body, &inputs); err != nil {
		// If unmarshalling as an array fails, try to unmarshal as a single object
		var singleInput dtos.UpdateProductDTO
		if errSingle := json.Unmarshal(body, &singleInput); errSingle != nil {
			// Log error and return 400 if the body format is invalid
			h.logger.Error("Invalid request body format", zap.Error(errSingle))
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request body format. Must be a product object or an array of product objects.",
				"details": errSingle.Error(),
			})
			return
		}
		// Wrap the single object in a slice to unify processing
		inputs = []dtos.UpdateProductDTO{singleInput}
	}

	// Retrieve user email from the context
	userEmailVal, exists := c.Get("userEmail")
	if !exists {
		h.logger.Error("User email not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User email not found"})
		return
	}
	userEmail, ok := userEmailVal.(string)
	if !ok {
		h.logger.Error("Invalid user email format in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user email data"})
		return
	}

	var results []batchResult     
	var products []*model.Product 

	// Loop through each input and validate it
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

		// Validate the product
		if errs := h.validator.ValidateUpdateProduct(product); errs != nil {
			// If validation fails, mark result as error with details
			results = append(results, batchResult{
				Index:  i,
				SKU:    product.SKU,
				Status: "error",
				Errors: errs,
			})
		} else {
			// If validation passes, mark result as ok and add to products list
			results = append(results, batchResult{
				Index:  i,
				SKU:    product.SKU,
				Status: "ok",
			})
			products = append(products, product)
		}
	}

	// Call the use case to perform the actual update in the database
	updateErrors := h.productUseCase.Update(c.Request.Context(), products, userEmail)

	// If the use case returned errors, update the results accordingly
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

	// Determine the final HTTP status based on the results
	status := determineHTTPStatus(results)

	// Log the processing result and send the response
	h.logger.Info("Products processed in updating", zap.Int("count", len(products)))
	c.JSON(status, gin.H{
		"message": "Products processed",
		"results": results,
	})
}

// Delete godoc
//
//	@Summary		Deleta um ou mais produtos
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
	body, err := h.readRequestBody(c)
	if err != nil {
		return
	}

	// Try to unmarshal as an array of SKUs
	var skus []int
	if err := json.Unmarshal(body, &skus); err != nil {
		// Try to unmarshal as a single SKU
		var singleSKU int
		if errSingle := json.Unmarshal(body, &singleSKU); errSingle != nil {
			h.logger.Error("Invalid request body format", zap.Error(errSingle))
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request body format. Must be a SKU or an array of SKUs.",
				"details": errSingle.Error(),
			})
			return
		}
		skus = []int{singleSKU}
	}

	var results []batchResult

	// Get user email from context
	userEmailVal, exists := c.Get("userEmail")
	if !exists {
		h.logger.Error("User email not found in context", zap.String("operation", "delete"))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User email not found"})
		return
	}
	userEmail, ok := userEmailVal.(string)
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
	deleteErrors := h.productUseCase.Delete(c.Request.Context(), skus, userEmail)
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

	// Determine final HTTP status using the helper
	status := determineHTTPStatus(results)

	h.logger.Info("Products processed for deletion", zap.Int("count", len(skus)))
	c.JSON(status, gin.H{
		"message": "Products processed for deletion",
		"results": results,
	})
}

func (h *ProductHandler) readRequestBody(c *gin.Context) ([]byte, error) {
	body, err := c.GetRawData()
	if err != nil {
		h.logger.Error("Failed to read request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return nil, err
	}
	return body, nil
}

func determineHTTPStatus(results []batchResult) int {
	allOk := true
	allConflicts := true
	for _, r := range results {
		if r.Status != "ok" {
			allOk = false
		}
		if r.Status != "conflict" {
			allConflicts = false
		}
	}
	switch {
	case allOk:
		return http.StatusCreated // 201
	case allConflicts:
		return http.StatusConflict // 409
	default:
		return http.StatusMultiStatus // 207
	}
}
