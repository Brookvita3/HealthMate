package permission

import (
	"context"

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
		return false, ErrSelfSubscription
	}

	sql := `
	SELECT EXISTS (
		-- Rule 1: Must be enabled in Global Rule for everyone
		SELECT 1
		FROM sharing_permissions sp_global
		WHERE sp_global.group_id IN (
			SELECT gm_observer.group_id
			FROM group_members gm_observer
			JOIN group_members gm_target ON gm_observer.group_id = gm_target.group_id
			WHERE gm_observer.user_id = $1::uuid
			  AND gm_target.user_id = $2::uuid
			  AND gm_observer.status = 'accepted'
			  AND gm_target.status = 'accepted'
		)
		AND sp_global.user_id = $2::uuid
		AND sp_global.metric_type = $3
		AND sp_global.shared_with_user_id IS NULL
	) AND (
		-- Rule 2: Either no member rule exists for this observer, OR it is specifically enabled
		NOT EXISTS (
			SELECT 1 FROM sharing_permissions 
			WHERE user_id = $2::uuid 
			  AND shared_with_user_id = $1::uuid
		)
		OR EXISTS (
			SELECT 1 FROM sharing_permissions 
			WHERE user_id = $2::uuid 
			  AND shared_with_user_id = $1::uuid
			  AND metric_type = $3
		)
	);
`

	var hasAccess bool
	err := r.pool.QueryRow(ctx, sql, observerUserID, targetUserID, metricType).Scan(&hasAccess)
	if err != nil {
		return false, ErrPermissionCheckFailed
	}

	return hasAccess, nil
}
