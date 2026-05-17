-- Add end_date to medications (optional, NULL = no end)
ALTER TABLE medications ADD COLUMN IF NOT EXISTS end_date DATE;

-- Add weight metric type and table
CREATE TABLE IF NOT EXISTS weight (
  "time"    TIMESTAMPTZ      NOT NULL,
  "user_id" UUID             NOT NULL,
  "value"   DOUBLE PRECISION NOT NULL,
  PRIMARY KEY ("time", "user_id")
);

SELECT create_hypertable('weight', 'time', if_not_exists => TRUE);

ALTER TABLE weight
  ADD CONSTRAINT weight_user_id_fkey FOREIGN KEY ("user_id")
  REFERENCES "users" ("id");

INSERT INTO metric_types (name, description, base_table, allowed_agg_funcs)
VALUES ('weight', 'Body weight in kg', 'weight', '["AVG"]')
ON CONFLICT (name) DO NOTHING;
