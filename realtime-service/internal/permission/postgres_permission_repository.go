package permission

import (
	"context"
	"log"

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

	// Strict Base + Filter Intersection Logic
	sql := `
	SELECT EXISTS (
		-- Base Rule Check (Must be enabled in at least one shared group)
		SELECT 1 FROM sharing_permissions base
		WHERE base.user_id = $2::uuid AND base.metric_type = $3
		  AND base.shared_with_user_id IS NULL
		  AND EXISTS (
			  -- Shared Group Check: Both must be in the group
			  SELECT 1 FROM group_members gm_observer
			  JOIN group_members gm_target ON gm_observer.group_id = gm_target.group_id
			  WHERE gm_observer.group_id = base.group_id
			    AND gm_observer.user_id = $1::uuid
			    AND gm_target.user_id = $2::uuid
			    AND gm_observer.status = 'accepted'
			    AND gm_target.status = 'accepted'
		  )
		  AND (
			  -- Specific Filter Check: Grant if specific match exists
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
	);`

	var hasAccess bool
	err := r.pool.QueryRow(ctx, sql, observerUserID, targetUserID, metricType).Scan(&hasAccess)
	if err != nil {
		return false, ErrPermissionCheckFailed
	}

	return hasAccess, nil
}

func (r *pgRepository) GetMetricWatchers(ctx context.Context, targetUserID, metricType string) (map[string][]string, error) {
	// Strict Base + Filter Intersection Logic for all group members
	// Includes check that the owner is still an accepted member of the group
	sql := `
	SELECT DISTINCT gm_watcher.user_id::text, base.group_id::text, g.name as group_name
	FROM sharing_permissions base
	JOIN group_members gm_watcher ON base.group_id = gm_watcher.group_id
	JOIN groups g ON base.group_id = g.id
	WHERE base.user_id = $1::uuid AND base.metric_type = $2
	  AND base.shared_with_user_id IS NULL -- Metric must be in Group Base
	  AND gm_watcher.status = 'accepted'
	  AND gm_watcher.user_id != $1::uuid -- Don't return the owner
	  AND (
		  -- Check Case 1: Specific Grant exists
		  EXISTS (
			  SELECT 1 FROM sharing_permissions spec
			  WHERE spec.group_id = base.group_id AND spec.user_id = base.user_id 
			    AND spec.metric_type = base.metric_type AND spec.shared_with_user_id = gm_watcher.user_id
		  )
		  OR
		  -- Check Case 2: No specific rules exist for this watcher (Fallback to Global)
		  NOT EXISTS (
			  SELECT 1 FROM sharing_permissions filter
			  WHERE filter.group_id = base.group_id AND filter.user_id = base.user_id 
			    AND filter.shared_with_user_id = gm_watcher.user_id
		  )
	  )
	  -- Ensure the owner is still an accepted member of the group
	  AND EXISTS (
		  SELECT 1 FROM group_members gm_owner
		  WHERE gm_owner.group_id = base.group_id AND gm_owner.user_id = $1::uuid AND gm_owner.status = 'accepted'
	  );`

	rows, err := r.pool.Query(ctx, sql, targetUserID, metricType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var observers = make(map[string][]string)
	for rows.Next() {
		var uid string
		var groupID string
		var groupName string
		if err := rows.Scan(&uid, &groupID, &groupName); err != nil {
			return nil, err
		}
		log.Printf("[Permission] User %s is authorized for %s/%s via group '%s' (%s)", uid, targetUserID, metricType, groupName, groupID)
		observers[uid] = append(observers[uid], groupID)
	}
	return observers, nil
}
