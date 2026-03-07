-- Script để test continuous aggregates (materialized views)
-- Chạy trong psql: docker exec -it timescaledb-storage psql -U healthmate -d healthmate

\timing on

-- ============================================
-- 1. XEM THÔNG TIN CONTINUOUS AGGREGATES
-- ============================================
SELECT '=== Continuous Aggregates Info ===' AS info;
SELECT 
    view_name,
    refresh_lag,
    refresh_interval
FROM timescaledb_information.continuous_aggregates
WHERE view_name LIKE '%_daily' OR view_name LIKE '%_hourly'
ORDER BY view_name;

-- ============================================
-- 2. XEM THỐNG KÊ DỮ LIỆU
-- ============================================
SELECT '=== Data Statistics ===' AS info;

-- Đếm số records trong raw tables
SELECT 'Heart Rates (raw)' AS table_name, COUNT(*) AS total_records FROM heart_rates;
SELECT 'Steps Counts (raw)' AS table_name, COUNT(*) AS total_records FROM steps_counts;
SELECT 'Calories Burnt (raw)' AS table_name, COUNT(*) AS total_records FROM calories_burnt;

-- Đếm số records trong materialized views
SELECT 'Heart Rates Daily (materialized)' AS view_name, COUNT(*) AS total_days FROM heart_rates_daily;
SELECT 'Heart Rates Hourly (materialized)' AS view_name, COUNT(*) AS total_hours FROM heart_rates_hourly;

-- ============================================
-- 3. XEM DỮ LIỆU GẦN NHẤT (RAW vs MATERIALIZED)
-- ============================================
SELECT '=== Latest Raw Data ===' AS info;
SELECT time, value FROM heart_rates ORDER BY time DESC LIMIT 5;

SELECT '=== Latest Materialized (Daily) ===' AS info;
SELECT bucket, avg_value, min_value, max_value, count 
FROM heart_rates_daily 
ORDER BY bucket DESC LIMIT 5;

SELECT '=== Latest Materialized (Hourly) ===' AS info;
SELECT bucket, avg_value, min_value, max_value, count 
FROM heart_rates_hourly 
ORDER BY bucket DESC LIMIT 5;

-- ============================================
-- 4. SO SÁNH RAW vs MATERIALIZED
-- ============================================
SELECT '=== Comparison: Today Data ===' AS info;

-- Raw count hôm nay
SELECT 'Today raw count' AS info, COUNT(*) AS count
FROM heart_rates
WHERE time >= DATE_TRUNC('day', NOW());

-- Materialized count hôm nay
SELECT 'Today materialized count' AS info, COALESCE(count, 0) AS count
FROM heart_rates_daily
WHERE bucket >= DATE_TRUNC('day', NOW());

-- ============================================
-- 5. TEST AUTO-REFRESH STATUS
-- ============================================
SELECT '=== Auto-Refresh Status ===' AS info;
SELECT 
    view_name,
    completed_threshold,
    invalidation_threshold,
    NOW() - completed_threshold AS last_refresh_age
FROM timescaledb_information.continuous_aggregate_stats
WHERE view_name LIKE '%_daily' OR view_name LIKE '%_hourly'
ORDER BY view_name;

-- Xem bucket gần nhất và độ trễ
SELECT '=== Latest Bucket Age ===' AS info;
SELECT 
    'heart_rates_daily' AS view_name,
    bucket,
    count,
    NOW() - bucket AS bucket_age
FROM heart_rates_daily
ORDER BY bucket DESC
LIMIT 1;

-- ============================================
-- 6. PERFORMANCE TEST (RAW vs MATERIALIZED)
-- ============================================
SELECT '=== Performance Test: Last 7 Days ===' AS info;

-- Query trên raw table
SELECT 'Query on RAW table:' AS info;
EXPLAIN ANALYZE
SELECT 
    DATE(time) AS date,
    AVG(value) AS avg_value,
    MIN(value) AS min_value,
    MAX(value) AS max_value,
    COUNT(*) AS count
FROM heart_rates
WHERE user_id = '00000000-0000-0000-0000-000000000001'
  AND time >= NOW() - INTERVAL '7 days'
GROUP BY DATE(time)
ORDER BY date DESC;

-- Query trên materialized view
SELECT 'Query on MATERIALIZED VIEW:' AS info;
EXPLAIN ANALYZE
SELECT 
    bucket AS date,
    avg_value,
    min_value,
    max_value,
    count
FROM heart_rates_daily
WHERE user_id = '00000000-0000-0000-0000-000000000001'
  AND bucket >= NOW() - INTERVAL '7 days'
ORDER BY bucket DESC;

-- ============================================
-- 7. MONITORING QUERY (chạy nhiều lần để xem update)
-- ============================================
SELECT '=== Real-time Monitoring ===' AS info;
SELECT 
    NOW() AS current_time,
    (SELECT COUNT(*) FROM heart_rates) AS raw_count,
    (SELECT COUNT(*) FROM heart_rates_daily) AS daily_count,
    (SELECT COUNT(*) FROM heart_rates_hourly) AS hourly_count,
    (SELECT bucket FROM heart_rates_daily ORDER BY bucket DESC LIMIT 1) AS latest_daily_bucket,
    (SELECT bucket FROM heart_rates_hourly ORDER BY bucket DESC LIMIT 1) AS latest_hourly_bucket;

-- ============================================
-- 8. MANUAL REFRESH (chạy nếu muốn force update)
-- ============================================
-- Uncomment để chạy manual refresh
-- SELECT 'Refreshing continuous aggregates...' AS info;
-- CALL refresh_continuous_aggregate('heart_rates_hourly', NULL, NULL);
-- CALL refresh_continuous_aggregate('heart_rates_daily', NULL, NULL);
-- CALL refresh_continuous_aggregate('steps_counts_hourly', NULL, NULL);
-- CALL refresh_continuous_aggregate('steps_counts_daily', NULL, NULL);
-- CALL refresh_continuous_aggregate('calories_burnt_hourly', NULL, NULL);
-- CALL refresh_continuous_aggregate('calories_burnt_daily', NULL, NULL);

\timing off
