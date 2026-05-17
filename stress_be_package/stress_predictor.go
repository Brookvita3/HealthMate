package stress

import (
	"encoding/json"
	"fmt"
	"math"
	"os"

	ort "github.com/yalue/onnxruntime_go"
)

// ─── Constants ───────────────────────────────────────────────────────────────

// Label mapping: 1 = Baseline, 2 = Stress
const (
	LabelBaseline = 1
	LabelStress   = 2
)

var featureOrder = []string{"hr_mean", "hr_std", "rmssd", "temp_mean"}

// ─── Types ───────────────────────────────────────────────────────────────────

// StressLabel is the classification output.
type StressLabel int

func (s StressLabel) String() string {
	if s == LabelStress {
		return "Stress"
	}
	return "Baseline"
}

// NormStat holds per-feature mean and std for one subject.
type NormStat struct {
	Mean float64 `json:"mean"`
	Std  float64 `json:"std"`
}

// SubjectNorm maps feature name → NormStat.
type SubjectNorm map[string]NormStat

// NormStats maps subject ID (or "_population") → SubjectNorm.
type NormStats map[string]SubjectNorm

// PredictResult is the output of Predict.
type PredictResult struct {
	Label       StressLabel
	LabelName   string
	ProbBaseline float64
	ProbStress   float64
}

// RawFeatures are the 4 values computed from Health Connect data (before normalization).
type RawFeatures struct {
	HRMean   float64 // mean of HR records (bpm) in the 60-second window
	HRStd    float64 // std  of HR records (bpm)
	RMSSD    float64 // HeartRateVariabilityRmssdRecord.heartRateVariabilityMillis (ms)
	TempMean float64 // SkinTemperatureRecord.temperature.inCelsius (°C)
}

// ─── Predictor ───────────────────────────────────────────────────────────────

// StressPredictor wraps the ONNX model and norm stats.
type StressPredictor struct {
	session   *ort.DynamicAdvancedSession
	normStats NormStats
}

// NewStressPredictor loads the ONNX model and norm-stats JSON.
//
//	onnxPath      — path to wesad_stress_hgbc.onnx
//	normStatsPath — path to wesad_norm_stats.json
func NewStressPredictor(onnxPath, normStatsPath string) (*StressPredictor, error) {
	// Init onnxruntime shared library (call once per process)
	if err := ort.InitializeEnvironment(); err != nil {
		return nil, fmt.Errorf("ort init: %w", err)
	}

	session, err := ort.NewDynamicAdvancedSession(
		onnxPath,
		[]string{"float_input"},
		[]string{"label", "probabilities"},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	raw, err := os.ReadFile(normStatsPath)
	if err != nil {
		return nil, fmt.Errorf("read norm stats: %w", err)
	}
	var ns NormStats
	if err := json.Unmarshal(raw, &ns); err != nil {
		return nil, fmt.Errorf("parse norm stats: %w", err)
	}

	return &StressPredictor{session: session, normStats: ns}, nil
}

// Close releases ONNX session resources.
func (p *StressPredictor) Close() {
	if p.session != nil {
		p.session.Destroy()
	}
}

// GetNorm returns the SubjectNorm for a subject ID.
// Falls back to "_population" if the subject is not found (new user before calibration).
func (p *StressPredictor) GetNorm(subjectID string) SubjectNorm {
	if n, ok := p.normStats[subjectID]; ok {
		return n
	}
	return p.normStats["_population"]
}

// ComputeUserNorm computes per-user norm stats from a slice of baseline RawFeatures.
// Call this during the ~5-minute onboarding baseline collection.
func ComputeUserNorm(baselineWindows []RawFeatures) SubjectNorm {
	n := float64(len(baselineWindows))
	if n == 0 {
		return nil
	}
	sums := map[string]float64{}
	for _, w := range baselineWindows {
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
	for _, w := range baselineWindows {
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

// Predict classifies a 60-second window as Baseline or Stress.
//
//	features  — raw values from Health Connect (not yet normalized)
//	userNorm  — from GetNorm(subjectID) or ComputeUserNorm(baselineWindows)
func (p *StressPredictor) Predict(features RawFeatures, userNorm SubjectNorm) (PredictResult, error) {
	raw := map[string]float64{
		"hr_mean":   features.HRMean,
		"hr_std":    features.HRStd,
		"rmssd":     features.RMSSD,
		"temp_mean": features.TempMean,
	}

	// Per-user z-score normalization
	input := make([]float32, len(featureOrder))
	for i, feat := range featureOrder {
		ns, ok := userNorm[feat]
		if !ok {
			return PredictResult{}, fmt.Errorf("norm stats missing for feature: %s", feat)
		}
		input[i] = float32((raw[feat] - ns.Mean) / ns.Std)
	}

	// Build input tensor [1, 4]
	inTensor, err := ort.NewTensor(ort.NewShape(1, int64(len(featureOrder))), input)
	if err != nil {
		return PredictResult{}, fmt.Errorf("create input tensor: %w", err)
	}
	defer inTensor.Destroy()

	labelTensor := ort.NewEmptyTensor[int64](ort.NewShape(1))
	defer labelTensor.Destroy()
	probaTensor := ort.NewEmptyTensor[float32](ort.NewShape(1, 2))
	defer probaTensor.Destroy()

	err = p.session.Run(
		[]ort.ArbitraryTensor{inTensor},
		[]ort.ArbitraryTensor{labelTensor, probaTensor},
	)
	if err != nil {
		return PredictResult{}, fmt.Errorf("run inference: %w", err)
	}

	label := StressLabel(labelTensor.GetData()[0])
	proba := probaTensor.GetData() // [p_baseline, p_stress]

	return PredictResult{
		Label:        label,
		LabelName:    label.String(),
		ProbBaseline: float64(proba[0]),
		ProbStress:   float64(proba[1]),
	}, nil
}

// ─── Health Connect helpers ───────────────────────────────────────────────────

// HRRecordsToFeatures computes hr_mean and hr_std from a slice of HR bpm values.
// hrBPM should contain all HeartRateRecord samples in a 60-second window.
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
