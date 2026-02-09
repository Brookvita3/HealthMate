package permission

import (
	"auth-service/internal/domain"
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) PermissionRepository {
	return &postgresRepository{pool: pool}
}

func (r *postgresRepository) SetPermission(ctx context.Context, groupID, userID uuid.UUID, metricType string) error {
	query := `
		INSERT INTO sharing_permissions (group_id, user_id, metric_type)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING`
	_, err := r.pool.Exec(ctx, query, groupID, userID, metricType)
	return err
}

func (r *postgresRepository) RevokePermission(ctx context.Context, groupID, userID uuid.UUID, metricType string) error {
	query := `DELETE FROM sharing_permissions WHERE group_id = $1 AND user_id = $2 AND metric_type = $3`
	_, err := r.pool.Exec(ctx, query, groupID, userID, metricType)
	return err
}

func (r *postgresRepository) ListUserPermissionsInGroup(ctx context.Context, groupID, userID uuid.UUID) ([]domain.Permission, error) {
	query := `SELECT group_id, user_id, metric_type FROM sharing_permissions WHERE group_id = $1 AND user_id = $2`
	rows, err := r.pool.Query(ctx, query, groupID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []domain.Permission
	for rows.Next() {
		var p domain.Permission
		if err := rows.Scan(&p.GroupId, &p.UserId, &p.MetricType); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, nil
}
func (r *postgresRepository) RevokeAllPermissions(ctx context.Context, groupID, userID uuid.UUID) error {
	query := `DELETE FROM sharing_permissions WHERE group_id = $1 AND user_id = $2`
	_, err := r.pool.Exec(ctx, query, groupID, userID)
	return err
}
func (r *postgresRepository) IsMember(ctx context.Context, groupID, userID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2 AND status = 'accepted')`
	var exists bool
	err := r.pool.QueryRow(ctx, query, groupID, userID).Scan(&exists)
	return exists, err
}

func (r *postgresRepository) IsValidMetricType(ctx context.Context, metricType string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM metric_types WHERE name = $1)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, metricType).Scan(&exists)
	return exists, err
}

func (r *postgresRepository) ListMetricTypes(ctx context.Context) ([]domain.MetricType, error) {
	query := `SELECT id, name, description, created_at FROM metric_types`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var types []domain.MetricType
	for rows.Next() {
		var t domain.MetricType
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.CreatedAt); err != nil {
			return nil, err
		}
		types = append(types, t)
	}
	return types, nil
}
