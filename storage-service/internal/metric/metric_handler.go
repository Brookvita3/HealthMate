package metric

import (
	"storage-service/internal/common"
	"storage-service/internal/web/helpers"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// GetChartData handles getting chart data for metrics
// @Summary Get chart data for metrics
// @Description Get aggregated chart data for a specific metric type and user
// @Tags metrics
// @Accept json
// @Produce json
// @Param user_id query string true "User ID"
// @Param metric_type query string true "Metric Type (e.g. heart_rate, steps)"
// @Param range query string true "Time Range (e.g. 24h, 7d, 30d, custom)"
// @Param start_time query string false "Start Time (RFC3339) - required for custom range"
// @Param end_time query string false "End Time (RFC3339) - required for custom range"
// @Success 200 {object} helpers.DataResponse
// @Failure 400 {object} helpers.ErrorResponse
// @Router /metrics/charts [get]
// @Security BearerAuth
func (h *Handler) GetChartData(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		helpers.HandleError(c, common.ErrInvalidRequest)
		return
	}

	metricType := c.Query("metric_type")
	if metricType == "" {
		helpers.HandleError(c, common.ErrInvalidRequest)
		return
	}

	timeRange := c.Query("range")
	var start, end *time.Time

	if timeRange == RangeCustom {
		startStr := c.Query("start_time")
		endStr := c.Query("end_time")

		startTime, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			helpers.HandleError(c, common.ErrInvalidRequest)
			return
		}
		endTime, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			helpers.HandleError(c, common.ErrInvalidRequest)
			return
		}
		start = &startTime
		end = &endTime
	}

	data, err := h.service.GetChartData(c.Request.Context(), userID, metricType, timeRange, start, end)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	helpers.RespondData(c, data)
}

// GetThresholds handles getting user health thresholds
// @Summary Get user health thresholds
// @Description Get current health thresholds for a user (including defaults)
// @Tags metrics
// @Accept json
// @Produce json
// @Param user_id query string true "User ID"
// @Success 200 {object} helpers.DataResponse
// @Failure 400 {object} helpers.ErrorResponse
// @Router /metrics/thresholds [get]
// @Security BearerAuth
func (h *Handler) GetThresholds(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		helpers.HandleError(c, common.ErrInvalidRequest)
		return
	}

	data, err := h.service.GetThresholds(c.Request.Context(), userID)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	helpers.RespondData(c, data)
}

// SetUserThreshold handles setting user health threshold
// @Summary Set user health threshold
// @Description Create or update a health threshold for a user
// @Tags metrics
// @Accept json
// @Produce json
// @Param threshold body UserThreshold true "Threshold definition"
// @Success 200 {object} helpers.OKResponse
// @Failure 400 {object} helpers.ErrorResponse
// @Router /metrics/thresholds [post]
// @Security BearerAuth
func (h *Handler) SetUserThreshold(c *gin.Context) {
	var threshold UserThreshold
	if err := c.ShouldBindJSON(&threshold); err != nil {
		helpers.HandleError(c, common.ErrInvalidRequest)
		return
	}

	if threshold.UserID == "" || threshold.MetricID == "" {
		helpers.HandleError(c, common.ErrInvalidRequest)
		return
	}

	err := h.service.SetUserThreshold(c.Request.Context(), threshold)
	if err != nil {
		helpers.HandleError(c, err)
		return
	}

	helpers.RespondOK(c, "Threshold updated successfully")
}
