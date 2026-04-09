-- Xóa cột cấu hình cho bảng metric_types
ALTER TABLE "metric_types" DROP COLUMN IF EXISTS "allowed_agg_funcs";
ALTER TABLE "metric_types" DROP COLUMN IF EXISTS "base_table";
