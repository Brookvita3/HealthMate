-- This migration reverts the changes made in the corresponding 'up' migration.
-- It is designed to return the schema to its state before version 2.

-- Remove the unique constraint from google_id.
-- Note: The constraint name might vary slightly depending on your Postgres version.
-- Use \d users in psql to verify the exact constraint name if this fails.
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_google_id_key;

-- Remove the 'role' and 'status' columns.
ALTER TABLE users DROP COLUMN role;
ALTER TABLE users DROP COLUMN status;

-- Revert column types back to TEXT.
ALTER TABLE users ALTER COLUMN email TYPE TEXT;
ALTER TABLE users ALTER COLUMN name TYPE TEXT;

-- Remove the NOT NULL constraint from the provider column.
ALTER TABLE users ALTER COLUMN provider DROP NOT NULL;