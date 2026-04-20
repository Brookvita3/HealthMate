import csv
import json
import re
import time
from html import unescape
from urllib.parse import urljoin

import requests

# ====== CHOOSE SOURCE MODE ======
# - "html": crawl trực tiếp trang tra cứu thuốc công khai
# - "api" : crawl endpoint JSON (nếu bạn có request Network)
SOURCE_MODE = "html"

# ====== MODE HTML (READY TO RUN) ======
HTML_START_URL = "https://dav.gov.vn/tra-cuu-thuoc.html"

# ====== MODE API (OPTIONAL) ======
API_URL = "PASTE_API_URL_HERE"
METHOD = "POST"  # POST or GET
HEADERS = {
    "User-Agent": "Mozilla/5.0",
    "Content-Type": "application/json;charset=UTF-8",
    # Optional:
    # "Cookie": "...",
    # "__RequestVerificationToken": "...",
}
BASE_PAYLOAD = {}
SKIP_KEY = "skipCount"
TAKE_KEY = "maxResultCount"
PAGE_SIZE = 200
# =================================================


def _clean_html_text(text: str) -> str:
    text = re.sub(r"<[^>]+>", " ", text)
    text = unescape(text)
    text = re.sub(r"\s+", " ", text)
    return text.strip()


def _extract_last_page(html: str) -> int:
    m = re.search(r"href=['\"](?P<href>[^'\"]*tra-cuu-thuoc-page(\d+)\.html)['\"][^>]*>\s*Trang cu", html, re.I)
    if m:
        n = re.search(r"page(\d+)\.html", m.group("href"), re.I)
        if n:
            return int(n.group(1))
    return 1


def _parse_html_rows(page_url: str, html: str) -> list[dict[str, str]]:
    tbody = re.search(r"<tbody>(.*?)</tbody>", html, re.I | re.S)
    if not tbody:
        return []

    rows: list[dict[str, str]] = []
    for tr in re.findall(r"<tr>(.*?)</tr>", tbody.group(1), re.I | re.S):
        tds = re.findall(r"<td[^>]*>(.*?)</td>", tr, re.I | re.S)
        if len(tds) < 8:
            continue

        attachments = re.findall(r"href=['\"]([^'\"]+)['\"]", tds[7], re.I)
        rows.append(
            {
                "stt": _clean_html_text(tds[0]),
                "so_giay_tiep_nhan": _clean_html_text(tds[1]),
                "nam_tiep_nhan": _clean_html_text(tds[2]),
                "ten_thuoc": _clean_html_text(tds[3]),
                "cong_ty_dang_ky_tttt_qc": _clean_html_text(tds[4]),
                "loai_hinh_thong_tin_quang_cao": _clean_html_text(tds[5]),
                "so_dang_ky_thuoc": _clean_html_text(tds[6]),
                "tai_lieu_dinh_kem": " | ".join(urljoin(page_url, u) for u in attachments),
            }
        )
    return rows


def crawl_html_source() -> tuple[list[dict], list[dict]]:
    session = requests.Session()
    session.headers.update({"User-Agent": "Mozilla/5.0"})

    first = session.get(HTML_START_URL, timeout=60)
    first.raise_for_status()
    first_html = first.text
    last_page = _extract_last_page(first_html)
    print("last_page =", last_page)

    all_items: list[dict] = []
    all_items.extend(_parse_html_rows(HTML_START_URL, first_html))
    print(f"fetched page 1/{last_page} -> {len(all_items)} rows")

    for page in range(2, last_page + 1):
        url = f"https://dav.gov.vn/tra-cuu-thuoc-page{page}.html"
        resp = session.get(url, timeout=60)
        resp.raise_for_status()
        all_items.extend(_parse_html_rows(url, resp.text))
        print(f"fetched page {page}/{last_page} -> {len(all_items)} rows")
        time.sleep(0.08)

    csv_rows = [
        x
        for x in all_items
        if x.get("ten_thuoc")
    ]
    return all_items, csv_rows


def fetch_api_page(skip: int) -> tuple[int, list[dict]]:
    payload = dict(BASE_PAYLOAD)
    payload[SKIP_KEY] = skip
    payload[TAKE_KEY] = PAGE_SIZE

    if METHOD.upper() == "POST":
        r = requests.post(API_URL, headers=HEADERS, json=payload, timeout=60)
    else:
        r = requests.get(API_URL, headers=HEADERS, params=payload, timeout=60)
    r.raise_for_status()

    data = r.json()
    result = data.get("result") or {}
    total = int(result.get("totalCount") or 0)
    items = result.get("items") or []
    return total, items


def crawl_api_source() -> tuple[list[dict], list[dict]]:
    all_items: list[dict] = []
    skip = 0
    total = None

    while True:
        t, items = fetch_api_page(skip)
        if total is None:
            total = t
            print("totalCount =", total)
        if not items:
            break

        all_items.extend(items)
        skip += len(items)
        print(f"fetched {len(all_items)}/{total}")
        time.sleep(0.15)
        if total and len(all_items) >= total:
            break

    rows = []
    for it in all_items:
        b = it.get("thongTinThuocCoBan") or {}
        rows.append(
            {
                "ten_thuoc": (it.get("tenThuoc") or "").strip(),
                "so_dang_ky": (it.get("soDangKy") or "").strip(),
                "hoat_chat_chinh": (b.get("hoatChatChinh") or "").strip(),
                "ham_luong": (b.get("hamLuong") or "").strip(),
                "dang_bao_che": (b.get("dangBaoChe") or "").strip(),
            }
        )
    rows = [r for r in rows if r["ten_thuoc"]]
    return all_items, rows


def write_outputs(raw_items: list[dict], csv_rows: list[dict]) -> None:
    with open("moh_raw.json", "w", encoding="utf-8") as f:
        json.dump({"totalCount": len(raw_items), "items": raw_items}, f, ensure_ascii=False, indent=2)

    fieldnames = list(csv_rows[0].keys()) if csv_rows else ["ten_thuoc"]
    with open("drugs_moh.csv", "w", newline="", encoding="utf-8-sig") as f:
        w = csv.DictWriter(f, fieldnames=fieldnames)
        w.writeheader()
        w.writerows(csv_rows)

    print("DONE")
    print("raw items:", len(raw_items))
    print("csv rows :", len(csv_rows))


def main():
    mode = SOURCE_MODE.strip().lower()
    if mode == "html":
        raw_items, csv_rows = crawl_html_source()
    elif mode == "api":
        raw_items, csv_rows = crawl_api_source()
    else:
        raise ValueError("SOURCE_MODE must be 'html' or 'api'")
    write_outputs(raw_items, csv_rows)


if __name__ == "__main__":
    main()
