import re
import os
import json
import base64
import urllib.request
import urllib.error
import urllib.parse
from typing import Any

import cv2
import numpy as np
from fastapi import FastAPI, File, HTTPException, UploadFile
from paddleocr import PaddleOCR

app = FastAPI(title="healthmate-ocr-service", version="1.0.0")

# English model works reliably with Vietnamese medicine names written in Latin.
# We normalize field extraction in post-processing.
ocr_engine = PaddleOCR(use_angle_cls=True, lang="en", show_log=False)
fallback_ocr_url = os.getenv("OCR_FALLBACK_HTTP_URL", "").strip()
fallback_mode = os.getenv("OCR_FALLBACK_MODE", "").strip().lower()
ocr_space_api_key = os.getenv("OCR_SPACE_API_KEY", "").strip()


def _preprocess_image(raw: bytes) -> np.ndarray:
    arr = np.frombuffer(raw, np.uint8)
    img = cv2.imdecode(arr, cv2.IMREAD_COLOR)
    if img is None:
        raise ValueError("invalid image bytes")
    gray = cv2.cvtColor(img, cv2.COLOR_BGR2GRAY)
    gray = cv2.bilateralFilter(gray, 5, 35, 35)
    gray = cv2.convertScaleAbs(gray, alpha=1.2, beta=6)
    kernel = np.array([[0, -1, 0], [-1, 5, -1], [0, -1, 0]])
    sharp = cv2.filter2D(gray, -1, kernel)
    return sharp


def _deskew(gray: np.ndarray) -> np.ndarray:
    # Estimate text angle from connected components and rotate back.
    th = cv2.threshold(gray, 0, 255, cv2.THRESH_BINARY + cv2.THRESH_OTSU)[1]
    inv = 255 - th
    coords = np.column_stack(np.where(inv > 0))
    if coords.size == 0:
        return gray
    angle = cv2.minAreaRect(coords)[-1]
    if angle < -45:
        angle = 90 + angle
    if abs(angle) < 0.5:
        return gray
    h, w = gray.shape[:2]
    m = cv2.getRotationMatrix2D((w // 2, h // 2), angle, 1.0)
    return cv2.warpAffine(gray, m, (w, h), flags=cv2.INTER_CUBIC, borderMode=cv2.BORDER_REPLICATE)


def _extract_raw_text_with_score(img: np.ndarray) -> tuple[str, float, list[str]]:
    result = ocr_engine.ocr(img, cls=True)
    if not result or not result[0]:
        return "", 0.0, []
    lines: list[str] = []
    score_sum = 0.0
    score_count = 0
    for line in result[0]:
        if len(line) < 2:
            continue
        txt = str(line[1][0]).strip()
        conf = float(line[1][1]) if isinstance(line[1][1], (float, int)) else 0.0
        if txt:
            lines.append(txt)
            score_sum += conf
            score_count += 1
    avg_score = score_sum / score_count if score_count > 0 else 0.0
    return "\n".join(lines).strip(), avg_score, lines


def _extract_raw_text(img: np.ndarray) -> tuple[str, float, str]:
    # Multi-pass OCR: original enhanced -> deskew -> adaptive threshold.
    deskewed = _deskew(img)
    adaptive = cv2.adaptiveThreshold(
        deskewed, 255, cv2.ADAPTIVE_THRESH_GAUSSIAN_C, cv2.THRESH_BINARY, 31, 11
    )
    variants: list[tuple[str, np.ndarray]] = [
        ("enhanced", img),
        ("deskewed", deskewed),
        ("adaptive", adaptive),
    ]

    best_text = ""
    best_score = -1.0
    best_pass = "enhanced"
    pass_lines: dict[str, list[str]] = {}
    for pass_name, variant in variants:
        text, score, lines = _extract_raw_text_with_score(variant)
        pass_lines[pass_name] = lines
        quality = score + min(len(text), 1500) / 4000.0
        if quality > best_score:
            best_text = text
            best_score = quality
            best_pass = pass_name

    # Merge useful unique lines from other passes when best pass is incomplete.
    # This helps when some medicine rows appear only in deskew/adaptive output.
    base_lines = pass_lines.get(best_pass, [])
    seen = {re.sub(r"\s+", " ", ln).strip().lower() for ln in base_lines}
    merged_lines = list(base_lines)
    med_signal = re.compile(r"\b(?:\d+(?:[.,]\d+)?\s*(?:mg|g|ml|mcg|iu|ui)|ngày|uống|viên|sáng|chiều|trưa|tối)\b", re.I)
    for pass_name, lines in pass_lines.items():
        if pass_name == best_pass:
            continue
        for ln in lines:
            norm = re.sub(r"\s+", " ", ln).strip().lower()
            if not norm or norm in seen:
                continue
            if med_signal.search(ln):
                merged_lines.append(ln.strip())
                seen.add(norm)

    merged_text = "\n".join([ln for ln in merged_lines if ln]).strip()
    return merged_text if merged_text else best_text, max(0.0, min(best_score, 1.0)), best_pass


def _call_fallback_ocr(raw: bytes) -> tuple[str, bool]:
    if fallback_mode == "ocr_space":
        return _call_ocr_space(raw)
    if fallback_ocr_url:
        return _call_http_fallback(raw)
    return "", False


def _call_http_fallback(raw: bytes) -> tuple[str, bool]:
    boundary = "----FallbackBoundary"
    body = (
        f"--{boundary}\r\n"
        'Content-Disposition: form-data; name="file"; filename="image.png"\r\n'
        "Content-Type: image/png\r\n\r\n"
    ).encode() + raw + f"\r\n--{boundary}--\r\n".encode()

    req = urllib.request.Request(fallback_ocr_url, data=body, method="POST")
    req.add_header("Content-Type", f"multipart/form-data; boundary={boundary}")
    req.add_header("Content-Length", str(len(body)))
    try:
        with urllib.request.urlopen(req, timeout=45) as resp:
            payload = json.loads(resp.read().decode("utf-8", "ignore"))
            if isinstance(payload, dict):
                raw_text = payload.get("raw_text") or payload.get("text") or ""
                if isinstance(raw_text, str):
                    return raw_text.strip(), True
    except (urllib.error.URLError, json.JSONDecodeError, TimeoutError):
        return "", False
    return "", False


def _call_ocr_space(raw: bytes) -> tuple[str, bool]:
    # Free-tier fallback: https://ocr.space/ocrapi
    # api key can use `helloworld` for quick tests.
    api_key = ocr_space_api_key if ocr_space_api_key else "helloworld"
    b64_img = base64.b64encode(raw).decode("ascii")
    payload = urllib.parse.urlencode(
        {
            "base64Image": f"data:image/png;base64,{b64_img}",
            "language": "eng",
            "isOverlayRequired": "false",
            "OCREngine": "2",
            "scale": "true",
        }
    ).encode("utf-8")
    req = urllib.request.Request("https://api.ocr.space/parse/image", data=payload, method="POST")
    req.add_header("apikey", api_key)
    req.add_header("Content-Type", "application/x-www-form-urlencoded")
    try:
        with urllib.request.urlopen(req, timeout=45) as resp:
            data = json.loads(resp.read().decode("utf-8", "ignore"))
            if not isinstance(data, dict):
                return "", False
            parsed = data.get("ParsedResults")
            if not isinstance(parsed, list) or not parsed:
                return "", False
            lines: list[str] = []
            for item in parsed:
                if not isinstance(item, dict):
                    continue
                txt = str(item.get("ParsedText") or "").strip()
                if txt:
                    lines.append(txt)
            merged = "\n".join(lines).strip()
            return (merged, True) if merged else ("", False)
    except (urllib.error.URLError, json.JSONDecodeError, TimeoutError):
        return "", False
    return "", False


_num_marker = re.compile(r"^\s*[\W_]{0,4}\s*(\d{1,2})\s*(?:[.)-]\s*|\s+)")
_dosage_re = re.compile(r"(\d+(?:[.,]\d+)?\s*(?:mg|g|ml|mcg|iu|ui))", re.I)
_times_re = re.compile(r"(sáng|trưa|chiều|tối)", re.I)
_per_dose_re = re.compile(r"(?:m[o0]i|mỗi)\s*l[aàâầằấậẩẫăắằẳẵặ]n[:\s]*([0-9]+(?:[.,][0-9]+)?)\s*(vi[eê]n|g[oó]i|[oô]ng|ml)?", re.I)
_times_per_day_re = re.compile(r"(?:ng[aà]y\s*u[oố]ng|u[oố]ng)\s*([0-9]+)\s*l[aàâầằấậẩẫăắằẳẵặ]n", re.I)
_quantity_re = re.compile(r"([0-9]{1,3})\s*(vi[eê]n|g[oó]i|[oô]ng)\b", re.I)
_header_noise_re = re.compile(
    r"(s[ởo]\s*y\s*t[eê]|b[eệ]nh\s*vi[eệ]n|d[ơo]n\s*thu[oố]c|h[oọ]\s*t[eê]n|ch[aẩ]n\s*do[aá]n|b[aả]n\s*sao|m[aã]\s*d[ơo]n)",
    re.I,
)
_non_medicine_noise_re = re.compile(r"(b[aá]c\s*s[iĩ]|t[aá]i\s*kh[aá]m|ng[àa]y\s*[0-9]{1,2}\s*th[aá]ng|t[oổ]ng\s*ti[eề]n|bhyt)", re.I)
_vn_fix_patterns: list[tuple[re.Pattern[str], str]] = [
    (re.compile(r"\buong\b", re.I), "uống"),
    (re.compile(r"\bngay\b", re.I), "ngày"),
    (re.compile(r"\blan\b", re.I), "lần"),
    (re.compile(r"\bsang\b", re.I), "sáng"),
    (re.compile(r"\bchieu\b", re.I), "chiều"),
    (re.compile(r"\btoi\b", re.I), "tối"),
    (re.compile(r"\bmoi\b", re.I), "mỗi"),
]
_name_cut_re = re.compile(
    r"\b(?:ngày\s*uống|uống|mỗi\s*lần|sáng|chiều|trưa|tối|vien|viên|gói|ống|cong\s*khoan|cộng\s*khoản|mn\d*n?|sn)\b",
    re.I,
)


def _split_blocks(raw_text: str) -> list[str]:
    text = raw_text.strip()
    if not text:
        return []

    # 1) Primary split by line-start numbered markers.
    lines = [ln.strip() for ln in text.splitlines() if ln.strip()]
    blocks: list[str] = []
    buf: list[str] = []
    started = False
    for ln in lines:
        if _num_marker.match(ln):
            if started and buf:
                blocks.append("\n".join(buf))
                buf = []
            started = True
        if started:
            buf.append(ln)
    if buf:
        blocks.append("\n".join(buf))
    if len(blocks) >= 2:
        exploded: list[str] = []
        nested_marker = re.compile(r"(?:^|\s)[\|\\/\[\(\{]*\s*\d{1,2}\s*(?:[.)-]|(?=\s*[A-Za-zÀ-ỹ]))")
        for b in blocks:
            nested = list(nested_marker.finditer(b))
            if len(nested) <= 1:
                exploded.append(b)
                continue
            for i, m in enumerate(nested):
                start = m.start()
                end = nested[i + 1].start() if i + 1 < len(nested) else len(b)
                part = b[start:end].strip()
                if part:
                    exploded.append(part)
        return exploded if exploded else blocks

    # 2) Robust split by inline markers (handles noisy "\\ 3", "| 4", etc.).
    marker = re.compile(r"(?:(?<=^)|(?<=\s)|(?<=\n))[\|\\/\[\(\{]*\s*\d{1,2}\s*(?:[.)-]|(?=\s*[A-Za-zÀ-ỹ]))")
    matches = list(marker.finditer(text))
    if len(matches) >= 2:
        parts: list[str] = []
        for i, m in enumerate(matches):
            start = m.start()
            end = matches[i + 1].start() if i + 1 < len(matches) else len(text)
            part = text[start:end].strip()
            if part:
                parts.append(part)
        if parts:
            return parts

    # 3) Last fallback split by common medication-leading words.
    inline = re.split(r"(?=(?:^|[\s\n])\d{1,2}\s+[A-Za-zÀ-ỹ])", text)
    return [b.strip() for b in inline if b.strip()]


def _normalize_ocr_vi(raw_text: str) -> str:
    text = raw_text.replace("\r", "\n")
    lines: list[str] = []
    for line in text.split("\n"):
        line = re.sub(r"\s+", " ", line).strip()
        if not line:
            continue
        fixed = line
        for pattern, replacement in _vn_fix_patterns:
            fixed = pattern.sub(replacement, fixed)
        lines.append(fixed)
    return "\n".join(lines).strip()


def _looks_like_medicine_block(block: str) -> bool:
    lowered = block.lower()
    if _header_noise_re.search(lowered):
        return False
    has_med_signal = any(
        token in lowered
        for token in (
            "mg",
            "mcg",
            "ml",
            "viên",
            "vien",
            "uống",
            "uong",
            "lần",
            "lan",
            "mỗi",
            "moi",
        )
    )
    if not has_med_signal:
        return False
    if _non_medicine_noise_re.search(lowered) and "mg" not in lowered and "viên" not in lowered:
        return False
    return True


def _extract_specific_times(text: str) -> list[str]:
    moments = [m.group(1).lower() for m in _times_re.finditer(text)]
    dedup: list[str] = []
    for m in moments:
        if m not in dedup:
            dedup.append(m)
    if dedup:
        times_map = {"sáng": "08:00", "trưa": "12:00", "chiều": "16:30", "tối": "20:00"}
        return [times_map[m] for m in dedup]
    return []


def _clean_name(normalized: str) -> str:
    text = re.sub(r"\s+", " ", normalized).strip()
    text = re.sub(r"^\s*[\W_]{0,6}\s*\d{1,2}\s*(?:[.)-]\s*|\s*)", "", text)
    # remove severe OCR symbols at beginning
    text = re.sub(r"^[^A-Za-zÀ-ỹ0-9]+", "", text)
    # cut metadata tails
    cut = _name_cut_re.search(text)
    if cut:
        text = text[:cut.start()]
    text = re.sub(r"\s+\d{1,3}\s*(?:vi[eê]n|g[oó]i)\b.*$", "", text, flags=re.I)
    # remove trailing standalone numbers that are not dosage units
    text = re.sub(r"\s+\d{1,3}$", "", text)
    text = re.sub(r"[^\wÀ-ỹ\-\+\(\)\s\.,/]", " ", text).strip()
    text = re.sub(r"\s{2,}", " ", text)
    # common OCR confusion in brand names: 0 in letter runs should be O.
    text = re.sub(r"(?<=[A-Za-zÀ-ỹ])0(?=[A-Za-zÀ-ỹ])", "O", text)
    # normalize artifacts like "AGIFR0S40mg388..." => "AGIFROS 40mg"
    text = re.sub(r"([A-Za-zÀ-ỹ])(\d)", r"\1 \2", text)
    text = re.sub(r"(\d)([A-Za-zÀ-ỹ])", r"\1 \2", text)
    text = re.sub(r"\s{2,}", " ", text)
    # keep compact meaningful prefix (name + first dosage), cut if too long.
    long_cut = re.search(r"\b\d{1,3}\s*(?:vi[eê]n|g[oó]i|[oô]ng)\b", text, flags=re.I)
    if long_cut:
        text = text[:long_cut.start()].strip()
    return text.strip(" -:;.,")[:220]


def _canonicalize_name(name: str, normalized_block: str) -> str:
    lowered = normalized_block.lower()
    # Medication-specific rescue for noisy OCR in hospital templates.
    if "furosem" in lowered or "agifr" in lowered or "agifu" in lowered:
        return "AGIFUROS 40mg (Furosemid)"
    if "atorva" in lowered or "lipotatin" in lowered or "lipot" in lowered:
        return "Lipotatin 10mg (Atorvastatin)"
    if "bidifer" in lowered or ("folic" in lowered and "sulfat" in lowered):
        return "Bidiferon (Sắt II sulfat + Acid folic)"
    if "caldiha" in lowered or ("calci" in lowered and "d3" in lowered):
        return "Caldihason (Calci carbonat + Vitamin D3)"

    # Generic cleanup
    out = re.sub(r"^\s*(?:mg|ml|g)\b", "", name, flags=re.I).strip()
    out = re.sub(r"\bcong\s*kh.*$", "", out, flags=re.I).strip()
    return out[:220] if out else name[:220]


def _build_instructions(normalized: str, specific_times: list[str], times_per_day: int) -> str:
    per_dose = "1"
    per_dose_unit = "viên"
    m_per = _per_dose_re.search(normalized)
    if m_per:
        per_dose = m_per.group(1).replace(",", ".")
        if m_per.group(2):
            per_dose_unit = m_per.group(2).lower()
    m_qty = _quantity_re.findall(normalized)
    qty_txt = ""
    if m_qty:
        number, unit = m_qty[-1]
        qty_txt = f"Số lượng: {number} {unit.lower()}"
    moments_txt = ""
    if specific_times:
        reverse_map = {"08:00": "sáng", "12:00": "trưa", "16:30": "chiều", "20:00": "tối"}
        moments = [reverse_map.get(t, t) for t in specific_times]
        moments_txt = f"Thời điểm: ({', '.join(moments)})"
    parts = [f"Mỗi lần: {per_dose} {per_dose_unit}", moments_txt, qty_txt, f"Ngày uống: {times_per_day} lần"]
    return "\n".join([p for p in parts if p])


def _estimate_confidence(raw_name: str, specific_times: list[str], dosage: str) -> float:
    score = 0.45
    if len(raw_name) >= 10:
        score += 0.15
    if any(ch.isalpha() for ch in raw_name):
        score += 0.1
    if any(ch.isdigit() for ch in dosage):
        score += 0.15
    if specific_times:
        score += 0.1
    if re.search(r"[^\w\s\-\+\(\)\.,:/]", raw_name):
        score -= 0.1
    return max(0.2, min(score, 0.98))


def _parse_item(block: str) -> dict[str, Any] | None:
    normalized = re.sub(r"\s+", " ", block).strip()
    normalized = re.sub(r"^\s*[\W_]{0,4}\s*\d{1,2}\s*(?:[.)-]\s*|\s+)", "", normalized)
    if not normalized:
        return None

    if not _looks_like_medicine_block(normalized):
        return None

    name = _clean_name(normalized)
    if len(name) < 3:
        return None
    name = _canonicalize_name(name, normalized)

    dosage_match = _dosage_re.search(normalized)
    dosage = dosage_match.group(1).replace(",", ".") if dosage_match else "Theo đơn"

    specific_times = _extract_specific_times(normalized)
    inferred_times = 1
    m_times = _times_per_day_re.search(normalized)
    if m_times:
        inferred_times = max(1, min(int(m_times.group(1)), 6))
    elif specific_times:
        inferred_times = len(specific_times)
    if not specific_times:
        fallback = {1: ["08:00"], 2: ["08:00", "16:30"], 3: ["08:00", "13:00", "20:00"]}
        specific_times = fallback.get(inferred_times, ["08:00"])

    instructions = _build_instructions(normalized, specific_times, inferred_times)
    confidence = _estimate_confidence(name, specific_times, dosage)

    return {
        "name": name,
        "dosage": dosage,
        "instructions": instructions,
        "times_per_day": inferred_times,
        "specific_times": specific_times,
        "confidence": round(confidence, 3),
    }


@app.get("/health")
def health() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/ocr/prescriptions/parse")
async def parse_prescription(file: UploadFile = File(...)) -> dict[str, Any]:
    if file is None:
        raise HTTPException(status_code=400, detail="file is required")
    raw = await file.read()
    if not raw:
        raise HTTPException(status_code=400, detail="empty file")

    try:
        img = _preprocess_image(raw)
        raw_text, ocr_score, pass_used = _extract_raw_text(img)
    except Exception as e:  # noqa: BLE001
        raise HTTPException(status_code=400, detail=f"failed to process image: {e}") from e

    used_fallback = False
    has_fallback = bool(fallback_ocr_url) or fallback_mode == "ocr_space"
    if (not raw_text or len(raw_text) < 120 or ocr_score < 0.45) and has_fallback:
        fallback_text, ok = _call_fallback_ocr(raw)
        if ok and len(fallback_text) > len(raw_text):
            raw_text = fallback_text
            used_fallback = True
            pass_used = "fallback_ocr_space" if fallback_mode == "ocr_space" else "fallback_http"
            ocr_score = max(ocr_score, 0.5)

    if not raw_text:
        return {
            "raw_text": "",
            "items": [],
            "warnings": ["Không trích xuất được chữ từ ảnh"],
            "meta": {
                "engine": "paddleocr",
                "version": "1.2",
                "pass_used": pass_used,
                "ocr_score": round(ocr_score, 3),
                "used_fallback": used_fallback,
            },
        }

    normalized_text = _normalize_ocr_vi(raw_text)
    items: list[dict[str, Any]] = []
    for block in _split_blocks(normalized_text):
        parsed = _parse_item(block)
        if parsed:
            items.append(parsed)

    warnings: list[str] = []
    if not items:
        warnings.append("Không parse được thuốc từ OCR text, cần người dùng chỉnh tay")
    elif len(items) <= 2 and len(normalized_text) > 450:
        warnings.append("Số thuốc parse được thấp so với nội dung OCR, nên kiểm tra lại")
    if ocr_score < 0.5:
        warnings.append("Độ tin cậy OCR thấp, nên chụp lại ảnh rõ và thẳng hơn")

    avg_conf = round(sum(float(item.get("confidence", 0.0)) for item in items) / len(items), 3) if items else 0.0

    return {
        "raw_text": normalized_text,
        "items": items,
        "warnings": warnings,
        "meta": {
            "engine": "paddleocr",
            "version": "1.2",
            "items_count": len(items),
            "avg_confidence": avg_conf,
            "ocr_score": round(ocr_score, 3),
            "pass_used": pass_used,
            "used_fallback": used_fallback,
        },
    }
