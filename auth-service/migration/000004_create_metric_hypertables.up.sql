-- 1. Bảng Nhịp tim (Heart Rates)
CREATE TABLE heart_rates (
    time TIMESTAMPTZ NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    value DOUBLE PRECISION NOT NULL
);
-- Kích hoạt TimescaleDB: Chuyển đổi thành Hypertable, phân mảnh theo cột 'time'
SELECT create_hypertable('heart_rates', 'time');


-- 2. Bảng Số bước chân (Step Counts)
CREATE TABLE steps_counts (
    time TIMESTAMPTZ NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    value DOUBLE PRECISION NOT NULL
);
-- Kích hoạt TimescaleDB
SELECT create_hypertable('steps_counts', 'time');


-- 3. Bảng Calo đã đốt (Calories Burned)
CREATE TABLE calories_burned (
    time TIMESTAMPTZ NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    value DOUBLE PRECISION NOT NULL
);
-- Kích hoạt TimescaleDB
SELECT create_hypertable('calories_burned', 'time');