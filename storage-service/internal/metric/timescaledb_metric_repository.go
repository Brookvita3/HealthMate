package metric

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) MetricRepository {
	return &pgRepository{
		pool: pool,
	}
}

// Config represents the dynamic configuration of a metric type
type MetricConfig struct {
	BaseTable       string   `json:"base_table"`
	AllowedAggFuncs []string `json:"allowed_agg_funcs"`
}

func (r *pgRepository) getMetricConfig(ctx context.Context, metricType string) (*MetricConfig, error) {
	query := `SELECT base_table, allowed_agg_funcs FROM metric_types WHERE name = $1`
	var baseTable string
	var allowedAggFuncs []byte

	err := r.pool.QueryRow(ctx, query, metricType).Scan(&baseTable, &allowedAggFuncs)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUnknownMetricType
		}
		return nil, fmt.Errorf("database error in getMetricConfig: %w", err)
	}

	var parsedFuncs []string
	if err := json.Unmarshal(allowedAggFuncs, &parsedFuncs); err != nil {
		return nil, fmt.Errorf("failed to parse allowed_agg_funcs for %s: %w", metricType, err)
	}

	config := &MetricConfig{
		BaseTable:       baseTable,
		AllowedAggFuncs: parsedFuncs,
	}

	return config, nil
}

func (r *pgRepository) Store(ctx context.Context, metric *HealthMetric) error {
	config, err := r.getMetricConfig(ctx, metric.Type)
	if err != nil {
		return err
	}

	sql := fmt.Sprintf(`INSERT INTO %s (time, user_id, value) VALUES ($1, $2, $3)`, config.BaseTable)
	_, err = r.pool.Exec(ctx, sql, metric.Timestamp, metric.UserID, metric.Value)
	if err != nil {
		return fmt.Errorf("failed to store metric: %w", err)
	}
	return nil
}

func (r *pgRepository) GetAggregatedMetrics(ctx context.Context, userID string, metricType string, bucketSize string, startTime, endTime time.Time) ([]MetricDataPoint, error) {
	config, err := r.getMetricConfig(ctx, metricType)
	if err != nil {
		return nil, err
	}

	// Default to the first allowed agg func if available, else AVG
	aggFunc := "AVG"
	if len(config.AllowedAggFuncs) > 0 {
		aggFunc = config.AllowedAggFuncs[0]
	}

	query := fmt.Sprintf(`
		SELECT time_bucket('%s', time) AS bucket,
		       %s(value) as val
		FROM %s
		WHERE user_id = $1 AND time BETWEEN $2 AND $3
		GROUP BY bucket
		ORDER BY bucket ASC`, bucketSize, aggFunc, config.BaseTable)

	rows, err := r.pool.Query(ctx, query, userID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query aggregated metrics: %w", err)
	}
	defer rows.Close()

	var results []MetricDataPoint
	for rows.Next() {
		var dp MetricDataPoint
		if err := rows.Scan(&dp.Timestamp, &dp.Value); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		results = append(results, dp)
	}

	return results, nil
}
func (r *pgRepository) GetUserThresholds(ctx context.Context, userID string) ([]UserThreshold, error) {
	query := `
		SELECT 
			$1 as user_id,
			mt.id as metric_id, 
			mt.name as metric_name,
			COALESCE(ut.min_value, mt.default_min_value) as min_value,
			COALESCE(ut.max_value, mt.default_max_value) as max_value,
			COALESCE(ut.is_enabled, TRUE) as is_enabled,
			COALESCE(ut.updated_at, mt.created_at) as updated_at
		FROM metric_types mt
		LEFT JOIN user_health_thresholds ut ON mt.id = ut.metric_id AND ut.user_id = $1
		WHERE mt.name IN ('spo2', 'blood_pressure') OR ut.user_id IS NOT NULL`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user thresholds: %w", err)
	}
	defer rows.Close()

	var thresholds []UserThreshold
	for rows.Next() {
		var t UserThreshold
		if err := rows.Scan(&t.UserID, &t.MetricID, &t.MetricName, &t.MinValue, &t.MaxValue, &t.IsEnabled, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan threshold row: %w", err)
		}
		thresholds = append(thresholds, t)
	}
	return thresholds, nil
}

func (r *pgRepository) GetThresholdByMetricName(ctx context.Context, userID string, metricName string) (*UserThreshold, error) {
	query := `
		SELECT 
			$1 as user_id,
			mt.id as metric_id, 
			mt.name as metric_name,
			COALESCE(ut.min_value, mt.default_min_value) as min_value,
			COALESCE(ut.max_value, mt.default_max_value) as max_value,
			COALESCE(ut.is_enabled, TRUE) as is_enabled,
			COALESCE(ut.updated_at, mt.created_at) as updated_at
		FROM metric_types mt
		LEFT JOIN user_health_thresholds ut ON mt.id = ut.metric_id AND ut.user_id = $1
		WHERE mt.name = $2`

	var t UserThreshold
	err := r.pool.QueryRow(ctx, query, userID, metricName).Scan(
		&t.UserID, &t.MetricID, &t.MetricName, &t.MinValue, &t.MaxValue, &t.IsEnabled, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w for metric %s", ErrMetricNotFound, metricName)
		}
		return nil, fmt.Errorf("database error in GetThresholdByMetricName: %w", err)
	}
	return &t, nil
}

func (r *pgRepository) UpsertThreshold(ctx context.Context, t *UserThreshold) error {
	query := `
		INSERT INTO user_health_thresholds (user_id, metric_id, min_value, max_value, is_enabled, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (user_id, metric_id) DO UPDATE SET
			min_value = EXCLUDED.min_value,
			max_value = EXCLUDED.max_value,
			is_enabled = EXCLUDED.is_enabled,
			updated_at = NOW()`

	_, err := r.pool.Exec(ctx, query, t.UserID, t.MetricID, t.MinValue, t.MaxValue, t.IsEnabled)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return ErrUnknownMetricType
		}
		return fmt.Errorf("failed to upsert user threshold: %w", err)
	}
	return nil
}
