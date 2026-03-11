package metric

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
		return nil, ErrUnknownMetricType
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

	if len(results) == 0 {
		return nil, ErrMetricNotFound
	}

	return results, nil
}
