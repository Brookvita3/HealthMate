import os
from dataclasses import dataclass


def _int_env(name: str, default: int, min_value: int = 1, max_value: int = 32) -> int:
    raw = os.getenv(name, "").strip()
    if not raw:
        return default
    try:
        value = int(raw)
    except ValueError:
        return default
    return max(min_value, min(max_value, value))


@dataclass(frozen=True)
class OCRRuntimeSettings:
    fallback_ocr_url: str
    fallback_mode: str
    ocr_space_api_key: str
    mistral_api_key: str
    enable_synthetic_hint_rows: bool
    enable_known_medicine_rescue: bool
    prioritize_name_recall: bool
    max_concurrent_requests: int


def load_runtime_settings() -> OCRRuntimeSettings:
    return OCRRuntimeSettings(
        fallback_ocr_url=os.getenv("OCR_FALLBACK_HTTP_URL", "").strip(),
        fallback_mode=os.getenv("OCR_FALLBACK_MODE", "").strip().lower(),
        ocr_space_api_key=os.getenv("OCR_SPACE_API_KEY", "").strip(),
        mistral_api_key=os.getenv("MISTRAL_API_KEY", "").strip(),
        enable_synthetic_hint_rows=os.getenv("OCR_ENABLE_SYNTHETIC_HINT_ROWS", "").strip().lower()
        in {"1", "true", "yes"},
        enable_known_medicine_rescue=os.getenv("OCR_ENABLE_KNOWN_MEDICINE_RESCUE", "")
        .strip()
        .lower()
        in {"1", "true", "yes"},
        prioritize_name_recall=os.getenv("OCR_PRIORITIZE_NAME_RECALL", "true").strip().lower()
        in {"1", "true", "yes", "on"},
        max_concurrent_requests=_int_env(
            "OCR_MAX_CONCURRENT_REQUESTS", default=2, min_value=1, max_value=16
        ),
    )


RUNTIME_SETTINGS = load_runtime_settings()
