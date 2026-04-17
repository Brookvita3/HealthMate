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
		LEFT JOIN user_health_thresholds ut ON mt.id = ut.metric_id AND ut.user_id = $1`

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

func (r *pgRepository) GetMetricWatchers(ctx context.Context, userID, metricType string) ([]Watcher, error) {
	// Simplified unified logic: Group Rule as Base + Per-user Filter
	query := `
		-- Strict Base + Filter Intersection Logic
		SELECT DISTINCT u.id, u.name as user_name, g.name as group_name
		FROM sharing_permissions base
		JOIN group_members gm_watcher ON base.group_id = gm_watcher.group_id
		JOIN users u ON gm_watcher.user_id = u.id
		JOIN groups g ON g.id = base.group_id
		WHERE base.user_id = $1 AND base.metric_type = $2
		  AND base.shared_with_user_id IS NULL -- Metric must be in Group Base
		  AND gm_watcher.status = 'accepted'
		  AND u.id != $1::uuid
		  AND (
			  -- Check Case 1: Specific Grant exists
			  EXISTS (
				  SELECT 1 FROM sharing_permissions spec
				  WHERE spec.group_id = base.group_id AND spec.user_id = base.user_id 
				    AND spec.metric_type = base.metric_type AND spec.shared_with_user_id = u.id
			  )
			  OR
			  -- Check Case 2: No specific rules exist for this watcher in this group (Fallback to Global)
			  NOT EXISTS (
				  SELECT 1 FROM sharing_permissions filter
				  WHERE filter.group_id = base.group_id AND filter.user_id = base.user_id 
				    AND filter.shared_with_user_id = u.id
			  )
		  )
		  -- Ensure the owner is still in the group
		  AND EXISTS (SELECT 1 FROM group_members WHERE group_id = base.group_id AND user_id = $1 AND status = 'accepted')`

	rows, err := r.pool.Query(ctx, query, userID, metricType)
	if err != nil {
		return nil, fmt.Errorf("failed to query metric watchers: %w", err)
	}
	defer rows.Close()

	var watchers []Watcher
	for rows.Next() {
		var w Watcher
		if err := rows.Scan(&w.UserID, &w.UserName, &w.GroupName); err != nil {
			return nil, fmt.Errorf("failed to scan watcher: %w", err)
		}
		watchers = append(watchers, w)
	}
	return watchers, nil
}

func (r *pgRepository) GetUserName(ctx context.Context, userID string) (string, error) {
	var name string
	err := r.pool.QueryRow(ctx, "SELECT name FROM users WHERE id = $1", userID).Scan(&name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "Unknown User", nil
		}
		return "", fmt.Errorf("failed to get user name: %w", err)
	}
	return name, nil
}

func (r *pgRepository) CheckAccess(ctx context.Context, observerID, targetID, metricType string) (bool, error) {
	if observerID == targetID {
		return true, nil
	}

	// Strict Base + Filter Intersection Logic
	query := `
	SELECT EXISTS (
		-- Base Rule Check (Must be enabled in at least one shared group)
		SELECT 1 FROM sharing_permissions base
		WHERE base.user_id = $2::uuid AND base.metric_type = $3
		  AND base.shared_with_user_id IS NULL
		  AND (
			  -- Case 1: Specific Filter Check: Grant if specific match exists
			  EXISTS (
				  SELECT 1 FROM sharing_permissions spec
				  WHERE spec.group_id = base.group_id AND spec.user_id = base.user_id 
				    AND spec.metric_type = base.metric_type AND spec.shared_with_user_id = $1::uuid
			  )
			  OR
			  -- Fallback to Global: Grant ONLY if NO specific rules exist for this observer in this group
			  NOT EXISTS (
				  SELECT 1 FROM sharing_permissions filter
				  WHERE filter.group_id = base.group_id AND filter.user_id = base.user_id 
				    AND filter.shared_with_user_id = $1::uuid
			  )
		  )
		  -- Shared Group Check
		  AND base.group_id IN (
			SELECT gm_observer.group_id
			FROM group_members gm_observer
			JOIN group_members gm_target ON gm_observer.group_id = gm_target.group_id
			WHERE gm_observer.user_id = $1::uuid
			  AND gm_target.user_id = $2::uuid
			  AND gm_observer.status = 'accepted'
			  AND gm_target.status = 'accepted'
		  )
	);`

	var hasAccess bool
	err := r.pool.QueryRow(ctx, query, observerID, targetID, metricType).Scan(&hasAccess)
	if err != nil {
		return false, fmt.Errorf("failed to check database access: %w", err)
	}
	return hasAccess, nil
}
