package readiness

import (
	"fmt"
	"math"

	ort "github.com/yalue/onnxruntime_go"
)

// Input is the raw data from the request; preprocessing is applied before inference.
type Input struct {
	HeartRate      float64
	SleepDuration  float64
	StressLevel    string  // "None"|"Low"|"Moderate"|"High"|"Very High"
	BloodOxygen    float64 // SpO2 %
	Steps          float64 // steps/day (raw)
	CaloriesBurned float64 // kcal (raw)
}

var (
	session      *ort.AdvancedSession
	inputTensor  *ort.Tensor[float32]
	outputTensor *ort.Tensor[float32]
)

// Init loads the ONNX Runtime shared library and creates the inference session.
// Must be called once at application startup before any Predict call.
func Init(onnxLibPath, modelPath string) error {
	ort.SetSharedLibraryPath(onnxLibPath)
	if err := ort.InitializeEnvironment(); err != nil {
		return fmt.Errorf("onnxruntime init: %w", err)
	}

	var err error
	inputTensor, err = ort.NewTensor(ort.NewShape(1, 6), make([]float32, 6))
	if err != nil {
		return fmt.Errorf("create input tensor: %w", err)
	}

	outputTensor, err = ort.NewEmptyTensor[float32](ort.NewShape(1, 1))
	if err != nil {
		inputTensor.Destroy()
		return fmt.Errorf("create output tensor: %w", err)
	}

	session, err = ort.NewAdvancedSession(
		modelPath,
		[]string{"features"},
		[]string{"predictions"},
		[]ort.Value{inputTensor},
		[]ort.Value{outputTensor},
		nil,
	)
	if err != nil {
		inputTensor.Destroy()
		outputTensor.Destroy()
		return fmt.Errorf("create onnx session: %w", err)
	}
	return nil
}

var stressMap = map[string]float32{
	"none": 0, "None": 0,
	"low": 1, "Low": 1,
	"moderate": 2, "Moderate": 2,
	"high": 3, "High": 3,
	"very high": 4, "Very High": 4,
}

// Predict runs inference and returns a readiness score in [0, 100].
// Feature order must match training: Heart_Rate, Sleep_Duration, Stress_Level_Num,
// Blood_Oxygen, Steps_log1p, Calories_log1p.
func Predict(inp Input) (float64, error) {
	if session == nil {
		return 0, fmt.Errorf("readiness model not initialised — call Init first")
	}

	spo2 := float32(math.Min(math.Max(inp.BloodOxygen, 85), 100))

	stress, ok := stressMap[inp.StressLevel]
	if !ok {
		stress = 2 // default: Moderate
	}

	features := []float32{
		float32(inp.HeartRate),                                // [0] Heart_Rate
		float32(inp.SleepDuration),                           // [1] Sleep_Duration
		stress,                                                // [2] Stress_Level_Num
		spo2,                                                  // [3] Blood_Oxygen
		float32(math.Log1p(math.Max(inp.Steps, 0))),          // [4] Steps_log1p
		float32(math.Log1p(math.Max(inp.CaloriesBurned, 0))), // [5] Calories_log1p
	}

	copy(inputTensor.GetData(), features)

	if err := session.Run(); err != nil {
		return 0, fmt.Errorf("onnx inference: %w", err)
	}

	score := float64(outputTensor.GetData()[0])
	return math.Min(math.Max(score, 0), 100), nil // clip to [0, 100]
}
