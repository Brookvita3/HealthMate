# Storage Service - Test Guide

Hướng dẫn test Continuous Aggregates (Materialized Views) với TimescaleDB.

---

## I. TEST LẦN ĐẦU TIÊN (First Time Setup)

### Bước 1: Khởi động TimescaleDB

```bash
cd C:\Hoc_Tap\HealthMate_BE\HealthMate\storage-service
docker compose up -d
```

Đợi 10-15 giây để database khởi động.

### Bước 2: Kiểm tra database sẵn sàng

```bash
docker exec timescaledb-storage pg_isready -U healthmate
```

Kết quả mong đợi: `/var/run/postgresql:5432 - accepting connections`

### Bước 3: Verify continuous aggregates đã được tạo

```bash
docker exec -it timescaledb-storage psql -U healthmate -d healthmate -c "\d+ heart_rates_daily"
```

Nếu thấy view details → OK. Nếu không → database chưa init đúng.

### Bước 4: Insert 10,000 records ban đầu

```bash
# Copy file SQL vào container
docker cp insert-bulk-data.sql timescaledb-storage:/tmp/insert-bulk-data.sql

# Chạy script insert
docker exec -it timescaledb-storage psql -U healthmate -d healthmate -f /tmp/insert-bulk-data.sql
```

**Kết quả mong đợi:**
- `INSERT 0 10000` (3 lần cho 3 bảng: heart_rates, steps_counts, calories_burnt)
- Total records: 10,000 cho mỗi bảng
- Daily aggregation: ~7 buckets (7 ngày)
- Hourly aggregation: ~168 buckets (7 ngày × 24 giờ)

### Bước 5: Verify data đã insert

```bash
# Kiểm tra raw data
docker exec -it timescaledb-storage psql -U healthmate -d healthmate -c "SELECT COUNT(*) FROM heart_rates;"

# Kiểm tra materialized view
docker exec -it timescaledb-storage psql -U healthmate -d healthmate -c "SELECT COUNT(*) FROM heart_rates_daily;"
```

**Kết quả mong đợi:**
- Raw table: 10,000 records
- Daily view: ~7 buckets

### Bước 6: Chạy streaming insert

**Mở terminal mới (Terminal 1)** và chạy:

```bash
docker run --rm --network storage-service_default -e TIMESCALEDB_URL=postgres://healthmate:healthmate123@timescaledb:5432/healthmate?sslmode=disable -e GOPROXY=direct -e GOSUMDB=off -v "C:\Hoc_Tap\HealthMate_BE\HealthMate\storage-service:/app" -w /app golang:1.23 sh -c "go mod download && go run cmd/stream_insert/main.go"
```

**Kết quả mong đợi:**
```
Successfully connected to TimescaleDB
==============================================
STREAMING INSERT - 30 records/minute for each metric
Press Ctrl+C to stop
==============================================
User ID: 00000000-0000-0000-0000-000000000001

[2026-01-23 15:30:00] Inserting batch...
  ✓ Inserted: 90 records (errors: 0)
  Total so far: 90 records
```

→ Script sẽ insert 90 records mỗi phút (30 cho mỗi metric). **Để script này chạy.**

### Bước 7: Monitor real-time (Terminal 2)

**Mở terminal mới (Terminal 2)** và kết nối vào database:

```bash
docker exec -it timescaledb-storage psql -U healthmate -d healthmate
```

#### Test 1: Xem số lượng records (chạy nhiều lần)

```sql
-- Xem raw count
SELECT 'Raw count' AS info, COUNT(*) FROM heart_rates;

-- Xem materialized count
SELECT 'Daily buckets' AS info, COUNT(*) FROM heart_rates_daily;
SELECT 'Hourly buckets' AS info, COUNT(*) FROM heart_rates_hourly;
```

Chạy lại query này sau mỗi 1-2 phút → thấy raw count tăng dần (+90 mỗi phút).

#### Test 2: Monitor real-time

```sql
SELECT 
    NOW() AS current_time,
    (SELECT COUNT(*) FROM heart_rates) AS raw_count,
    (SELECT COUNT(*) FROM heart_rates_daily) AS daily_count,
    (SELECT time FROM heart_rates ORDER BY time DESC LIMIT 1) AS latest_raw_time;
```

Hoặc auto-refresh mỗi 30 giây:

```sql
\watch 30
```

Press `Ctrl+C` để dừng auto-refresh.

#### Test 3: Xem data gần nhất

```sql
-- Raw data
SELECT time, value FROM heart_rates ORDER BY time DESC LIMIT 10;

-- Materialized view (chưa refresh)
SELECT bucket, avg_value, min_value, max_value, count 
FROM heart_rates_daily 
ORDER BY bucket DESC 
LIMIT 5;
```

#### Test 4: Manual refresh continuous aggregate

```sql
-- Force refresh để cập nhật data mới
CALL refresh_continuous_aggregate('heart_rates_hourly', NULL, NULL);
CALL refresh_continuous_aggregate('heart_rates_daily', NULL, NULL);
```

#### Test 5: Xem lại sau khi refresh

```sql
-- Xem lại daily aggregation
SELECT bucket, avg_value, min_value, max_value, count 
FROM heart_rates_daily 
ORDER BY bucket DESC 
LIMIT 5;
```

→ Thấy `count` tăng lên, data mới đã được aggregate.

#### Test 6: So sánh performance (RAW vs MATERIALIZED)

```sql
-- Query trên RAW table (chậm)
EXPLAIN ANALYZE
SELECT 
    DATE(time) AS date,
    AVG(value) AS avg_value,
    COUNT(*) AS count
FROM heart_rates
WHERE time >= NOW() - INTERVAL '7 days'
GROUP BY DATE(time)
ORDER BY date DESC;

-- Query trên MATERIALIZED VIEW (nhanh)
EXPLAIN ANALYZE
SELECT 
    bucket AS date,
    avg_value,
    count
FROM heart_rates_daily
WHERE bucket >= NOW() - INTERVAL '7 days'
ORDER BY bucket DESC;
```

So sánh `Execution Time` → materialized view nhanh hơn 100-250x.

### Bước 8: Hoặc chạy tất cả tests từ file

Copy và chạy toàn bộ file `test-materialized-view.sql`:

```bash
docker cp test-materialized-view.sql timescaledb-storage:/tmp/test-materialized-view.sql
docker exec -it timescaledb-storage psql -U healthmate -d healthmate -f /tmp/test-materialized-view.sql
```

### Bước 9: Dừng test

1. **Terminal 1** (streaming insert): Press `Ctrl+C`
2. **Terminal 2** (psql): Gõ `\q` để thoát

### Bước 10: Tổng kết

```bash
# Xem tổng số records cuối cùng
docker exec -it timescaledb-storage psql -U healthmate -d healthmate -c "
SELECT 
    (SELECT COUNT(*) FROM heart_rates) AS raw_total,
    (SELECT COUNT(*) FROM heart_rates_daily) AS daily_buckets,
    (SELECT COUNT(*) FROM heart_rates_hourly) AS hourly_buckets;
"
```

---

## II. TEST CÁC LẦN SAU (Subsequent Tests)

Database đã có data từ lần test trước. Không cần insert 10k records lại.

### Bước 1: Khởi động TimescaleDB (nếu đã tắt)

```bash
cd C:\Hoc_Tap\HealthMate_BE\HealthMate\storage-service
docker compose up -d
```

### Bước 2: Kiểm tra data hiện tại

```bash
docker exec -it timescaledb-storage psql -U healthmate -d healthmate -c "
SELECT 
    'Heart Rates' AS metric,
    (SELECT COUNT(*) FROM heart_rates) AS raw_count,
    (SELECT COUNT(*) FROM heart_rates_daily) AS daily_count;
"
```

### Bước 3: (Optional) Xóa data cũ nếu muốn test lại từ đầu

```bash
# Xóa tất cả data
docker exec -it timescaledb-storage psql -U healthmate -d healthmate -c "
DELETE FROM heart_rates;
DELETE FROM steps_counts;
DELETE FROM calories_burnt;
"

# Sau đó quay lại phần I để insert 10k records
```

### Bước 4: Chạy streaming insert

```bash
docker run --rm --network storage-service_default -e TIMESCALEDB_URL=postgres://healthmate:healthmate123@timescaledb:5432/healthmate?sslmode=disable -e GOPROXY=direct -e GOSUMDB=off -v "C:\Hoc_Tap\HealthMate_BE\HealthMate\storage-service:/app" -w /app golang:1.23 sh -c "go mod download && go run cmd/stream_insert/main.go"
```

### Bước 5: Monitor và test

Mở terminal mới:

```bash
docker exec -it timescaledb-storage psql -U healthmate -d healthmate
```

Chạy các query test như phần I, Bước 7.

### Bước 6: Quick test - Copy/Paste các query này

```sql
-- 1. Xem current status
SELECT 
    NOW() AS time,
    (SELECT COUNT(*) FROM heart_rates) AS raw,
    (SELECT COUNT(*) FROM heart_rates_daily) AS daily;

-- 2. Xem data mới nhất
SELECT time, value FROM heart_rates ORDER BY time DESC LIMIT 5;

-- 3. Refresh continuous aggregate
CALL refresh_continuous_aggregate('heart_rates_daily', NULL, NULL);

-- 4. Xem aggregated data
SELECT bucket, avg_value, count 
FROM heart_rates_daily 
ORDER BY bucket DESC 
LIMIT 5;

-- 5. Monitor auto (mỗi 30s)
\watch 30
```

Press `Ctrl+C` để dừng.

---

## III. TROUBLESHOOTING

### Lỗi: "relation does not exist"

```bash
# Kiểm tra continuous aggregates
docker exec -it timescaledb-storage psql -U healthmate -d healthmate -c "\d"

# Nếu không thấy heart_rates_daily → chạy lại init
docker compose down -v
docker compose up -d
```

### Lỗi: "duplicate key value violates unique constraint"

Data đã tồn tại. Xóa data cũ:

```bash
docker exec -it timescaledb-storage psql -U healthmate -d healthmate -c "
DELETE FROM heart_rates WHERE user_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM steps_counts WHERE user_id = '00000000-0000-0000-0000-000000000001';
DELETE FROM calories_burnt WHERE user_id = '00000000-0000-0000-0000-000000000001';
"
```

### Lỗi: Docker network not found

```bash
# Tạo lại network
docker network create storage-service_default
```

### Lỗi: Port 5432 đã được sử dụng

Sửa `docker-compose.yml`:
```yaml
ports:
  - "5433:5432"  # Đổi port
```

Và cập nhật `TIMESCALEDB_URL`:
```
postgres://healthmate:healthmate123@timescaledb:5432/healthmate?sslmode=disable
```

(Giữ nguyên `:5432` vì đó là port bên trong container)

---

## IV. EXPECTED RESULTS

### Sau khi insert 10k records:
- Raw table: 10,000 records
- Daily buckets: ~7 (7 ngày)
- Hourly buckets: ~168 (7 × 24)

### Sau 5 phút streaming insert:
- Raw table: 10,450 records (+450 = 5 phút × 90 records/phút)
- Daily buckets: 7-8 (tùy thời gian trong ngày)
- Hourly buckets: 168-169

### Performance:
- RAW query: ~10-20ms với 10k records
- MATERIALIZED query: ~0.05-0.1ms
- **Speed up: 100-250x**

---

## V. FILES OVERVIEW

| File | Mục đích |
|------|----------|
| `insert-bulk-data.sql` | Insert 10,000 records ban đầu |
| `cmd/stream_insert/main.go` | Streaming insert 90 records/phút |
| `test-materialized-view.sql` | Test queries cho continuous aggregates |
| `docker-compose.yml` | TimescaleDB setup |
| `init-db/01-init.sql` | Database initialization + continuous aggregates |

---

## VI. NOTES

1. **Auto-refresh policy:**
   - Hourly: refresh mỗi 1 giờ
   - Daily: refresh mỗi 1 ngày
   - Data mới nhất có thể chưa có trong materialized view (lag)

2. **Manual refresh:**
   - Dùng `CALL refresh_continuous_aggregate()` để force update
   - Chỉ cần khi muốn xem data mới ngay lập tức

3. **Production:**
   - Auto-refresh policy đủ cho production
   - Không cần manual refresh
   - Nếu cần data real-time mới nhất → query raw table (có fallback logic trong Go code)

4. **Storage:**
   - Data được compress sau 7 ngày (compression policy)
   - Raw data có thể được xóa sau X năm (retention policy - đang comment)
   - Continuous aggregates giữ lại data aggregate vĩnh viễn

---

## VII. DEMO SCENARIOS

### Scenario 1: Quick test (5 phút)
1. Insert 10k records
2. Chạy streaming insert 5 phút
3. Monitor real-time
4. Manual refresh và xem kết quả

### Scenario 2: Performance test
1. Insert 10k records
2. Compare query performance RAW vs MATERIALIZED
3. Verify speedup 100-250x

### Scenario 3: Auto-refresh test
1. Insert 10k records
2. Chạy streaming insert
3. Monitor mỗi 30 giây trong 1 giờ
4. Xem hourly continuous aggregate tự động refresh

### Scenario 4: Long-term test (1 ngày)
1. Insert 10k records
2. Chạy streaming insert cả ngày
3. Xem daily continuous aggregate tự động refresh vào cuối ngày
