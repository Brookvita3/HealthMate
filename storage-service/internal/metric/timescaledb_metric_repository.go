package metric

import (
	"context"
	"fmt"

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

var metricTable = map[string]string{
	"heart_rate":      "heart_rates",
	"steps_count":     "steps_counts",
	"calories_burned": "calories_burned",
}

func (r *pgRepository) Store(ctx context.Context, metric *HealthMetric) error {
	table, ok := metricTable[metric.Type]
	if !ok {
		return fmt.Errorf("unknown metric type: %s", metric.Type)
	}

	sql := fmt.Sprintf(`INSERT INTO %s (time, user_id, value) VALUES ($1, $2, $3)`, table)
	_, err := r.pool.Exec(ctx, sql, metric.Timestamp, metric.UserID, metric.Value)
	if err != nil {
		return fmt.Errorf("failed to store metric: %w", err)
	}
	return nil
}
