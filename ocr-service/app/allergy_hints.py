"""
Generic drug-class allergy hints for OCR-service items.

These are NOT user-specific -- they classify drugs by well-known allergy groups
so the Flutter client can match against the user's allergy profile on-device.

The hint strings are kept in Latin script and match the common allergy names
used in the FE _commonAllergies list (Penicillin, Aspirin, Cephalosporins, etc.).
"""

from __future__ import annotations

# Each entry: (canonical_key, frozenset_of_name_fragments, list_of_hint_strings)
# Fragments are checked via `in` against the lowercase normalized drug name.
_DRUG_GROUP_RULES: list[tuple[str, frozenset[str], list[str]]] = [
    (
        "penicillin",
        frozenset([
            "amoxicillin", "ampicillin", "penicillin", "unasyn",
            "sultamicillin", "augmentin", "amoxiclav", "oxacillin",
            "cloxacillin", "dicloxacillin", "piperacillin", "ticarcillin",
        ]),
        ["Penicillin"],
    ),
    (
        "cephalosporin",
        frozenset([
            "cephalosporin", "cefixim", "cefalexin", "ceftriaxone",
            "cefuroxim", "cefadroxil", "cefpodoxim", "cefdinir",
            "cefazolin", "cephalothin", "cefepime", "ceftazidime",
            "cefoperazone", "cefotaxime",
        ]),
        ["Cephalosporins"],
    ),
    (
        "sulfonamide",
        frozenset([
            "sulfamethoxazol", "trimethoprim", "bactrim", "biseptol",
            "sulfadiazine", "sulfisoxazole", "sulfonamide",
        ]),
        ["Sulfonamides"],
    ),
    (
        "nsaid",
        frozenset([
            "aspirin", "ibuprofen", "naproxen", "diclofenac", "ketoprofen",
            "indomethacin", "meloxicam", "celecoxib", "piroxicam",
            "ketorolac", "mefenamic",
        ]),
        ["Aspirin", "Ibuprofen"],
    ),
    (
        "quinolone",
        frozenset([
            "ciprofloxacin", "levofloxacin", "ofloxacin", "norfloxacin",
            "moxifloxacin", "gemifloxacin", "enrofloxacin",
        ]),
        ["Quinolones"],
    ),
    (
        "tetracycline",
        frozenset([
            "tetracycline", "doxycycline", "minocycline", "tigecycline",
        ]),
        ["Tetracyclines"],
    ),
    (
        "codeine_opioid",
        frozenset([
            "codeine", "morphine", "tramadol", "oxycodone", "hydrocodone",
            "fentanyl", "buprenorphine",
        ]),
        ["Codeine", "Morphine"],
    ),
    (
        "warfarin_anticoagulant",
        frozenset(["warfarin"]),
        ["Warfarin"],
    ),
    (
        "insulin",
        frozenset(["insulin", "mixtard", "insuact", "lantus", "novorapid", "humalog"]),
        ["Insulin"],
    ),
    (
        "vancomycin",
        frozenset(["vancomycin"]),
        ["Vancomycin"],
    ),
    (
        "lidocaine",
        frozenset(["lidocaine", "lignocaine", "xylocaine"]),
        ["Lidocaine"],
    ),
]


def get_allergy_hints(normalized_name: str) -> list[str]:
    """Return drug-class allergy hint strings for *normalized_name*.

    Args:
        normalized_name: Drug name already normalized by drug_dictionary
                         (lowercase, stripped of obvious OCR noise).

    Returns:
        List of hint strings (may be empty).  Duplicates are removed while
        preserving insertion order.
    """
    if not normalized_name:
        return []

    lower = normalized_name.lower()
    hints: list[str] = []

    for _key, fragments, group_hints in _DRUG_GROUP_RULES:
        if any(frag in lower for frag in fragments):
            for h in group_hints:
                if h not in hints:
                    hints.append(h)

    return hints
