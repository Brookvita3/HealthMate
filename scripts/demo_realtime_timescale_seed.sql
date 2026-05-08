-- Demo: day du lieu time-series gan day de xem chart / realtime (storage TimescaleDB).
-- Mac dinh user = Alice trong bo E2E (scripts/e2e_sharing_permissions_seed.sql).
--
-- Chay (tu thu muc HealthMate_BE/HealthMate):
--   Get-Content scripts\demo_realtime_timescale_seed.sql -Raw | docker compose exec -T timescaledb psql -U postgres -d healthmate
--
-- Luu y: Chi insert vao bang da ton tai. Neu DB chua co blood_pressure/spo2 trong metric_types,
-- bo qua (DO $$ ... IF EXISTS table metric_types name ... khong can - chi can bang du lieu).

BEGIN;

-- UUID Alice E2E (doi neu can)
-- psql khong ho tro bien tu shell; dung literal:
-- 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'

DELETE FROM heart_rates
WHERE user_id = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid
  AND time > NOW() - INTERVAL '48 hours';

DELETE FROM steps_counts
WHERE user_id = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid
  AND time > NOW() - INTERVAL '48 hours';

DELETE FROM calories_burnt
WHERE user_id = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid
  AND time > NOW() - INTERVAL '48 hours';

INSERT INTO heart_rates (time, user_id, value)
SELECT
  NOW() - (n * INTERVAL '30 minutes'),
  'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid,
  (72 + 8 * sin(n / 4.0) + (random() * 4 - 2))::double precision
FROM generate_series(1, 96) AS n;

INSERT INTO steps_counts (time, user_id, value)
SELECT
  NOW() - (n * INTERVAL '30 minutes'),
  'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid,
  (2000 + (n * 50) % 800 + (random() * 200)::int)::int
FROM generate_series(1, 96) AS n;

INSERT INTO calories_burnt (time, user_id, value)
SELECT
  NOW() - (n * INTERVAL '30 minutes'),
  'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid,
  (120 + (n % 20) * 3 + (random() * 15))::double precision
FROM generate_series(1, 96) AS n;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'blood_pressure'
  ) AND EXISTS (
    SELECT 1 FROM metric_types WHERE name = 'blood_pressure'
  ) THEN
    DELETE FROM blood_pressure
    WHERE user_id = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid
      AND time > NOW() - INTERVAL '48 hours';
    INSERT INTO blood_pressure (time, user_id, value)
    SELECT
      NOW() - (n * INTERVAL '30 minutes'),
      'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid,
      (115 + (random() * 12))::double precision
    FROM generate_series(1, 48) AS n;
  END IF;
END $$;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'spo2'
  ) AND EXISTS (
    SELECT 1 FROM metric_types WHERE name = 'spo2'
  ) THEN
    DELETE FROM spo2
    WHERE user_id = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid
      AND time > NOW() - INTERVAL '48 hours';
    INSERT INTO spo2 (time, user_id, value)
    SELECT
      NOW() - (n * INTERVAL '30 minutes'),
      'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid,
      (96 + (random() * 3))::double precision
    FROM generate_series(1, 48) AS n;
  END IF;
END $$;

COMMIT;
