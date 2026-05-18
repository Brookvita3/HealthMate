CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    name TEXT,
    google_id TEXT,
    password TEXT,
    picture TEXT,
    provider TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- This migration updates the users table to align with the refactored user.User model.
-- It adds role and status columns, enforces constraints, and adjusts column types.

-- Add the new 'role' and 'status' columns with sane defaults and NOT NULL constraints.
ALTER TABLE users ADD COLUMN role VARCHAR(50) NOT NULL DEFAULT 'user';
ALTER TABLE users ADD COLUMN status VARCHAR(50) NOT NULL DEFAULT 'active';

-- Ensure the provider is always set.
ALTER TABLE users ALTER COLUMN provider SET NOT NULL;

-- Make the google_id unique, as it's a primary identifier for Google logins.
-- It can be NULL for users who register with email/password.
ALTER TABLE users ADD CONSTRAINT users_google_id_key UNIQUE (google_id);

-- Change TEXT to VARCHAR(255) for email and name for better indexing and data integrity.
ALTER TABLE users ALTER COLUMN email TYPE VARCHAR(255);
ALTER TABLE users ALTER COLUMN name TYPE VARCHAR(255);
CREATE TABLE "groups" (
  "id" uuid PRIMARY KEY,
  "name" varchar(255) NOT NULL,
  "description" text,
  "owner_id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "group_members" (
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

ALTER TABLE "groups" ADD FOREIGN KEY ("owner_id") REFERENCES "users" ("id");

ALTER TABLE "group_members" ADD FOREIGN KEY ("invited_by") REFERENCES "users" ("id");

ALTER TABLE "group_members" ADD FOREIGN KEY ("group_id") REFERENCES "groups" ("id");

ALTER TABLE "group_members" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

CREATE TABLE "sharing_permissions" (
  "group_id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "metric_type" varchar(50) NOT NULL,
  PRIMARY KEY ("group_id", "user_id", "metric_type")
);

ALTER TABLE "sharing_permissions" ADD FOREIGN KEY ("group_id", "user_id") REFERENCES "group_members" ("group_id", "user_id");
-- Táº¡o báº£ng heart_rate
CREATE TABLE "heart_rates" (
  "time" TIMESTAMPTZ NOT NULL,
  "user_id" UUID NOT NULL,
  "value" DOUBLE PRECISION NOT NULL,
  PRIMARY KEY ("time", "user_id")
);

-- Chuyá»ƒn thÃ nh hypertable
SELECT create_hypertable('heart_rates', 'time', if_not_exists => TRUE);


-- Táº¡o báº£ng steps_counts
CREATE TABLE "steps_counts" (
  "time" TIMESTAMPTZ NOT NULL,
  "user_id" UUID NOT NULL,
  "value" INTEGER NOT NULL,
  PRIMARY KEY ("time", "user_id")
);

SELECT create_hypertable('steps_counts', 'time', if_not_exists => TRUE);


-- Táº¡o báº£ng calories_burnt
CREATE TABLE "calories_burnt" (
  "time" TIMESTAMPTZ NOT NULL,
  "user_id" UUID NOT NULL,
  "value" DOUBLE PRECISION NOT NULL,
  PRIMARY KEY ("time", "user_id")
);

SELECT create_hypertable('calories_burnt', 'time', if_not_exists => TRUE);


-- FOREIGN KEYS
ALTER TABLE "heart_rates"
  ADD CONSTRAINT heart_rates_user_id_fkey FOREIGN KEY ("user_id") 
  REFERENCES "users" ("id");

ALTER TABLE "steps_counts"
  ADD CONSTRAINT steps_counts_user_id_fkey FOREIGN KEY ("user_id") 
  REFERENCES "users" ("id");

ALTER TABLE "calories_burnt"
  ADD CONSTRAINT calories_burnt_user_id_fkey FOREIGN KEY ("user_id") 
  REFERENCES "users" ("id");
CREATE TABLE "metric_types" (
  "id" uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  "name" varchar(100) UNIQUE NOT NULL,
  "description" text,
  "created_at" timestamp DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO "metric_types" ("name", "description") VALUES
('heart_rate', 'Heart rate monitor data'),
('steps_count', 'Steps taken count per day'),
('calories_burned', 'Estimated calories burned during activities');

-- Optionally add a foreign key to sharing_permissions
-- But first we might need to ensure all current values in sharing_permissions exist in metric_types
-- For now, let's just create the table and seed it.
-- ThÃªm cá»™t cáº¥u hÃ¬nh cho báº£ng metric_types
ALTER TABLE "metric_types" ADD COLUMN "base_table" varchar(100);
ALTER TABLE "metric_types" ADD COLUMN "allowed_agg_funcs" jsonb;

-- Cáº­p nháº­t cáº¥u hÃ¬nh hiá»‡n táº¡i
UPDATE "metric_types" SET "base_table" = 'heart_rates', "allowed_agg_funcs" = '["AVG"]' WHERE "name" = 'heart_rate';
UPDATE "metric_types" SET "base_table" = 'steps_counts', "allowed_agg_funcs" = '["SUM"]' WHERE "name" = 'steps_count';
UPDATE "metric_types" SET "base_table" = 'calories_burnt', "allowed_agg_funcs" = '["SUM"]' WHERE "name" = 'calories_burned';
ALTER TABLE users
ADD COLUMN phone TEXT,
ADD COLUMN address TEXT,
ADD COLUMN gender TEXT CHECK (gender IN ('male', 'female', 'other')),
ADD COLUMN birthday DATE,
ADD COLUMN weight NUMERIC(5, 2),
ADD COLUMN height NUMERIC(5, 2),
ADD COLUMN blood_group TEXT;
-- Táº¡o báº£ng spo2
CREATE TABLE "spo2" (
  "time" TIMESTAMPTZ NOT NULL,
  "user_id" UUID NOT NULL,
  "value" DOUBLE PRECISION NOT NULL,
  PRIMARY KEY ("time", "user_id")
);

SELECT create_hypertable('spo2', 'time', if_not_exists => TRUE);

ALTER TABLE "spo2"
  ADD CONSTRAINT spo2_user_id_fkey FOREIGN KEY ("user_id")
  REFERENCES "users" ("id");


-- Táº¡o báº£ng blood_pressure
CREATE TABLE "blood_pressure" (
  "time" TIMESTAMPTZ NOT NULL,
  "user_id" UUID NOT NULL,
  "value" DOUBLE PRECISION NOT NULL,
  PRIMARY KEY ("time", "user_id")
);

SELECT create_hypertable('blood_pressure', 'time', if_not_exists => TRUE);

ALTER TABLE "blood_pressure"
  ADD CONSTRAINT blood_pressure_user_id_fkey FOREIGN KEY ("user_id")
  REFERENCES "users" ("id");


-- Seed metric_types
INSERT INTO "metric_types" ("name", "description", "base_table", "allowed_agg_funcs") VALUES
('spo2', 'Blood oxygen saturation (SpO2)', 'spo2', '["AVG"]'),
('blood_pressure', 'Blood pressure measurement', 'blood_pressure', '["AVG"]');
-- medications: parent table
CREATE TABLE medications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    dosage VARCHAR(100) NOT NULL,
    instructions TEXT DEFAULT '',
    prescribed_by VARCHAR(255) DEFAULT '',
    is_active BOOLEAN DEFAULT true,
    frequency JSONB NOT NULL DEFAULT '{}',
    start_date DATE NOT NULL,
    timezone VARCHAR(50) DEFAULT 'UTC',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_medications_user_id ON medications (user_id);
CREATE INDEX idx_medications_active ON medications (user_id) WHERE is_active = true;

-- medication_reminders: child table, CASCADE on medication delete
CREATE TABLE medication_reminders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    medication_id UUID NOT NULL REFERENCES medications(id) ON DELETE CASCADE,
    time VARCHAR(8) NOT NULL,
    is_enabled BOOLEAN DEFAULT true,
    last_taken TIMESTAMPTZ,
    missed_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_reminders_medication_id ON medication_reminders (medication_id);
CREATE INDEX idx_reminders_enabled ON medication_reminders (medication_id) WHERE is_enabled = true;
CREATE TABLE IF NOT EXISTS user_device_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL,
    platform TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, token)
);
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
-- Step 1: Add new column
ALTER TABLE "sharing_permissions" ADD COLUMN "shared_with_user_id" uuid;

-- Step 2: Since standard PK can't have NULLs and we are on PG14, we drop old PK and use Unique Indexes
ALTER TABLE "sharing_permissions" DROP CONSTRAINT "sharing_permissions_pkey";

-- Step 3: Global Rule Index (shared_with_user_id IS NULL)
CREATE UNIQUE INDEX "sharing_permissions_global_idx" ON "sharing_permissions" ("group_id", "user_id", "metric_type") WHERE "shared_with_user_id" IS NULL;

-- Step 4: Specific Rule Index (shared_with_user_id IS NOT NULL)
CREATE UNIQUE INDEX "sharing_permissions_member_idx" ON "sharing_permissions" ("group_id", "user_id", "metric_type", "shared_with_user_id") WHERE "shared_with_user_id" IS NOT NULL;

-- Step 5: Add Foreign Key for shared_with_user_id
ALTER TABLE "sharing_permissions" ADD FOREIGN KEY ("group_id", "shared_with_user_id") REFERENCES "group_members" ("group_id", "user_id");
-- medication_shares: allows sharing a medication schedule with specific group members
-- and setting a notification offset for missed pills.
CREATE TABLE medication_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    medication_id UUID NOT NULL REFERENCES medications(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    shared_with_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    notify_offset_minutes INT NOT NULL DEFAULT 15,
    
    -- Tracking for missed notifications to avoid duplicates
    last_notified_reminder_id UUID REFERENCES medication_reminders(id) ON DELETE SET NULL,
    last_notified_date DATE, -- The date (in medication's timezone) we last notified for this share

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Ensure a medication is only shared once with a specific user
    UNIQUE(medication_id, shared_with_user_id)
);

CREATE INDEX idx_medication_shares_medication_id ON medication_shares(medication_id);
CREATE INDEX idx_medication_shares_shared_with ON medication_shares(shared_with_user_id);
CREATE INDEX idx_medication_shares_group ON medication_shares(group_id);
-- Add timezone to users table
ALTER TABLE users ADD COLUMN timezone VARCHAR(50) DEFAULT 'UTC';

-- Migrate existing timezone data from medications to users (best effort)
-- If a user has multiple medications with different timezones, we'll just pick one.
UPDATE users u
SET timezone = (
    SELECT m.timezone 
    FROM medications m 
    WHERE m.user_id = u.id 
    ORDER BY m.created_at DESC 
    LIMIT 1
)
WHERE EXISTS (SELECT 1 FROM medications m WHERE m.user_id = u.id);

-- Optional: Drop timezone from medications table
-- ALTER TABLE medications DROP COLUMN timezone;
-- We'll keep it for now but deprecate it in code to avoid breaking things immediately.
-- Drop timezone from medications as it is now in users
ALTER TABLE medications DROP COLUMN IF EXISTS timezone;
