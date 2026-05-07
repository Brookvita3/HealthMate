"""
Prescription text parser.

All medication extraction logic extracted from the original main.py.
Uses constants from app.constants so drug-signal lists are maintained
in one place instead of scattered across multiple files.
"""
from __future__ import annotations

import re
from typing import Any

from app.constants import (
    DEFAULT_TIMES_BY_COUNT,
    KNOWN_DRUG_SIGNALS,
    KNOWN_MEDICINE_RESCUE,
    RE_AFTER_MEAL,
    RE_BEFORE_MEAL,
    RE_DOSE_AFTER_TAKE,
    RE_DOSAGE,
    RE_HEADER_NOISE,
    RE_LINE_NUM_MARKER,
    RE_NAME_CUT,
    RE_NON_MEDICINE_NOISE,
    RE_NUM_MARKER,
    RE_PER_DOSE,
    RE_QUANTITY,
    RE_SUPPLY_QTY,
    RE_TIMES_OF_DAY,
    RE_TIMES_PER_DAY,
    REVERSE_TIME_MAP,
    TIMES_OF_DAY_MAP,
    VN_FIX_PATTERNS,
)
from app.drug_dictionary import normalize_name_with_dictionary_meta
from app.template_hints import (
    looks_like_agifuros_family,
    prescription_item_identity_key,
)

# ---------------------------------------------------------------------------
# Text normalisation
# ---------------------------------------------------------------------------

def normalize_ocr_vi(raw_text: str) -> str:
    """Fix missing Vietnamese diacritics, OCR digit/letter confusions, and whitespace."""
    text = raw_text.replace("\r", "\n")
    lines: list[str] = []
    for line in text.split("\n"):
        line = re.sub(r"\s+", " ", line).strip()
        if not line:
            continue
        for pat, repl in VN_FIX_PATTERNS:
            line = pat.sub(repl, line)
        # OCR commonly confuses 'ố'→'6', 'á'→'4', 'ê'→'e' in time-of-day words.
        # Fix time tokens so dosage extraction can parse them reliably.
        line = re.sub(r"\b[Tt][0O6]\s*i\b", "tối", line)          # T6i/T0i → tối
        line = re.sub(r"\b[Ss][4a][Nn][Gg]\b", "sáng", line, flags=re.I)
        # Normalize parenthesised unit words: "01 (Vien)" → "01 viên"
        line = re.sub(
            r"\(\s*(vi[eê]n|g[oó]i|[oô]ng|ml|gi[oọ]t|chai|l[oọ])\s*\)",
            lambda m: _normalize_dose_unit(m.group(1)),
            line, flags=re.I,
        )
        # Normalize fractional doses written with slash: "1/2 viên" stays as-is;
        # "1/3" is unusual but valid — keep it.
        lines.append(line)
    return "\n".join(lines).strip()


# ---------------------------------------------------------------------------
# Low-level field extractors
# ---------------------------------------------------------------------------

def _normalize_dose_unit(raw_unit: str) -> str:
    u = (raw_unit or "").strip().lower()
    if not u:
        return "viên"
    mapping = {
        "vien": "viên", "viên": "viên",
        "tab": "viên", "tabs": "viên", "tablet": "viên", "tablets": "viên",
        "cap": "viên", "caps": "viên", "capsule": "viên", "capsules": "viên",
        "goi": "gói", "gói": "gói", "sachet": "gói", "sachets": "gói",
        "ong": "ống", "ống": "ống", "ampoule": "ống", "ampoules": "ống",
        "ml": "ml",
        "drop": "giọt", "drops": "giọt", "giot": "giọt", "giọt": "giọt",
        "chai": "chai", "lo": "lọ", "lọ": "lọ",
    }
    return mapping.get(u, u)


def extract_specific_times(text: str) -> list[str]:
    moments = [m.group(1).lower() for m in RE_TIMES_OF_DAY.finditer(text)]
    seen: list[str] = []
    for m in moments:
        if m not in seen:
            seen.append(m)
    return [TIMES_OF_DAY_MAP[m] for m in seen if m in TIMES_OF_DAY_MAP]


def extract_dosage_display(text: str) -> str:
    """
    Extract per-intake quantity in physical units (viên/gói/ống/ml/giọt).

    Never returns mg/strength — the caller wants "how many units to take",
    not "what concentration".  Returns "Theo đơn" when no quantity unit is
    found so the UI can prompt the user to check the original prescription.
    """
    _UNIT_WORDS = {"viên", "gói", "ống", "ml", "giọt", "chai", "lọ"}

    # 1. Explicit "mỗi lần X viên" phrasing
    m = RE_PER_DOSE.search(text)
    if m:
        amount = m.group(1).replace(",", ".")
        unit = _normalize_dose_unit(m.group(2) or "viên")
        return f"{amount} {unit}"

    # 2. "sáng/tối/trưa: X viên" phrasing
    m = RE_DOSE_AFTER_TAKE.search(text)
    if m:
        unit = _normalize_dose_unit(m.group(2))
        if unit in _UNIT_WORDS:
            return f"{m.group(1).replace(',', '.')} {unit}"

    # 3. Any quantity + unit token (strip supply total first)
    no_supply = RE_SUPPLY_QTY.sub(" ", text)
    qty_matches = RE_QUANTITY.findall(no_supply)
    if qty_matches:
        parsed: list[tuple[float, str]] = []
        for amt, unit in qty_matches:
            try:
                parsed.append((float(amt.replace(",", ".")), unit))
            except ValueError:
                continue
        if parsed:
            # Threshold varies by unit: liquid units allow up to 20 per dose
            # (e.g. 5ml siro, 10 giọt), solid units (viên/gói/ống) stay at 4.
            def _is_per_dose(amt: float, raw_unit: str) -> bool:
                u = _normalize_dose_unit(raw_unit)
                return 0 < amt <= (20 if u in {"ml", "giọt"} else 4)

            per_dose = next((p for p in parsed if _is_per_dose(p[0], p[1])), parsed[0])
            unit = _normalize_dose_unit(per_dose[1])
            if unit in _UNIT_WORDS:
                amount = str(int(per_dose[0])) if float(per_dose[0]).is_integer() else str(per_dose[0])
                return f"{amount} {unit}"

    # 4. No unit information found — do NOT guess "1 viên" or expose mg
    return "Theo đơn"


def extract_strength_display(text: str) -> str:
    matches = RE_DOSAGE.findall(text)
    if not matches:
        return ""
    strengths: list[str] = []
    seen: set[str] = set()
    for m in matches:
        val = re.sub(r"\s+", " ", str(m)).replace(",", ".").strip()
        low = val.lower()
        if low in seen:
            continue
        seen.add(low)
        strengths.append(val)
        if len(strengths) >= 3:
            break
    return " + ".join(strengths)


def _extract_meal_timing(text: str) -> str:
    if RE_BEFORE_MEAL.search(text):
        return "trước ăn"
    if RE_AFTER_MEAL.search(text):
        return "sau ăn"
    return "không rõ"


def build_instructions(text: str, specific_times: list[str], times_per_day: int) -> str:
    meal = f"Cách dùng: {_extract_meal_timing(text)}"
    slots = ""
    if specific_times:
        labels = [REVERSE_TIME_MAP.get(t, t) for t in specific_times]
        slots = f"Giờ dùng: {', '.join(labels)}"
    freq = f"Ngày uống: {times_per_day} lần"
    return "\n".join(p for p in [meal, slots, freq] if p)


def estimate_confidence(name: str, specific_times: list[str], dosage: str) -> float:
    score = 0.45
    if len(name) >= 10:
        score += 0.15
    if any(ch.isalpha() for ch in name):
        score += 0.10
    if any(ch.isdigit() for ch in dosage):
        score += 0.15
    if specific_times:
        score += 0.10
    if re.search(r"[^\w\s\-\+\(\)\.,:/]", name):
        score -= 0.10
    return max(0.2, min(score, 0.98))


# ---------------------------------------------------------------------------
# Name cleaning / canonicalisation
# ---------------------------------------------------------------------------

def clean_name(raw: str) -> str:
    text = re.sub(r"\s+", " ", raw).strip()
    # Strip numbered list prefix
    text = re.sub(r"^\s*[\W_]{0,6}\s*\d{1,2}\s*(?:[.)/\-]\s*|\s*)", "", text)
    # Drop hospital/order codes at head: "(182)…", "711)…"
    text = re.sub(r"^\s*[\(\[]?\s*\d{2,4}\s*[\)\].:/-]\s*", "", text)
    # Remove leading non-alphanumeric characters
    text = re.sub(r"^[^A-Za-zÀ-ỹ0-9]+", "", text)
    # Cut at scheduling keywords
    cut = RE_NAME_CUT.search(text)
    if cut:
        text = text[: cut.start()]
    # Remove trailing quantity tail: "06 viên", "03 gói"
    text = re.sub(r"\s+\d{1,3}\s*(?:vi[eê]n|g[oó]i)\b.*$", "", text, flags=re.I)
    text = re.sub(r"\s+\d{1,3}$", "", text)
    text = re.sub(r"[^\wÀ-ỹ\-\+\(\)\s\.,/]", " ", text).strip()
    text = re.sub(r"\s{2,}", " ", text)
    # OCR confusion: '0' inside a letter run is usually 'O'
    text = re.sub(r"(?<=[A-Za-zÀ-ỹ])0(?=[A-Za-zÀ-ỹ])", "O", text)
    # Normalise letter/digit boundaries
    text = re.sub(r"([A-Za-zÀ-ỹ])(\d)", r"\1 \2", text)
    text = re.sub(r"(\d)([A-Za-zÀ-ỹ])", r"\1 \2", text)
    text = re.sub(r"\s{2,}", " ", text)
    # Truncate before trailing quantity if any
    long_cut = re.search(r"\b\d{1,3}\s*(?:vi[eê]n|g[oó]i|[oô]ng)\b", text, flags=re.I)
    if long_cut:
        text = text[: long_cut.start()].strip()
    return text.strip(" -:;.,")[:220]


def canonicalize_name(name: str, block: str) -> str:
    low_block = block.lower()
    if looks_like_agifuros_family(low_block) or looks_like_agifuros_family(name.lower()):
        return "AGIFUROS 40mg (Furosemid)"
    out = re.sub(r"^\s*(?:mg|ml|g)\b", "", name, flags=re.I).strip()
    out = re.sub(r"\bcong\s*kh.*$", "", out, flags=re.I).strip()
    return (out or name)[:220]


def _strip_line_number_marker(line: str) -> str:
    return re.sub(r"^\s*[\W_]{0,6}\s*\d{1,2}\s*(?:[.)/\-]\s*|\s+)", "", line).strip()


# ---------------------------------------------------------------------------
# Block / line classification
# ---------------------------------------------------------------------------

def _line_looks_like_numbered_med(line: str) -> bool:
    return bool(re.match(r"^\s*[\W_]{0,4}\s*\d{1,2}\s*[.)/\-]\s*(?=[A-Za-zÀ-ỹ])", line))


def line_has_drug_signal(line: str) -> bool:
    """True when this line should be kept as a potential medication line."""
    low = line.lower()
    if any(k in low for k in KNOWN_DRUG_SIGNALS):
        return True
    if re.search(r"\b(?:\d+(?:[.,]\d+)?\s*(?:mg|ml|mcg|iu|ui)|viên|vien)\b", line, re.I):
        return True
    if re.search(r"\bSL\s*[:.]?\s*\d", line, re.I):
        return True
    return _line_looks_like_numbered_med(line)


def _looks_like_medicine_block(block: str) -> bool:
    low = block.lower()
    has_signal = any(k in low for k in KNOWN_DRUG_SIGNALS)
    if not has_signal:
        has_signal = bool(re.search(r"\d+(?:[.,]\d+)?\s*(?:mg|ml|mcg|iu|ui)\b", low))
        if has_signal and len(low) <= 25:
            has_signal = False
    if not has_signal:
        return False
    # Long blocks with a header noise token are still valid if they also contain mg/ml
    if RE_HEADER_NOISE.search(low):
        if len(low) < 90 or not RE_DOSAGE.search(low):
            return False
    if RE_NON_MEDICINE_NOISE.search(low) and "mg" not in low and "viên" not in low:
        return False
    return True


def _line_is_possible_medication(line: str) -> bool:
    low = line.lower()
    if len(low) < 7:
        return False
    if RE_NUM_MARKER.match(low):
        return True
    if re.search(r"\d+(?:[.,]\d+)?\s*(?:mg|g|ml|mcg|iu|ui)\b", low):
        return True
    if line_has_drug_signal(low):
        return True
    letters = re.sub(r"[^a-zà-ỹ]+", "", low)
    if 6 <= len(letters) <= 24 and len(letters) >= int(len(low.replace(" ", "")) * 0.65):
        if not RE_HEADER_NOISE.search(low) and not RE_NON_MEDICINE_NOISE.search(low):
            return True
    return False


def _is_plausible_medication_item(item: dict[str, Any]) -> bool:
    """Filter out junk blocks — keep only items with clear medication signals."""
    name = (item.get("name") or "").strip()
    if len(name) < 5:
        return False
    if re.fullmatch(r"[\d\s\W_()]+", name):
        return False
    nlow = name.lower()
    if any(k in nlow for k in KNOWN_DRUG_SIGNALS):
        return True
    alpha_chunks = re.findall(r"[a-zà-ỹ]+", nlow)
    alpha_join = "".join(alpha_chunks)
    if len(alpha_join) < 6:
        return False
    if alpha_chunks and max(len(p) for p in alpha_chunks) < 4:
        return False
    if re.search(r"\d+(?:[.,]\d+)?\s*(?:mg|ml|mcg|iu|ui)\b", nlow):
        return True
    if re.search(r"\bviên\b|\bvien\b|\bgói\b|\bống\b", nlow):
        return True
    return False


# ---------------------------------------------------------------------------
# Single-item parser
# ---------------------------------------------------------------------------

def _build_item(
    name: str, block: str, *, confidence_override: float | None = None
) -> dict[str, Any] | None:
    """Assemble a medication item dict from name + surrounding block text."""
    if len(name) < 3:
        return None
    dosage = extract_dosage_display(block)
    strength = extract_strength_display(block)
    specific_times = extract_specific_times(block)
    times_per_day = 1
    m = RE_TIMES_PER_DAY.search(block)
    if m:
        times_per_day = max(1, min(int(m.group(1)), 6))
    elif specific_times:
        times_per_day = len(specific_times)
    if not specific_times:
        specific_times = DEFAULT_TIMES_BY_COUNT.get(times_per_day, ["08:00"])

    conf = confidence_override if confidence_override is not None else (
        estimate_confidence(name, specific_times, dosage)
    )
    return {
        "name": name,
        "dosage": dosage,
        "strength": strength,
        "instructions": build_instructions(block, specific_times, times_per_day),
        "times_per_day": times_per_day,
        "specific_times": specific_times,
        "confidence": round(conf, 3),
    }


def _parse_item(block: str) -> dict[str, Any] | None:
    normalized = re.sub(r"\s+", " ", block).strip()
    normalized = re.sub(r"^\s*[\W_]{0,4}\s*\d{1,2}\s*(?:[.)/\-]\s*|\s+)", "", normalized)
    if not normalized or not _looks_like_medicine_block(normalized):
        return None
    name = canonicalize_name(clean_name(normalized), normalized)
    return _build_item(name, normalized)


# ---------------------------------------------------------------------------
# Block splitters
# ---------------------------------------------------------------------------

def _split_blocks(raw_text: str) -> list[str]:
    text = raw_text.strip()
    if not text:
        return []

    # 1. Primary split: line-start numbered markers
    lines = [ln.strip() for ln in text.splitlines() if ln.strip()]
    blocks: list[str] = []
    buf: list[str] = []
    started = False
    for ln in lines:
        if RE_NUM_MARKER.match(ln):
            if started and buf:
                blocks.append("\n".join(buf))
                buf = []
            started = True
        if started:
            buf.append(ln)
    if buf:
        blocks.append("\n".join(buf))
    if len(blocks) >= 2:
        nested_pat = re.compile(
            r"(?:^|\s)[\|\\/\[\(\{]*\s*\d{1,2}\s*(?:[.)/\-]|(?=\s*[A-Za-zÀ-ỹ]))"
        )
        exploded: list[str] = []
        for b in blocks:
            nested = list(nested_pat.finditer(b))
            if len(nested) <= 1:
                exploded.append(b)
                continue
            for i, m in enumerate(nested):
                end = nested[i + 1].start() if i + 1 < len(nested) else len(b)
                part = b[m.start():end].strip()
                if part:
                    exploded.append(part)
        return exploded or blocks

    # 2. Inline marker split
    inline_pat = re.compile(
        r"(?:(?<=^)|(?<=\s)|(?<=\n))[\|\\/\[\(\{]*\s*\d{1,2}\s*(?:[.)/\-]|(?=\s*[A-Za-zÀ-ỹ]))"
    )
    matches = list(inline_pat.finditer(text))
    if len(matches) >= 2:
        parts: list[str] = []
        for i, m in enumerate(matches):
            end = matches[i + 1].start() if i + 1 < len(matches) else len(text)
            part = text[m.start():end].strip()
            if part:
                parts.append(part)
        if parts:
            return parts

    # 3. Last-resort: split on leading digit + letter
    fallback = re.split(r"(?=(?:^|[\s\n])\d{1,2}\s+[A-Za-zÀ-ỹ])", text)
    return [b.strip() for b in fallback if b.strip()]


def _split_blocks_aggressive(text: str) -> list[str]:
    """Aggressive split by numbered marker — used when expected_count > found."""
    t = text.strip()
    if not t:
        return []
    pat = re.compile(
        r"(?=^\s*[\W_]{0,4}\s*\d{1,2}\s*(?:[.)]\s*|\s+(?=[A-Za-zÀ-ỹ]{3,})))",
        re.MULTILINE,
    )
    starts = [m.start() for m in pat.finditer(t)]
    if len(starts) < 2:
        return []
    parts: list[str] = []
    for i, start in enumerate(starts):
        end = starts[i + 1] if i + 1 < len(starts) else len(t)
        chunk = t[start:end].strip()
        if chunk:
            parts.append(chunk)
    return parts


# ---------------------------------------------------------------------------
# Block-based parser
# ---------------------------------------------------------------------------

def parse_items_from_text(normalized_text: str) -> list[dict[str, Any]]:
    items: list[dict[str, Any]] = []
    for block in _split_blocks(normalized_text):
        parsed = _parse_item(block)
        if parsed:
            items.append(parsed)
    return items


# ---------------------------------------------------------------------------
# Numbered-group parsers
# ---------------------------------------------------------------------------

def _split_numbered_line_groups(text: str) -> list[list[str]]:
    lines = [re.sub(r"\s+", " ", ln).strip() for ln in text.splitlines() if ln.strip()]
    groups: list[list[str]] = []
    current: list[str] = []
    for ln in lines:
        if RE_LINE_NUM_MARKER.match(ln):
            if current:
                groups.append(current)
            current = [ln]
        elif current:
            current.append(ln)
    if current:
        groups.append(current)
    return groups


def _group_marker_no(group: list[str]) -> int | None:
    if not group:
        return None
    m = RE_LINE_NUM_MARKER.match(group[0])
    if not m:
        return None
    try:
        return int(m.group(1))
    except (TypeError, ValueError):
        return None


def extract_numbered_markers(text: str) -> list[int]:
    markers: list[int] = []
    for group in _split_numbered_line_groups(text):
        no = _group_marker_no(group)
        if no is not None:
            markers.append(no)
    return markers


def is_reliable_numbered_sequence(markers: list[int]) -> bool:
    """Return True only for 1,2,3,… sequences — avoids mistaking hospital codes."""
    if len(markers) < 2:
        return False
    uniq = sorted(set(markers))
    if len(uniq) < 2 or uniq[0] != 1:
        return False
    if uniq[-1] > len(uniq) + 3:
        return False
    small = sum(1 for i in range(1, len(uniq)) if 1 <= uniq[i] - uniq[i - 1] <= 2)
    return small >= max(1, len(uniq) - 2)


def _extract_name_from_group(lines: list[str]) -> str:
    if not lines:
        return ""
    first = _strip_line_number_marker(lines[0])
    first = re.sub(r"\s+\d{1,3}\s*(?:vi[eê]n|g[oó]i|[oô]ng)\b.*$", "", first, flags=re.I)
    first = clean_name(first)
    if len(first) >= 4:
        return first

    def _score(text: str) -> tuple[int, int]:
        has_strength = 1 if re.search(r"\d+(?:[.,]\d+)?\s*(?:mg|g|ml|mcg|iu|ui)\b", text.lower()) else 0
        looks_med = 1 if _line_is_possible_medication(text) else 0
        return (has_strength + looks_med, len(text))

    candidates: list[str] = []
    for ln in lines[1:]:
        cand = re.sub(r"\s+\d{1,3}\s*(?:vi[eê]n|g[oó]i|[oô]ng)\b.*$", "", ln.strip(), flags=re.I)
        cand = clean_name(cand)
        if len(cand) >= 4:
            candidates.append(cand)
    return max(candidates, key=_score) if candidates else ""


def _parse_item_from_group(lines: list[str]) -> dict[str, Any] | None:
    if not lines:
        return None
    block = "\n".join(lines).strip()
    if not block:
        return None
    med_like = _looks_like_medicine_block(block)
    raw_name = _extract_name_from_group(lines)
    if not med_like and len(raw_name) < 4:
        return None
    name = canonicalize_name(raw_name, block) if raw_name else ""
    return _build_item(name, block)


def parse_items_from_numbered_lines(text: str) -> list[dict[str, Any]]:
    groups = _split_numbered_line_groups(text)
    if not groups:
        return []
    by_marker: dict[int, dict[str, Any]] = {}
    order: list[int] = []
    extra: list[dict[str, Any]] = []

    for group in groups:
        item = _parse_item_from_group(group)
        if not item:
            continue
        no = _group_marker_no(group)
        if no is None:
            extra.append(item)
            continue
        if no not in by_marker:
            by_marker[no] = item
            order.append(no)
        else:
            prev = by_marker[no]
            if float(item.get("confidence", 0)) > float(prev.get("confidence", 0)):
                by_marker[no] = item
            elif float(item.get("confidence", 0)) == float(prev.get("confidence", 0)):
                if len(str(item.get("name", ""))) > len(str(prev.get("name", ""))):
                    by_marker[no] = item

    result = [by_marker[m] for m in sorted(order)] + extra
    return dedupe_prescription_items(result)


def parse_items_from_numbered_head_lines(text: str) -> list[dict[str, Any]]:
    """
    Minimal parser: only the marker line itself (1. DrugName…) — not the
    subsequent context lines.  Fewer false positives for noisy OCR output.
    """
    lines = [re.sub(r"\s+", " ", ln).strip() for ln in text.splitlines() if ln.strip()]
    seen: set[int] = set()
    rows: list[tuple[int, dict[str, Any]]] = []

    for ln in lines:
        m = RE_LINE_NUM_MARKER.match(ln)
        if not m:
            continue
        try:
            no = int(m.group(1))
        except (TypeError, ValueError):
            continue
        if no in seen:
            continue

        raw_name = clean_name(_strip_line_number_marker(ln))
        if len(raw_name) < 3:
            continue
        letters_only = re.sub(r"[^a-zà-ỹ]+", "", raw_name.lower())
        known = any(k in raw_name.lower() for k in KNOWN_DRUG_SIGNALS)
        if len(letters_only) < 5 and not known:
            continue

        name = canonicalize_name(raw_name, ln)
        item = _build_item(name, ln)
        if item:
            rows.append((no, item))
            seen.add(no)

    rows.sort(key=lambda x: x[0])
    return dedupe_prescription_items([it for _, it in rows])


# ---------------------------------------------------------------------------
# Relaxed / tabular parsers (fallback)
# ---------------------------------------------------------------------------

def parse_items_relaxed_from_lines(text: str) -> list[dict[str, Any]]:
    """Line-by-line fallback for heavily fragmented OCR output."""
    picked: list[dict[str, Any]] = []
    for ln in (ln.strip() for ln in text.splitlines() if ln.strip()):
        if not _line_is_possible_medication(ln):
            continue
        candidate = _parse_item(ln)
        if candidate is None:
            raw_name = clean_name(ln)
            if len(raw_name) < 4:
                continue
            canon = canonicalize_name(raw_name, ln)
            dm = RE_DOSAGE.search(ln)
            dosage = dm.group(1).replace(",", ".") if dm else "Theo đơn"
            times = extract_specific_times(ln) or ["08:00"]
            candidate = _build_item(canon, ln, confidence_override=0.46)
            if candidate:
                candidate["dosage"] = dosage
        if candidate:
            picked.append(candidate)
    return dedupe_prescription_items(picked)


def parse_items_from_tabular_lines(text: str) -> list[dict[str, Any]]:
    """Last-resort recovery for table-format prescriptions."""
    lines = [re.sub(r"\s+", " ", ln).strip() for ln in text.splitlines() if ln.strip()]
    if not lines:
        return []

    qty_re = re.compile(r"\b(?:s[oố]\s*l[uư][oợ]ng|sl|sig)\b", re.I)
    header_re = re.compile(
        r"\b(?:đơn thuốc|toa thuốc|ngày tái khám|bác sĩ|họ tên|địa chỉ)\b", re.I
    )
    candidates: list[dict[str, Any]] = []
    for i, ln in enumerate(lines):
        if header_re.search(ln) or not qty_re.search(ln):
            continue
        left = re.split(r"\b(?:s[oố]\s*l[uư][oợ]ng|sl|sig)\b", ln, maxsplit=1, flags=re.I)[0].strip()
        prev = lines[i - 1] if i > 0 else ""
        next_ln = lines[i + 1] if i + 1 < len(lines) else ""
        for opt in [left, prev, next_ln]:
            if not opt or len(opt) < 5:
                continue
            raw = clean_name(opt)
            if len(raw) < 4 or header_re.search(raw):
                continue
            canon = canonicalize_name(raw, opt)
            times = extract_specific_times(f"{opt} {ln}") or ["08:00"]
            item = _build_item(canon, f"{opt} {ln}", confidence_override=0.42)
            if item:
                candidates.append(item)
            break
    return dedupe_prescription_items(candidates)


# ---------------------------------------------------------------------------
# Deduplication
# ---------------------------------------------------------------------------

def _name_signature(name: str) -> str:
    s = (name or "").lower()
    s = re.sub(r"\([^)]*\)", " ", s)
    s = re.sub(r"\b\d+(?:[.,]\d+)?\s*(?:mg|g|ml|mcg|iu|ui)\b", " ", s)
    s = re.sub(r"[^a-z0-9à-ỹ]+", " ", s)
    return re.sub(r"\s+", " ", s).strip()


def dedupe_items_by_signature(items: list[dict[str, Any]]) -> list[dict[str, Any]]:
    order: list[str] = []
    best: dict[str, dict[str, Any]] = {}
    for item in items:
        sig = _name_signature(str(item.get("name", "")))
        if not sig:
            sig = f"raw:{str(item.get('name', ''))[:40].lower()}"
        if sig not in best:
            best[sig] = item
            order.append(sig)
            continue
        prev = best[sig]
        prev_c = float(prev.get("confidence", 0))
        cur_c  = float(item.get("confidence", 0))
        if cur_c > prev_c:
            best[sig] = item
        elif cur_c == prev_c and len(str(item.get("name", ""))) > len(str(prev.get("name", ""))):
            best[sig] = item
    return [best[k] for k in order]


def dedupe_prescription_items(items: list[dict[str, Any]]) -> list[dict[str, Any]]:
    order: list[str] = []
    best: dict[str, dict[str, Any]] = {}
    for it in items:
        if not _is_plausible_medication_item(it):
            continue
        key = prescription_item_identity_key(it)
        if key not in best:
            order.append(key)
            best[key] = it
            continue
        prev = best[key]
        prev_c = float(prev.get("confidence", 0))
        new_c  = float(it.get("confidence", 0))
        if new_c > prev_c:
            best[key] = it
        elif new_c == prev_c and len(str(it.get("name", ""))) > len(str(prev.get("name", ""))):
            best[key] = it
    return [best[k] for k in order]


# ---------------------------------------------------------------------------
# Recovery helpers
# ---------------------------------------------------------------------------

def recover_items_from_dictionary_lines(
    text: str,
    items: list[dict[str, Any]],
    *,
    max_added: int = 6,
) -> list[dict[str, Any]]:
    """
    Recall-first rescue: scan each line, match against the drug dictionary,
    and add any medication found that isn't already in *items*.
    """
    lines = [re.sub(r"\s+", " ", ln).strip() for ln in text.splitlines() if ln.strip()]
    if not lines:
        return items

    existing_keys = {
        re.sub(r"[^a-z0-9à-ỹ]+", "", str(it.get("name", "")).lower())
        for it in items
        if str(it.get("name", "")).strip()
    }
    recovered: list[dict[str, Any]] = []
    added = 0

    for ln in lines:
        if added >= max_added:
            break
        if not (_line_is_possible_medication(ln) or RE_LINE_NUM_MARKER.match(ln)):
            continue
        raw = clean_name(_strip_line_number_marker(ln))
        if len(raw) < 4:
            continue
        normalized, matched, kind = normalize_name_with_dictionary_meta(raw)
        if not matched or kind not in {"exact", "exact_ascii", "exact_simplified", "fuzzy", "fuzzy_weak"}:
            continue
        key = re.sub(r"[^a-z0-9à-ỹ]+", "", str(normalized).lower())
        if len(key) < 3 or key in existing_keys:
            continue

        item = _build_item(normalized, ln, confidence_override=0.42)
        if item:
            recovered.append(item)
            existing_keys.add(key)
            added += 1

    return dedupe_items_by_signature([*items, *recovered]) if recovered else items


def recover_known_meds_from_text(
    text: str, items: list[dict[str, Any]]
) -> list[dict[str, Any]]:
    """
    Light heuristic rescue: when very few items found, pattern-match raw text
    against a pre-compiled list of known medications.
    Only adds, never removes existing items.
    """
    if len(items) > 2 or len(text) < 500:
        return items

    low = text.lower()
    existing_keys = {prescription_item_identity_key(it) for it in items}
    recovered: list[dict[str, Any]] = []

    for kid, pat, canonical in KNOWN_MEDICINE_RESCUE:
        if kid in existing_keys:
            continue
        if not pat.search(low):
            continue
        item = _build_item(canonical, canonical, confidence_override=0.43)
        if item:
            recovered.append(item)
            existing_keys.add(kid)
        if len(items) + len(recovered) >= 6:
            break

    return dedupe_prescription_items([*items, *recovered]) if recovered else items


# ---------------------------------------------------------------------------
# Parser selection — single function replacing the tangled if/elif chain
# ---------------------------------------------------------------------------

def select_best_items(
    block_items: list[dict[str, Any]],
    numbered_items: list[dict[str, Any]],
    head_items: list[dict[str, Any]],
    *,
    marker_reliable: bool,
    marker_count: int,
    marker_rich: bool,
    prioritize_recall: bool,
) -> list[dict[str, Any]]:
    """
    Deterministic selection logic (replaces the original 40-line spaghetti).

    Priority (highest first):
      1. If marker sequence is reliable → prefer the parser that found the most
         items; in recall mode also merge all three parsers.
      2. If markers exist but sequence is unreliable (marker_rich) → use the
         largest result set.
      3. Otherwise → use block_items as-is.

    After selection, cap the list at marker_count when the sequence is reliable
    (prevents adding phantom drugs beyond what the prescription lists).
    """
    if not marker_reliable and not marker_rich:
        return block_items

    best = block_items

    # Prefer any parser that found at least 2 items and more than current best
    for candidate in (head_items, numbered_items):
        if len(candidate) >= 2 and len(candidate) >= len(best):
            best = candidate

    if prioritize_recall:
        # Merge all three parsers: maximise how many drugs we detect
        merged = dedupe_items_by_signature([*head_items, *numbered_items, *block_items])
        if len(merged) > len(best):
            best = merged

    # Hard cap: reliable sequence means we know exactly how many drugs there are
    if marker_reliable and marker_count >= 2 and len(best) > marker_count:
        best = best[:marker_count]

    return best
