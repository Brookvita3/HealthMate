-- Set default thresholds for heart_rate
UPDATE "metric_types" SET "default_min_value" = 50.0, "default_max_value" = 120.0 WHERE "name" = 'heart_rate';
