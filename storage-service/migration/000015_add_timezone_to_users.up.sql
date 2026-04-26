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
