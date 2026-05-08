"""
Central constants for HealthMate OCR service.

All drug-signal sets and compiled regex patterns are defined here —
single source of truth instead of scattered across multiple files.
"""
from __future__ import annotations

import re

# ---------------------------------------------------------------------------
# Drug-name fragments used as OCR signal detection across all parsers.
# A line containing any of these is treated as a potential medication line.
# ---------------------------------------------------------------------------
KNOWN_DRUG_SIGNALS: frozenset[str] = frozenset({
    # Original set
    "adalat",      "agidopa",    "agifu",       "agitrith",   "amlodip",
    "aspirin",     "atorva",     "bidifer",      "bisoprol",   "caldihason",
    "calci",       "carbonat",   "domela",       "folic",      "furosem",
    "haxium",      "hyvalor",    "insuact",      "lipotatin",  "methyldop",
    "mixtard",     "nebivol",    "pantopraz",    "paracetamol","piracetam",
    "rivarox",     "sucralf",    "sulfat",       "vagastat",   "vitamin",
    "xaravix",     "xarelto",    "xerax",
    # Cardiovascular / antihypertensive
    "losartan",    "valsartan",  "candesartan",  "telmisartan","irbesartan",
    "perindopril", "ramipril",   "enalapril",    "lisinopril", "captopril",
    "nifedipin",   "amlodipin",  "felodipine",   "diltiazem",  "verapamil",
    "metoprolol",  "carvedilol", "nebivolol",    "propranolol","atenolol",
    "spironolact", "indapamid",  "hydrochloroth","torsemide",
    # Lipid-lowering
    "rosuvastatin","simvastatin","pravastatin",  "fluvastatin","pitavastatin",
    "fenofibrat",  "gemfibrozil","ezetimibe",    "ezetimib",
    # Antiplatelet / anticoagulant
    "clopidogrel", "ticagrelor", "aspirin",      "warfarin",   "dabigatran",
    "apixaban",    "edoxaban",
    # Diabetes
    "metformin",   "glibencla",  "gliclazid",    "glipizide",  "sitagliptin",
    "empaglifloz", "dapaglifloz","insulin",      "glargine",   "liraglutide",
    # Antibiotics
    "amoxicillin", "amoxiclav",  "clavulanat",   "azithromycin","clarithro",
    "ciprofloxac", "levofloxac", "moxifloxac",   "doxycycline","tetracyclin",
    "cephalexin",  "cefuroxim",  "ceftriaxon",   "cefixim",    "metronidazol",
    # GI / antiulcer
    "omeprazol",   "esomeprazol","lansoprazol",  "rabeprazol", "famotidin",
    "ranitidine",  "domperidon", "metoclopram",  "ondansetron","loperamid",
    "lactulose",   "probiotic",  "saccharomyc",  "simethicone",
    # CNS / psychiatric
    "mirtazapin",  "sertralin",  "fluoxetin",    "paroxetin",  "venlafaxin",
    "amitriptylin","lorazepam",  "diazepam",     "alprazolam", "zolpidem",
    "haloperidol", "risperidon", "quetiapine",   "olanzapine", "clozapine",
    "levodopa",    "carbidopa",  "trihexyphen",  "memantine",  "donepezil",
    "levomepromazin","tisercin",
    # Pain / anti-inflammatory
    "ibuprofen",   "naproxen",   "diclofenac",   "celecoxib",  "meloxicam",
    "loxoprofen",  "eperisone",  "thiocolchicosid",
    # Vitamins / supplements
    "vitamin",     "vitamine",   "acid folic",   "ferrous",    "iron",
    "calci",       "calcium",    "magnesi",      "kali",       "zinc",
    "omega",       "ginkgo",     "bilomag",
    # Other common in VN prescriptions
    "allopurinol", "colchicin",  "prednisolon",  "dexamethasone","budesonide",
    "salbutamol",  "montelukast","cetirizin",    "loratadin",  "fexofenadine",
    "acyclovir",   "oseltamivir","paracetam",    "acemuc",     "acetylcystein",
    "ridlor",      "sterolow",   "troysar",      "norvasc",    "concor",
    "saviprolol",  "kavasdin",   "slmetformin",
})

# ---------------------------------------------------------------------------
# Last-resort rescue candidates when parser finds too few items.
# Format: (identity_key, compiled_regex, canonical_name)
# The identity_key must match what prescription_item_identity_key() returns.
# ---------------------------------------------------------------------------
KNOWN_MEDICINE_RESCUE: list[tuple[str, re.Pattern[str], str]] = [
    (
        "id:agidopa",
        re.compile(r"agidopa|methyldop|melhyldopa|melyldop", re.I),
        "Agidopa 250 mg (Methyldopa)",
    ),
    (
        "id:paracetamol",
        re.compile(r"paracetamol|paracelamol|paracetamo", re.I),
        "Paracetamol 500 mg",
    ),
    (
        "id:pantoprazole",
        re.compile(r"pantop|pantopraz|pantoprez", re.I),
        "Pantoprazole 40 mg",
    ),
    (
        "id:bidiferon",
        re.compile(
            r"bidifer|sulfat.{0,24}folic|folic.{0,24}sulfat|s[aăâ]t.{0,12}folic",
            re.I,
        ),
        "Bidiferon (Sắt II sulfat + Acid folic)",
    ),
    (
        "id:caldihasan",
        re.compile(
            r"caldiha|calci.{0,24}vitamin|calci.{0,24}d3|carbonat.{0,24}vitamin",
            re.I,
        ),
        "Caldihason (Calci carbonat + Vitamin D3)",
    ),
    (
        "id:vagastat",
        re.compile(r"vagastat|sucralf", re.I),
        "Vagastat (Sucralfate)",
    ),
    (
        "id:bisoprolol",
        re.compile(r"bisoprol|bisopro", re.I),
        "Bisoprolol 2.5 mg",
    ),
    (
        "id:xarelto",
        re.compile(r"xarelto|rivarox|xaravix|xerax", re.I),
        "Xarelto 15 mg (Rivaroxaban)",
    ),
]

# ---------------------------------------------------------------------------
# Time-of-day token → clock time
# ---------------------------------------------------------------------------
TIMES_OF_DAY_MAP: dict[str, str] = {
    "sáng":  "08:00",
    "trưa":  "12:00",
    "chiều": "16:30",
    "tối":   "20:00",
}

# Default specific_times list when count is known but slots aren't explicit
DEFAULT_TIMES_BY_COUNT: dict[int, list[str]] = {
    1: ["08:00"],
    2: ["08:00", "16:30"],
    3: ["08:00", "13:00", "20:00"],
}

# Reverse map: clock time → Vietnamese label (for building instruction strings)
REVERSE_TIME_MAP: dict[str, str] = {v: k for k, v in TIMES_OF_DAY_MAP.items()}

# ---------------------------------------------------------------------------
# Compiled regex patterns — compiled once at import, reused everywhere.
# ---------------------------------------------------------------------------

# Numbered list marker: "1.", "2)", "3 ", "| 1.", etc.
RE_NUM_MARKER      = re.compile(r"^\s*[\W_]{0,4}\s*(\d{1,2})\s*[.)/\-]\s*")
RE_LINE_NUM_MARKER = re.compile(r"^\s*[\W_]{0,4}(\d{1,2})\s*[.)/\-]\s*")

# Dosage strength: "500mg", "10ml", "2.5mg/5ml"
RE_DOSAGE = re.compile(
    r"(\d+(?:[.,]\d+)?\s*(?:mg|g|ml|mcg|iu|ui)"
    r"(?:\s*/\s*\d+(?:[.,]\d+)?\s*(?:mg|g|ml|mcg|iu|ui))?)",
    re.I,
)

# Time-of-day tokens in Vietnamese
RE_TIMES_OF_DAY = re.compile(r"(sáng|trưa|chiều|tối)", re.I)

# Per-dose phrasing: "mỗi lần 1 viên"
RE_PER_DOSE = re.compile(
    r"(?:m[o0]i|mỗi)\s*l[aàâầằấậẩẫăắằẳẵặ]n[:\s]*([0-9]+(?:[.,][0-9]+)?)\s*"
    r"(vi[eê]n|tabs?|tablets?|caps?|capsules?|g[oó]i|sachets?|[oô]ng|ampoules?"
    r"|ml|drops?|gi[oọ]t|chai|l[oọ])?",
    re.I,
)

# Times-per-day phrasing: "uống 2 lần"
RE_TIMES_PER_DAY = re.compile(
    r"(?:ng[aà]y\s*u[oố]ng|u[oố]ng)\s*([0-9]+)\s*l[aàâầằấậẩẫăắằẳẵặ]n",
    re.I,
)

# Dispensed quantity: "2 viên", "10 gói"
RE_QUANTITY = re.compile(
    r"([0-9]{1,3})\s*(vi[eê]n|tabs?|tablets?|caps?|capsules?|g[oó]i|sachets?"
    r"|[oô]ng|ampoules?|ml|drops?|gi[oọ]t|chai|l[oọ])\b",
    re.I,
)

# Dose following a time-of-day word: "sáng 1 viên", "tối: 2 viên"
RE_DOSE_AFTER_TAKE = re.compile(
    r"(?:u[oố]ng|s[aá]ng|tr[ưu]a|chi[eề]u|t[oố]i)\s*[:\-]?\s*"
    r"([0-9]+(?:[.,][0-9]+)?)\s*"
    r"(vi[eê]n|tabs?|tablets?|caps?|capsules?|g[oó]i|sachets?|[oô]ng|ampoules?"
    r"|ml|drops?|gi[oọ]t|chai|l[oọ])\b",
    re.I,
)

# Meal timing
RE_BEFORE_MEAL = re.compile(r"(?:tr[uư][oớ]c\s*a[nă]n|trc\s*a[nă]n)", re.I)
RE_AFTER_MEAL  = re.compile(r"(?:sau\s*a[nă]n)", re.I)

# Administrative / header text that is NOT a medication line
RE_HEADER_NOISE = re.compile(
    r"(s[ởo]\s*y\s*t[eê]|b[eệ]nh\s*vi[eệ]n|d[ơo]n\s*thu[oố]c"
    r"|h[oọ]\s*t[eê]n|ch[aẩ]n\s*do[aá]n|b[aả]n\s*sao|m[aã]\s*d[ơo]n)",
    re.I,
)
RE_NON_MEDICINE_NOISE = re.compile(
    r"(b[aá]c\s*s[iĩ]|t[aá]i\s*kh[aá]m|ng[àa]y\s*[0-9]{1,2}\s*th[aá]ng"
    r"|t[oổ]ng\s*ti[eề]n|bhyt)",
    re.I,
)

# Vietnamese unaccented → accented normalisation (applied line by line)
VN_FIX_PATTERNS: list[tuple[re.Pattern[str], str]] = [
    (re.compile(r"\buong\b",  re.I), "uống"),
    (re.compile(r"\bngay\b",  re.I), "ngày"),
    (re.compile(r"\blan\b",   re.I), "lần"),
    (re.compile(r"\bsang\b",  re.I), "sáng"),
    (re.compile(r"\bchieu\b", re.I), "chiều"),
    (re.compile(r"\btoi\b",   re.I), "tối"),
    (re.compile(r"\bmoi\b",   re.I), "mỗi"),
]

# Cut-point marker inside a drug-name token (where the name ends)
RE_NAME_CUT = re.compile(
    r"\b(?:ngày\s*uống|uống|mỗi\s*lần|sáng|chiều|trưa|tối"
    r"|vien|viên|gói|ống|cong\s*khoan|cộng\s*khoản|mn\d*n?|sn)\b",
    re.I,
)

# Supply-quantity prefix to strip before looking for per-dose quantity
RE_SUPPLY_QTY = re.compile(
    r"\b(?:s[oố]\s*l[uư][oợ]ng|sl)\s*[:\-]?\s*[0-9]{1,3}\s*"
    r"(?:vi[eê]n|tabs?|tablets?|caps?|capsules?|g[oó]i|sachets?"
    r"|[oô]ng|ampoules?|ml|drops?|gi[oọ]t|chai|l[oọ])\b",
    re.I,
)
