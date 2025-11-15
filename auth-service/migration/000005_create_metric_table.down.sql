-- DROP foreign keys
ALTER TABLE "heart_rates" DROP CONSTRAINT IF EXISTS heart_rates_user_id_fkey;
ALTER TABLE "steps_counts" DROP CONSTRAINT IF EXISTS steps_counts_user_id_fkey;
ALTER TABLE "calories_burnt" DROP CONSTRAINT IF EXISTS calories_burnt_user_id_fkey;

-- DROP tables
DROP TABLE IF EXISTS "heart_rates";
DROP TABLE IF EXISTS "steps_counts";
DROP TABLE IF EXISTS "calories_burnt";
