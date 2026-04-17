package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"go-seckill/internal/errs"
	"go-seckill/internal/service"
	httpresponse "go-seckill/internal/transport/http/response"
)

type ActivityHandler struct {
	activityService *service.ActivityService
}

type CreateActivityRequest struct {
	ProductID  uint64    `json:"product_id" binding:"required"`
	Name       string    `json:"name" binding:"required,min=2,max=128"`
	StartTime  time.Time `json:"start_time" binding:"required"`
	EndTime    time.Time `json:"end_time" binding:"required"`
	Status     int8      `json:"status"`
	TotalStock int       `json:"total_stock" binding:"required,min=1"`
}

func NewActivityHandler(activityService *service.ActivityService) *ActivityHandler {
	return &ActivityHandler{activityService: activityService}
}

// Create godoc
// @Summary 创建秒杀活动
// @Tags activity
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body handler.CreateActivityRequest true "创建秒杀活动请求"
// @Success 200 {object} response.Envelope
// @Router /api/v1/activities [post]
func (h *ActivityHandler) Create(c *gin.Context) {
	var request CreateActivityRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httpresponse.Error(c, http.StatusBadRequest, errs.CodeBadRequest, err.Error())
		return
	}

	err := h.activityService.Create(c.Request.Context(), service.CreateActivityInput{
		ProductID:  request.ProductID,
		Name:       request.Name,
		StartTime:  request.StartTime,
		EndTime:    request.EndTime,
		Status:     request.Status,
		TotalStock: request.TotalStock,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrProductMissing):
			httpresponse.Error(c, http.StatusBadRequest, errs.CodeBadRequest, "product does not exist")
		case errors.Is(err, service.ErrInvalidActivityTime):
			httpresponse.Error(c, http.StatusBadRequest, errs.CodeBadRequest, "start_time must be before end_time")
		default:
			httpresponse.Error(c, http.StatusInternalServerError, errs.CodeInternalError, "")
		}
		return
	}

	httpresponse.Success(c, gin.H{"created": true})
}

// List godoc
// @Summary 秒杀活动列表
// @Tags activity
// @Produce json
// @Success 200 {object} response.Envelope
// @Router /api/v1/activities [get]
func (h *ActivityHandler) List(c *gin.Context) {
	activities, err := h.activityService.List(c.Request.Context())
	if err != nil {
		httpresponse.Error(c, http.StatusInternalServerError, errs.CodeInternalError, "")
		return
	}

	httpresponse.Success(c, activities)
}

// Detail godoc
// @Summary 秒杀活动详情
// @Tags activity
// @Produce json
// @Success 200 {object} response.Envelope
// @Router /api/v1/activities/{id} [get]
func (h *ActivityHandler) Detail(c *gin.Context) {
	activityID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpresponse.Error(c, http.StatusBadRequest, errs.CodeBadRequest, "invalid activity id")
		return
	}

	activity, err := h.activityService.GetByID(c.Request.Context(), activityID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrActivityNotFound):
			httpresponse.Error(c, http.StatusBadRequest, errs.CodeBadRequest, "activity does not exist")
		default:
			httpresponse.Error(c, http.StatusInternalServerError, errs.CodeInternalError, "")
		}
		return
	}

	httpresponse.Success(c, activity)
}

// Preheat godoc
// @Summary 预热秒杀活动缓存
// @Tags activity
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Envelope
// @Router /api/v1/activities/{id}/preheat [post]
func (h *ActivityHandler) Preheat(c *gin.Context) {
	activityID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpresponse.Error(c, http.StatusBadRequest, errs.CodeBadRequest, "invalid activity id")
		return
	}

	if err := h.activityService.Preheat(c.Request.Context(), activityID); err != nil {
		switch {
		case errors.Is(err, service.ErrActivityNotFound):
			httpresponse.Error(c, http.StatusBadRequest, errs.CodeBadRequest, "activity does not exist")
		default:
			httpresponse.Error(c, http.StatusInternalServerError, errs.CodeInternalError, "")
		}
		return
	}

	httpresponse.Success(c, gin.H{"preheated": true, "activity_id": activityID})
}
