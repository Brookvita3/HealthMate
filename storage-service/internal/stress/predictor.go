package stress

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sync"

	ort "github.com/yalue/onnxruntime_go"
)

const (
	LabelBaseline = 1
	LabelStress   = 2
)

var featureOrder = []string{"hr_mean", "hr_std", "rmssd", "temp_mean"}

type StressLabel int

func (s StressLabel) String() string {
	if s == LabelStress {
		return "Stress"
	}
	return "Baseline"
}

type NormStat struct {
	Mean float64 `json:"mean"`
	Std  float64 `json:"std"`
}

// SubjectNorm maps feature name → NormStat for one user.
type SubjectNorm map[string]NormStat

// NormStats maps subject ID (or "_population") → SubjectNorm.
type NormStats map[string]SubjectNorm

type PredictResult struct {
	Label        StressLabel
	LabelName    string
	ProbBaseline float64
	ProbStress   float64
}

type RawFeatures struct {
	HRMean   float64
	HRStd    float64
	RMSSD    float64
	TempMean float64
}

// Predictor wraps the ONNX session, population norms, and per-user calibration cache.
type Predictor struct {
	session    *ort.DynamicAdvancedSession
	population NormStats
	userNorms  map[string]SubjectNorm
	mu         sync.RWMutex
}

var global *Predictor

// Init loads the stress model and norm stats.
// onnxLibPath is accepted for API compatibility but ORT environment must already
// be initialised by the readiness package before this is called.
func Init(onnxLibPath, modelPath, normStatsPath string) error {
	session, err := ort.NewDynamicAdvancedSession(
		modelPath,
		[]string{"float_input"},
		[]string{"label", "probabilities"},
		nil,
	)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	raw, err := os.ReadFile(normStatsPath)
	if err != nil {
		return fmt.Errorf("read norm stats: %w", err)
	}
	var ns NormStats
	if err := json.Unmarshal(raw, &ns); err != nil {
		return fmt.Errorf("parse norm stats: %w", err)
	}

	global = &Predictor{
		session:    session,
		population: ns,
		userNorms:  make(map[string]SubjectNorm),
	}
	return nil
}

// IsReady reports whether the stress predictor was successfully initialized.
func IsReady() bool { return global != nil }

// Close releases ONNX session resources.
func Close() {
	if global != nil && global.session != nil {
		global.session.Destroy()
	}
}

// SetUserNorm stores calibration data for a user in memory.
// Data is lost on restart; for persistence, callers should also save to DB.
func SetUserNorm(userID string, norm SubjectNorm) {
	if global == nil {
		return
	}
	global.mu.Lock()
	defer global.mu.Unlock()
	global.userNorms[userID] = norm
}

// GetUserNorm returns the calibrated norm for userID, or the population fallback.
func GetUserNorm(userID string) (SubjectNorm, bool) {
	if global == nil {
		return nil, false
	}
	global.mu.RLock()
	defer global.mu.RUnlock()
	if n, ok := global.userNorms[userID]; ok {
		return n, true
	}
	return global.population["_population"], false
}

// Predict classifies a 60-second window as Baseline or Stress.
func Predict(features RawFeatures, userNorm SubjectNorm) (PredictResult, error) {
	if global == nil {
		return PredictResult{}, fmt.Errorf("stress model not initialised — call Init first")
	}

	raw := map[string]float64{
		"hr_mean":   features.HRMean,
		"hr_std":    features.HRStd,
		"rmssd":     features.RMSSD,
		"temp_mean": features.TempMean,
	}

	input := make([]float32, len(featureOrder))
	for i, feat := range featureOrder {
		ns, ok := userNorm[feat]
		if !ok {
			return PredictResult{}, fmt.Errorf("norm stats missing for feature: %s", feat)
		}
		input[i] = float32((raw[feat] - ns.Mean) / ns.Std)
	}

	inTensor, err := ort.NewTensor(ort.NewShape(1, int64(len(featureOrder))), input)
	if err != nil {
		return PredictResult{}, fmt.Errorf("create input tensor: %w", err)
	}
	defer inTensor.Destroy()

	labelTensor, err := ort.NewEmptyTensor[int64](ort.NewShape(1))
	if err != nil {
		return PredictResult{}, fmt.Errorf("create label tensor: %w", err)
	}
	defer labelTensor.Destroy()
	probaTensor, err := ort.NewEmptyTensor[float32](ort.NewShape(1, 2))
	if err != nil {
		return PredictResult{}, fmt.Errorf("create proba tensor: %w", err)
	}
	defer probaTensor.Destroy()

	if err := global.session.Run(
		[]ort.ArbitraryTensor{inTensor},
		[]ort.ArbitraryTensor{labelTensor, probaTensor},
	); err != nil {
		return PredictResult{}, fmt.Errorf("run inference: %w", err)
	}

	label := StressLabel(labelTensor.GetData()[0])
	proba := probaTensor.GetData()

	return PredictResult{
		Label:        label,
		LabelName:    label.String(),
		ProbBaseline: float64(proba[0]),
		ProbStress:   float64(proba[1]),
	}, nil
}

// ComputeUserNorm computes per-user norm stats from baseline RawFeatures windows.
func ComputeUserNorm(windows []RawFeatures) SubjectNorm {
	n := float64(len(windows))
	if n == 0 {
		return nil
	}
	sums := map[string]float64{}
	for _, w := range windows {
		sums["hr_mean"] += w.HRMean
		sums["hr_std"] += w.HRStd
		sums["rmssd"] += w.RMSSD
		sums["temp_mean"] += w.TempMean
	}
	means := map[string]float64{}
	for k, v := range sums {
		means[k] = v / n
	}
	vars := map[string]float64{}
	for _, w := range windows {
		vals := map[string]float64{
			"hr_mean": w.HRMean, "hr_std": w.HRStd,
			"rmssd": w.RMSSD, "temp_mean": w.TempMean,
		}
		for k, v := range vals {
			d := v - means[k]
			vars[k] += d * d
		}
	}
	norm := SubjectNorm{}
	for _, feat := range featureOrder {
		std := math.Sqrt(vars[feat]/n) + 1e-8
		norm[feat] = NormStat{Mean: means[feat], Std: std}
	}
	return norm
}

// HRRecordsToFeatures computes hr_mean and hr_std from a slice of HR bpm values.
func HRRecordsToFeatures(hrBPM []float64) (mean, std float64) {
	if len(hrBPM) == 0 {
		return 0, 0
	}
	sum := 0.0
	for _, v := range hrBPM {
		sum += v
	}
	mean = sum / float64(len(hrBPM))
	variance := 0.0
	for _, v := range hrBPM {
		d := v - mean
		variance += d * d
	}
	std = math.Sqrt(variance / float64(len(hrBPM)))
	return
}
