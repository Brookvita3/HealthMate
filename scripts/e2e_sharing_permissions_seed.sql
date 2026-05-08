-- E2E seed: quyền chia sẻ chỉ số + 1 medication share (dev only).
-- Chạy: docker compose exec -T timescaledb psql -U postgres -d healthmate -f /path/to/this/file
-- Hoặc: Get-Content scripts\e2e_sharing_permissions_seed.sql | docker compose exec -T timescaledb psql -U postgres -d healthmate

BEGIN;

DELETE FROM medication_shares WHERE id = '33333333-3333-4333-8333-333333333333'::uuid;
DELETE FROM medication_reminders WHERE medication_id = '22222222-2222-4222-8222-222222222222'::uuid;
DELETE FROM medications WHERE id = '22222222-2222-4222-8222-222222222222'::uuid;
DELETE FROM sharing_permissions WHERE group_id = '11111111-1111-4111-8111-111111111111'::uuid;
DELETE FROM group_members WHERE group_id = '11111111-1111-4111-8111-111111111111'::uuid;
DELETE FROM groups WHERE id = '11111111-1111-4111-8111-111111111111'::uuid;
DELETE FROM heart_rates WHERE user_id = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid;
DELETE FROM steps_counts WHERE user_id = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid;
DELETE FROM calories_burnt WHERE user_id = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid;
DO $$ BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'blood_pressure'
  ) THEN
    DELETE FROM blood_pressure WHERE user_id = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid;
  END IF;
END $$;
-- spo2: migration 000009 có thể chưa áp dụng trên DB dev
DO $$ BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'spo2'
  ) THEN
    DELETE FROM spo2 WHERE user_id = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid;
  END IF;
END $$;
DELETE FROM users WHERE id IN (
  'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid,
  'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'::uuid,
  'cccccccc-cccc-4ccc-8ccc-cccccccccccc'::uuid,
  'dddddddd-dddd-4ddd-8ddd-dddddddddddd'::uuid,
  'eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee'::uuid,
  'ffffffff-ffff-4fff-8fff-ffffffffffff'::uuid
);

INSERT INTO users (id, email, name, provider, role, status, password, created_at, updated_at)
VALUES
  ('aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid, 'e2e_share_alice@healthmate.test', 'E2E Alice', 'HealthMate', 'user', 'verified', NULL, NOW(), NOW()),
  ('bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'::uuid, 'e2e_share_bob@healthmate.test', 'E2E Bob', 'HealthMate', 'user', 'verified', NULL, NOW(), NOW()),
  ('cccccccc-cccc-4ccc-8ccc-cccccccccccc'::uuid, 'e2e_share_charlie@healthmate.test', 'E2E Charlie', 'HealthMate', 'user', 'verified', NULL, NOW(), NOW()),
  ('dddddddd-dddd-4ddd-8ddd-dddddddddddd'::uuid, 'e2e_share_diana@healthmate.test', 'E2E Diana', 'HealthMate', 'user', 'verified', NULL, NOW(), NOW()),
  ('eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee'::uuid, 'e2e_share_eve@healthmate.test', 'E2E Eve', 'HealthMate', 'user', 'verified', NULL, NOW(), NOW()),
  ('ffffffff-ffff-4fff-8fff-ffffffffffff'::uuid, 'e2e_share_frank@healthmate.test', 'E2E Frank', 'HealthMate', 'user', 'verified', NULL, NOW(), NOW());

INSERT INTO groups (id, name, description, owner_id, created_at, updated_at)
VALUES (
  '11111111-1111-4111-8111-111111111111'::uuid,
  'E2E Sharing Permissions',
  'Seed cho test quyền /metrics/charts',
  'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid,
  NOW(), NOW()
);

INSERT INTO group_members (group_id, user_id, role, status, invited_by, joined_at, created_at, updated_at)
VALUES
  ('11111111-1111-4111-8111-111111111111'::uuid, 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid, 'owner', 'accepted', 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid, NOW(), NOW(), NOW()),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'::uuid, 'member', 'accepted', 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid, NOW(), NOW(), NOW()),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'cccccccc-cccc-4ccc-8ccc-cccccccccccc'::uuid, 'member', 'accepted', 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid, NOW(), NOW(), NOW()),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee'::uuid, 'member', 'accepted', 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid, NOW(), NOW(), NOW()),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'ffffffff-ffff-4fff-8fff-ffffffffffff'::uuid, 'member', 'accepted', 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid, NOW(), NOW(), NOW());
-- Diana KHÔNG trong nhóm → test từ chối người ngoài

-- Alice: base (shared_with NULL) + filter/specific theo luật CheckAccess trong storage-service.
-- Eve: có access_control + chỉ HR/BP cụ thể → không được bước/calo/spo2 (không base spo2).
-- Frank: chỉ member, không có hàng filter → hưởng mọi base.
-- Charlie: filter + chỉ steps cụ thể → không HR/calo/BP.
INSERT INTO sharing_permissions (group_id, user_id, metric_type, shared_with_user_id)
VALUES
  ('11111111-1111-4111-8111-111111111111'::uuid, 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid, 'heart_rate', NULL),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid, 'steps_count', NULL),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid, 'calories_burned', NULL),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid, 'blood_pressure', NULL),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid, 'access_control', 'cccccccc-cccc-4ccc-8ccc-cccccccccccc'::uuid),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid, 'steps_count', 'cccccccc-cccc-4ccc-8ccc-cccccccccccc'::uuid),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid, 'access_control', 'eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee'::uuid),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid, 'heart_rate', 'eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee'::uuid),
  ('11111111-1111-4111-8111-111111111111'::uuid, 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid, 'blood_pressure', 'eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee'::uuid);

INSERT INTO heart_rates (time, user_id, value)
VALUES
  (NOW() - INTERVAL '2 hours', 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid, 72.0),
  (NOW() - INTERVAL '1 hour', 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid, 75.0);

INSERT INTO steps_counts (time, user_id, value)
VALUES
  (NOW() - INTERVAL '30 minutes', 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid, 4200);

INSERT INTO calories_burnt (time, user_id, value)
VALUES
  (NOW() - INTERVAL '25 minutes', 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid, 180.5);

DO $$ BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'blood_pressure'
  ) THEN
    INSERT INTO blood_pressure (time, user_id, value)
    VALUES
      (NOW() - INTERVAL '15 minutes', 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid, 118.0);
  END IF;
END $$;

DO $$ BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'spo2'
  ) THEN
    INSERT INTO spo2 (time, user_id, value)
    VALUES
      (NOW() - INTERVAL '20 minutes', 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid, 98.0);
  END IF;
END $$;

INSERT INTO medications (id, user_id, name, dosage, instructions, prescribed_by, is_active, frequency, start_date, created_at, updated_at)
VALUES (
  '22222222-2222-4222-8222-222222222222'::uuid,
  'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid,
  'E2E Paracetamol',
  '1 viên',
  'Sau ăn',
  '',
  true,
  '{}'::jsonb,
  CURRENT_DATE,
  NOW(), NOW()
);

INSERT INTO medication_reminders (medication_id, time, is_enabled)
VALUES ('22222222-2222-4222-8222-222222222222'::uuid, '08:00', true);

INSERT INTO medication_shares (id, medication_id, group_id, shared_with_user_id, notify_offset_minutes, created_at, updated_at)
VALUES (
  '33333333-3333-4333-8333-333333333333'::uuid,
  '22222222-2222-4222-8222-222222222222'::uuid,
  '11111111-1111-4111-8111-111111111111'::uuid,
  'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'::uuid,
  15,
  NOW(), NOW()
);

COMMIT;
