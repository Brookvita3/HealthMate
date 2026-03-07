-- Script để insert 10,000 records ban đầu cho mỗi metric
-- Chạy: docker cp insert-bulk-data.sql timescaledb-storage:/tmp/insert-bulk-data.sql
--       docker exec -it timescaledb-storage psql -U healthmate -d healthmate -f /tmp/insert-bulk-data.sql

-- User ID mẫu
-- '00000000-0000-0000-0000-000000000001'

\timing on

-- Insert 10,000 heart rate records (dữ liệu từ ~7 ngày trước đến hiện tại)
INSERT INTO heart_rates (time, user_id, value)
SELECT 
    NOW() - (INTERVAL '1 minute' * generate_series(0, 9999)),
    '00000000-0000-0000-0000-000000000001'::UUID,
    60.0 + random() * 40.0  -- Heart rate: 60-100 bpm
FROM generate_series(1, 1);

-- Insert 10,000 steps count records
INSERT INTO steps_counts (time, user_id, value)
SELECT 
    NOW() - (INTERVAL '1 minute' * generate_series(0, 9999)),
    '00000000-0000-0000-0000-000000000001'::UUID,
    floor(100 + random() * 500)::INTEGER  -- Steps: 100-600 per minute
FROM generate_series(1, 1);

-- Insert 10,000 calories burnt records
INSERT INTO calories_burnt (time, user_id, value)
SELECT 
    NOW() - (INTERVAL '1 minute' * generate_series(0, 9999)),
    '00000000-0000-0000-0000-000000000001'::UUID,
    1.0 + random() * 4.0  -- Calories: 1-5 per minute
FROM generate_series(1, 1);

\timing off

-- Đếm số lượng records
SELECT 'Heart Rates' AS metric_type, COUNT(*) AS total_records FROM heart_rates;
SELECT 'Steps Counts' AS metric_type, COUNT(*) AS total_records FROM steps_counts;
SELECT 'Calories Burnt' AS metric_type, COUNT(*) AS total_records FROM calories_burnt;

-- Xem dữ liệu gần nhất
SELECT 'Latest 5 heart_rates:' AS info;
SELECT time, value FROM heart_rates ORDER BY time DESC LIMIT 5;

-- Refresh continuous aggregates
SELECT 'Refreshing continuous aggregates...' AS info;
CALL refresh_continuous_aggregate('heart_rates_hourly', NULL, NULL);
CALL refresh_continuous_aggregate('heart_rates_daily', NULL, NULL);
CALL refresh_continuous_aggregate('steps_counts_hourly', NULL, NULL);
CALL refresh_continuous_aggregate('steps_counts_daily', NULL, NULL);
CALL refresh_continuous_aggregate('calories_burnt_hourly', NULL, NULL);
CALL refresh_continuous_aggregate('calories_burnt_daily', NULL, NULL);

-- Kiểm tra aggregation
SELECT 'Daily aggregation count:' AS info, COUNT(*) AS days FROM heart_rates_daily;
SELECT 'Hourly aggregation count:' AS info, COUNT(*) AS hours FROM heart_rates_hourly;

-- Xem mẫu daily aggregation
SELECT 'Latest 5 daily aggregations:' AS info;
SELECT bucket, avg_value, min_value, max_value, count FROM heart_rates_daily ORDER BY bucket DESC LIMIT 5;
