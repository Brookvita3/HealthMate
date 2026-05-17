package stress

import (
	"math"
	"path/filepath"
	"runtime"
	"testing"
)

// newTestPredictor loads the predictor using the models/ directory next to this file.
// If the ONNX Runtime shared library is not available, the test is skipped.
func newTestPredictor(t *testing.T) *StressPredictor {
	t.Helper()
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	p, err := NewStressPredictor(
		filepath.Join(dir, "models", "wesad_stress_hgbc.onnx"),
		filepath.Join(dir, "models", "wesad_norm_stats.json"),
	)
	if err != nil {
		t.Skipf("cannot load predictor (onnxruntime shared library missing?): %v", err)
	}
	return p
}

// ---------------------------------------------------------------------------
// Predict — reference vectors verified against Python ONNX runtime
// ---------------------------------------------------------------------------

// TestPredict_Baseline: typical resting physiology with population norm stats.
//
// Raw input (Health Connect scale):
//
//	hr_mean=65 bpm, hr_std=3, rmssd=45 ms, temp_mean=33.5 °C
//
// After population z-score → normalized ≈ [-1.58, -2.76, -3.02, +0.69]
// Expected: Baseline (label=1), ProbBaseline > 0.99
func TestPredict_Baseline(t *testing.T) {
	p := newTestPredictor(t)
	defer p.Close()

	norm := p.GetNorm("_population")
	result, err := p.Predict(RawFeatures{
		HRMean:   65.0,
		HRStd:    3.0,
		RMSSD:    45.0,
		TempMean: 33.5,
	}, norm)
	if err != nil {
		t.Fatalf("Predict: %v", err)
	}
	if result.Label != LabelBaseline {
		t.Errorf("want Baseline(1), got %d (%s) — ProbStress=%.4f",
			result.Label, result.LabelName, result.ProbStress)
	}
	if result.ProbBaseline < 0.90 {
		t.Errorf("ProbBaseline too low: %.4f", result.ProbBaseline)
	}
}

// TestPredict_Stress: elevated HR, low RMSSD, slightly cool skin.
//
// Raw input:
//
//	hr_mean=92 bpm, hr_std=8.5, rmssd=18 ms, temp_mean=32.8 °C
//
// After population z-score → normalized ≈ [+1.51, -1.92, -3.34, -0.54]
// Expected: Stress (label=2), ProbStress > 0.90
func TestPredict_Stress(t *testing.T) {
	p := newTestPredictor(t)
	defer p.Close()

	norm := p.GetNorm("_population")
	result, err := p.Predict(RawFeatures{
		HRMean:   92.0,
		HRStd:    8.5,
		RMSSD:    18.0,
		TempMean: 32.8,
	}, norm)
	if err != nil {
		t.Fatalf("Predict: %v", err)
	}
	if result.Label != LabelStress {
		t.Errorf("want Stress(2), got %d (%s) — ProbBaseline=%.4f",
			result.Label, result.LabelName, result.ProbBaseline)
	}
	if result.ProbStress < 0.90 {
		t.Errorf("ProbStress too low: %.4f", result.ProbStress)
	}
}

// TestPredict_ProbSum: probabilities must sum to 1.
func TestPredict_ProbSum(t *testing.T) {
	p := newTestPredictor(t)
	defer p.Close()

	norm := p.GetNorm("_population")
	result, err := p.Predict(RawFeatures{HRMean: 75, HRStd: 5, RMSSD: 35, TempMean: 33.2}, norm)
	if err != nil {
		t.Fatalf("Predict: %v", err)
	}
	sum := result.ProbBaseline + result.ProbStress
	if math.Abs(sum-1.0) > 1e-4 {
		t.Errorf("probabilities do not sum to 1: %.6f + %.6f = %.6f",
			result.ProbBaseline, result.ProbStress, sum)
	}
	if result.LabelName != "Baseline" && result.LabelName != "Stress" {
		t.Errorf("unexpected label name: %q", result.LabelName)
	}
}

// ---------------------------------------------------------------------------
// GetNorm — population fallback
// ---------------------------------------------------------------------------

func TestGetNorm_PopulationFallback(t *testing.T) {
	p := newTestPredictor(t)
	defer p.Close()

	norm := p.GetNorm("__nonexistent_user__")
	if norm == nil {
		t.Fatal("expected non-nil norm from population fallback")
	}
	for _, feat := range []string{"hr_mean", "hr_std", "rmssd", "temp_mean"} {
		if _, ok := norm[feat]; !ok {
			t.Errorf("population norm missing feature %q", feat)
		}
		if norm[feat].Std <= 0 {
			t.Errorf("population norm Std for %q must be positive, got %.6f", feat, norm[feat].Std)
		}
	}
}

// ---------------------------------------------------------------------------
// HRRecordsToFeatures
// ---------------------------------------------------------------------------

func TestHRRecordsToFeatures_Normal(t *testing.T) {
	hrBPM := []float64{70, 72, 68, 75, 73, 71, 69, 74}
	mean, std := HRRecordsToFeatures(hrBPM)

	wantMean := (70 + 72 + 68 + 75 + 73 + 71 + 69 + 74) / 8.0
	if math.Abs(mean-wantMean) > 0.01 {
		t.Errorf("mean: want %.4f, got %.4f", wantMean, mean)
	}
	if std <= 0 {
		t.Errorf("std should be positive, got %.4f", std)
	}
}

func TestHRRecordsToFeatures_SingleValue(t *testing.T) {
	mean, std := HRRecordsToFeatures([]float64{72.0})
	if mean != 72.0 {
		t.Errorf("mean: want 72.0, got %.4f", mean)
	}
	if std != 0 {
		t.Errorf("std with single value: want 0, got %.4f", std)
	}
}

func TestHRRecordsToFeatures_Empty(t *testing.T) {
	mean, std := HRRecordsToFeatures(nil)
	if mean != 0 || std != 0 {
		t.Errorf("empty slice: want (0,0), got (%.4f, %.4f)", mean, std)
	}
}

// ---------------------------------------------------------------------------
// ComputeUserNorm
// ---------------------------------------------------------------------------

func TestComputeUserNorm(t *testing.T) {
	windows := []RawFeatures{
		{HRMean: 65, HRStd: 2.0, RMSSD: 45, TempMean: 33.5},
		{HRMean: 67, HRStd: 3.0, RMSSD: 42, TempMean: 33.6},
		{HRMean: 64, HRStd: 2.5, RMSSD: 48, TempMean: 33.4},
	}
	norm := ComputeUserNorm(windows)

	if norm == nil {
		t.Fatal("ComputeUserNorm returned nil")
	}
	for _, feat := range []string{"hr_mean", "hr_std", "rmssd", "temp_mean"} {
		ns, ok := norm[feat]
		if !ok {
			t.Errorf("missing feature %q", feat)
			continue
		}
		if ns.Std <= 0 {
			t.Errorf("Std for %q must be positive, got %.6f", feat, ns.Std)
		}
	}

	// hr_mean mean = (65+67+64)/3 = 65.333...
	wantMean := (65.0 + 67.0 + 64.0) / 3.0
	if math.Abs(norm["hr_mean"].Mean-wantMean) > 0.01 {
		t.Errorf("hr_mean.Mean: want %.4f, got %.4f", wantMean, norm["hr_mean"].Mean)
	}
}

func TestComputeUserNorm_Empty(t *testing.T) {
	norm := ComputeUserNorm(nil)
	if norm != nil {
		t.Errorf("expected nil for empty windows, got %v", norm)
	}
}
