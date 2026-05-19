import re
from typing import Any
from app.allergy_hints import get_allergy_hints
from app.constants import KNOWN_DRUG_SIGNALS
from app.drug_dictionary import normalize_name_with_dictionary_meta

_med_name_typo_fixes: list[tuple[re.Pattern[str], str]] = [
    (re.compile(r"\baumgmentin\b", re.I), "Augmentin"),
    (re.compile(r"\bauqmentin\b", re.I), "Augmentin"),
    (re.compile(r"\baugmentln\b", re.I), "Augmentin"),
    (re.compile(r"\bacemuc\b", re.I), "Acemuc"),
    (re.compile(r"\bpropanolol\b", re.I), "Propranolol"),
    (re.compile(r"\bvag[a3e]?sta[ti7]?\b", re.I), "Sucralfat"),
    (re.compile(r"\bpanto[a-z0-9]{0,15}pantoc[a-z0-9]{0,15}\b", re.I), "Pantoprazole"),
    (re.compile(r"\bpantopra\w*\b", re.I), "Pantoprazole"),
    (re.compile(r"\bpantoc\b|\bpantoloc\b|\bpantozol\b|\bpantac\b", re.I), "Pantoprazole"),
    (re.compile(r"\bsu?x?ra?l?a?t?\b", re.I), "Sucralfat"),
    # PPI generic names: VN spelling drops trailing 'e' but labels use international INN
    (re.compile(r"\besomeprazol(?!e)\b", re.I), "Esomeprazole"),
    (re.compile(r"\bomeprazol(?!e)\b", re.I), "Omeprazole"),
    (re.compile(r"\blansoprazol(?!e)\b", re.I), "Lansoprazole"),
    (re.compile(r"\brabeprazol(?!e)\b", re.I), "Rabeprazole"),
]


# ---------------------------------------------------------------------------
# Generic-from-parentheses extraction
# ---------------------------------------------------------------------------

# Matches full parenthesised block: (content)
_PAREN_FULL_RE = re.compile(r"\(([A-Za-zÀ-ỹ][^)]{5,60})\)")
# Matches partial: content ending with ')' after digit/space (OCR-dropped opening paren)
_PAREN_PARTIAL_RE = re.compile(r"(?:^|\d|\s)([A-Za-zÀ-ỹ][A-Za-zÀ-ỹ]{4,30})\)")


def _extract_paren_generic(text: str) -> str | None:
    """Return a generic drug name from parenthesised content, or None.

    Tries two strategies:
      1. Full parens  (GENERIC NAME) — all-caps or has a known drug signal.
      2. Partial paren GenericName)  — OCR dropped the opening '('; must have a signal.
    """
    # Strategy 1: full parentheses
    for m in _PAREN_FULL_RE.finditer(text):
        content = m.group(1).strip()
        alpha_len = sum(1 for c in content if c.isalpha())
        if alpha_len < 5:
            continue
        c_lower = content.lower()
        has_signal = any(k in c_lower for k in KNOWN_DRUG_SIGNALS)
        is_all_caps = sum(1 for c in content if c.isupper()) >= alpha_len * 0.7
        if has_signal or is_all_caps:
            return content
    # Strategy 2: partial paren (OCR-dropped opening)
    for m in _PAREN_PARTIAL_RE.finditer(text):
        content = m.group(1).strip()
        if any(k in content.lower() for k in KNOWN_DRUG_SIGNALS):
            return content
    return None


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
# Names that start with a Vietnamese schedule/dosage keyword (e.g. "ngày U ng").
_QG_SCHEDULE_START = re.compile(
    r"^\s*(?:ng[aà]y|u[oố]ng|m[oỗ]i\s*l[aầ]n|s[aá]ng|tr[ưu]a|chi[eề]u|t[oố]i|c[aá]ch\s*d[uù]ng)\b",
    re.I,
)
_QG_ALPHA_ONLY = re.compile(r"[^a-zà-ỹ]", re.I)


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
    if _QG_SCHEDULE_START.match(s):
        # Only reject if there's no long non-schedule token (>5 alpha chars).
        # Labels like "Toi Trua Celecoxib" contain a real drug name — keep them.
        tokens = s.split()
        has_drug_token = any(
            len(_QG_ALPHA_ONLY.sub("", t)) > 5 for t in tokens
        )
        if not has_drug_token:
            return True, "schedule_noise"
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
        brand_part = text[: dose_match.start()].strip()
        # When the brand has no known drug signal, prefer generic name from parentheses
        # (e.g. "Megistan 300mg (URSODEOXYCHOLIC ACID)" → "URSODEOXYCHOLIC ACID")
        brand_has_signal = any(k in brand_part.lower() for k in KNOWN_DRUG_SIGNALS)
        if not brand_has_signal:
            after_dose = text[dose_match.end():]
            generic = _extract_paren_generic(after_dose)
            if generic:
                return generic.strip(" -:;.,")
        text = brand_part

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
    # Guard: chỉ compact khi token đầu ngắn (≤4 ký tự) — tránh ghép tên 2 từ
    # hợp lệ như "Bisoprolol fumarat" thành "Bisoprololfumarat".
    parts = [p for p in text.split(" ") if p]
    if 2 <= len(parts) <= 3 and len(parts[0]) <= 4:
        if not any(re.match(r"^[^A-Za-zÀ-ỹ0-9]", p) for p in parts):
            compact = "".join(parts)
            if 6 <= len(compact) <= 20:
                text = compact

    text = re.sub(r"\s{2,}", " ", text).strip(" -:;.,")

    # Strip trailing dangling open bracket produced by OCR (e.g. "Longn(").
    text = re.sub(r"\s*\(\s*$", "", text).strip()

    # Second pass of typo fixes: catches variants that only appear after compact/trim
    # (e.g. "Vag 3 stat" → truncate → digit-strip → compact → "Vagstat" → fixed here).
    for pat, repl in _med_name_typo_fixes:
        text = pat.sub(repl, text)

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
            # Schedule-noise items are unsalvageable — drop entirely.
            if reason == "schedule_noise":
                continue
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

    # Subset-dedup: remove items whose simplified token set is a strict subset of
    # another item's token set.  Prevents emitting both "Amlodipine" and
    # "Amlodipine besylate" (the latter is more specific and subsumes the former).
    def _tok_set(name: str) -> frozenset[str]:
        s = re.sub(r"\d+(?:[.,]\d+)?\s*(?:mg|g|ml|mcg|iu|ui)(?:/\S+)?", " ", name, flags=re.I)
        s = re.sub(r"[^a-zà-ỹ]+", " ", s.lower())
        return frozenset(t for t in s.split() if len(t) >= 2)

    tok_sets = [_tok_set(str(it.get("name", ""))) for it in deduped]
    dominated = set()
    for i in range(len(deduped)):
        if not tok_sets[i]:
            continue
        for j in range(len(deduped)):
            if i == j or j in dominated:
                continue
            if tok_sets[i] < tok_sets[j]:   # strict subset → i is less specific than j
                dominated.add(i)
                break
    deduped = [it for k, it in enumerate(deduped) if k not in dominated]

    unique_warnings: list[str] = []
    for w in warnings:
        if w not in unique_warnings:
            unique_warnings.append(w)
    return deduped, unique_warnings[:8]
