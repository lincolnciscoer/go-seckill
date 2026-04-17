package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"go-seckill/internal/errs"
	"go-seckill/internal/service"
	httpmiddleware "go-seckill/internal/transport/http/middleware"
	httpresponse "go-seckill/internal/transport/http/response"
)

type SeckillHandler struct {
	seckillService *service.SeckillService
}

func NewSeckillHandler(seckillService *service.SeckillService) *SeckillHandler {
	return &SeckillHandler{seckillService: seckillService}
}

// Attempt godoc
// @Summary 发起同步版秒杀
// @Tags seckill
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Envelope
// @Router /api/v1/seckill/activities/{id}/attempt [post]
func (h *SeckillHandler) Attempt(c *gin.Context) {
	currentUser, ok := httpmiddleware.GetCurrentUser(c)
	if !ok {
		httpresponse.Error(c, http.StatusUnauthorized, errs.CodeUnauthorized, "")
		return
	}

	activityID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpresponse.Error(c, http.StatusBadRequest, errs.CodeBadRequest, "invalid activity id")
		return
	}

	order, err := h.seckillService.Attempt(c.Request.Context(), currentUser.UserID, activityID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrActivityNotFound):
			httpresponse.Error(c, http.StatusBadRequest, errs.CodeBadRequest, "activity does not exist")
		case errors.Is(err, service.ErrActivityInactive):
			httpresponse.Error(c, http.StatusBadRequest, errs.CodeActivityInactive, "")
		case errors.Is(err, service.ErrActivityNotStarted):
			httpresponse.Error(c, http.StatusBadRequest, errs.CodeActivityNotStarted, "")
		case errors.Is(err, service.ErrActivityEnded):
			httpresponse.Error(c, http.StatusBadRequest, errs.CodeActivityEnded, "")
		case errors.Is(err, service.ErrSoldOut):
			httpresponse.Error(c, http.StatusConflict, errs.CodeSoldOut, "")
		case errors.Is(err, service.ErrRepeatOrder):
			httpresponse.Error(c, http.StatusConflict, errs.CodeDuplicateOrder, "")
		default:
			httpresponse.Error(c, http.StatusInternalServerError, errs.CodeInternalError, "")
		}
		return
	}

	httpresponse.Success(c, order)
}
