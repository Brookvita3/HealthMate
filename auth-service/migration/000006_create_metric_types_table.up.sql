CREATE TABLE "metric_types" (
  "id" uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  "name" varchar(100) UNIQUE NOT NULL,
  "description" text,
  "created_at" timestamp DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO "metric_types" ("name", "description") VALUES
('heart_rate', 'Heart rate monitor data'),
('steps_count', 'Steps taken count per day'),
('calories_burned', 'Estimated calories burned during activities');

-- Optionally add a foreign key to sharing_permissions
-- But first we might need to ensure all current values in sharing_permissions exist in metric_types
-- For now, let's just create the table and seed it.
