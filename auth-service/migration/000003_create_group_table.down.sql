-- 1. DROP foreign keys on group_members
ALTER TABLE "group_members" DROP CONSTRAINT IF EXISTS group_members_invited_by_fkey;
ALTER TABLE "group_members" DROP CONSTRAINT IF EXISTS group_members_group_id_fkey;
ALTER TABLE "group_members" DROP CONSTRAINT IF EXISTS group_members_user_id_fkey;

-- 2. DROP foreign key on groups
ALTER TABLE "groups" DROP CONSTRAINT IF EXISTS groups_owner_id_fkey;

-- 3. DROP tables (child first)
DROP TABLE IF EXISTS "group_members";
DROP TABLE IF EXISTS "groups";