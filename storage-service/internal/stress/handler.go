package stress

import (
	"log"
	"net/http"
	"storage-service/internal/common"
	"storage-service/internal/web/helpers"

	"github.com/gin-gonic/gin"
)

// ─── Predict ─────────────────────────────────────────────────────────────────

type predictRequest struct {
	HRMean   float64 `json:"hr_mean"   binding:"required"`
	HRStd    float64 `json:"hr_std"`
	RMSSD    float64 `json:"rmssd"     binding:"required"`
	TempMean float64 `json:"temp_mean" binding:"required"`
}

type predictResponse struct {
	Label        int     `json:"label"`
	LabelName    string  `json:"label_name"`
	ProbStress   float64 `json:"prob_stress"`
	ProbBaseline float64 `json:"prob_baseline"`
	Calibrated   bool    `json:"calibrated"`
}

// PredictHandler handles POST /metrics/stress/predict
// @Summary Predict stress level
// @Description Classify a 60-second window as Baseline or Stress using the WESAD model
// @Tags stress
// @Accept json
// @Produce json
// @Param body body predictRequest true "60-second window features"
// @Success 200 {object} predictResponse
// @Failure 400 {object} helpers.ErrorResponse
// @Failure 503 {object} helpers.ErrorResponse
// @Router /metrics/stress/predict [post]
// @Security BearerAuth
func PredictHandler(c *gin.Context) {
	if !IsReady() {
		c.JSON(http.StatusServiceUnavailable, helpers.ErrorResponse{Error: "stress model not available"})
		return
	}

	var req predictRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.HandleError(c, common.ErrInvalidRequest)
		return
	}

	userID := c.GetString("sub")
	norm, calibrated := GetUserNorm(userID)
	if norm == nil {
		c.JSON(http.StatusInternalServerError, helpers.ErrorResponse{Error: "population norm stats not loaded"})
		return
	}

	result, err := Predict(RawFeatures{
		HRMean:   req.HRMean,
		HRStd:    req.HRStd,
		RMSSD:    req.RMSSD,
		TempMean: req.TempMean,
	}, norm)
	if err != nil {
		log.Printf("stress: inference error: %v", err)
		c.JSON(http.StatusInternalServerError, helpers.ErrorResponse{Error: "inference failed"})
		return
	}

	c.JSON(http.StatusOK, predictResponse{
		Label:        int(result.Label),
		LabelName:    result.LabelName,
		ProbStress:   result.ProbStress,
		ProbBaseline: result.ProbBaseline,
		Calibrated:   calibrated,
	})
}

// ─── Calibrate ───────────────────────────────────────────────────────────────

type calibrateWindow struct {
	HRMean   float64 `json:"hr_mean"   binding:"required"`
	HRStd    float64 `json:"hr_std"`
	RMSSD    float64 `json:"rmssd"     binding:"required"`
	TempMean float64 `json:"temp_mean" binding:"required"`
}

type calibrateRequest struct {
	Windows []calibrateWindow `json:"windows" binding:"required,min=1"`
}

// CalibrateHandler handles POST /metrics/stress/calibrate
// @Summary Calibrate stress model for a user
// @Description Submit ~5 minutes of baseline windows (each 60 s) to compute per-user norm stats
// @Tags stress
// @Accept json
// @Produce json
// @Param body body calibrateRequest true "Baseline windows"
// @Success 200 {object} helpers.OKResponse
// @Failure 400 {object} helpers.ErrorResponse
// @Router /metrics/stress/calibrate [post]
// @Security BearerAuth
func CalibrateHandler(c *gin.Context) {
	if !IsReady() {
		c.JSON(http.StatusServiceUnavailable, helpers.ErrorResponse{Error: "stress model not available"})
		return
	}

	var req calibrateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.HandleError(c, common.ErrInvalidRequest)
		return
	}

	windows := make([]RawFeatures, len(req.Windows))
	for i, w := range req.Windows {
		windows[i] = RawFeatures{
			HRMean:   w.HRMean,
			HRStd:    w.HRStd,
			RMSSD:    w.RMSSD,
			TempMean: w.TempMean,
		}
	}

	userID := c.GetString("sub")
	norm := ComputeUserNorm(windows)
	SetUserNorm(userID, norm)

	log.Printf("stress: calibration saved for user %s (%d windows)", userID, len(windows))
	helpers.RespondOK(c, "calibration saved")
}
