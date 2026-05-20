# OCR Service (HealthMate)

Dịch vụ OCR **tự host** cho đơn thuốc (FastAPI + PaddleOCR): nhận ảnh, trích chữ, parse danh sách thuốc gợi ý. **Không** thay thế tư vấn y tế; chỉ hỗ trợ nhập liệu. Client gọi qua **API Gateway** (`/ocr/...`, có JWT), không bắt buộc qua `storage-service`.

**English (one-liner):** `POST /ocr/prescriptions/parse` — multipart `file` → JSON with `raw_text`, `items`, `warnings`, `meta`.

---

## Chạy nhanh

**Local (thư mục `ocr-service`):**

```bash
pip install -r requirements.txt
uvicorn app.main:app --host 0.0.0.0 --port 8010
```

**Docker Compose** (từ thư mục `HealthMate`):

```bash
docker compose up -d ocr-service api-gateway
```

Đặt `OCR_HTTP_URL` của gateway trỏ đúng host/port OCR (xem `api-gateway/.env.example`).

---

## 1. Kiến trúc tổng quan

```
Client (FE)
    → HTTPS / HTTP tới Nginx (8080) hoặc trực tiếp API Gateway (5002)
        → API Gateway (Gin) — route có JWT
            → Reverse proxy tới ocr-service
                → FastAPI + PaddleOCR (cổng 8010 trong Docker)
```

- **ocr-service**: xử lý ảnh, OCR, parse cấu trúc đơn → JSON.
- **api-gateway**: không implement logic OCR; chỉ **proxy** có bảo vệ **Bearer JWT** (cùng nhóm route với `/users`, `/medications`, …).
- **storage-service**: có thể cấu hình `OCR_HTTP_URL` trong `.env` cho tích hợp khác; **luồng quét đơn chính** trên FE gọi qua gateway (`/ocr/...`).

---

## 2. Stack & endpoint

| Thành phần | Mô tả |
|------------|--------|
| Runtime | Python 3.11, `uvicorn` |
| Engine OCR chính | **PaddleOCR** (`lang="en"`, `use_angle_cls=True`) — phù hợp tên thuốc Latin trên đơn VN |
| Xử lý ảnh | OpenCV: tiền xử lý, deskew, nhiều pass (enhanced / deskew / adaptive threshold / blur), chọn pass theo chất lượng |
| Parse văn bản | Regex + heuristics: marker số thứ tự, gom dòng thuốc, chuẩn hóa tiếng Việt sau OCR, dedupe, khôi phục dòng khi parser thiếu |

| Method | Path | Body |
|--------|------|------|
| `POST` | `/ocr/prescriptions/parse` | `multipart/form-data`, field **`file`** (ảnh đơn) |

**Dockerfile**: `python:3.11-slim`, dependency hệ thống cho OpenCV/Paddle, `uvicorn app.main:app --port 8010`.

**Dependencies** (`requirements.txt`): `fastapi`, `uvicorn`, `paddleocr`, `paddlepaddle`, `opencv-python-headless`, `numpy`.

---

## 3. Hành vi xử lý (trong `app/main.py`)

1. **Tiền xử lý** (`_preprocess_image`): có thể **crop phần thân đơn** (bỏ đầu giấy dọc); có thể tắt crop khi retry.
2. **OCR đa pass** + merge dòng từ các pass phụ khi thiếu dòng thuốc.
3. **Fallback** (tùy env): text ngắn / điểm thấp → HTTP URL ngoài hoặc OCR.Space (mục 6).
4. **Chuẩn hóa** (`_normalize_ocr_vi`), marker đánh số, parse **`items`**.
5. **Giới hạn số thuốc** khi chuỗi marker tin cậy.
6. **Retry full ảnh** (không crop) nếu parser ra quá ít item.
7. **`warnings`**, **`meta`** (engine, `ocr_score`, `pass_used`, fallback…).

---

## 4. API Gateway — proxy & JWT

File `../api-gateway/app/router.go`:

```text
protected.Any("/ocr/*proxyPath", handlers.ReverseProxy(cfg.OCRHTTPURL, "/ocr"))
```

Client gọi:

```text
POST {API_GATEWAY_BASE}/ocr/prescriptions/parse
```

→ proxy tới `{OCR_HTTP_URL}/ocr/prescriptions/parse` (Docker: `http://ocr-service:8010/ocr/prescriptions/parse`).

Header: **`Authorization: Bearer <access_token>`**.

---

## 5. JSON phản hồi (đại diện)

```json
{
  "raw_text": "…",
  "items": [{ "name": "…", "dosage": "…", "confidence": 0.0 }],
  "warnings": ["Cần kiểm tra lại đơn thuốc"],
  "meta": {
    "engine": "paddleocr",
    "version": "2.5",
    "items_count": 0,
    "avg_confidence": 0.0,
    "ocr_score": 0.0,
    "pass_used": "…",
    "used_fallback": false,
    "fallback_stage": ""
  }
}
```

---

## 6. Biến môi trường (ocr-service)

| Biến | Ý nghĩa |
|------|---------|
| `OCR_FALLBACK_HTTP_URL` | URL OCR HTTP khác (custom) khi Paddle yếu |
| `OCR_FALLBACK_MODE` | `ocr_space` / `mistral_ocr` / để trống (không fallback cloud) |
| `OCR_SPACE_API_KEY` | Key OCR.Space (có thể `helloworld` để thử) |
| `MISTRAL_API_KEY` | API key cho Mistral OCR fallback |
| `OCR_MISTRAL_MODEL` | Model Mistral OCR (mặc định `mistral-ocr-latest`) |
| `OCR_MISTRAL_TIMEOUT_SEC` | Timeout gọi Mistral OCR (mặc định `45`) |
| `OCR_ENABLE_SYNTHETIC_HINT_ROWS` | Gợi ý dòng tổng hợp (khi bật) |
| `OCR_BODY_TOP_SKIP_FRAC` | Tỷ lệ cắt đầu ảnh dọc (~0.18 mặc định trong code) |
| `OCR_DRUG_DICT_ENABLED` | Bật/tắt chuẩn hóa tên thuốc theo từ điển (mặc định `true`) |
| `OCR_DRUG_DICT_CSV_PATH` | Đường dẫn file CSV danh mục thuốc chuẩn (khuyên dùng dữ liệu Cục Quản lý Dược) |
| `OCR_DRUG_DICT_FUZZY_CUTOFF` | Ngưỡng fuzzy "chắc chắn" để tự chuẩn hóa tên thuốc (mặc định `0.88`) |
| `OCR_DRUG_DICT_WEAK_CUTOFF` | Ngưỡng fuzzy "gần đúng nhưng chưa chắc" để phát cảnh báo (mặc định `0.80`) |

**Ghi chú:** PaddleOCR vẫn là **primary**; fallback chỉ khi kết quả yếu (text ngắn / điểm thấp). Trong `docker-compose.yml` thường để trống fallback để chỉ self-hosted.

### Chuẩn hóa tên thuốc theo nguồn chính phủ Việt Nam (khuyến nghị)

- Nguồn ưu tiên: cổng công bố thuốc của **Cục Quản lý Dược - Bộ Y tế**.
- Export dữ liệu về CSV (chứa cột tên thuốc), mount vào container và set:

```env
OCR_DRUG_DICT_ENABLED=true
OCR_DRUG_DICT_CSV_PATH=/app/data/drugs_moh.csv
OCR_DRUG_DICT_FUZZY_CUTOFF=0.88
OCR_DRUG_DICT_WEAK_CUTOFF=0.80
```

- Service sẽ fuzzy-match tên OCR về tên chuẩn trong CSV khi độ khớp đủ cao.
- Nếu chỉ "gần đúng" hoặc không có trong danh mục, API thêm warning để FE yêu cầu người dùng kiểm tra thủ công.

### Ví dụ cấu hình fallback Mistral OCR

```env
OCR_FALLBACK_MODE=mistral_ocr
MISTRAL_API_KEY=your_real_key_here
OCR_MISTRAL_MODEL=mistral-ocr-latest
OCR_MISTRAL_TIMEOUT_SEC=45
```

- Chỉ dùng fallback khi OCR chính yếu (text ngắn / điểm thấp hoặc parse ra 0 item).
- Nếu không set `MISTRAL_API_KEY`, mode `mistral_ocr` sẽ tự bỏ qua fallback.

---

## 7. Docker Compose & Nginx

- Service **`ocr-service`**: build `./ocr-service`, port **`8010:8010`**, network `backend`; `api-gateway` `depends_on` có `ocr-service`.
- **`nginx/nginx.conf`**: `location /` → API Gateway; path `/ocr/prescriptions/parse` đi chung luồng. Có thể cần `client_max_body_size` và `proxy_read_timeout` đủ lớn cho ảnh và lần load model đầu.

---

## 8. Frontend (tham chiếu)

```text
POST {base}/ocr/prescriptions/parse
```

Multipart `file` + Bearer — FE: `ApiOcrService`, `ApiEndpoints.ocrPrescriptionParse`.

---

## 9. Hạn chế

- Phụ thuộc chất lượng ảnh (sáng, nét, không che chữ).
- Fallback bên thứ ba (OCR.Space / HTTP) — cân nhắc bảo mật trước khi bật.
- Container OCR **tốn RAM**; cần giới hạn tài nguyên phù hợp.

---

## 10. Hướng dẫn test (unit test trong repo này)

Nguồn đồng bộ từ repo R&D `HealthMate_OCR` (cùng `app/prescription_parser.py`).

### 10.1. Pytest — parser (50 test, không cần Paddle/Docker)

```powershell
cd ocr-service
python -m venv .venv
.\.venv\Scripts\Activate.ps1
pip install -r requirements-dev.txt
python -m pytest
```

Chụp màn hình báo cáo:

```powershell
.\scripts\run_unit_tests.ps1
```

Linux/macOS: `./scripts/run_unit_tests.sh`

**Kỳ vọng:** `50 passed in ~0.1s`

| Nhóm | Số case |
|------|---------|
| `TestStripMarkdownFence` | 7 |
| `TestVlJsonToItems` | 13 |
| `TestParsePrescriptionText` | 25 |
| `test_real_world_samples` | 5 |

`tests/` và `scripts/run_unit_tests.*` **không** đưa vào image Docker (`.dockerignore`).

### 10.2. Smoke API — Docker (OCR thật)

```bash
docker build -t healthmate-ocr:local .
docker run --rm -p 8010:8010 \
  -e OCR_POOL_SIZE=1 \
  -e OCR_MAX_CONCURRENT_REQUESTS=1 \
  -e OCR_EARLY_EXIT_QUALITY=99.0 \
  -e OCR_DRUG_DICT_ENABLED=true \
  -e OCR_DRUG_DICT_CSV_PATH=/app/data/drugs_moh.csv \
  healthmate-ocr:local
```

```bash
curl http://localhost:8010/health
curl -X POST http://localhost:8010/ocr/prescriptions/parse -F "file=@don-thuoc.jpg"
```

Qua gateway (JWT): `POST {GATEWAY}/ocr/prescriptions/parse` — mục 4.

Benchmark F1 + dataset 66 ảnh: repo `HealthMate_OCR` (`scripts/benchmark.py`).

---

*Tài liệu căn cứ `app/main.py`, `app/prescription_parser.py`, `tests/test_prescription_parser.py`, `../api-gateway/app/router.go`.*
