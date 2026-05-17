package metric_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"storage-service/internal/metric"
	"storage-service/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// allowThresholdGoroutine sets up the mock to tolerate the background
// checkThresholds goroutine that RecordMetric spawns. We return
// ErrMetricNotFound so the goroutine exits immediately without side-effects.
// Using Maybe() means AssertExpectations won't fail if the goroutine finishes
// before the mock is torn down.
func allowThresholdGoroutine(repo *mocks.MetricRepository, userID, metricType string) {
	repo.On("GetThresholdByMetricName", mock.Anything, userID, metricType).
		Return((*metric.UserThreshold)(nil), metric.ErrMetricNotFound).Maybe()
}

type metricTestEnv struct {
	repo    *mocks.MetricRepository
	notif   *mocks.NotificationService
	service metric.Service
}

func newMetricTestEnv() *metricTestEnv {
	repo := new(mocks.MetricRepository)
	notif := new(mocks.NotificationService)
	return &metricTestEnv{
		repo:    repo,
		notif:   notif,
		service: metric.NewMetricService(repo, notif),
	}
}

// fPtr returns a pointer to a float64 – helper for threshold tests.
func fPtr(v float64) *float64 { return &v }

// ─── RecordMetric ─────────────────────────────────────────────────────────────

func TestRecordMetric(t *testing.T) {
	t.Run("success: metric is stored and nil is returned", func(t *testing.T) {
		// The service stores the metric via the repository; threshold checks run
		// in the background goroutine and must not affect the return value.
		// We allow the goroutine's GetThresholdByMetricName call with Maybe() so
		// it can exit cleanly regardless of scheduling order.
		env := newMetricTestEnv()
		m := metric.HealthMetric{
			UserID:    "user-1",
			Type:      "heart_rate",
			Value:     75,
			Timestamp: time.Now(),
		}

		env.repo.On("Store", mock.Anything, &m).Return(nil).Once()
		allowThresholdGoroutine(env.repo, "user-1", "heart_rate")

		err := env.service.RecordMetric(context.Background(), m)

		assert.NoError(t, err)
		// Brief sleep lets the background goroutine finish before mock teardown.
		time.Sleep(50 * time.Millisecond)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: repository Store failure is returned immediately", func(t *testing.T) {
		// When Store fails the error must be returned without starting the
		// threshold goroutine (goroutine is never launched on error path).
		env := newMetricTestEnv()
		m := metric.HealthMetric{UserID: "user-1", Type: "heart_rate", Value: 75}
		dbErr := errors.New("write failed")

		env.repo.On("Store", mock.Anything, &m).Return(dbErr).Once()

		err := env.service.RecordMetric(context.Background(), m)

		assert.ErrorIs(t, err, dbErr)
		env.repo.AssertExpectations(t)
	})
}

// ─── GetChartData ─────────────────────────────────────────────────────────────

func TestGetChartData(t *testing.T) {
	t.Run("success: 24h range uses 15-minute bucket", func(t *testing.T) {
		// For a 24-hour window the service must choose a "15 minutes" bucket size.
		env := newMetricTestEnv()
		observerID, userID := "obs-1", "user-1"

		env.repo.On("CheckAccess", mock.Anything, observerID, userID, "heart_rate").Return(true, nil).Once()
		env.repo.On("GetAggregatedMetrics", mock.Anything, userID, "heart_rate", "15 minutes",
			mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
			Return([]metric.MetricDataPoint{{Value: 72}}, nil).Once()

		got, err := env.service.GetChartData(context.Background(), observerID, userID, "heart_rate", metric.RangeLast24Hours, nil, nil)

		assert.NoError(t, err)
		assert.Len(t, got, 1)
		env.repo.AssertExpectations(t)
	})

	t.Run("success: 7d range uses 1-day bucket", func(t *testing.T) {
		// For a 7-day window the service chooses "1 day" buckets.
		env := newMetricTestEnv()
		observerID, userID := "obs-1", "user-1"

		env.repo.On("CheckAccess", mock.Anything, observerID, userID, "steps_count").Return(true, nil).Once()
		env.repo.On("GetAggregatedMetrics", mock.Anything, userID, "steps_count", "1 day",
			mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
			Return([]metric.MetricDataPoint{}, nil).Once()

		_, err := env.service.GetChartData(context.Background(), observerID, userID, "steps_count", metric.RangeLast7Days, nil, nil)

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("success: 30d range uses 1-day bucket", func(t *testing.T) {
		// A 30-day window still uses "1 day" buckets (2–30 days range).
		env := newMetricTestEnv()
		observerID, userID := "obs-1", "user-1"

		env.repo.On("CheckAccess", mock.Anything, observerID, userID, "heart_rate").Return(true, nil).Once()
		env.repo.On("GetAggregatedMetrics", mock.Anything, userID, "heart_rate", "1 day",
			mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
			Return([]metric.MetricDataPoint{}, nil).Once()

		_, err := env.service.GetChartData(context.Background(), observerID, userID, "heart_rate", metric.RangeLast30Days, nil, nil)

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("success: custom range with explicit start/end", func(t *testing.T) {
		// A 2-day custom window falls into the "1 hour" bucket category (1 < days ≤ 2).
		env := newMetricTestEnv()
		observerID, userID := "obs-1", "user-1"
		end := time.Now()
		start := end.Add(-48 * time.Hour) // exactly 2 days

		env.repo.On("CheckAccess", mock.Anything, observerID, userID, "heart_rate").Return(true, nil).Once()
		env.repo.On("GetAggregatedMetrics", mock.Anything, userID, "heart_rate", "1 hour", start, end).
			Return([]metric.MetricDataPoint{}, nil).Once()

		_, err := env.service.GetChartData(context.Background(), observerID, userID, "heart_rate", metric.RangeCustom, &start, &end)

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: permission denied when observer has no access", func(t *testing.T) {
		// CheckAccess returning false must cause ErrPermissionDenied before any
		// data query is executed.
		env := newMetricTestEnv()
		observerID, userID := "spy", "user-1"

		env.repo.On("CheckAccess", mock.Anything, observerID, userID, "heart_rate").Return(false, nil).Once()

		got, err := env.service.GetChartData(context.Background(), observerID, userID, "heart_rate", metric.RangeLast24Hours, nil, nil)

		assert.Nil(t, got)
		assert.ErrorIs(t, err, metric.ErrPermissionDenied)
		env.repo.AssertNotCalled(t, "GetAggregatedMetrics", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("error: custom range missing start/end returns ErrMissingTimes", func(t *testing.T) {
		// A custom range with no explicit times must be rejected.
		env := newMetricTestEnv()
		observerID, userID := "obs-1", "user-1"

		env.repo.On("CheckAccess", mock.Anything, observerID, userID, "heart_rate").Return(true, nil).Once()

		got, err := env.service.GetChartData(context.Background(), observerID, userID, "heart_rate", metric.RangeCustom, nil, nil)

		assert.Nil(t, got)
		assert.ErrorIs(t, err, metric.ErrMissingTimes)
	})

	t.Run("error: range exceeding 365 days returns ErrInvalidRange", func(t *testing.T) {
		// Custom windows longer than a year are rejected.
		env := newMetricTestEnv()
		observerID, userID := "obs-1", "user-1"
		end := time.Now()
		start := end.AddDate(-2, 0, 0) // 2 years

		env.repo.On("CheckAccess", mock.Anything, observerID, userID, "heart_rate").Return(true, nil).Once()

		got, err := env.service.GetChartData(context.Background(), observerID, userID, "heart_rate", metric.RangeCustom, &start, &end)

		assert.Nil(t, got)
		assert.ErrorIs(t, err, metric.ErrInvalidRange)
	})

	t.Run("error: CheckAccess repository failure propagates", func(t *testing.T) {
		env := newMetricTestEnv()
		observerID, userID := "obs-1", "user-1"
		dbErr := errors.New("connection refused")

		env.repo.On("CheckAccess", mock.Anything, observerID, userID, "heart_rate").Return(false, dbErr).Once()

		_, err := env.service.GetChartData(context.Background(), observerID, userID, "heart_rate", metric.RangeLast24Hours, nil, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "permission check failed")
	})
}

// ─── GetThresholds ────────────────────────────────────────────────────────────

func TestGetThresholds(t *testing.T) {
	t.Run("success: returns all thresholds for user", func(t *testing.T) {
		// Service must delegate directly to the repository.
		env := newMetricTestEnv()
		userID := "user-1"
		want := []metric.UserThreshold{
			{UserID: userID, MetricName: "heart_rate", MinValue: fPtr(50), MaxValue: fPtr(120), IsEnabled: true},
		}

		env.repo.On("GetUserThresholds", mock.Anything, userID).Return(want, nil).Once()

		got, err := env.service.GetThresholds(context.Background(), userID)

		assert.NoError(t, err)
		assert.Equal(t, want, got)
		env.repo.AssertExpectations(t)
	})

	t.Run("success: empty slice when no thresholds configured", func(t *testing.T) {
		// No configured thresholds is a valid state, not an error.
		env := newMetricTestEnv()

		env.repo.On("GetUserThresholds", mock.Anything, "new-user").Return([]metric.UserThreshold{}, nil).Once()

		got, err := env.service.GetThresholds(context.Background(), "new-user")

		assert.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("error: repository failure propagates", func(t *testing.T) {
		env := newMetricTestEnv()
		dbErr := errors.New("query timeout")

		env.repo.On("GetUserThresholds", mock.Anything, "user-1").Return(nil, dbErr).Once()

		_, err := env.service.GetThresholds(context.Background(), "user-1")

		assert.ErrorIs(t, err, dbErr)
		env.repo.AssertExpectations(t)
	})
}

// ─── SetUserThreshold ─────────────────────────────────────────────────────────

func TestSetUserThreshold(t *testing.T) {
	t.Run("success: upserts threshold via repository", func(t *testing.T) {
		// Service must convert the value-type threshold to a pointer and call UpsertThreshold.
		env := newMetricTestEnv()
		threshold := metric.UserThreshold{
			UserID:     "user-1",
			MetricName: "heart_rate",
			MinValue:   fPtr(50),
			MaxValue:   fPtr(120),
			IsEnabled:  true,
		}

		env.repo.On("UpsertThreshold", mock.Anything, &threshold).Return(nil).Once()

		err := env.service.SetUserThreshold(context.Background(), threshold)

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("success: threshold with only MaxValue (no min)", func(t *testing.T) {
		// Partial thresholds with only an upper bound are valid.
		env := newMetricTestEnv()
		threshold := metric.UserThreshold{
			UserID:     "user-1",
			MetricName: "steps_count",
			MaxValue:   fPtr(20000),
			IsEnabled:  true,
		}

		env.repo.On("UpsertThreshold", mock.Anything, &threshold).Return(nil).Once()

		err := env.service.SetUserThreshold(context.Background(), threshold)

		assert.NoError(t, err)
		env.repo.AssertExpectations(t)
	})

	t.Run("error: repository failure propagates", func(t *testing.T) {
		env := newMetricTestEnv()
		threshold := metric.UserThreshold{UserID: "user-1", MetricName: "heart_rate"}
		dbErr := errors.New("constraint violation")

		env.repo.On("UpsertThreshold", mock.Anything, &threshold).Return(dbErr).Once()

		err := env.service.SetUserThreshold(context.Background(), threshold)

		assert.ErrorIs(t, err, dbErr)
		env.repo.AssertExpectations(t)
	})
}
