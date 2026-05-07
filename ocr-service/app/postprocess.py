import re
from typing import Any
from app.allergy_hints import get_allergy_hints
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

# ── Quality-gate patterns ────────────────────────────────────────────────────
# Matches a letter immediately followed (no mandatory space) by ) and then another letter.
# Catches OCR-glued names like "Metformin)Glumetorm" or "Glielazid ) Glumeron".
_QG_GLUED = re.compile(r"[A-Za-zÀ-ỹ]\s*\)\s*[A-Za-zÀ-ỹ]")
# Symbols that inflate punctuation density.
_QG_SYMBOLS = re.compile(r"[|()/\\]")


def _score_name_quality(name: str) -> tuple[bool, str]:
    """Return (is_suspicious, reason_code) for a drug name string.

    Heuristics — ordered from cheapest to most expensive:
      dangling_bracket   — name ends with a bare '(' (e.g. "Longn(")
      glued_names        — letter ) letter pattern (e.g. "Metformin)Glumetorm")
      high_symbol_density — >18 % non-space chars are punctuation symbols
      too_few_letters    — fewer than 4 alphabetic characters total
    """
    if not name:
        return True, "empty"
    s = name.strip()
    if re.search(r"\(\s*$", s):
        return True, "dangling_bracket"
    if _QG_GLUED.search(s):
        return True, "glued_names"
    nonspace = s.replace(" ", "")
    if nonspace:
        sym_ratio = len(_QG_SYMBOLS.findall(nonspace)) / len(nonspace)
        if sym_ratio > 0.18:
            return True, "high_symbol_density"
    alpha = sum(1 for c in s if c.isalpha())
    if alpha < 4:
        return True, "too_few_letters"
    return False, ""


def _salvage_glued_name(name: str) -> str:
    """Extract the leading clean token from a glued OCR name.

    Splits on ) ( | / \\ and returns the first part with ≥4 alpha chars.
    For "Metformin)Glumetorm" this yields "Metformin" — the generic name
    that usually appears first on Vietnamese prescriptions.
    """
    parts = re.split(r"[)(|/\\]", name)
    for p in parts:
        p = p.strip(" -:;.,")
        if sum(1 for c in p if c.isalpha()) >= 4:
            return p
    return name


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

    # Strip leading non-letter noise (/, (, |, \, etc.) that OCR often prepends.
    text = re.sub(r"^[^A-Za-zÀ-ỹ0-9]+", "", text)
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
    # Guard: không compact khi có token là ký tự đặc biệt (ví dụ ")", "(").
    parts = [p for p in text.split(" ") if p]
    if 2 <= len(parts) <= 3:
        if not any(re.match(r"^[^A-Za-zÀ-ỹ0-9]", p) for p in parts):
            compact = "".join(parts)
            if 6 <= len(compact) <= 20:
                text = compact

    text = re.sub(r"\s{2,}", " ", text).strip(" -:;.,")

    # Strip trailing dangling open bracket produced by OCR (e.g. "Longn(").
    text = re.sub(r"\s*\(\s*$", "", text).strip()

    return text[:220]


def post_process_items(items: list[dict[str, Any]]) -> tuple[list[dict[str, Any]], list[str]]:
    out: list[dict[str, Any]] = []
    warnings: list[str] = []
    for item in items:
        fixed = dict(item)
        raw_name = str(item.get("name", ""))

        # Pre-check: ONLY for trailing dangling "(" which clean_medication_name_surface
        # strips before the quality gate sees it.  We intentionally do NOT inherit the
        # broader glued_names / high_symbol_density checks here, because raw item names
        # often contain trailing schedule text (e.g. "875mg+Acid) mỗi Jan 1") that
        # produces false-positive pattern matches; clean_medication_name_surface already
        # truncates that noise before the post-clean gate runs.
        pre_dangling = bool(re.search(r"\(\s*$", raw_name.strip()))

        cleaned = clean_medication_name_surface(raw_name)
        normalized, matched, match_kind = normalize_name_with_dictionary_meta(cleaned)
        fixed["name"] = normalized
        fixed["name_match_kind"] = match_kind
        # Generic drug-class hints -- not user-specific; FE matches against user profile.
        fixed["allergy_hints"] = get_allergy_hints(normalized)

        # ── Name quality gate ────────────────────────────────────────────────
        # Post-clean check on the final name; also inherits pre-clean flag so
        # that names whose garbage was stripped (e.g. "Longn(" → "Longn") are
        # still flagged for review.
        suspicious, reason = _score_name_quality(fixed["name"])
        if not suspicious and pre_dangling:
            suspicious, reason = True, "dangling_bracket"

        if suspicious:
            # Attempt to salvage glued names before giving up.
            if reason == "glued_names":
                fixed["name"] = _salvage_glued_name(fixed["name"])
            elif reason == "dangling_bracket":
                # Trailing bracket strip may not have fired if the name arrived
                # already post-dict (e.g. dict returned the raw cleaned string).
                fixed["name"] = re.sub(r"\s*\(\s*$", "", fixed["name"]).strip()

            orig_conf = float(fixed.get("confidence", 0.5))
            fixed["confidence"] = round(max(0.2, orig_conf * 0.55), 3)
            fixed["review_needed"] = True
            warnings.append(
                f"Tên thuốc có độ tin cậy thấp, cần kiểm tra lại đơn gốc: '{fixed['name']}'"
            )
        else:
            fixed["review_needed"] = False

        # ── Dictionary-miss warnings ─────────────────────────────────────────
        if not matched and cleaned:
            if match_kind == "fuzzy_below_cutoff":
                warnings.append(
                    f"Tên thuốc '{cleaned}' gần đúng nhưng chưa đủ chắc chắn, cần kiểm tra thủ công"
                )
                fixed["review_needed"] = True
            elif match_kind == "not_found":
                # Emit the "low confidence" phrasing so it appears even when
                # avg_confidence from PaddleOCR is high.
                warnings.append(
                    f"Tên thuốc có độ tin cậy thấp, cần kiểm tra lại đơn gốc: '{cleaned}'"
                )
                fixed["review_needed"] = True
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
