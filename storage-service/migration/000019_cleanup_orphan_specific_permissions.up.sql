-- Legacy POST /permissions could create per-viewer rows without access_control,
-- which broke CheckAccess vs visible-metrics. Remove orphan specific rows.
DELETE FROM sharing_permissions sp
WHERE sp.shared_with_user_id IS NOT NULL
  AND sp.metric_type != 'access_control'
  AND NOT EXISTS (
    SELECT 1 FROM sharing_permissions ac
    WHERE ac.group_id = sp.group_id
      AND ac.user_id = sp.user_id
      AND ac.shared_with_user_id = sp.shared_with_user_id
      AND ac.metric_type = 'access_control'
  );
