-- Revert default thresholds for heart_rate to NULL
UPDATE "metric_types" SET "default_min_value" = NULL, "default_max_value" = NULL WHERE "name" = 'heart_rate';
