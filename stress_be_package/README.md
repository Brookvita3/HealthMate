# stress_be_package — Stress Classification for Go Backend

Binary stress classifier (Baseline / Stress) trained on the WESAD dataset.
Uses only features available from the **Health Connect API** on Samsung Galaxy Watch 6.

Model: `HistGradientBoostingClassifier` (sklearn) → ONNX  
LOSO cross-validation: **93.8% accuracy ± 9.7%, F1-macro 92.8% ± 11.8%**

---

## Package contents

```
stress_be_package/
├── stress_predictor.go          # Go package (package stress)
├── stress_predictor_test.go     # Unit tests — run: go test ./...
├── models/
│   ├── wesad_stress_hgbc.onnx  # ONNX model (float_input [1,4] → label, probabilities)
│   └── wesad_norm_stats.json   # Per-subject z-score stats + _population fallback
└── README.md
```

---

## Dependencies

Add to your `go.mod`:

```go
require (
    github.com/yalue/onnxruntime_go v1.13.0
)
```

The package wraps `onnxruntime` via CGO. You must have the ONNX Runtime shared library
available at runtime:

- **Linux/Android**: `libonnxruntime.so`
- **Windows**: `onnxruntime.dll`

Download from: https://github.com/microsoft/onnxruntime/releases (v1.18+ recommended)

> **Note:** `ort.InitializeEnvironment()` is called inside `NewStressPredictor`. If your
> process already calls it elsewhere (e.g., for another ONNX model), call
> `ort.SetSharedLibraryPath(...)` before either `NewStressPredictor` call to avoid
> double-initialization.

---

## Quick start

```go
import stress "your_module/stress_be_package"

// 1. Initialize once at startup
predictor, err := stress.NewStressPredictor(
    "stress_be_package/models/wesad_stress_hgbc.onnx",
    "stress_be_package/models/wesad_norm_stats.json",
)
if err != nil {
    log.Fatal(err)
}
defer predictor.Close()

// 2. Get norm stats for a user (falls back to _population for new users)
userNorm := predictor.GetNorm(userID) // userID from your user table

// 3. For each 60-second Health Connect window, build RawFeatures and predict
hrBPM := []float64{72, 74, 73, 75, 71} // HeartRateRecord.bpm values in the window
hrMean, hrStd := stress.HRRecordsToFeatures(hrBPM)

features := stress.RawFeatures{
    HRMean:   hrMean,
    HRStd:    hrStd,
    RMSSD:    38.5,  // HeartRateVariabilityRmssdRecord.heartRateVariabilityMillis
    TempMean: 33.2,  // SkinTemperatureRecord.temperature.inCelsius
}

result, err := predictor.Predict(features, userNorm)
if err != nil {
    log.Fatal(err)
}

fmt.Println(result.LabelName)    // "Baseline" or "Stress"
fmt.Println(result.ProbStress)   // e.g. 0.82
```

---

## Health Connect API → RawFeatures mapping

| `RawFeatures` field | Health Connect record type | Field |
|---|---|---|
| `HRMean` | `HeartRateRecord` | mean of `.samples[].beatsPerMinute` in window |
| `HRStd` | `HeartRateRecord` | std of `.samples[].beatsPerMinute` in window |
| `RMSSD` | `HeartRateVariabilityRmssdRecord` | `.heartRateVariabilityMillis` |
| `TempMean` | `SkinTemperatureRecord` | mean of `.deltas[].temperature.inCelsius` + baseline |

Use `HRRecordsToFeatures(hrBPM []float64)` to compute `HRMean` and `HRStd` from the raw bpm slice.

Window size: **60 seconds**. Run prediction every 30–60 seconds.

---

## New user onboarding (calibration) — REQUIRED for RMSSD accuracy

**Calibration is strongly recommended.** The RMSSD values from Health Connect
(`HeartRateVariabilityRmssdRecord`, typically 15–80 ms) are on a different absolute scale
than the WESAD wrist BVP signal used during training. The `_population` fallback norm stats
encode the WESAD scale and will produce systematically negative RMSSD z-scores for
Health Connect users, which may reduce precision on RMSSD-driven predictions.

With a per-user calibration, the user's own resting Health Connect values define the
normalization baseline, making the z-scores scale-invariant. **Accuracy numbers above
assume per-user calibration is applied.**

```go
var baselineWindows []stress.RawFeatures

// Collect one RawFeatures per 60-second window during the baseline session (~5 min)
for _, window := range collectedBaselineData {
    hrMean, hrStd := stress.HRRecordsToFeatures(window.HRSamples)
    baselineWindows = append(baselineWindows, stress.RawFeatures{
        HRMean:   hrMean,
        HRStd:    hrStd,
        RMSSD:    window.RMSSD,
        TempMean: window.TempMean,
    })
}

// Compute and persist the user's norm stats
userNorm := stress.ComputeUserNorm(baselineWindows)
// Save userNorm to your database as JSON (it's a map[string]NormStat)

// On subsequent sessions, load from DB and use directly with predictor.Predict()
```

Before calibration is complete, `predictor.GetNorm(unknownUserID)` returns the population-level
fallback (`_population`) computed from all 15 WESAD subjects.

---

## Output format

```go
type PredictResult struct {
    Label        StressLabel // 1 = Baseline, 2 = Stress
    LabelName    string      // "Baseline" or "Stress"
    ProbBaseline float64     // probability of Baseline (0–1)
    ProbStress   float64     // probability of Stress   (0–1)
}
```

---

## Running tests

```bash
# From the BE repo root (where go.mod lives):
go test ./stress_be_package/...

# Tests requiring ONNX Runtime (stress_predictor_test.go) will be skipped
# automatically if libonnxruntime.so / onnxruntime.dll is not available.
```

---

## Model details

- **Training data**: WESAD dataset, 15 subjects, wrist sensor (BVP + TEMP)
- **Window**: 60 s, 50% overlap, 90% purity threshold
- **Features**: `hr_mean`, `hr_std`, `rmssd`, `temp_mean` (Health Connect compatible)
- **Normalization**: per-user z-score applied before inference
- **Classifier**: `HistGradientBoostingClassifier` (max_depth=4, max_iter=300, lr=0.08)
- **Evaluation**: Leave-One-Subject-Out CV → **93.8% ± 9.7% accuracy, F1-macro 92.8% ± 11.8%**
- **Classes**: 1 = Baseline, 2 = Stress (WESAD protocol labels)
- **Regenerate models**: run `wesad_stress_classification.ipynb` cells 1–14, then copy
  `models/wesad_stress_hgbc.onnx` and `models/wesad_norm_stats.json` from the Python repo.
  Cell 14 does this copy automatically.
