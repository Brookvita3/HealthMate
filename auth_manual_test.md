# Hướng dẫn Kiểm thử Thủ công Auth Service & Realtime Integration

Tài liệu này cung cấp các kịch bản kiểm thử (test cases) cho các tính năng xác thực (Authentication) và tích hợp với Realtime Service.

## 1. Môi trường (Pre-requisites)
- **Cài đặt công cụ**: Postman, Insomnia hoặc `curl`.
- **Base URL (Auth Service)**: `http://localhost:5000/api/v1` (Mặc định).
- **Base URL (Realtime Service)**: `ws://localhost:5001`.
- **Cơ sở dữ liệu**: Đảm bảo Postgres và Redis đang chạy.

---

## 2. Các Kịch bản Kiểm thử (Test Cases)

### TC-01: Đăng ký tài khoản mới (Email/Password)
- **Endpoint**: `POST /auth/register`
- **Body (JSON)**:
  ```json
  {
    "email": "test-user@example.com",
    "password": "password123",
    "name": "Test User"
  }
  ```
- **Kết quả mong đợi**:
  - HTTP Status: `201 Created`
  - Body chứa thông tin user và thông báo yêu cầu xác thực OTP gửi qua email.
  - Lưu ý: Vì đang test local, kiểm tra log của `auth-service` để lấy mã OTP nếu chưa cấu hình Gmail thực tế.

### TC-02: Xác thực tài khoản với OTP (Verify Account)
- **Endpoint**: `POST /auth/otp/verify`
- **Body (JSON)**:
  ```json
  {
    "email": "test-user@example.com",
    "otp": "123456" 
  }
  ```
- **Kết quả mong đợi**:
  - HTTP Status: `200 OK`
  - Body trả về `access_token` và `refresh_token`.
  - Account status chuyển thành `active` trong DB.

### TC-03: Đăng nhập (App Login)
- **Endpoint**: `POST /auth/app`
- **Body (JSON)**:
  ```json
  {
    "email": "test-user@example.com",
    "password": "password123"
  }
  ```
- **Kết quả mong đợi**:
  - HTTP Status: `200 OK`
  - Body trả về `access_token` (JWT) và `refresh_token`.

### TC-04: Làm mới Token (Refresh Token)
- **Endpoint**: `POST /auth/refresh`
- **Body (JSON)**:
  ```json
  {
    "refresh_token": "<REFRESH_TOKEN_FROM_LOGIN>"
  }
  ```
- **Kết quả mong đợi**:
  - HTTP Status: `200 OK`
  - Trả về `access_token` mới.

### TC-05: Đổi/Thiết lập mật khẩu
- **Endpoint**: `POST /auth/password`
- **Header**: `Authorization: Bearer <ACCESS_TOKEN>`
- **Body (JSON)**:
  ```json
  {
    "password": "new-strong-password"
  }
  ```
- **Kết quả mong đợi**:
  - HTTP Status: `200 OK`
  - Thử đăng nhập lại với mật khẩu mới thành công.

### TC-06: Đăng xuất (Logout)
- **Endpoint**: `POST /auth/logout`
- **Header**: `Authorization: Bearer <ACCESS_TOKEN>`
- **Body (JSON)**:
  ```json
  {
    "refresh_token": "<REFRESH_TOKEN>"
  }
  ```
- **Kết quả mong đợi**:
  - HTTP Status: `200 OK`
  - Thử dùng `refresh_token` cũ để `/refresh` sẽ bị reject (401).

---

## 3. Tích hợp với Realtime Service

Đây là kịch bản kiểm tra xem token từ Auth Service có hoạt động với Realtime Service hay không.

### TC-07: Kết nối WebSocket với JWT
- **URL**: `ws://localhost:5001?token=<ACCESS_TOKEN>`
- **Công cụ**: Sử dụng `wscat` hoặc Browser Console:
  ```javascript
  const ws = new WebSocket("ws://localhost:5001?token=YOUR_JWT_TOKEN");
  ws.onopen = () => console.log("Connected!");
  ws.onerror = (e) => console.error("Failed:", e);
  ```
- **Kết quả mong đợi**:
  - Kết nối thành công (Status 101).
  - Nếu token sai/hết hạn: Server trả về 401 Unauthorized hoặc đóng kết nối ngay lập tức.

### TC-08: Subscribe & Unsubscribe Metrics
- **Thực hiện**: Sau khi kết nối thành công ở TC-07, gửi message subscribe:
  ```json
  {
    "action": "subscribe",
    "items": [
      { "target_user_id": "<UUID_CỦA_BẠN>", "metric_type": "heart_rate" }
    ]
  }
  ```
- **Kết quả mong đợi**: Server trả về thành công.
- **Tiếp theo**: Gửi message unsubscribe:
  ```json
  {
    "action": "unsubscribe",
    "items": [
      { "target_user_id": "<UUID_CỦA_BẠN>", "metric_type": "heart_rate" }
    ]
  }
  ```
- **Kết quả mong đợi**: Server ngừng gửi dữ liệu cho metric đó.

---

## 4. Các trường hợp lỗi cần lưu ý (Negative Testing)
1. **Invalid Token**: Truy cập WS với token linh tinh -> Mong đợi 401.
2. **Expired Token**: Chờ token hết hạn rồi gọi `/refresh` -> Mong đợi thành công. Dùng access token hết hạn gọi WS -> Mong đợi 401.
3. **Wrong OTP**: Nhập sai OTP 3 lần -> Kiểm tra logic khóa hoặc báo lỗi.
4. **Duplicate Email**: Đăng ký với email đã tồn tại -> Mong đợi 409 Conflict.
