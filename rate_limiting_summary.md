# Tóm tắt thay đổi: Thêm tính năng Rate Limiting

Tôi đã hoàn thành việc thêm tính năng rate limiting vào API Gateway sử dụng Redis. Dưới đây là tóm tắt các thay đổi:

## 1. Dependencies
- Đã thêm thư viện `github.com/redis/go-redis/v9` vào `go.mod`.

## 2. Cấu hình (Configuration)
- **`api-gateway/.env`**: Thêm các biến môi trường:
  - `REDIS_HOST="redis"` (tên service trong docker-compose)
  - `REDIS_PORT="6379"`
  - `RATE_LIMIT_LIMIT="100"`
  - `RATE_LIMIT_WINDOW="60"`
- **`api-gateway/config/config.go`**: Cập nhật struct `Config` và hàm `LoadConfig` để đọc các biến trên.

## 3. Khởi tạo Redis
- **`api-gateway/app/app.go`**: Thêm field `RedisClient` vào struct `App` và khởi tạo kết nối trong `NewApp`.

## 4. Middleware Rate Limiting
- **`api-gateway/internal/middleware/ratelimit.go`**: Tạo mới file này.
  - Sử dụng thuật toán **Fixed Window** với lệnh `INCR` và `EXPIRE` của Redis.
  - **Xác định Client**:
    1. Ưu tiên `user_id` (lấy từ claim `sub` sau khi qua JWT middleware).
    2. Nếu không có, tìm trong header `X-API-Key`.
    3. Nếu không có, dùng Client IP.
  - **Headers trả về**:
    - `X-RateLimit-Limit`: Giới hạn tối đa.
    - `X-RateLimit-Remaining`: Số request còn lại.
    - `Retry-After`: Thời gian chờ (giây) trước khi thử lại (khi bị block).
  - **HTTP Status**: Trả về `429 Too Many Requests` khi vượt giới hạn.
  - **Fail-open**: Nếu Redis gặp sự cố, middleware sẽ cho qua (Next) để không làm gián đoạn dịch vụ.

## 5. Áp dụng Middleware
- **`api-gateway/app/router.go`**:
  - Áp dụng cho nhóm route `/auth` (public): Sử dụng Rate Limit trước (sẽ nhận diện theo API Key hoặc IP).
  - Áp dụng cho nhóm route `protected` (đã qua JWT): Sử dụng Rate Limit sau JWT middleware (sẽ nhận diện theo User ID `sub`).

## Kết quả
- Đã chạy `go build ./...` thành công, không có lỗi compile.

Bạn có thể review trực tiếp các file trên. Nếu cần điều chỉnh gì, hãy báo cho tôi biết!
