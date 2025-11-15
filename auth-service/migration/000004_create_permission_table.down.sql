-- Drop foreign key on sharing_permissions
ALTER TABLE "sharing_permissions" 
  DROP CONSTRAINT IF EXISTS sharing_permissions_group_id_user_id_fkey;

-- Drop table
DROP TABLE IF EXISTS "sharing_permissions";
