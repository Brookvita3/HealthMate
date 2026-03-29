-- Tạo bảng spo2
CREATE TABLE "spo2" (
  "time" TIMESTAMPTZ NOT NULL,
  "user_id" UUID NOT NULL,
  "value" DOUBLE PRECISION NOT NULL,
  PRIMARY KEY ("time", "user_id")
);

SELECT create_hypertable('spo2', 'time', if_not_exists => TRUE);

ALTER TABLE "spo2"
  ADD CONSTRAINT spo2_user_id_fkey FOREIGN KEY ("user_id")
  REFERENCES "users" ("id");


-- Tạo bảng blood_pressure
CREATE TABLE "blood_pressure" (
  "time" TIMESTAMPTZ NOT NULL,
  "user_id" UUID NOT NULL,
  "value" DOUBLE PRECISION NOT NULL,
  PRIMARY KEY ("time", "user_id")
);

SELECT create_hypertable('blood_pressure', 'time', if_not_exists => TRUE);

ALTER TABLE "blood_pressure"
  ADD CONSTRAINT blood_pressure_user_id_fkey FOREIGN KEY ("user_id")
  REFERENCES "users" ("id");


-- Seed metric_types
INSERT INTO "metric_types" ("name", "description", "base_table", "allowed_agg_funcs") VALUES
('spo2', 'Blood oxygen saturation (SpO2)', 'spo2', '["AVG"]'),
('blood_pressure', 'Blood pressure measurement', 'blood_pressure', '["AVG"]');
