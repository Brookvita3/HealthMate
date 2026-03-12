CREATE TABLE "sharing_permissions" (
  "group_id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "metric_type" varchar(50) NOT NULL,
  PRIMARY KEY ("group_id", "user_id", "metric_type")
);

ALTER TABLE "sharing_permissions" ADD FOREIGN KEY ("group_id", "user_id") REFERENCES "group_members" ("group_id", "user_id");
