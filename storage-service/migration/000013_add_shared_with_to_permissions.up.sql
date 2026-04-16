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
