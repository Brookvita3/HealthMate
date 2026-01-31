package permission

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type pgRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{pool: pool}
}

func (r *pgRepository) CheckAccess(ctx context.Context, observerUserID, targetUserID string, metricType string) (bool, error) {

	if observerUserID == targetUserID {
		return true, nil
	}

	sql := `
	SELECT EXISTS (
		SELECT 1
		FROM group_members gm_observer
		JOIN group_members gm_target
		  ON gm_observer.group_id = gm_target.group_id
		JOIN groups g
		  ON g.id = gm_target.group_id
		JOIN sharing_permissions sp
		  ON sp.group_id = gm_target.group_id
		 AND sp.user_id  = gm_target.user_id
		 AND sp.metric_type = $3
		WHERE
			gm_observer.user_id = $1::uuid
			AND gm_target.user_id = $2::uuid
			AND gm_observer.status = 'accepted'
			AND gm_target.status = 'accepted'
	);
`

	var hasAccess bool
	err := r.pool.QueryRow(ctx, sql, observerUserID, targetUserID, metricType).Scan(&hasAccess)
	if err != nil {
		return false, fmt.Errorf("error checking access: %w", err)
	}

	return hasAccess, nil
}
