-- Drop timezone from medications as it is now in users
ALTER TABLE medications DROP COLUMN IF EXISTS timezone;
