"""
prescription_parser.py — Raw OCR string → Structured JSON (HealthMate schema)
==============================================================================

Public API
----------
    parse_prescription_text(raw_text: str) -> dict

Input:  raw OCR text (plain string from any OCR engine or VL model)
Output: HealthMate prescription dict (same schema as /ocr/prescriptions/parse)

Schema
------
{
    "raw_text": str,                  # normalized OCR text
    "items": [
        {
            "name": str,              # normalized drug name
            "dosage": str,            # "1 viên", "2 viên", ...
            "instructions": str,      # human-readable instructions
            "times_per_day": int,
            "specific_times": list[str],  # ["08:00", "20:00", ...]
            "confidence": float,      # 0.0 – 1.0
            "name_match_kind": str,   # "exact"/"fuzzy"/"dictionary_unavailable"
        }, ...
    ],
    "warnings": list[str],
    "meta": {
        "items_count": int,
        "parser_used": str,           # which parser strategy won
    }
}

Usage example
-------------
    from app.prescription_parser import parse_prescription_text

    result = parse_prescription_text(raw_ocr_output)
    items = result["items"]
    assert isinstance(result, dict)
    assert all(isinstance(i["name"], str) for i in items)

Note on VL model output
-----------------------
PaddleOCR-VL (d-dev) may return markdown-fenced JSON like:

    ```json
    { "drugs": [ ... ] }
    ```

This module automatically strips fences and falls back to text parsing if the
payload is not recognisable JSON (or if it uses a non-standard schema).

Constraints
-----------
- No side effects: pure text → dict, no I/O.
- Thread-safe: no global mutable state.
- Raises only ValueError for completely empty/invalid input.
"""
from __future__ import annotations

import json
import re
from typing import Any

from app.parser import (
    extract_numbered_markers,
    is_reliable_numbered_sequence,
    normalize_ocr_vi,
    parse_items_from_numbered_head_lines,
    parse_items_from_numbered_lines,
    parse_items_from_text,
    parse_items_relaxed_from_lines,
)
from app.postprocess import post_process_items
from app.template_hints import prescription_item_identity_key  # noqa: F401 (may be unused)

__all__ = ["parse_prescription_text", "strip_markdown_fence", "vl_json_to_items"]


# ---------------------------------------------------------------------------
# 1. Markdown fence stripper
# ---------------------------------------------------------------------------

_FENCE_RE = re.compile(
    r"^```(?:json|JSON)?\s*\n(.*?)\n?```\s*$",
    re.DOTALL,
)


def strip_markdown_fence(text: str) -> str:
    """Remove ``` ... ``` code fences that a VL model may wrap around its response.

    Returns the inner content unchanged if no fence is present.

    >>> strip_markdown_fence("```json\\n{}\n```")
    '{}'
    >>> strip_markdown_fence("plain text")
    'plain text'
    """
    m = _FENCE_RE.match(text.strip())
    return m.group(1).strip() if m else text.strip()


# ---------------------------------------------------------------------------
# 2. VL JSON → items (when the model returns structured JSON directly)
# ---------------------------------------------------------------------------

# Expected keys from a VL model that returns JSON instead of OCR text
_VL_JSON_NAME_KEYS = ("name", "drug_name", "ten_thuoc", "medication")
_VL_JSON_DOSE_KEYS = ("dosage", "dose", "lieu_dung", "lieu")
_VL_JSON_INST_KEYS = ("instructions", "instruction", "huong_dan", "cach_dung")


def _coerce_item(raw: Any) -> dict[str, Any] | None:
    """Convert a raw VL JSON item (dict) to HealthMate item schema."""
    if not isinstance(raw, dict):
        return None

    name = ""
    for k in _VL_JSON_NAME_KEYS:
        if raw.get(k):
            name = str(raw[k]).strip()
            break
    if not name:
        return None

    dosage = ""
    for k in _VL_JSON_DOSE_KEYS:
        if raw.get(k):
            dosage = str(raw[k]).strip()
            break

    instructions = ""
    for k in _VL_JSON_INST_KEYS:
        if raw.get(k):
            instructions = str(raw[k]).strip()
            break

    times_per_day = 1
    for k in ("times_per_day", "frequency", "tan_suat"):
        try:
            times_per_day = int(raw[k])
            break
        except (KeyError, TypeError, ValueError):
            pass

    return {
        "name": name,
        "dosage": dosage or "1 liều",
        "instructions": instructions,
        "times_per_day": times_per_day,
        "specific_times": raw.get("specific_times", []),
        "confidence": float(raw.get("confidence", 0.80)),
    }


def vl_json_to_items(json_text: str) -> list[dict[str, Any]]:
    """Parse VL model JSON response into HealthMate item list.

    Tries to find a list of drug items under several common key names.
    Returns [] if the JSON has an unrecognised schema.

    >>> vl_json_to_items('{"items": [{"name": "Paracetamol", "dosage": "1 viên"}]}')
    [{'name': 'Paracetamol', 'dosage': '1 viên', ...}]
    """
    try:
        payload = json.loads(json_text)
    except (json.JSONDecodeError, ValueError):
        return []

    # Find the list of drugs under common wrapper keys
    candidates: list[Any] = []
    if isinstance(payload, list):
        candidates = payload
    elif isinstance(payload, dict):
        for k in ("items", "drugs", "medications", "prescriptions", "thuoc", "don_thuoc"):
            v = payload.get(k)
            if isinstance(v, list):
                candidates = v
                break

    items = [c for raw in candidates if (c := _coerce_item(raw)) is not None]
    return items


# ---------------------------------------------------------------------------
# 3. Core orchestrator: text → items
# ---------------------------------------------------------------------------

def _select_best_items(
    text_items: list[dict],
    numbered_items: list[dict],
    numbered_head_items: list[dict],
    relaxed_items: list[dict],
    marker_sequence_reliable: bool,
) -> tuple[list[dict], str]:
    """Pick the best item list from competing parsers.

    Returns (items, parser_name_str).
    """
    # When numbering is reliable, prefer numbered parsers (less noise)
    if marker_sequence_reliable:
        if len(numbered_head_items) >= 2:
            base = numbered_head_items
            parser = "numbered_head"
            if len(numbered_items) > len(base):
                base = numbered_items
                parser = "numbered"
            return base, parser
        if len(numbered_items) >= 2:
            return numbered_items, "numbered"

    # Fall back to block parser, then relaxed
    if text_items:
        base = text_items
        parser = "block"
    elif relaxed_items:
        base = relaxed_items
        parser = "relaxed"
    else:
        return [], "none"

    # Supplement with numbered if it found more
    if marker_sequence_reliable and len(numbered_items) > len(base):
        return numbered_items, "numbered"
    if marker_sequence_reliable and len(numbered_head_items) > len(base):
        return numbered_head_items, "numbered_head"

    return base, parser


def _dedupe(items: list[dict]) -> list[dict]:
    """Remove exact-duplicate items by (name, dosage) fingerprint."""
    seen: set[str] = set()
    out: list[dict] = []
    for it in items:
        key = f"{it.get('name','').lower().strip()}|{it.get('dosage','').lower().strip()}"
        if key not in seen:
            seen.add(key)
            out.append(it)
    return out


def parse_prescription_text(raw_text: str) -> dict[str, Any]:
    """Convert raw OCR string to structured HealthMate prescription dict.

    This is the single entry point for the text → JSON pipeline.
    It handles:
      - Markdown code fence stripping (VL model output)
      - VL JSON response (if model returns JSON)
      - Plain OCR text (classic PaddleOCR or VL in text mode)
      - Multi-strategy item extraction with fallback chain
      - Name normalisation and post-processing

    Parameters
    ----------
    raw_text : str
        Raw string from OCR engine or VL model.
        May be empty string — will return empty items list.

    Returns
    -------
    dict matching HealthMate prescription schema.
    """
    if not isinstance(raw_text, str):
        raw_text = str(raw_text)

    # ── Step 1: strip markdown fences ────────────────────────────────────
    cleaned = strip_markdown_fence(raw_text)

    # ── Step 2: try structured VL JSON shortcut ───────────────────────────
    vl_items = vl_json_to_items(cleaned)
    if vl_items:
        vl_items, warnings = post_process_items(vl_items)
        return {
            "raw_text": raw_text,
            "items": _dedupe(vl_items),
            "warnings": warnings,
            "meta": {
                "items_count": len(vl_items),
                "parser_used": "vl_json",
            },
        }

    # ── Step 3: normalize Vietnamese OCR text ────────────────────────────
    if not cleaned:
        return {
            "raw_text": raw_text,
            "items": [],
            "warnings": ["Không có văn bản để phân tích"],
            "meta": {"items_count": 0, "parser_used": "none"},
        }

    normalized = normalize_ocr_vi(cleaned)

    # ── Step 4: run all parsers ───────────────────────────────────────────
    markers                  = extract_numbered_markers(normalized)
    marker_seq_reliable      = is_reliable_numbered_sequence(markers)

    text_items               = parse_items_from_text(normalized)
    numbered_items           = parse_items_from_numbered_lines(normalized)
    numbered_head_items      = parse_items_from_numbered_head_lines(normalized)

    items, parser_used = _select_best_items(
        text_items, numbered_items, numbered_head_items, [], marker_seq_reliable
    )

    # ── Step 5: relaxed fallback when parsers yield nothing / too few ─────
    if len(items) == 0 and len(normalized) >= 120:
        relaxed = parse_items_relaxed_from_lines(normalized)
        if relaxed:
            items = relaxed
            parser_used = "relaxed"
    elif len(items) <= 2 and len(normalized) >= 650:
        relaxed = parse_items_relaxed_from_lines(normalized)
        if len(relaxed) > len(items):
            items = relaxed
            parser_used = "relaxed"

    # ── Step 6: deduplicate ───────────────────────────────────────────────
    items = _dedupe(items)

    # ── Step 7: post-process (name normalisation, quality gate, allergy) ─
    items, warnings = post_process_items(items)

    return {
        "raw_text": raw_text,
        "items": items,
        "warnings": warnings,
        "meta": {
            "items_count": len(items),
            "parser_used": parser_used,
        },
    }
