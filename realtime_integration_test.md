# Toàn diện Kịch bản Kiểm thử Realtime Service (WebSocket)

Tài liệu này mô tả chi tiết luồng vận hành của Realtime Service, từ lúc kết nối, đăng ký theo dõi (subscribe) đến khi nhận dữ liệu thực tế.

---

## 1. Luồng Kết nối & Xác thực (Handshake)

Realtime Service yêu cầu xác thực qua JWT. Token phải được gửi qua URL query parameter khi khởi tạo kết nối WebSocket.

- **URL**: `ws://localhost:5005/ws?token={ACCESS_TOKEN}`
- **Xác thực**: Backend thực hiện validate JWT. Nếu token không hợp lệ hoặc hết hạn, kết nối sẽ bị đóng ngay lập tức (Status code 401).

---

## 2. Luồng Điều khiển (Subscribe/Unsubscribe)

Sau khi kết nối thành công, Client (Người theo dõi - ví dụ: User A) gửi các lệnh điều khiển để đăng ký theo dõi các bộ chỉ số của người dùng khác (Người được theo dõi - ví dụ: User B).

### 2.1 Đăng ký theo dõi (Subscribe)
Client gửi message dạng JSON:
```json
{
  "action": "subscribe",
  "items": [
    {
      "target_user_id": "UUID_NGƯỜI_ĐƯỢC_THEO_DÕI",
      "metric_type": "heart_rate"
    }
  ]
}
```

**Hành động của Server đối với luồng đăng ký:**
1. Server duyệt qua từng item trong mảng `items`.
2. Kiểm tra quyền truy cập của `viewer_id` (người đang kết nối) đối với `target_user_id` và `metric_type`.
3. Nếu không có quyền hoặc có lỗi xảy ra: Server trả về một message lỗi riêng biệt cho item đó và đánh dấu request có lỗi.
4. Nếu hợp lệ: Cập nhật danh sách quan sát trong bộ nhớ (Hub) và cache quyền.
5. **Chỉ khi tất cả các items đều thành công (không có item nào bị lỗi):** Server mới trả về một message "Subscribe success".

**Phản hồi từ Server:**

*Thành công toàn bộ:*
```json
{
  "type": "success",
  "payload": "Subscribe success"
}
```

*Lỗi (Ví dụ không có quyền, message này trả về cho từng item bị lỗi và DO NOT trả về success chung):*
```json
{
  "type": "error",
  "payload": "No permission for UUID_NGƯỜI_ĐƯỢC_THEO_DÕI/heart_rate"
}
```

### 2.2 Hủy theo dõi (Unsubscribe)
```json
{
  "action": "unsubscribe",
  "items": [
    {
      "target_user_id": "UUID_NGƯỜI_ĐƯỢC_THEO_DÕI",
      "metric_type": "heart_rate"
    }
  ]
}
```

---

## 3. Luồng Dữ liệu Realtime (Data Streaming)

Có hai cách để dữ liệu được đẩy tới người quan sát:

### Luồng A: Đẩy dữ liệu trực tiếp từ thiết bị qua WebSocket
Thiết bị của người được theo dõi kết nối tới Realtime Service và đẩy dữ liệu trực tiếp:

**Message từ thiết bị (Người được theo dõi đẩy lên):**
```json
{
  "user_id": "UUID_NGƯỜI_ĐƯỢC_THEO_DÕI",
  "metric_type": "heart_rate",
  "value": 75.5,
  "timestamp": "2026-03-11T09:40:00Z"
}
```
*Lưu ý: `user_id` bắt buộc phải khớp với ID trong Token của chính kết nối đẩy dữ liệu đó (Security check). Nếu không khớp, server trả về lỗi `Forbidden: You can only push data for your own UserID`.*

**Hành động của Server:**
1. Nhận dữ liệu → đẩy vào Kafka topic `health_metrics`.
2. Kafka Consumer của Realtime Service nhận lại dữ liệu từ Kafka.
3. Broadcast tới tất cả các WebSocket Client đang subscribe `UUID_NGƯỜI_ĐƯỢC_THEO_DÕI` với type đó và check lại quyền (có cache tại client state).

### Luồng B: Dữ liệu từ các nguồn khác (API, Batch, Third-party)
Dữ liệu được đẩy vào Kafka thông qua `storage-service` hoặc bất kỳ phương thức nào khác vào chung topic `health_metrics`. Realtime Service Kafka Consumer luôn lắng nghe và thực hiện bước Broadcast tương tự như trên.

---

## 4. Nội dung Message Dữ liệu Nhận được mẫu

Khi có dữ liệu mới, Client (Người theo dõi) sẽ nhận được message trực tiếp là raw model của `HealthMetric` (KHÔNG bọc trong format `"type": "metric"` như message điều khiển).

**Cấu trúc message broadcast (Nhận bởi Observer):**
```json
{
  "user_id": "UUID_NGƯỜI_ĐƯỢC_THEO_DÕI",
  "metric_type": "heart_rate",
  "value": 75.5,
  "timestamp": "2026-03-11T09:40:00Z"
}
```

---

## 5. Kịch bản Kiểm thử Từng bước (Test Cases)

### TC-RT-01: Full Flow - Từ đầu đến cuối
1. **User A** (Người theo dõi) kết nối tới `ws://localhost:5005/ws?token=TOKEN_A`.
2. **User A** gửi subscribe **User B** (Người được theo dõi) cho metric `heart_rate`.
3. Kiểm tra User A nhận được `{"type": "success", "payload": "Subscribe success"}`.
4. **User B** kết nối WebSocket bằng `TOKEN_B` và gửi message metric `heart_rate` lên WebSocket của mình.
5. **Kỳ vọng**: User A nhận ngay lập tức message dữ liệu có `user_id` của User B và `value` tương ứng, không bị bọc thêm `payload`.

### TC-RT-02: Security Check
1. **User B** cố tình gửi message metric lên WebSocket của B nhưng điền `user_id` là của **User C**.
2. **Kỳ vọng**: Server phản hồi lỗi qua client của B: `{"type": "error", "payload": "Forbidden: You can only push data for your own UserID"}`.

### TC-RT-03: Permission Check & Bug fix validation
1. **User A** subscribe `steps_count` của **User C** (Nhưng C chưa cấp quyền này hoặc không tồn tại).
2. **Kỳ vọng**: Server trả về `{"type": "error", "payload": "No permission for UUID_C/steps_count"}` và **KHÔNG** trả về message `Subscribe success`.

---

## 6. SQL Check (Gỡ lỗi khi không nhận được dữ liệu)

Nếu User A đã subscribe thành công nhưng không thấy dữ liệu nhảy về, hãy kiểm tra quyền trong DB:

```sql
-- Kiểm tra quyền xem cụ thể
SELECT 
    gm_observer.user_id as observer_id,
    gm_target.user_id as target_id,
    sp.metric_type,
    g.name as group_name
FROM group_members gm_observer
JOIN group_members gm_target ON gm_observer.group_id = gm_target.group_id
JOIN groups g ON g.id = gm_observer.group_id
JOIN sharing_permissions sp ON sp.group_id = gm_target.group_id AND sp.user_id = gm_target.user_id
WHERE 
    gm_observer.user_id = 'ID_CỦA_A' 
    AND gm_target.user_id = 'ID_CỦA_B'
    AND gm_observer.status = 'accepted'
    AND gm_target.status = 'accepted'
    AND sp.metric_type = 'heart_rate';
```
