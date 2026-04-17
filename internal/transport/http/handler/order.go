package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"go-seckill/internal/errs"
	"go-seckill/internal/service"
	httpmiddleware "go-seckill/internal/transport/http/middleware"
	httpresponse "go-seckill/internal/transport/http/response"
)

type OrderHandler struct {
	orderService *service.OrderService
}

func NewOrderHandler(orderService *service.OrderService) *OrderHandler {
	return &OrderHandler{orderService: orderService}
}

// Detail godoc
// @Summary 订单详情
// @Tags order
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Envelope
// @Router /api/v1/orders/{orderNo} [get]
func (h *OrderHandler) Detail(c *gin.Context) {
	order, err := h.orderService.GetByOrderNo(c.Request.Context(), c.Param("orderNo"))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrderNotFound):
			httpresponse.Error(c, http.StatusNotFound, errs.CodeOrderNotFound, "")
		default:
			httpresponse.Error(c, http.StatusInternalServerError, errs.CodeInternalError, "")
		}
		return
	}

	currentUser, ok := httpmiddleware.GetCurrentUser(c)
	if !ok || order.UserID != currentUser.UserID {
		httpresponse.Error(c, http.StatusUnauthorized, errs.CodeUnauthorized, "order does not belong to current user")
		return
	}

	httpresponse.Success(c, order)
}

// ListMine godoc
// @Summary 当前用户订单列表
// @Tags order
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Envelope
// @Router /api/v1/orders/me [get]
func (h *OrderHandler) ListMine(c *gin.Context) {
	currentUser, ok := httpmiddleware.GetCurrentUser(c)
	if !ok {
		httpresponse.Error(c, http.StatusUnauthorized, errs.CodeUnauthorized, "")
		return
	}

	orders, err := h.orderService.ListByUserID(c.Request.Context(), currentUser.UserID)
	if err != nil {
		httpresponse.Error(c, http.StatusInternalServerError, errs.CodeInternalError, "")
		return
	}

	httpresponse.Success(c, orders)
}
