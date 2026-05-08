"""
Ẩn/mask thông tin nhạy cảm trước khi gửi ảnh đơn thuốc ra dịch vụ OCR bên thứ ba,
và làm sạch phần text trả về để giảm rò rỉ PII sau bước fallback.

PaddleOCR chạy local không qua module này; chỉ dùng trong app/fallback.py.
"""
from __future__ import annotations

import os
import re
from typing import Final

import cv2
import numpy as np

_FLOAT_ENV = re.compile(r"^\d+(\.\d+)?$")


def _frac(name: str, default: float) -> float:
    raw = os.getenv(name, "").strip()
    if not raw or not _FLOAT_ENV.match(raw):
        return default
    val = float(raw)
    return max(0.0, min(0.45, val))


def redact_image_bytes_for_external(raw: bytes) -> bytes:
    """
    Che vùng đầu/cuối ảnh (thường chứa họ tên, địa chỉ, chữ ký, số liên hệ).
    Ảnh lỗi decode thì trả nguyên bytes (không chặn OCR).
    """
    if not raw:
        return raw
    enabled = os.getenv("OCR_EXTERNAL_REDACT_ENABLED", "true").strip().lower() in {
        "1",
        "true",
        "yes",
        "on",
    }
    if not enabled:
        return raw

    top = _frac("OCR_EXTERNAL_REDACT_TOP_FRAC", 0.22)
    bottom = _frac("OCR_EXTERNAL_REDACT_BOTTOM_FRAC", 0.12)
    sides = _frac("OCR_EXTERNAL_REDACT_SIDES_FRAC", 0.02)

    arr = np.frombuffer(raw, dtype=np.uint8)
    img = cv2.imdecode(arr, cv2.IMREAD_COLOR)
    if img is None:
        return raw

    h, w = img.shape[:2]
    out = img.copy()

    if top > 0:
        y = max(1, int(h * top))
        out[0:y, :] = 0
    if bottom > 0:
        y = max(1, int(h * bottom))
        out[h - y : h, :] = 0
    if sides > 0:
        x = max(1, int(w * sides))
        out[:, 0:x] = 0
        out[:, w - x : w] = 0

    ok, buf = cv2.imencode(".png", out)
    if not ok:
        return raw
    return buf.tobytes()


# Dòng có khả năng cao là PII trên đơn (tiếng Việt không dấu / có dấu OCR)
_PII_LINE_HINT: Final = re.compile(
    r"(?i)^\s*(?:"
    r"h[ọo]\s*t[êe]n|h[oọ]\s+v[aă]|\bhoten\b|"
    r"đ[ịi]a\s*ch[ỉi]|dia\s*chi|"
    r"ng[aà]y\s*sinh|tu[oổ]i|dob|"
    r"c[ăa]n\s*c[ưu][ớo]c|cccd|cmnd|"
    r"bhyt|b[aả]o\s*hi[eể]m|m[aã]\s*(?:th[eẻ]|bh)|s[oố]\s*(?:the|bh)|"
    r"đi[eệ]n\s*tho[aạ]i|s[oố]\s*đt|mobile|tel|"
    r"email|e-mail|@"
    r")\b"
)


# Số điện thoại VN / quốc tế rút gọn
_PHONE: Final = re.compile(
    r"(?:(?:\+?84|0)(?:3|5|7|8|9|28)\d{8})|(?:\b0\d{9,10}\b)"
)

# Email
_EMAIL: Final = re.compile(
    r"[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}",
    re.I,
)

# Chuỗi số dài (CCCD 12 số, một số mã thẻ)
_LONG_DIGITS: Final = re.compile(r"\b\d{10,}\b")


def scrub_external_http_json_payload(payload: dict) -> dict:
    """
    Chỉ redact các field rõ ràng là thông tin bệnh nhân (tránh xóa nhầm name thuốc).
    """
    patient_keys = {
        "patient_name",
        "patient",
        "patient_full_name",
        "benh_nhan",
        "full_name_patient",
        "dia_chi",
        "address",
        "phone",
        "so_dien_thoai",
        "email",
        "id_number",
        "cmnd",
        "cccd",
        "bhyt",
        "insurance",
        "insurance_id",
    }
    out = dict(payload)
    for k in list(out.keys()):
        lk = str(k).lower()
        if lk in patient_keys:
            out[k] = "[REDACTED]"
        if lk in {"raw_text", "text"} and isinstance(out.get(k), str):
            out[k] = scrub_external_ocr_text(str(out[k]))
    return out


def scrub_external_ocr_text(text: str) -> str:
    if not text:
        return text
    lines_out: list[str] = []
    for ln in text.splitlines():
        s = ln.strip()
        if not s:
            continue
        if _PII_LINE_HINT.search(s):
            lines_out.append("[Đã che thông tin cá nhân]")
            continue
        s = _EMAIL.sub("[EMAIL_REDACTED]", s)
        s = _PHONE.sub("[PHONE_REDACTED]", s)
        s = _LONG_DIGITS.sub("[ID_REDACTED]", s)
        lines_out.append(s)
    return "\n".join(lines_out).strip()
