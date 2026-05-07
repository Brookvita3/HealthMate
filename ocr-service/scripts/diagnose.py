"""Diagnostic script — run inside container to investigate failing cases."""
import json
import urllib.request
from pathlib import Path

IMG_DIR = Path("/app/scripts/dataset/images")
LAB_DIR = Path("/app/scripts/dataset/labels")

CASES = [
    "0760c8f6-adbd-4254-9418-a62ba6d6916d",             # VACOMUC, Savi C -> Got []
    "642339050_1455056389447802_3264942585587069227_n",  # Sucralfat -> got Vagastat (synonym mismatch)
    "041fec48-0760-48d5-a8fb-73b72843c8c7",             # Levomepromazin -> Got []
    "659813677_2416353085548154_7601677991925691977_n",  # Amoxicillin, Sacch -> Got []
    "660337762_1950367022512887_6912860012122313822_n",  # 7 drugs -> Got []
]

def call(img_path: Path) -> dict:
    boundary = "----BenchBoundary"
    body = (
        f"--{boundary}\r\n"
        f'Content-Disposition: form-data; name="file"; filename="{img_path.name}"\r\n'
        f"Content-Type: image/jpeg\r\n\r\n"
    ).encode() + img_path.read_bytes() + f"\r\n--{boundary}--\r\n".encode()
    req = urllib.request.Request(
        "http://localhost:8010/ocr/prescriptions/parse",
        data=body,
        headers={"Content-Type": f"multipart/form-data; boundary={boundary}"},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=120) as r:
        return json.loads(r.read())

for stem in CASES:
    img = IMG_DIR / (stem + ".jpg")
    lab_file = LAB_DIR / (stem + ".json")
    label = json.loads(lab_file.read_text()) if lab_file.exists() else {}

    resp = call(img)
    print(f"\n{'='*70}")
    print(f"IMAGE : {stem[:55]}")
    print(f"EXPECT: {[i['name'] for i in label.get('items', [])]}")
    print(f"GOT   : {[i['name'] for i in resp['items']]}")
    print(f"SCORE : ocr={resp['meta']['ocr_score']}  pass={resp['meta']['pass_used']}  ms={resp['meta']['processing_ms']}")
    print(f"RAW TEXT (400 chars):\n{resp['raw_text'][:400]!r}")
