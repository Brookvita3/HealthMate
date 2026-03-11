-- Thêm cột cấu hình cho bảng metric_types
ALTER TABLE "metric_types" ADD COLUMN "base_table" varchar(100);
ALTER TABLE "metric_types" ADD COLUMN "allowed_agg_funcs" jsonb;

-- Cập nhật cấu hình hiện tại
UPDATE "metric_types" SET "base_table" = 'heart_rates', "allowed_agg_funcs" = '["AVG"]' WHERE "name" = 'heart_rate';
UPDATE "metric_types" SET "base_table" = 'steps_counts', "allowed_agg_funcs" = '["SUM"]' WHERE "name" = 'steps_count';
UPDATE "metric_types" SET "base_table" = 'calories_burnt', "allowed_agg_funcs" = '["SUM"]' WHERE "name" = 'calories_burned';
