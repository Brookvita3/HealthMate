import re
from typing import Any
from app.drug_dictionary import normalize_name_with_dictionary_meta

_med_name_typo_fixes: list[tuple[re.Pattern[str], str]] = [
    (re.compile(r"\baumgmentin\b", re.I), "Augmentin"),
    (re.compile(r"\bauqmentin\b", re.I), "Augmentin"),
    (re.compile(r"\baugmentln\b", re.I), "Augmentin"),
    (re.compile(r"\bacemuc\b", re.I), "Acemuc"),
    (re.compile(r"\bpropanolol\b", re.I), "Propranolol"),
    (re.compile(r"\bva?g\s*[-_ ]*\d*\s*sta?t\b", re.I), "Vagastat"),
    (re.compile(r"\bpanto\w*pantoc\w*\b", re.I), "Pantoc"),
    (re.compile(r"\bpantopra\w*\b", re.I), "Pantoprazol"),
    (re.compile(r"\bsu?x?ra?l?a?t?\b", re.I), "Sucralfat"),
]


_med_name_cut_markers = re.compile(
    r"\b(?:ng[aà]y|s[aá]ng|tr[ưu]a|chi[eề]u|t[oố]i|m[oỗ]i|l[aà]n|u[oố]ng|c[aá]ch d[uù]ng|sig)\b",
    re.I,
)


def _truncate_name_noise(text: str) -> str:
    """Cắt phần đuôi không thuộc tên thuốc (lịch dùng/cách dùng)."""
    if not text:
        return text

    # Ưu tiên cắt theo marker semantic.
    m = _med_name_cut_markers.search(text)
    if m and m.start() > 4:
        text = text[: m.start()].strip()

    dose_match = re.search(r"\b\d+(?:[.,]\d+)?\s*(?:mg|mcg|g|ml)\b", text, flags=re.I)
    if dose_match and dose_match.start() > 3:
        text = text[: dose_match.start()].strip()

    # Cắt theo dấu phân tách thường xuất hiện khi OCR trộn nhiều cột.
    text = re.split(r"\s*[|•;]{1,}\s*", text, maxsplit=1)[0].strip()

    # Nếu chuỗi có lặp token bất thường, giữ nửa đầu để tránh dính hai thuốc.
    toks = [t for t in re.split(r"\s+", text) if t]
    if len(toks) >= 8:
        uniq_ratio = len(set(t.lower() for t in toks)) / len(toks)
        if uniq_ratio < 0.72:
            text = " ".join(toks[: max(3, len(toks) // 2)])

    return text.strip(" -:;.,")


def clean_medication_name_surface(name: str) -> str:
    """Dọn nhiễu tên thuốc sau parse (không thay đổi logic tách item)."""
    text = re.sub(r"\s+", " ", (name or "")).strip()
    if not text:
        return text

    for pat, repl in _med_name_typo_fixes:
        text = pat.sub(repl, text)

    text = _truncate_name_noise(text)
    text = re.sub(r"\b\d+\b", " ", text)
    text = re.sub(r"\b(?:mt|nt)\b", " ", text, flags=re.I)

    # Bỏ các cụm ký tự nhiễu thường xuất hiện cuối tên.
    text = re.sub(r"\b(?:g\s*6\s*i|g6i)\b", "", text, flags=re.I)
    text = re.sub(r"\((?:mt|nt)\)\s*$", "", text, flags=re.I)
    text = re.sub(r"\s{2,}", " ", text).strip(" -:;.,")

    # Trường hợp OCR ghép nhiều từ không tự nhiên, ưu tiên cụm đầu sạch hơn.
    tokens = [t for t in text.split(" ") if t]
    if len(tokens) > 7:
        text = " ".join(tokens[:7])

    # Nếu token đầu bị OCR tách rời (vd: "Vag stat"), thử ghép lại.
    parts = [p for p in text.split(" ") if p]
    if 2 <= len(parts) <= 3:
        compact = "".join(parts)
        if 6 <= len(compact) <= 20:
            text = compact

    text = re.sub(r"\s{2,}", " ", text).strip(" -:;.,")
    return text[:220]


def post_process_items(items: list[dict[str, Any]]) -> tuple[list[dict[str, Any]], list[str]]:
    out: list[dict[str, Any]] = []
    warnings: list[str] = []
    for item in items:
        fixed = dict(item)
        cleaned = clean_medication_name_surface(str(item.get("name", "")))
        normalized, matched, match_kind = normalize_name_with_dictionary_meta(cleaned)
        fixed["name"] = normalized
        fixed["name_match_kind"] = match_kind

        if not matched and cleaned:
            if match_kind == "fuzzy_below_cutoff":
                warnings.append(
                    f"Tên thuốc '{cleaned}' gần đúng nhưng chưa đủ chắc chắn, cần kiểm tra thủ công"
                )
            elif match_kind == "not_found":
                warnings.append(f"Không tìm thấy '{cleaned}' trong danh mục thuốc, cần kiểm tra thủ công")
        out.append(fixed)

    # Dedupe nhẹ theo tên thuốc đã normalize để tránh trùng item
    # khi OCR tách cùng 1 thuốc thành nhiều dòng gần giống nhau.
    deduped: list[dict[str, Any]] = []
    seen_name_keys: set[str] = set()
    for item in out:
        key = re.sub(r"[^a-z0-9à-ỹ]+", "", str(item.get("name", "")).lower())
        if len(key) >= 3 and key in seen_name_keys:
            continue
        if len(key) >= 3:
            seen_name_keys.add(key)
        deduped.append(item)

    unique_warnings: list[str] = []
    for w in warnings:
        if w not in unique_warnings:
            unique_warnings.append(w)
    return deduped, unique_warnings[:8]
