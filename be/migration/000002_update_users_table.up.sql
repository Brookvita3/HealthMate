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