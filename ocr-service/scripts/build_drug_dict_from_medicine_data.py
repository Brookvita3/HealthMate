#!/usr/bin/env python3
"""Build OCR drug dictionary CSV from medicine_data JSON files.

Input format expected:
- Directory contains multiple *.json files.
- Each file is a JSON array of objects with field `ten_thuoc`.

Output format:
- CSV with one column: ten_thuoc
- Cleaned canonical names, deduplicated, sorted.
"""

from __future__ import annotations

import argparse
import csv
import json
import re
from pathlib import Path
from typing import Iterable


_CUT_PATTERNS: tuple[re.Pattern[str], ...] = (
    re.compile(
        r"\b(?:điều trị|dự phòng|hỗ trợ|dùng trong|chỉ định|giảm đau|kháng viêm)\b",
        flags=re.I,
    ),
)

_NOISE_PREFIX = re.compile(r"^\s*thuốc\s+", flags=re.I)
_NOISE_SUFFIX = re.compile(r"\s*[-–—:,;|]+\s*$")
_SPACE_RE = re.compile(r"\s+")


def _normalize_spaces(text: str) -> str:
    return _SPACE_RE.sub(" ", text).strip()


def _clean_name(raw: str) -> str:
    """Extract canonical-ish drug name from noisy product title."""
    text = _normalize_spaces(raw or "")
    if not text:
        return ""

    # Keep left side before indication phrases.
    for pat in _CUT_PATTERNS:
        m = pat.search(text)
        if m and m.start() > 0:
            text = text[: m.start()].strip()

    # Drop package/count details in trailing parentheses.
    text = re.sub(r"\([^)]*\)$", "", text).strip()

    # Remove generic "Thuốc" prefix (for cleaner matching key).
    text = _NOISE_PREFIX.sub("", text).strip()

    # Remove trailing punctuation/noise.
    text = _NOISE_SUFFIX.sub("", text).strip()

    # Remove duplicated separators and normalize spaces.
    text = _normalize_spaces(text)

    # Hard cap to avoid accidental long sentences.
    if len(text) > 140:
        text = text[:140].strip()

    return text


def _iter_names(json_file: Path) -> Iterable[str]:
    with json_file.open("r", encoding="utf-8") as f:
        data = json.load(f)
    if not isinstance(data, list):
        return
    for row in data:
        if not isinstance(row, dict):
            continue
        name = str(row.get("ten_thuoc", "")).strip()
        if name:
            yield name


def build_dictionary(input_dir: Path, output_csv: Path) -> dict[str, int]:
    if not input_dir.exists():
        raise FileNotFoundError(f"Input directory not found: {input_dir}")

    files = sorted(input_dir.glob("*.json"))
    if not files:
        raise FileNotFoundError(f"No JSON files found in: {input_dir}")

    total_raw = 0
    cleaned_names: list[str] = []
    dropped_empty = 0

    for jf in files:
        for raw in _iter_names(jf):
            total_raw += 1
            cleaned = _clean_name(raw)
            if not cleaned:
                dropped_empty += 1
                continue
            cleaned_names.append(cleaned)

    # Deduplicate by normalized key, keep shortest candidate (usually cleaner).
    best_by_key: dict[str, str] = {}
    for name in cleaned_names:
        key = re.sub(r"[^a-z0-9à-ỹ]+", " ", name.lower()).strip()
        key = _normalize_spaces(key)
        if len(key) < 2:
            continue
        cur = best_by_key.get(key)
        if cur is None or len(name) < len(cur):
            best_by_key[key] = name

    final_names = sorted(best_by_key.values(), key=lambda s: s.lower())
    output_csv.parent.mkdir(parents=True, exist_ok=True)
    with output_csv.open("w", encoding="utf-8", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=["ten_thuoc"])
        writer.writeheader()
        for name in final_names:
            writer.writerow({"ten_thuoc": name})

    return {
        "json_files": len(files),
        "total_raw_names": total_raw,
        "cleaned_names": len(cleaned_names),
        "dropped_empty": dropped_empty,
        "final_unique_names": len(final_names),
    }


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Build OCR drug dictionary CSV from medicine_data JSON files."
    )
    parser.add_argument(
        "--input-dir",
        default=r"C:\Hoc_Tap\ĐATN\medicine_data",
        help="Directory containing *.json medicine files.",
    )
    parser.add_argument(
        "--output-csv",
        default=r"C:\Hoc_Tap\ĐATN\HealthMate_BE\HealthMate\ocr-service\data\drugs_moh.csv",
        help="Output CSV path for OCR drug dictionary.",
    )
    args = parser.parse_args()

    stats = build_dictionary(Path(args.input_dir), Path(args.output_csv))
    print("Dictionary generated successfully.")
    for k, v in stats.items():
        print(f"{k}: {v}")
    print(f"output_csv: {args.output_csv}")


if __name__ == "__main__":
    main()
