import csv
import difflib
import os
import re
import unicodedata
from dataclasses import dataclass
from functools import lru_cache


@dataclass(frozen=True)
class DrugEntry:
    canonical_name: str
    normalized_key: str


def _normalize_drug_key(text: str) -> str:
    s = (text or "").strip().lower()
    s = re.sub(r"\([^)]*\)", " ", s)
    s = re.sub(r"[^a-z0-9à-ỹ]+", " ", s)
    s = re.sub(r"\s+", " ", s).strip()
    return s


def _strip_vietnamese_accents(text: str) -> str:
    if not text:
        return ""
    text = text.replace("đ", "d").replace("Đ", "D")
    return "".join(ch for ch in unicodedata.normalize("NFD", text) if unicodedata.category(ch) != "Mn")


def _simplify_key_for_match(key: str) -> str:
    """Chuẩn hóa key cho fuzzy để giảm nhiễu từ hàm lượng/dạng bào chế."""
    s = key
    s = re.sub(
        r"\b\d+(?:[.,]\d+)?\s*(?:mg|mcg|g|kg|ml|l|ui|iu|%)\b",
        " ",
        s,
        flags=re.I,
    )
    s = re.sub(r"\b(?:tab|tabs|tablet|cap|caps|vien|viên|ống|goi|gói|chai)\b", " ", s, flags=re.I)
    s = re.sub(r"\s+", " ", s).strip()
    return s


def _get_csv_path() -> str:
    return os.getenv("OCR_DRUG_DICT_CSV_PATH", "").strip()


def _is_enabled() -> bool:
    flag = os.getenv("OCR_DRUG_DICT_ENABLED", "true").strip().lower()
    return flag in {"1", "true", "yes", "on"}


@lru_cache(maxsize=1)
def load_drug_entries() -> tuple[DrugEntry, ...]:
    if not _is_enabled():
        return ()
    path = _get_csv_path()
    if not path or not os.path.exists(path):
        return ()

    entries: list[DrugEntry] = []
    with open(path, "r", encoding="utf-8-sig", newline="") as f:
        reader = csv.DictReader(f)
        # Hỗ trợ vài tên cột phổ biến khi export dữ liệu thuốc.
        candidate_cols = [
            "ten_thuoc",
            "tenthoc",
            "ten thuoc",
            "name",
            "drug_name",
            "thuoc",
        ]
        header_map = {h.lower().strip(): h for h in (reader.fieldnames or [])}
        col = next((header_map[c] for c in candidate_cols if c in header_map), None)
        if not col:
            return ()

        for row in reader:
            raw = str(row.get(col) or "").strip()
            if len(raw) < 3:
                continue
            key = _normalize_drug_key(raw)
            if len(key) < 3:
                continue
            entries.append(DrugEntry(canonical_name=raw, normalized_key=key))

    # Loại trùng theo key, ưu tiên tên ngắn hơn (thường sạch hơn).
    best: dict[str, str] = {}
    for e in entries:
        cur = best.get(e.normalized_key)
        if cur is None or len(e.canonical_name) < len(cur):
            best[e.normalized_key] = e.canonical_name
    return tuple(
        DrugEntry(canonical_name=v, normalized_key=k) for k, v in best.items()
    )


@lru_cache(maxsize=1)
def _build_lookup() -> tuple[dict[str, str], dict[str, str], tuple[str, ...], dict[str, tuple[str, ...]]]:
    entries = load_drug_entries()
    key_to_name: dict[str, str] = {}
    simplified_to_name: dict[str, str] = {}
    prefix_buckets: dict[str, list[str]] = {}
    all_keys: list[str] = []

    for e in entries:
        key_to_name[e.normalized_key] = e.canonical_name
        all_keys.append(e.normalized_key)

        simple = _simplify_key_for_match(e.normalized_key)
        if simple and simple not in simplified_to_name:
            simplified_to_name[simple] = e.canonical_name

        prefix = e.normalized_key[:1]
        prefix_buckets.setdefault(prefix, []).append(e.normalized_key)

    frozen_buckets: dict[str, tuple[str, ...]] = {k: tuple(v) for k, v in prefix_buckets.items()}
    return key_to_name, simplified_to_name, tuple(all_keys), frozen_buckets


def normalize_name_with_dictionary(name: str) -> str:
    normalized, _, _ = normalize_name_with_dictionary_meta(name)
    return normalized


def normalize_name_with_dictionary_meta(name: str) -> tuple[str, bool, str]:
    entries = load_drug_entries()
    if not entries:
        return name, False, "dictionary_unavailable"

    key_to_name, simplified_to_name, all_keys, prefix_buckets = _build_lookup()
    src_key = _normalize_drug_key(name)
    if len(src_key) < 3:
        return name, False, "name_too_short"

    # Exact key O(1)
    exact = key_to_name.get(src_key)
    if exact:
        return exact, True, "exact"

    # Exact key sau khi bỏ dấu tiếng Việt.
    ascii_src = _strip_vietnamese_accents(src_key)
    for key, canonical in key_to_name.items():
        if _strip_vietnamese_accents(key) == ascii_src:
            return canonical, True, "exact_ascii"

    # Match theo key rút gọn (bỏ bớt hàm lượng/dạng bào chế).
    simple_src = _simplify_key_for_match(src_key)
    if len(simple_src) >= 3 and simple_src in simplified_to_name:
        return simplified_to_name[simple_src], True, "exact_simplified"

    # Fuzzy key trên tập candidate hẹp hơn để nhanh hơn và ít false positive.
    candidates = list(prefix_buckets.get(src_key[:1], ()))
    if len(candidates) < 10:
        candidates = list(all_keys)
    len_src = len(src_key)
    narrow = [k for k in candidates if abs(len(k) - len_src) <= 8]
    if len(narrow) >= 10:
        candidates = narrow

    strict_cutoff = float(os.getenv("OCR_DRUG_DICT_FUZZY_CUTOFF", "0.88"))
    weak_cutoff = float(os.getenv("OCR_DRUG_DICT_WEAK_CUTOFF", "0.80"))
    match = difflib.get_close_matches(src_key, candidates, n=1, cutoff=strict_cutoff)
    if not match:
        fallback_match = difflib.get_close_matches(src_key, candidates, n=1, cutoff=weak_cutoff)
        if fallback_match:
            return name, False, "fuzzy_below_cutoff"
        return name, False, "not_found"
    best_key = match[0]
    canonical = key_to_name.get(best_key)
    if canonical:
        return canonical, True, "fuzzy"
    return name, False, "not_found"
