package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"go-seckill/internal/errs"
	"go-seckill/internal/service"
	httpresponse "go-seckill/internal/transport/http/response"
)

type ProductHandler struct {
	productService *service.ProductService
}

type CreateProductRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=128"`
	Description string `json:"description" binding:"max=1000"`
	Price       int64  `json:"price" binding:"required,min=1"`
	Status      int8   `json:"status"`
}

func NewProductHandler(productService *service.ProductService) *ProductHandler {
	return &ProductHandler{productService: productService}
}

// Create godoc
// @Summary 创建商品
// @Tags product
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body handler.CreateProductRequest true "创建商品请求"
// @Success 200 {object} response.Envelope
// @Router /api/v1/products [post]
func (h *ProductHandler) Create(c *gin.Context) {
	var request CreateProductRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httpresponse.Error(c, http.StatusBadRequest, errs.CodeBadRequest, err.Error())
		return
	}

	product, err := h.productService.Create(c.Request.Context(), service.CreateProductInput{
		Name:        request.Name,
		Description: request.Description,
		Price:       request.Price,
		Status:      request.Status,
	})
	if err != nil {
		httpresponse.Error(c, http.StatusInternalServerError, errs.CodeInternalError, "")
		return
	}

	httpresponse.Success(c, product)
}

// List godoc
// @Summary 商品列表
// @Tags product
// @Produce json
// @Success 200 {object} response.Envelope
// @Router /api/v1/products [get]
func (h *ProductHandler) List(c *gin.Context) {
	products, err := h.productService.List(c.Request.Context())
	if err != nil {
		httpresponse.Error(c, http.StatusInternalServerError, errs.CodeInternalError, "")
		return
	}

	httpresponse.Success(c, products)
}
