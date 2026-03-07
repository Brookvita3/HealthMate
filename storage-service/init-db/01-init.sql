-- Enable TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255),
    google_id VARCHAR(255) UNIQUE,
    password VARCHAR(255),
    picture VARCHAR(255),
    provider VARCHAR(50) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    sex VARCHAR(10),
    height DOUBLE PRECISION,
    weight DOUBLE PRECISION,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create groups table
CREATE TABLE IF NOT EXISTS "groups" (
  "id" uuid PRIMARY KEY,
  "name" varchar(255) NOT NULL,
  "description" text,
  "owner_id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

-- Create group_members table
CREATE TABLE IF NOT EXISTS "group_members" (
  "group_id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "role" varchar(50) NOT NULL DEFAULT 'member',
  "status" varchar(50) NOT NULL DEFAULT 'pending',
  "invited_by" uuid NOT NULL,
  "joined_at" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now()),
  PRIMARY KEY ("group_id", "user_id")
);

-- Create sharing_permissions table
CREATE TABLE IF NOT EXISTS "sharing_permissions" (
  "group_id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "metric_type" varchar(50) NOT NULL,
  PRIMARY KEY ("group_id", "user_id", "metric_type")
);

-- Add foreign keys
ALTER TABLE "groups" ADD CONSTRAINT groups_owner_id_fkey FOREIGN KEY ("owner_id") REFERENCES "users" ("id");
ALTER TABLE "group_members" ADD CONSTRAINT group_members_invited_by_fkey FOREIGN KEY ("invited_by") REFERENCES "users" ("id");
ALTER TABLE "group_members" ADD CONSTRAINT group_members_group_id_fkey FOREIGN KEY ("group_id") REFERENCES "groups" ("id");
ALTER TABLE "group_members" ADD CONSTRAINT group_members_user_id_fkey FOREIGN KEY ("user_id") REFERENCES "users" ("id");
ALTER TABLE "sharing_permissions" ADD CONSTRAINT sharing_permissions_fkey FOREIGN KEY ("group_id", "user_id") REFERENCES "group_members" ("group_id", "user_id");

-- Create metric tables
CREATE TABLE IF NOT EXISTS "heart_rates" (
  "time" TIMESTAMPTZ NOT NULL,
  "user_id" UUID NOT NULL,
  "value" DOUBLE PRECISION NOT NULL,
  PRIMARY KEY ("time", "user_id")
);

CREATE TABLE IF NOT EXISTS "steps_counts" (
  "time" TIMESTAMPTZ NOT NULL,
  "user_id" UUID NOT NULL,
  "value" INTEGER NOT NULL,
  PRIMARY KEY ("time", "user_id")
);

CREATE TABLE IF NOT EXISTS "calories_burnt" (
  "time" TIMESTAMPTZ NOT NULL,
  "user_id" UUID NOT NULL,
  "value" DOUBLE PRECISION NOT NULL,
  PRIMARY KEY ("time", "user_id")
);

-- Convert to hypertables
SELECT create_hypertable('heart_rates', 'time', if_not_exists => TRUE, chunk_time_interval => INTERVAL '1 day');
SELECT create_hypertable('steps_counts', 'time', if_not_exists => TRUE, chunk_time_interval => INTERVAL '1 day');
SELECT create_hypertable('calories_burnt', 'time', if_not_exists => TRUE, chunk_time_interval => INTERVAL '1 day');

-- Add indexes for faster queries (user_id + time range queries)
CREATE INDEX IF NOT EXISTS idx_heart_rates_user_time ON heart_rates (user_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_steps_counts_user_time ON steps_counts (user_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_calories_burnt_user_time ON calories_burnt (user_id, time DESC);

-- Add foreign keys for metric tables
ALTER TABLE "heart_rates" ADD CONSTRAINT heart_rates_user_id_fkey FOREIGN KEY ("user_id") REFERENCES "users" ("id");
ALTER TABLE "steps_counts" ADD CONSTRAINT steps_counts_user_id_fkey FOREIGN KEY ("user_id") REFERENCES "users" ("id");
ALTER TABLE "calories_burnt" ADD CONSTRAINT calories_burnt_user_id_fkey FOREIGN KEY ("user_id") REFERENCES "users" ("id");

-- Create continuous aggregates for hourly aggregations (auto-refresh every hour)
CREATE MATERIALIZED VIEW IF NOT EXISTS heart_rates_hourly
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 hour', time) AS bucket,
    user_id,
    AVG(value) AS avg_value,
    MIN(value) AS min_value,
    MAX(value) AS max_value,
    STDDEV(value) AS stddev_value,
    COUNT(*) AS count
FROM heart_rates
GROUP BY bucket, user_id;

CREATE MATERIALIZED VIEW IF NOT EXISTS steps_counts_hourly
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 hour', time) AS bucket,
    user_id,
    AVG(value) AS avg_value,
    SUM(value) AS sum_value,
    MIN(value) AS min_value,
    MAX(value) AS max_value,
    COUNT(*) AS count
FROM steps_counts
GROUP BY bucket, user_id;

CREATE MATERIALIZED VIEW IF NOT EXISTS calories_burnt_hourly
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 hour', time) AS bucket,
    user_id,
    AVG(value) AS avg_value,
    SUM(value) AS sum_value,
    MIN(value) AS min_value,
    MAX(value) AS max_value,
    COUNT(*) AS count
FROM calories_burnt
GROUP BY bucket, user_id;

-- Create continuous aggregates for daily aggregations (auto-refresh every day)
CREATE MATERIALIZED VIEW IF NOT EXISTS heart_rates_daily
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 day', time) AS bucket,
    user_id,
    AVG(value) AS avg_value,
    MIN(value) AS min_value,
    MAX(value) AS max_value,
    STDDEV(value) AS stddev_value,
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY value) AS median_value,
    PERCENTILE_CONT(0.25) WITHIN GROUP (ORDER BY value) AS p25_value,
    PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY value) AS p75_value,
    COUNT(*) AS count
FROM heart_rates
GROUP BY bucket, user_id;

CREATE MATERIALIZED VIEW IF NOT EXISTS steps_counts_daily
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 day', time) AS bucket,
    user_id,
    AVG(value) AS avg_value,
    SUM(value) AS sum_value,
    MIN(value) AS min_value,
    MAX(value) AS max_value,
    STDDEV(value) AS stddev_value,
    COUNT(*) AS count
FROM steps_counts
GROUP BY bucket, user_id;

CREATE MATERIALIZED VIEW IF NOT EXISTS calories_burnt_daily
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 day', time) AS bucket,
    user_id,
    AVG(value) AS avg_value,
    SUM(value) AS sum_value,
    MIN(value) AS min_value,
    MAX(value) AS max_value,
    STDDEV(value) AS stddev_value,
    COUNT(*) AS count
FROM calories_burnt
GROUP BY bucket, user_id;

-- Add refresh policies for continuous aggregates
SELECT add_continuous_aggregate_policy('heart_rates_hourly',
    start_offset => INTERVAL '3 hours',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour',
    if_not_exists => TRUE);

SELECT add_continuous_aggregate_policy('steps_counts_hourly',
    start_offset => INTERVAL '3 hours',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour',
    if_not_exists => TRUE);

SELECT add_continuous_aggregate_policy('calories_burnt_hourly',
    start_offset => INTERVAL '3 hours',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour',
    if_not_exists => TRUE);

SELECT add_continuous_aggregate_policy('heart_rates_daily',
    start_offset => INTERVAL '3 days',
    end_offset => INTERVAL '1 day',
    schedule_interval => INTERVAL '1 day',
    if_not_exists => TRUE);

SELECT add_continuous_aggregate_policy('steps_counts_daily',
    start_offset => INTERVAL '3 days',
    end_offset => INTERVAL '1 day',
    schedule_interval => INTERVAL '1 day',
    if_not_exists => TRUE);

SELECT add_continuous_aggregate_policy('calories_burnt_daily',
    start_offset => INTERVAL '3 days',
    end_offset => INTERVAL '1 day',
    schedule_interval => INTERVAL '1 day',
    if_not_exists => TRUE);

-- Enable compression for old data (compress data older than 7 days)
ALTER TABLE heart_rates SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'user_id',
    timescaledb.compress_orderby = 'time DESC'
);

ALTER TABLE steps_counts SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'user_id',
    timescaledb.compress_orderby = 'time DESC'
);

ALTER TABLE calories_burnt SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'user_id',
    timescaledb.compress_orderby = 'time DESC'
);

-- Add compression policy (compress data older than 7 days)
SELECT add_compression_policy('heart_rates', INTERVAL '7 days', if_not_exists => TRUE);
SELECT add_compression_policy('steps_counts', INTERVAL '7 days', if_not_exists => TRUE);
SELECT add_compression_policy('calories_burnt', INTERVAL '7 days', if_not_exists => TRUE);

-- Create continuous aggregates for monthly aggregations (for long-term analysis)
CREATE MATERIALIZED VIEW IF NOT EXISTS heart_rates_monthly
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 month', time) AS bucket,
    user_id,
    AVG(value) AS avg_value,
    MIN(value) AS min_value,
    MAX(value) AS max_value,
    STDDEV(value) AS stddev_value,
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY value) AS median_value,
    PERCENTILE_CONT(0.25) WITHIN GROUP (ORDER BY value) AS p25_value,
    PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY value) AS p75_value,
    COUNT(*) AS count
FROM heart_rates
GROUP BY bucket, user_id;

CREATE MATERIALIZED VIEW IF NOT EXISTS steps_counts_monthly
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 month', time) AS bucket,
    user_id,
    AVG(value) AS avg_value,
    SUM(value) AS sum_value,
    MIN(value) AS min_value,
    MAX(value) AS max_value,
    STDDEV(value) AS stddev_value,
    COUNT(*) AS count
FROM steps_counts
GROUP BY bucket, user_id;

CREATE MATERIALIZED VIEW IF NOT EXISTS calories_burnt_monthly
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 month', time) AS bucket,
    user_id,
    AVG(value) AS avg_value,
    SUM(value) AS sum_value,
    MIN(value) AS min_value,
    MAX(value) AS max_value,
    STDDEV(value) AS stddev_value,
    COUNT(*) AS count
FROM calories_burnt
GROUP BY bucket, user_id;

-- Create continuous aggregates for yearly aggregations (for long-term trends)
CREATE MATERIALIZED VIEW IF NOT EXISTS heart_rates_yearly
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 year', time) AS bucket,
    user_id,
    AVG(value) AS avg_value,
    MIN(value) AS min_value,
    MAX(value) AS max_value,
    STDDEV(value) AS stddev_value,
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY value) AS median_value,
    PERCENTILE_CONT(0.25) WITHIN GROUP (ORDER BY value) AS p25_value,
    PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY value) AS p75_value,
    COUNT(*) AS count
FROM heart_rates
GROUP BY bucket, user_id;

CREATE MATERIALIZED VIEW IF NOT EXISTS steps_counts_yearly
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 year', time) AS bucket,
    user_id,
    AVG(value) AS avg_value,
    SUM(value) AS sum_value,
    MIN(value) AS min_value,
    MAX(value) AS max_value,
    STDDEV(value) AS stddev_value,
    COUNT(*) AS count
FROM steps_counts
GROUP BY bucket, user_id;

CREATE MATERIALIZED VIEW IF NOT EXISTS calories_burnt_yearly
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 year', time) AS bucket,
    user_id,
    AVG(value) AS avg_value,
    SUM(value) AS sum_value,
    MIN(value) AS min_value,
    MAX(value) AS max_value,
    STDDEV(value) AS stddev_value,
    COUNT(*) AS count
FROM calories_burnt
GROUP BY bucket, user_id;

-- Add refresh policies for monthly aggregates (refresh daily)
SELECT add_continuous_aggregate_policy('heart_rates_monthly',
    start_offset => INTERVAL '3 months',
    end_offset => INTERVAL '1 day',
    schedule_interval => INTERVAL '1 day',
    if_not_exists => TRUE);

SELECT add_continuous_aggregate_policy('steps_counts_monthly',
    start_offset => INTERVAL '3 months',
    end_offset => INTERVAL '1 day',
    schedule_interval => INTERVAL '1 day',
    if_not_exists => TRUE);

SELECT add_continuous_aggregate_policy('calories_burnt_monthly',
    start_offset => INTERVAL '3 months',
    end_offset => INTERVAL '1 day',
    schedule_interval => INTERVAL '1 day',
    if_not_exists => TRUE);

-- Add refresh policies for yearly aggregates (refresh weekly)
SELECT add_continuous_aggregate_policy('heart_rates_yearly',
    start_offset => INTERVAL '3 years',
    end_offset => INTERVAL '1 week',
    schedule_interval => INTERVAL '1 week',
    if_not_exists => TRUE);

SELECT add_continuous_aggregate_policy('steps_counts_yearly',
    start_offset => INTERVAL '3 years',
    end_offset => INTERVAL '1 week',
    schedule_interval => INTERVAL '1 week',
    if_not_exists => TRUE);

SELECT add_continuous_aggregate_policy('calories_burnt_yearly',
    start_offset => INTERVAL '3 years',
    end_offset => INTERVAL '1 week',
    schedule_interval => INTERVAL '1 week',
    if_not_exists => TRUE);

-- Optional: Add retention policy to automatically delete raw data older than X years
-- Uncomment and adjust interval as needed (e.g., keep 5 years of raw data)
-- SELECT add_retention_policy('heart_rates', INTERVAL '5 years', if_not_exists => TRUE);
-- SELECT add_retention_policy('steps_counts', INTERVAL '5 years', if_not_exists => TRUE);
-- SELECT add_retention_policy('calories_burnt', INTERVAL '5 years', if_not_exists => TRUE);

-- Note: Continuous aggregates will keep aggregated data even after raw data is deleted
-- This allows long-term analysis while managing storage costs

-- Insert sample user for testing
INSERT INTO users (id, email, name, provider, role, status) 
VALUES ('00000000-0000-0000-0000-000000000001', 'test@example.com', 'Test User', 'email', 'user', 'active')
ON CONFLICT (id) DO NOTHING;
