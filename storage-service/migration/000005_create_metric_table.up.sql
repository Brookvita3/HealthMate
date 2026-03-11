-- Tạo bảng heart_rate
CREATE TABLE "heart_rates" (
  "time" TIMESTAMPTZ NOT NULL,
  "user_id" UUID NOT NULL,
  "value" DOUBLE PRECISION NOT NULL,
  PRIMARY KEY ("time", "user_id")
);

-- Chuyển thành hypertable
SELECT create_hypertable('heart_rates', 'time', if_not_exists => TRUE);


-- Tạo bảng steps_counts
CREATE TABLE "steps_counts" (
  "time" TIMESTAMPTZ NOT NULL,
  "user_id" UUID NOT NULL,
  "value" INTEGER NOT NULL,
  PRIMARY KEY ("time", "user_id")
);

SELECT create_hypertable('steps_counts', 'time', if_not_exists => TRUE);


-- Tạo bảng calories_burnt
CREATE TABLE "calories_burnt" (
  "time" TIMESTAMPTZ NOT NULL,
  "user_id" UUID NOT NULL,
  "value" DOUBLE PRECISION NOT NULL,
  PRIMARY KEY ("time", "user_id")
);

SELECT create_hypertable('calories_burnt', 'time', if_not_exists => TRUE);


-- FOREIGN KEYS
ALTER TABLE "heart_rates"
  ADD CONSTRAINT heart_rates_user_id_fkey FOREIGN KEY ("user_id") 
  REFERENCES "users" ("id");

ALTER TABLE "steps_counts"
  ADD CONSTRAINT steps_counts_user_id_fkey FOREIGN KEY ("user_id") 
  REFERENCES "users" ("id");

ALTER TABLE "calories_burnt"
  ADD CONSTRAINT calories_burnt_user_id_fkey FOREIGN KEY ("user_id") 
  REFERENCES "users" ("id");
