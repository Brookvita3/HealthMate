DELETE FROM metric_types WHERE name = 'weight';
DROP TABLE IF EXISTS weight;
ALTER TABLE medications DROP COLUMN IF EXISTS end_date;
