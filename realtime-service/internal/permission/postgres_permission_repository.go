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
	// Đây là câu SQL mấu chốt, join 3 bảng
	// 1. Tìm các nhóm chung mà cả 2 user đều là thành viên (status = 'accepted')
	// 2. Trong các nhóm chung đó, kiểm tra xem 'targetUserID'
	//    có cho phép chia sẻ 'metricType' hay không.
	sql := `
		SELECT EXISTS (
			SELECT 1
			FROM group_members gm_observer
			JOIN group_members gm_target ON gm_observer.group_id = gm_target.group_id
			JOIN sharing_permissions sp ON sp.group_id = gm_target.group_id AND sp.user_id = gm_target.user_id
			WHERE
				gm_observer.user_id = $1 AND gm_observer.status = 'accepted'
				AND gm_target.user_id = $2 AND gm_target.status = 'accepted'
				AND sp.metric_type = $3
		);
	`
	var hasAccess bool
	err := r.pool.QueryRow(ctx, sql, observerUserID, targetUserID, metricType).Scan(&hasAccess)
	if err != nil {
		return false, fmt.Errorf("error checking access: %w", err)
	}

	return hasAccess, nil
}
