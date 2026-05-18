# Đặc tả: Lịch uống thuốc & nhắc giờ (HealthMate)

**Phiên bản tài liệu:** 1.0  
**Cập nhật:** 2026-03-28  
**Phạm vi:** Flutter FE + storage-service (medications / medication_reminders) + api-gateway.

> Trong codebase hiện **không** có module “lịch hẹn khám bác sĩ” riêng. Phần tương ứng “lịch hẹn” trong UI là **lịch nhắc uống thuốc trong ngày**. Nếu sau này thêm lịch khám, nên tách entity/API và cập nhật phạm vi tài liệu này.

---

## 1. Thuật ngữ & phạm vi

| Thuật ngữ UI / nói chuyện | Ý nghĩa trong hệ thống                                                                                                                |
| ------------------------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| Lịch hôm nay / lịch uống  | Danh sách **mốc nhắc** (`medication_reminders`) của các thuốc **đang active**, nhóm theo buổi trong ngày **theo giờ local thiết bị**. |
| Nhắc / lượt uống          | Một dòng trong lịch = một cặp (thuốc + `reminder` tại giờ `HH:mm`).                                                                   |
| Đã uống                   | `last_taken` của reminder được set (toggle theo quy tắc BE).                                                                          |
| Thêm vào lịch             | Tạo bản ghi `medications` + các `medication_reminders` (POST).                                                                        |
| Xóa khỏi lịch             | Xóa cả thuốc (cascade xóa reminders) — DELETE medication.                                                                             |

**Trong phạm vi:** quản lý thuốc + nhắc giờ + đánh dấu uống + quét đơn (OCR) tạo nhiều thuốc có nhắc.

**Ngoài phạm vi (chưa có / chưa đầy đủ trong code):** lịch tái khám độc lập; chỉnh sửa thuốc/reminder từng field (PATCH); thông báo đẩy (FCM) theo giờ; đồng bộ realtime; logic `weekly`/`monthly` thực sự lọc ngày trong lịch.

---

## 2. Actor & điều kiện

- Người dùng **đã đăng nhập** (JWT access).
- Mọi API `/medications` gắn `user_id` từ claim `sub`; **không** tin `user_id` do client tự gửi khi tạo bản ghi.
- Client gọi qua **api-gateway** (path public thường là `/medications`, …).

---

## 3. Mô hình dữ liệu

### 3.1 Bảng `medications`

- `id` (UUID), `user_id` (FK → `users`, ON DELETE CASCADE).
- `name`, `dosage`, `instructions`, `prescribed_by`, `is_active`.
- `frequency` (JSONB): object kiểu `{ type, times_per_day, specific_times[] }` — BE **lưu nguyên**, không validate sâu lịch theo tuần/tháng.
- `start_date`, `end_date` (DATE; `end_date` nullable).

### 3.2 Bảng `medication_reminders`

- `id` (UUID), `medication_id` (FK CASCADE).
- `time` — chuỗi `HH:mm` (VARCHAR(8)).
- `is_enabled` — FE **ẩn** reminder tắt khỏi lịch; BE vẫn lưu.
- `last_taken` (TIMESTAMPTZ, nullable) — phục vụ toggle “đã uống”.
- `missed_count` — **cột có nhưng chưa có logic cập nhật** (mặc định 0).

**Migration:** `storage-service/migration/000010_create_medications.up.sql`.

---

## 4. Quy tắc hiển thị “Lịch hôm nay” (FE)

**Nguồn:** `MedicationView`, hàm `_buildSchedule`.

1. Chỉ thuốc `is_active == true`.
2. Với mỗi thuốc, duyệt `reminders` có `is_enabled == true`.
3. **Phân buổi** theo **giờ local** (parse từ `time`):
   - **Sáng:** 6:00–11:59
   - **Trưa:** 12:00–16:59
   - **Chiều:** 17:00–20:59
   - **Tối:** 21:00–5:59 (gồm 0–5h sáng).
4. Trong mỗi buổi, sắp xếp theo `time` tăng dần (chuỗi `HH:mm` zero-pad).
5. **Đã uống hôm nay (FE):** `MedicationReminder.isTakenToday` so `last_taken` với **ngày lịch local** (`DateTime.now()`).
6. **Quá hạn / “Cần chú ý”:** chưa “đã uống” theo FE **và** `reminder.time < currentLocalTime` (`HH:mm`).

### 4.1 Lệch UTC vs local (quan trọng)

- BE khi `take` so sánh “cùng ngày” theo **UTC**.
- FE hiển thị “đã uống hôm nay” theo **local**.

→ Có thể **lệch một ngày** giữa UI và server ở một số múi giờ. Cần quyết định: timezone user (profile), hoặc thống nhất một chuẩn (ví dụ chỉ local end-to-end).

### 4.2 Chưa áp dụng trên FE

- Lọc theo `start_date` / `end_date` khi dựng lịch (thuốc có thể vẫn hiện nếu `is_active` và còn reminder).

---

## 5. Toggle “Đã uống” (BE + FE)

**Endpoint:** `POST /medications/:medicationId/reminders/:reminderId/take`  
**Header:** `Authorization: Bearer <access_token>`.

**Logic server (tóm tắt):**

- Kiểm tra thuốc tồn tại và `user_id` khớp; sai user → **403**.
- `now` = UTC.
- Nếu `last_taken` **cùng ngày lịch UTC** với `now` → set `last_taken = NULL` (huỷ đã uống).
- Ngược lại → set `last_taken = now` (UTC).

**Response:** `200`, body = **mảng** toàn bộ thuốc của user (giống GET).

**FE:** optimistic update rồi gọi API; lỗi thì revert + SnackBar.

**Lỗi:** `404` — không tìm thấy thuốc/reminder hoặc không thuộc user; `403` — thuốc user khác.

---

## 6. API HTTP (contract)

Base qua gateway (ví dụ): `http://<host>:8080/medications` … Gateway rewrite sang storage (`API_PREFIX`, ví dụ `/api/v1`).

| Method | Path                                                    | Mô tả                                                               |
| ------ | ------------------------------------------------------- | ------------------------------------------------------------------- |
| GET    | `/medications`                                          | Danh sách thuốc + reminders. Body: **array** JSON trực tiếp.        |
| POST   | `/medications`                                          | Tạo thuốc + reminders (server sinh UUID). `201` + object một thuốc. |
| POST   | `/medications/:medicationId/reminders/:reminderId/take` | Toggle đã uống. `200` + array toàn bộ thuốc.                        |
| DELETE | `/medications/:medicationId`                            | Xóa thuốc (và reminders theo CASCADE).                              |

**Validation POST create (BE):** bắt buộc `name`, `dosage`, `frequency` JSON hợp lệ, `start_date`. Nếu không suy ra giờ từ `reminders[]` hoặc `frequency.specific_times` → mặc định một reminder `08:00`. `id` / `reminders[].id` từ client **không** tin để ghi DB.

**Body POST create (mẫu):**

```json
{
  "name": "Paracetamol",
  "dosage": "500mg",
  "frequency": {
    "type": "daily",
    "times_per_day": 1,
    "specific_times": ["08:00"]
  },
  "start_date": "2026-03-28",
  "end_date": null,
  "instructions": "",
  "prescribed_by": "",
  "is_active": true,
  "reminders": [{ "time": "08:00", "is_enabled": true }]
}
```

**Định dạng phần tử response:** tương thích `Medication.fromJson` / `MEDICATIONS-API.md` (snake_case, `last_taken` RFC3339 hoặc null).

**Tài liệu BE chi tiết:** `HealthMate_BE/HealthMate/storage-service/MEDICATIONS-API.md`.

---

## 7. Luồng người dùng (màn Thuốc & lịch uống)

1. Vào màn: `FetchMedications` — loading “Đang tải lịch thuốc” / lỗi có retry.
2. Thống kê: Lượt uống / Đã uống / Cần chú ý (theo logic mục 4).
3. Quét đơn: OCR → chọn thuốc, giờ, “Thêm vào lịch uống” → `AddMedicationsBatch` (nhiều POST). Cần ít nhất một mục bật “Thêm vào lịch uống”.
4. Thêm thủ công: form tên, liều, tần suất, nhiều giờ, ngày → `AddMedication`.
5. Lịch hôm nay: theo buổi; tap → `TakeMedication`.
6. Chỉnh sửa: dialog **chỉ xóa thuốc** (không sửa giờ/tên tại chỗ) → `DeleteMedication`.
7. Kéo refresh: `FetchMedications`.
8. Disclaimer: nhắc không thay thế chỉ định bác sĩ/dược sĩ.

---

## 8. FE — kiến trúc liên quan

| Thành phần    | Đường dẫn gợi ý                                          |
| ------------- | -------------------------------------------------------- |
| Màn + lịch    | `lib/presentation/medications/view/medication_view.dart` |
| Bloc          | `lib/presentation/medications/bloc/medication_bloc.dart` |
| API           | `lib/data/services/api_medication_service.dart`          |
| Model         | `lib/data/models/medication/`                            |
| Thêm thủ công | `add_medication_dialog.dart`                             |
| Quản lý / xóa | `manage_medications_dialog.dart`                         |
| Quét đơn      | `prescription_scan_dialog.dart`                          |

**Sự kiện bloc:** `FetchMedications`, `TakeMedication`, `AddMedication`, `AddMedicationsBatch`, `DeleteMedication`, `ClearMedicationFeedback`.

---

## 9. BE — vị trí code

| Thành phần               | Đường dẫn                              |
| ------------------------ | -------------------------------------- |
| Handler / service / repo | `storage-service/internal/medication/` |
| Route                    | `storage-service/app/app.go`           |
| Proxy                    | `api-gateway` — nhóm `/medications`    |

---

## 10. Phi chức năng & bảo mật

- JWT access; claim `type == access`; `sub` = user id.
- `JWT_SECRET` và `API_PREFIX` đồng bộ gateway, auth-service, storage-service.
- Rate limit / audit: chưa có trong phạm vi hiện tại.

---

## 11. Hạng mục mở / cần quyết định

1. Thống nhất **UTC vs local** cho “cùng ngày” khi take.
2. **`missed_count`:** có dùng không, quy tắc tăng.
3. **`start_date` / `end_date`:** có ẩn khỏi lịch khi ngoài khoảng.
4. **`weekly` / `monthly` / `asNeeded`:** UI có nhưng lịch trong ngày không lọc theo loại.
5. **PATCH** thuốc/reminder; bật/tắt từng reminder trên server.
6. **Push notification** tại giờ nhắc.

---

## 12. Smoke test (curl)

Cần access token hợp lệ (`sub` = user có trong DB).

```bash
curl -sS -H "Authorization: Bearer $TOKEN" http://localhost:8080/medications

curl -sS -X POST http://localhost:8080/medications \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"name":"Test","dosage":"1 viên","frequency":{"type":"daily","times_per_day":1,"specific_times":["08:00"]},"start_date":"2026-03-28","reminders":[{"time":"08:00"}]}'
```

Sau đó gọi `take` và `DELETE` với `medicationId` / `reminderId` từ response.

---

_Tài liệu phản ánh implementation tại thời điểm soạn; khi đổi API hoặc UI, cập nhật file này cùng PR._
