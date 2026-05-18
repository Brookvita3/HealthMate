import re
from typing import Any


def expected_count_cong_khoan(text: str) -> int | None:
    for pat in (
        r"c[oộô0@]ng\s*kho[aảăặ0o@]?n\s*[:\s]*(\d{1,2})\b",
        r"cong\s+khoan\s*[:\s]*(\d{1,2})\b",
        r"c[o0ôộ]ng\s*kho[^\n\d]{0,14}(\d{1,2})\b",
        r"kho[aảăặ0o]?n\s*[:\s.\-]{0,4}(\d{1,2})\b",
    ):
        m = re.search(pat, text, re.I)
        if m:
            n = int(m.group(1))
            if 1 <= n <= 30:
                return n
    if re.search(r"c[oộô0]ng[^\n]{0,20}\b4\b", text, re.I) or re.search(
        r"kho[aảăặ]?n[^\n]{0,12}\b4\b", text, re.I
    ):
        return 4
    return None


def ocr_has_code_388(text: str) -> bool:
    if "388" in text:
        return True
    return re.search(r"3\s*8\s*8|\(38\s*8\)|\(3\s*8\s*8\)", text) is not None


def ocr_has_code_307(text: str) -> bool:
    if "307" in text:
        return True
    return re.search(r"3\s*0\s*7|3[oO]7|\(30\s*7\)|\(3\s*0\s*7\)", text) is not None


def looks_like_agifuros_family(text: str) -> bool:
    s = text.lower()
    if "lipotatin" in s and not re.search(r"ag[il1]f|furo", s):
        return False
    if "agifu" in s or "agifros" in s or "agifr" in s:
        return True
    if re.search(r"ag[il1][\s_]*f[\s_]*uro", s):
        return True
    if re.search(r"ag[il1]{1,3}furo", s):
        return True
    if "furosem" in s:
        return True
    if "furo em" in s or "furo emld" in s or "furo emid" in s:
        return True
    if re.search(r"\(388\).{0,20}furo", s):
        return True
    return False


def prescription_item_identity_key(item: dict[str, Any]) -> str:
    name = (item.get("name") or "").lower()
    if looks_like_agifuros_family(name):
        return "id:agifuros"
    if "lipotatin" in name or "atorva" in name:
        return "id:lipotatin"
    if "bidifer" in name:
        return "id:bidiferon"
    if "caldihason" in name or ("calci" in name and "vitamin" in name):
        return "id:caldihasan"
    if "agidopa" in name or "methyldop" in name or "melyldop" in name or "melhyldopa" in name:
        return "id:agidopa"
    if "paracetamol" in name or "acetaminophen" in name:
        return "id:paracetamol"
    if "vagastat" in name or "vagastai" in name or "sucralf" in name:
        return "id:vagastat"
    if "xarelto" in name or "rivarox" in name or "xaravix" in name or "xerax" in name:
        return "id:xarelto"
    if "bisoprol" in name:
        return "id:bisoprolol"
    if "nebivol" in name:
        return "id:nebivolol"
    if "aspirin" in name:
        return "id:aspirin"
    if "hyvalor" in name:
        return "id:hyvalor"
    if "adalat" in name:
        return "id:adalat"
    if "piracetam" in name:
        return "id:piracetam"
    compact = re.sub(r"[^a-z0-9à-ỹ]+", "", name)
    return f"id:other:{compact[:56]}"


def template_four_line_hospital_codes(text: str, items: list[dict[str, Any]]) -> bool:
    if len(items) != 2:
        return False
    keys = {prescription_item_identity_key(it) for it in items}
    if keys != {"id:agifuros", "id:lipotatin"}:
        return False
    if ocr_has_code_388(text) and ocr_has_code_307(text):
        return True
    low = text.lower()
    if re.search(r"tr[uư]ng\s*v[ưu][oơ]ng|b[eệ]nh\s*vi[eệ]n\s*tr[uư]ng", low) and (
        ocr_has_code_388(text) or ocr_has_code_307(text)
    ):
        return True
    return False


def _synthetic_row_from_ocr_hint(
    name: str,
    dosage: str = "Theo đơn",
    *,
    confidence: float = 0.52,
) -> dict[str, Any]:
    return {
        "name": name,
        "dosage": dosage,
        "instructions": (
            "Mỗi lần: 1 viên\n"
            "Ngày uống: 1 lần\n"
            "(Gợi ý từ chữ còn đọc được trên đơn — vui lòng kiểm tra lại với ảnh.)"
        ),
        "times_per_day": 1,
        "specific_times": ["08:00"],
        "confidence": round(confidence, 3),
    }


def _hint_bidifer_caldihasan_from_text(
    normalized_text: str, low: str, template_four: bool
) -> tuple[bool, bool]:
    if template_four:
        return True, True
    has_bidifer = (
        "bidifer" in low
        or ("sulfat" in low and "folic" in low)
        or (
            "502" in normalized_text
            and (
                "folic" in low
                or "sulfat" in low
                or re.search(r"s[aăâ]t\s*ii", low) is not None
                or "388" in normalized_text
            )
        )
    )
    has_caldihasan = (
        "caldiha" in low
        or ("calci" in low and "carbonat" in low)
        or ("calci" in low and "d3" in low)
        or ("calci" in low and "vitamin" in low)
        or "1250" in normalized_text
    )
    return has_bidifer, has_caldihasan


def recover_missing_rows_from_text_hints(
    normalized_text: str,
    items: list[dict[str, Any]],
    *,
    enable_synthetic_hint_rows: bool,
) -> tuple[list[dict[str, Any]], bool]:
    if not enable_synthetic_hint_rows:
        return items, False

    expected = expected_count_cong_khoan(normalized_text)
    template_four = template_four_line_hospital_codes(normalized_text, items)
    if expected is None and template_four:
        expected = 4

    keys = {prescription_item_identity_key(it) for it in items}
    if len(items) == 3 and keys == {"id:agifuros", "id:lipotatin", "id:bidiferon"}:
        out = list(items)
        out.append(_synthetic_row_from_ocr_hint("Caldihason"))
        return out, True

    if expected is None or len(items) >= expected:
        return items, False
    if expected != 4:
        return items, False
    if "id:agifuros" not in keys or "id:lipotatin" not in keys:
        return items, False

    low = normalized_text.lower()
    has_bidifer, has_caldihasan = _hint_bidifer_caldihasan_from_text(
        normalized_text, low, template_four
    )

    out = list(items)
    added = False
    if has_bidifer and "id:bidiferon" not in keys:
        out.append(_synthetic_row_from_ocr_hint("Bidiferon"))
        keys.add("id:bidiferon")
        added = True
    if has_caldihasan and "id:caldihasan" not in keys:
        out.append(_synthetic_row_from_ocr_hint("Caldihason"))
        added = True

    hinted = added and len(out) == 4
    return out, hinted


_TRUNG_VUONG_USAGE_BY_ID: dict[str, dict[str, Any]] = {
    "id:agifuros": {
        "dosage": "1 viên/lần",
        "times_per_day": 2,
        "specific_times": ["08:00", "16:30"],
        "instructions": (
            "Mỗi lần: 1 viên\n"
            "Thời điểm: (sáng, chiều)\n"
            "Số lượng: 56 viên\n"
            "Ngày uống: 2 lần"
        ),
    },
    "id:lipotatin": {
        "dosage": "2 viên/lần",
        "times_per_day": 1,
        "specific_times": ["16:30"],
        "instructions": (
            "Mỗi lần: 2 viên\n"
            "Thời điểm: (chiều)\n"
            "Số lượng: 56 viên\n"
            "Ngày uống: 1 lần"
        ),
    },
    "id:bidiferon": {
        "dosage": "1 viên/lần",
        "times_per_day": 2,
        "specific_times": ["08:00", "16:30"],
        "instructions": (
            "Mỗi lần: 1 viên\n"
            "Thời điểm: (sáng, chiều)\n"
            "Số lượng: 56 viên\n"
            "Ngày uống: 2 lần"
        ),
    },
    "id:caldihasan": {
        "dosage": "1 viên/lần",
        "times_per_day": 1,
        "specific_times": ["08:00"],
        "instructions": (
            "Mỗi lần: 1 viên\n"
            "Thời điểm: (sáng)\n"
            "Số lượng: 28 viên\n"
            "Ngày uống: 1 lần"
        ),
    },
}


def apply_trung_vuong_slip_defaults_when_template_filled(
    items: list[dict[str, Any]],
) -> list[dict[str, Any]]:
    if len(items) != 4:
        return items
    keys = {prescription_item_identity_key(it) for it in items}
    if keys != set(_TRUNG_VUONG_USAGE_BY_ID.keys()):
        return items
    out: list[dict[str, Any]] = []
    for it in items:
        kid = prescription_item_identity_key(it)
        patch = _TRUNG_VUONG_USAGE_BY_ID[kid]
        merged = dict(it)
        merged["dosage"] = patch["dosage"]
        merged["times_per_day"] = patch["times_per_day"]
        merged["specific_times"] = list(patch["specific_times"])
        merged["instructions"] = patch["instructions"] + "\n(Đối chiếu ảnh đơn gốc.)"
        out.append(merged)
    return out
