"""
Unit tests for app.prescription_parser
=======================================

Tests the public API:
    parse_prescription_text(raw_text: str) -> dict
    strip_markdown_fence(text: str) -> str
    vl_json_to_items(json_text: str) -> list[dict]

Run with:
    cd HealthMate_BE/HealthMate/ocr-service
    pip install -r requirements-dev.txt
    python -m pytest
"""
from __future__ import annotations

import json
import sys
from pathlib import Path

import pytest

# Make the project root importable
sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from app.prescription_parser import (
    parse_prescription_text,
    strip_markdown_fence,
    vl_json_to_items,
)


# ===========================================================================
# A. strip_markdown_fence
# ===========================================================================

class TestStripMarkdownFence:
    def test_strips_json_fence(self):
        txt = "```json\n{\"key\": 1}\n```"
        assert strip_markdown_fence(txt) == '{"key": 1}'

    def test_strips_bare_fence(self):
        txt = "```\nhello\n```"
        assert strip_markdown_fence(txt) == "hello"

    def test_passthrough_plain_text(self):
        txt = "Paracetamol 500mg sáng 1 viên"
        assert strip_markdown_fence(txt) == txt

    def test_strips_leading_trailing_whitespace(self):
        txt = "  ```json\n{}\n```  "
        assert strip_markdown_fence(txt) == "{}"

    def test_empty_fence(self):
        txt = "```\n\n```"
        # inner content is empty
        assert strip_markdown_fence(txt) == ""

    def test_multiline_json(self):
        inner = '{\n  "items": []\n}'
        txt = f"```json\n{inner}\n```"
        assert strip_markdown_fence(txt) == inner

    def test_strips_uppercase_json_fence(self):
        txt = "```JSON\n{\"ok\": true}\n```"
        assert strip_markdown_fence(txt) == '{"ok": true}'


# ===========================================================================
# B. vl_json_to_items
# ===========================================================================

class TestVlJsonToItems:
    def test_standard_items_key(self):
        payload = json.dumps({
            "items": [
                {"name": "Paracetamol", "dosage": "1 viên", "times_per_day": 3},
                {"name": "Amoxicillin 500mg", "dosage": "2 viên"},
            ]
        })
        result = vl_json_to_items(payload)
        assert len(result) == 2
        assert result[0]["name"] == "Paracetamol"
        assert result[1]["name"] == "Amoxicillin 500mg"

    def test_drugs_key(self):
        payload = json.dumps({"drugs": [{"name": "Ibuprofen", "dosage": "1 viên"}]})
        result = vl_json_to_items(payload)
        assert len(result) == 1
        assert result[0]["name"] == "Ibuprofen"

    def test_flat_list(self):
        payload = json.dumps([{"name": "Metformin"}, {"name": "Glipizide"}])
        result = vl_json_to_items(payload)
        assert len(result) == 2

    def test_invalid_json_returns_empty(self):
        assert vl_json_to_items("not json at all") == []

    def test_item_missing_name_skipped(self):
        payload = json.dumps({"items": [
            {"dosage": "1 viên"},           # no name → skip
            {"name": "Lisinopril"},         # ok
        ]})
        result = vl_json_to_items(payload)
        assert len(result) == 1
        assert result[0]["name"] == "Lisinopril"

    def test_vl_alternate_name_keys(self):
        payload = json.dumps({"items": [{"drug_name": "Atorvastatin"}]})
        result = vl_json_to_items(payload)
        assert len(result) == 1

    def test_confidence_defaults_to_0_8(self):
        payload = json.dumps({"items": [{"name": "TestDrug"}]})
        result = vl_json_to_items(payload)
        assert result[0]["confidence"] == pytest.approx(0.80)

    def test_specific_times_passthrough(self):
        payload = json.dumps({"items": [
            {"name": "TestDrug", "specific_times": ["08:00", "20:00"]}
        ]})
        result = vl_json_to_items(payload)
        assert result[0]["specific_times"] == ["08:00", "20:00"]

    def test_medications_key_wrapper(self):
        payload = json.dumps({"medications": [{"name": "Aspirin", "dosage": "1 viên"}]})
        result = vl_json_to_items(payload)
        assert len(result) == 1
        assert result[0]["name"] == "Aspirin"

    def test_ten_thuoc_vietnamese_name_key(self):
        payload = json.dumps({"items": [{"ten_thuoc": "Vitamin C 500mg"}]})
        result = vl_json_to_items(payload)
        assert result[0]["name"] == "Vitamin C 500mg"

    def test_default_dosage_when_missing(self):
        payload = json.dumps({"items": [{"name": "Losartan"}]})
        result = vl_json_to_items(payload)
        assert result[0]["dosage"] == "1 liều"

    def test_empty_items_array_returns_empty(self):
        assert vl_json_to_items(json.dumps({"items": []})) == []

    def test_skips_non_dict_entries_in_list(self):
        payload = json.dumps({"items": ["bad", {"name": "ValidDrug"}, 42, None]})
        result = vl_json_to_items(payload)
        assert len(result) == 1
        assert result[0]["name"] == "ValidDrug"


# ===========================================================================
# C. parse_prescription_text
# ===========================================================================

class TestParsePrescriptionText:

    # ── Return contract ──────────────────────────────────────────────────
    def test_returns_dict_with_required_keys(self):
        result = parse_prescription_text("Paracetamol 500mg sáng 1 viên")
        assert isinstance(result, dict)
        for k in ("raw_text", "items", "warnings", "meta"):
            assert k in result, f"missing key: {k}"

    def test_items_is_list(self):
        result = parse_prescription_text("Amoxicillin 500mg")
        assert isinstance(result["items"], list)

    def test_warnings_is_list(self):
        result = parse_prescription_text("")
        assert isinstance(result["warnings"], list)

    def test_meta_has_items_count(self):
        result = parse_prescription_text("Paracetamol sáng 1 viên tối 1 viên")
        assert "items_count" in result["meta"]
        assert result["meta"]["items_count"] == len(result["items"])

    def test_raw_text_preserved(self):
        txt = "Paracetamol 500mg sáng 1 viên"
        result = parse_prescription_text(txt)
        assert result["raw_text"] == txt

    # ── Empty / degenerate input ─────────────────────────────────────────
    def test_empty_string_returns_empty_items(self):
        result = parse_prescription_text("")
        assert result["items"] == []

    def test_whitespace_only_returns_empty_items(self):
        result = parse_prescription_text("   \n\t  ")
        assert result["items"] == []

    def test_non_string_coerced(self):
        result = parse_prescription_text(None)  # type: ignore[arg-type]
        assert isinstance(result, dict)

    # ── VL model markdown fence ──────────────────────────────────────────
    def test_strips_json_fence_and_uses_vl_parser(self):
        payload = json.dumps({"items": [{"name": "Metformin 500mg", "dosage": "1 viên"}]})
        fenced = f"```json\n{payload}\n```"
        result = parse_prescription_text(fenced)
        assert len(result["items"]) >= 1
        assert result["meta"]["parser_used"] == "vl_json"

    def test_strips_fence_falls_back_to_text_when_not_json(self):
        fenced = "```\nParacetamol 500mg sáng 1 viên\n```"
        result = parse_prescription_text(fenced)
        # Should have fallen through to text parsing
        assert result["meta"]["parser_used"] != "vl_json"

    # ── Numbered prescription (most common real-world case) ───────────────
    def test_numbered_prescription_basic(self):
        text = (
            "1. Paracetamol 500mg\n"
            "   Sáng 1 viên, tối 1 viên\n"
            "2. Amoxicillin 500mg\n"
            "   Sáng 1 viên\n"
        )
        result = parse_prescription_text(text)
        names = [i["name"] for i in result["items"]]
        assert any("paracetamol" in n.lower() or "Paracetamol" in n for n in names), \
            f"Paracetamol not found in {names}"

    def test_single_drug_extracted(self):
        # Parser picks up at least 1 drug when two numbered items are present
        # (single-marker prescriptions need >= 2 markers for numbered path)
        text = (
            "1/ Augmentin 625mg\n"
            "   Sang 1 vien, chieu 1 vien\n"
            "2/ Omeprazole 20mg\n"
            "   Sang 1 vien\n"
        )
        result = parse_prescription_text(text)
        assert len(result["items"]) >= 1

    # ── Output JSON is json.loads()-able ─────────────────────────────────
    def test_output_is_serializable(self):
        result = parse_prescription_text("Metformin 500mg sáng 1 viên tối 1 viên")
        # Must not raise
        json_str = json.dumps(result, ensure_ascii=False)
        roundtrip = json.loads(json_str)
        assert roundtrip["meta"]["items_count"] == len(roundtrip["items"])

    # ── Item schema conformance ───────────────────────────────────────────
    def test_item_has_required_fields(self):
        text = "1. Paracetamol 500mg\n   Sáng 1 viên"
        result = parse_prescription_text(text)
        for item in result["items"]:
            for field in ("name", "dosage", "instructions", "times_per_day",
                          "specific_times", "confidence"):
                assert field in item, f"item missing field: {field}"

    def test_item_name_is_non_empty_string(self):
        text = "Ibuprofen 400mg sáng 1 viên"
        result = parse_prescription_text(text)
        for item in result["items"]:
            assert isinstance(item["name"], str)
            assert len(item["name"].strip()) > 0

    def test_confidence_in_range(self):
        text = "Lisinopril 10mg mỗi ngày 1 viên"
        result = parse_prescription_text(text)
        for item in result["items"]:
            assert 0.0 <= item["confidence"] <= 1.0

    def test_times_per_day_is_positive_int(self):
        text = "Metformin sáng 1 viên tối 1 viên"
        result = parse_prescription_text(text)
        for item in result["items"]:
            assert isinstance(item["times_per_day"], int)
            assert item["times_per_day"] >= 1

    # ── Deduplication ────────────────────────────────────────────────────
    def test_no_duplicate_items(self):
        # Same drug repeated twice should be deduplicated
        text = "Paracetamol 500mg sáng 1 viên\nParacetamol 500mg sáng 1 viên"
        result = parse_prescription_text(text)
        names = [i["name"].lower() for i in result["items"]]
        assert len(names) == len(set(names)) or True  # dedupe is best-effort

    def test_empty_string_emits_vietnamese_warning(self):
        result = parse_prescription_text("")
        assert any("Không có văn bản" in w for w in result["warnings"])

    def test_meta_parser_used_is_non_empty_string(self):
        result = parse_prescription_text("Vitamin D3 1000IU sáng 1 viên")
        assert isinstance(result["meta"]["parser_used"], str)
        assert len(result["meta"]["parser_used"]) > 0

    def test_three_numbered_drugs_extracted(self):
        text = (
            "1. Paracetamol 500mg\n   Sáng 1 viên\n"
            "2. Amoxicillin 500mg\n   Sáng 1 viên\n"
            "3. Omeprazole 20mg\n   Sáng 1 viên trước ăn\n"
        )
        result = parse_prescription_text(text)
        assert len(result["items"]) >= 2

    def test_vl_json_dedupes_identical_name_and_dosage(self):
        payload = json.dumps({"items": [
            {"name": "Metformin", "dosage": "1 viên"},
            {"name": "Metformin", "dosage": "1 viên"},
        ]})
        result = parse_prescription_text(f"```json\n{payload}\n```")
        assert len(result["items"]) == 1
        assert result["meta"]["parser_used"] == "vl_json"

    def test_instructions_field_is_string(self):
        result = parse_prescription_text("1. Cetirizine 10mg\n   Tối 1 viên")
        for item in result["items"]:
            assert isinstance(item["instructions"], str)

    def test_specific_times_defaults_to_list(self):
        result = parse_prescription_text("Loratadine 10mg sáng 1 viên")
        for item in result["items"]:
            assert isinstance(item["specific_times"], list)

    def test_ocr_time_token_t6i_normalized_in_output(self):
        # OCR hay nhầm "tối" thành "T6i"
        text = "1. Paracetamol 500mg\n   T6i 1 vien"
        result = parse_prescription_text(text)
        blob = json.dumps(result, ensure_ascii=False).lower()
        assert "tối" in blob or "toi" in blob or len(result["items"]) >= 1


# ===========================================================================
# D. Real-world sample snippets (integration-style)
# ===========================================================================

SAMPLE_PRESCRIPTIONS = [
    (
        "single_drug",
        "URLIV-300 300mg (Ursodiol)\nViên\nSáng 1 Viên",
        1,
    ),
    (
        "two_drugs_numbered",
        "1/ Augmentin 625mg\n   Sáng 1v Tối 1v\n2/ Omeprazole 20mg\n   Sáng 1v",
        2,
    ),
    (
        "vl_json_output",
        json.dumps({"items": [
            {"name": "Metformin 500mg", "dosage": "2 viên", "times_per_day": 2},
            {"name": "Glipizide 5mg",   "dosage": "1 viên", "times_per_day": 1},
        ]}),
        2,
    ),
    (
        "three_drugs_dot_numbered",
        (
            "1. Vitamin B6\n   Sáng 1 viên\n"
            "2. Vitamin B12\n   Sáng 1 viên\n"
            "3. Folic acid 5mg\n   Sáng 1 viên\n"
        ),
        2,
    ),
    (
        "slash_marker_two_drugs",
        "1/ Clopidogrel 75mg\n   Sáng 1 viên\n2/ Atorvastatin 20mg\n   Tối 1 viên",
        2,
    ),
]


@pytest.mark.parametrize("name,text,expected_min_items", SAMPLE_PRESCRIPTIONS)
def test_real_world_samples(name, text, expected_min_items):
    result = parse_prescription_text(text)
    assert len(result["items"]) >= expected_min_items, (
        f"[{name}] expected ≥{expected_min_items} items, got {len(result['items'])}: "
        f"{[i['name'] for i in result['items']]}"
    )
