ALTER TABLE "sharing_permissions" DROP CONSTRAINT IF EXISTS "sharing_permissions_shared_with_user_id_fkey";
DROP INDEX IF EXISTS "sharing_permissions_member_idx";
DROP INDEX IF EXISTS "sharing_permissions_global_idx";

-- Restore original PK
-- Note: This might fail if there are specific member permissions (non-NULL shared_with_user_id)
-- But typically down migrations are for rolling back to a clean state.
DELETE FROM "sharing_permissions" WHERE "shared_with_user_id" IS NOT NULL;
ALTER TABLE "sharing_permissions" DROP COLUMN "shared_with_user_id";
ALTER TABLE "sharing_permissions" ADD PRIMARY KEY ("group_id", "user_id", "metric_type");
