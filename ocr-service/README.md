# OCR Service (Self-hosted)

This service provides OCR API for Vietnamese prescriptions and is intended to be called by `storage-service`.

## Endpoint

- `POST /ocr/prescriptions/parse`
  - multipart form-data: `file`
  - response:
    - `raw_text`: OCR text
    - `items`: parsed medication list

## Local run (without Docker)

```bash
pip install -r requirements.txt
uvicorn app.main:app --host 0.0.0.0 --port 8010
```

## Docker run

Use the root `docker-compose.yml`, service name: `ocr-service`.

## Optional free fallback OCR

You can keep PaddleOCR as primary and use free-tier OCR only when output quality is low.

- HTTP fallback (custom OCR API):
  - `OCR_FALLBACK_HTTP_URL=http://your-ocr-service/ocr/prescriptions/parse`
- OCR.Space free-tier fallback:
  - `OCR_FALLBACK_MODE=ocr_space`
  - `OCR_SPACE_API_KEY=helloworld` (or your own key)

Notes:
- Primary OCR is still self-hosted PaddleOCR.
- Fallback is invoked only for weak OCR results (`short text` / `low score`).
