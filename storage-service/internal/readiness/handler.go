package readiness

import (
	"net/http"
	"storage-service/internal/common"
	"storage-service/internal/web/helpers"

	"github.com/gin-gonic/gin"
)

type predictRequest struct {
	HeartRate      float64 `json:"heart_rate"      binding:"required"`
	SleepDuration  float64 `json:"sleep_duration"  binding:"required"`
	StressLevel    string  `json:"stress_level"    binding:"required"`
	BloodOxygen    float64 `json:"blood_oxygen"    binding:"required"`
	Steps          float64 `json:"steps"`
	CaloriesBurned float64 `json:"calories_burned"`
}

type predictResponse struct {
	ReadinessScore float64 `json:"readiness_score"`
}

// PredictHandler handles readiness score prediction requests.
// @Summary Predict physical readiness score
// @Description Predict a readiness score (0–100) from health metrics using the on-device ML model
// @Tags readiness
// @Accept json
// @Produce json
// @Param body body predictRequest true "Health metrics input"
// @Success 200 {object} predictResponse
// @Failure 400 {object} helpers.ErrorResponse
// @Failure 500 {object} helpers.ErrorResponse
// @Router /metrics/readiness [post]
// @Security BearerAuth
func PredictHandler(c *gin.Context) {
	var req predictRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.HandleError(c, common.ErrInvalidRequest)
		return
	}

	score, err := Predict(Input{
		HeartRate:      req.HeartRate,
		SleepDuration:  req.SleepDuration,
		StressLevel:    req.StressLevel,
		BloodOxygen:    req.BloodOxygen,
		Steps:          req.Steps,
		CaloriesBurned: req.CaloriesBurned,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, helpers.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, predictResponse{ReadinessScore: score})
}
