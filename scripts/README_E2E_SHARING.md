# E2E: mock data + test quyền chia sẻ chỉ số

## Điều kiện

- Đã `docker compose up` trong `HealthMate_BE/HealthMate` (ít nhất: **timescaledb**, **auth-service**, **storage-service**, **api-gateway**; JWT_SECRET trùng `.env` gateway).
- DB đã chạy migration storage (có `metric_types` đủ: `heart_rate`, `steps_count`, `calories_burned`, `blood_pressure`, `spo2`, …).
- Port gateway mặc định: **5002**.

## Bước 1 — Nạp dữ liệu mẫu

Từ thư mục `HealthMate_BE/HealthMate`:

```powershell
Get-Content scripts\e2e_sharing_permissions_seed.sql -Raw | docker compose exec -T timescaledb psql -U postgres -d healthmate
```

(Linux/mac: `docker compose exec -T timescaledb psql -U postgres -d healthmate < scripts/e2e_sharing_permissions_seed.sql`)

**Seed (nhóm `1111…`, Alice owner):**

| User   | Vai trò trong nhóm | Quyền xem chỉ số Alice (tóm tắt) |
|--------|--------------------|-----------------------------------|
| Bob    | member, không hàng `shared_with` | Mọi **base**: HR, steps, calo, BP; **không** spo2 (không có base spo2) |
| Charlie| member + `access_control` + chỉ `steps_count` cụ thể | Chỉ **steps**; không HR / calo / BP / spo2 |
| Eve    | member + `access_control` + **HR + BP** cụ thể | Chỉ **HR, BP**; không steps / calo / spo2 |
| Frank  | member, không filter | Giống Bob (full base, không spo2) |
| Diana  | **không** trong nhóm | Mọi request Alice → 403 |

Có dữ liệu mẫu Timescale cho Alice: `heart_rates`, `steps_counts`, `calories_burnt`, `blood_pressure` (và `spo2` nếu bảng tồn tại).

## Bước 2 — Chạy test HTTP tự động

```powershell
pip install -r scripts\e2e_requirements.txt
python scripts\e2e_sharing_permissions_test.py
```

Biến môi trường tùy chọn:

- `E2E_GATEWAY_URL` (mặc định `http://127.0.0.1:5002`)
- `JWT_SECRET` (mặc định `healthmate_jwt_secret_dev` — bỏ dấu ngoặc nếu có trong `.env`)

### Kỳ vọng (tóm tắt)

- **Charts:** ~25 assertion (Bob/Charlie/Diana/Alice/Eve/Frank × nhiều metric; edge: `weight`, `calories_burnt`; Alice `weight` → 400).
- **Thuốc:** Alice list shares **200**; Bob / Charlie / Eve list **403**.
- **Auth — chỉ số ngoài Base nhóm:** Trong `auth-service`, “chỉ số được phép của nhóm” với chủ số liệu chính là tập **global** trong `sharing_permissions` (`shared_with_user_id IS NULL`). `UpdateSharing` / `EnableSharing` cho một thành viên cụ thể **không** được chứa metric chưa bật ở Base (ví dụ seed không có base `spo2`):
  - `PUT /groups/{id}/permissions` với `target_user_id` + danh sách có `spo2` → **400**
  - `POST /groups/{id}/permissions` bật `spo2` riêng cho thành viên khi chưa có global `spo2` → **403**

Các request này đi qua **gateway** tới **auth-service**; nếu chỉ chạy storage sẽ fail phần Auth.

## Gỡ seed (tùy chọn)

Xóa thủ công các UUID trong file SQL phần `DELETE` đầu file, hoặc chạy lại seed (transaction `BEGIN`/`COMMIT` đã xóa trước khi insert lại).

---

## Realtime (WebSocket + Kafka)

Test **cùng bộ seed** nhưng kiểm tra luồng: Alice push chỉ số qua WS → Kafka → `realtime-service` → hub gọi `GetMetricWatchers` → chỉ subscriber **được phép** mới nhận `{"type":"metric",...}`.

### Điều kiện

- `docker compose` có **zookeeper**, **kafka**, **timescaledb**, **realtime-service** (port WS mặc định **5001**).
- Đã nạp `e2e_sharing_permissions_seed.sql`.
- `JWT_SECRET` trùng `.env` của **realtime-service** (và gateway nếu dùng).

### Chạy

```powershell
pip install -r scripts\e2e_requirements.txt
python scripts\e2e_realtime_ws_test.py
```

Biến môi trường tùy chọn:

- `REALTIME_WS_URL` — mặc định `ws://127.0.0.1:5001/ws` (URL đầy đủ tới path `/ws`, script tự thêm `token=`).
- `JWT_SECRET` — như mục HTTP ở trên.

### Kỳ vọng (khớp seed)

1. **heart_rate:** Bob nhận; Charlie **không** nhận.
2. **steps_count:** Bob và Charlie **đều** nhận.
3. **blood_pressure:** Bob, Frank, Eve nhận; Charlie **không** nhận.
4. **calories_burned:** Bob và Frank nhận; Charlie và Eve **không** nhận.

Chi tiết luồng xem `realtime-service/README.md` (hub kiểm tra DB **mỗi lần** broadcast).

---

## Demo: mock data để xem chart / realtime

- **Time-series trong Timescale (Alice E2E):** `scripts/demo_realtime_timescale_seed.sql` — chèn ~48h dữ liệu `heart_rate`, `steps_count`, `calories_burnt` (và `blood_pressure` / `spo2` nếu DB đã có bảng + `metric_types`).
- **Đẩy metric sống qua WebSocket:** `scripts/demo_push_realtime_metrics.py` — `--mode publish` (Alice) / `--mode watch` (Bob + `--target-id` Alice). Xem docstring trong file.

```powershell
Get-Content scripts\demo_realtime_timescale_seed.sql -Raw | docker compose exec -T timescaledb psql -U postgres -d healthmate
python scripts\demo_push_realtime_metrics.py --help
```
