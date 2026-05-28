import base64
import json
import os
import re
import urllib.error
import urllib.parse
import urllib.request
from typing import Union


def _get_bytes(img_input: Union[bytes, str]) -> bytes:
    if isinstance(img_input, bytes):
        return img_input
    with open(img_input, "rb") as f:
        return f.read()


def call_fallback_ocr(
    img_input: Union[bytes, str],
    *,
    fallback_mode: str,
    fallback_ocr_url: str,
    ocr_space_api_key: str,
) -> tuple[str, bool]:
    if fallback_mode == "ocr_space":
        return call_ocr_space(img_input, ocr_space_api_key=ocr_space_api_key)
    if fallback_mode == "mistral_ocr":
        txt, ok = call_mistral_ocr(img_input)
        if ok and _is_usable_ocr_text(txt):
            return txt, True
        # Safety net: when Mistral output is noisy, try OCR.Space.
        txt2, ok2 = call_ocr_space(img_input, ocr_space_api_key=ocr_space_api_key)
        if ok2:
            return txt2, True
        return txt, ok
    if fallback_ocr_url:
        return call_http_fallback(img_input, fallback_ocr_url=fallback_ocr_url)
    return "", False


def call_http_fallback(
    img_input: Union[bytes, str], *, fallback_ocr_url: str
) -> tuple[str, bool]:
    raw = _get_bytes(img_input)
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


def call_ocr_space(
    img_input: Union[bytes, str], *, ocr_space_api_key: str
) -> tuple[str, bool]:
    raw = _get_bytes(img_input)
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


def call_mistral_ocr(img_input: Union[bytes, str]) -> tuple[str, bool]:
    raw = _get_bytes(img_input)
    api_key = os.getenv("MISTRAL_API_KEY", "").strip()
    if not api_key:
        return "", False

    model = os.getenv("OCR_MISTRAL_MODEL", "mistral-ocr-latest").strip() or "mistral-ocr-latest"
    timeout_sec = int(os.getenv("OCR_MISTRAL_TIMEOUT_SEC", "45") or "45")

    try:
        try:
            from mistralai import Mistral  # type: ignore[reportMissingImports]
        except Exception:
            from mistralai.client import Mistral  # type: ignore[reportMissingImports,no-redef]

        try:
            client = Mistral(api_key=api_key, timeout=timeout_sec)
        except TypeError:
            # Some SDK versions do not accept timeout in constructor.
            client = Mistral(api_key=api_key)
        uploaded_file = client.files.upload(
            file={"file_name": "prescription.png", "content": raw},
            purpose="ocr",
        )
        # SDK expects expiry in hours (1..168), not seconds.
        signed_url = client.files.get_signed_url(file_id=uploaded_file.id, expiry=1)
        response = client.ocr.process(
            model=model,
            document={"type": "document_url", "document_url": signed_url.url},
            include_image_base64=False,
            table_format=None,
        )
    except Exception:
        return "", False

    pages = getattr(response, "pages", None) or []
    texts: list[str] = []
    for page in pages:
        markdown = getattr(page, "markdown", None)
        if markdown is None and isinstance(page, dict):
            markdown = page.get("markdown")
        txt = str(markdown or "").strip()
        if txt:
            texts.append(txt)

    merged = _clean_mistral_markdown("\n".join(texts))
    return (merged, True) if merged else ("", False)


def _clean_mistral_markdown(text: str) -> str:
    if not text:
        return ""

    lines_in = [ln.strip() for ln in text.splitlines() if ln.strip()]
    cleaned: list[str] = []
    in_prescription_block = False

    rx_header = re.compile(r"\b(?:đơn thuốc|toa thuốc)\b", re.I)
    item_line = re.compile(
        r"^\s*(?:\d{1,2}\s*[\)./-]|[-*])\s*|"
        r"\b(?:mg|mcg|g|ml|viên|goi|gói|ống|sáng|trưa|chiều|tối|uống)\b",
        re.I,
    )
    noise_line = re.compile(
        r"^(?:#+|[*_`>-]+)$|"
        r"^\s*(?:thông tin|scan|barcode|mã đơn|ms|số phiếu|địa chỉ|dt)\b",
        re.I,
    )

    for ln in lines_in:
        ln = re.sub(r"^[#>\-\*\s]+", "", ln)
        ln = re.sub(r"\*\*(.*?)\*\*", r"\1", ln)
        ln = ln.replace("|", " ")
        ln = re.sub(r"\s+", " ", ln).strip(" -:;")
        if not ln:
            continue
        if rx_header.search(ln):
            in_prescription_block = True
            cleaned.append(ln)
            continue
        if noise_line.search(ln):
            continue
        if in_prescription_block:
            cleaned.append(ln)
        elif item_line.search(ln):
            cleaned.append(ln)

    if not cleaned:
        cleaned = [re.sub(r"\s+", " ", ln).strip() for ln in lines_in]
    return "\n".join([ln for ln in cleaned if ln]).strip()


def _is_usable_ocr_text(text: str) -> bool:
    s = (text or "").strip()
    if len(s) < 100:
        return False
    lines = [ln.strip() for ln in s.splitlines() if ln.strip()]
    if len(lines) < 4:
        return False

    uniq_ratio = len(set(lines)) / max(1, len(lines))
    if uniq_ratio < 0.35:
        return False

    med_signal = re.compile(
        r"\b(?:mg|mcg|g|ml|viên|goi|gói|ống|sáng|trưa|chiều|tối|uống|ngày)\b",
        re.I,
    )
    score = sum(1 for ln in lines if med_signal.search(ln))
    return score >= 2
