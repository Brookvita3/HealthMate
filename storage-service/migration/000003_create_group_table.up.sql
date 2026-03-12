CREATE TABLE "groups" (
  "id" uuid PRIMARY KEY,
  "name" varchar(255) NOT NULL,
  "description" text,
  "owner_id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "group_members" (
  "group_id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "role" varchar(50) NOT NULL DEFAULT 'member',
  "status" varchar(50) NOT NULL DEFAULT 'pending',
  "invited_by" uuid NOT NULL,
  "joined_at" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now()),
  PRIMARY KEY ("group_id", "user_id")
);

ALTER TABLE "groups" ADD FOREIGN KEY ("owner_id") REFERENCES "users" ("id");

ALTER TABLE "group_members" ADD FOREIGN KEY ("invited_by") REFERENCES "users" ("id");

ALTER TABLE "group_members" ADD FOREIGN KEY ("group_id") REFERENCES "groups" ("id");

ALTER TABLE "group_members" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

