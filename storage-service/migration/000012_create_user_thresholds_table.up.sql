-- Add default threshold columns to metric_types
ALTER TABLE "metric_types" ADD COLUMN "default_min_value" DOUBLE PRECISION;
ALTER TABLE "metric_types" ADD COLUMN "default_max_value" DOUBLE PRECISION;

-- Set default thresholds for spo2 and blood_pressure
UPDATE "metric_types" SET "default_min_value" = 95.0 WHERE "name" = 'spo2';
UPDATE "metric_types" SET "default_max_value" = 140.0 WHERE "name" = 'blood_pressure';

-- Create user_health_thresholds table
CREATE TABLE "user_health_thresholds" (
    "user_id" UUID NOT NULL,
    "metric_id" UUID NOT NULL,
    "min_value" DOUBLE PRECISION,
    "max_value" DOUBLE PRECISION,
    "is_enabled" BOOLEAN DEFAULT TRUE,
    "created_at" TIMESTAMPTZ DEFAULT NOW(),
    "updated_at" TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY ("user_id", "metric_id"),
    FOREIGN KEY ("user_id") REFERENCES "users" ("id"),
    FOREIGN KEY ("metric_id") REFERENCES "metric_types" ("id")
);
