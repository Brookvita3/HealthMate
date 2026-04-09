-- Xóa dữ liệu trong metric_types
DELETE FROM "metric_types" WHERE "name" IN ('spo2', 'blood_pressure');

-- Xóa bảng blood_pressure
DROP TABLE IF EXISTS "blood_pressure";

-- Xóa bảng spo2
DROP TABLE IF EXISTS "spo2";
