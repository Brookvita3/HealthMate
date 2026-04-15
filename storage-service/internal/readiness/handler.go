package readiness

import (
	"fmt"
	"log"
	"net/http"
	"storage-service/internal/common"
	"storage-service/internal/web/helpers"

	"github.com/gin-gonic/gin"
)

type predictRequest struct {
	HeartRate      float64 `json:"heart_rate"      binding:"required"`
	SleepDuration  float64 `json:"sleep_duration"` // 0 is valid (no sleep data)
	StressLevel    string  `json:"stress_level"    binding:"required"`
	BloodOxygen    float64 `json:"blood_oxygen"    binding:"required"`
	Steps          float64 `json:"steps"`
	CaloriesBurned float64 `json:"calories_burned"`
}

type predictResponse struct {
	ReadinessScore *float64 `json:"readiness_score"`
}

func validatePredictRequest(req *predictRequest) error {
	if req.HeartRate < 30 || req.HeartRate > 300 {
		return fmt.Errorf("heart_rate must be between 30 and 300")
	}
	if req.SleepDuration < 0 || req.SleepDuration > 24 {
		return fmt.Errorf("sleep_duration must be between 0 and 24")
	}
	if req.BloodOxygen < 50 || req.BloodOxygen > 100 {
		return fmt.Errorf("blood_oxygen must be between 50 and 100")
	}
	if req.Steps < 0 {
		return fmt.Errorf("steps must be non-negative")
	}
	if req.CaloriesBurned < 0 {
		return fmt.Errorf("calories_burned must be non-negative")
	}
	return nil
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
// @Router /metrics/readiness [post]
// @Security BearerAuth
func PredictHandler(c *gin.Context) {
	var req predictRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.HandleError(c, common.ErrInvalidRequest)
		return
	}

	if err := validatePredictRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, helpers.ErrorResponse{Error: err.Error()})
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
		log.Printf("readiness: onnx inference error: %v", err)
		c.JSON(http.StatusOK, predictResponse{ReadinessScore: nil})
		return
	}

	c.JSON(http.StatusOK, predictResponse{ReadinessScore: &score})
}
