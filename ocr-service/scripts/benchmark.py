"""
Benchmark script for the OCR service.

Sends every image in scripts/dataset/images/ to the running service,
compares results against the stored labels in scripts/dataset/labels/,
and prints a performance report.

Usage:
    python scripts/benchmark.py [--url http://localhost:8010] [--concurrency 4]
"""
from __future__ import annotations

import argparse
import csv
import datetime
import json
import re
import sys

# Ensure UTF-8 output on Windows terminals
if hasattr(sys.stdout, "reconfigure"):
    sys.stdout.reconfigure(encoding="utf-8", errors="replace")
import statistics
import time
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path

import urllib.request
import urllib.error

# ---------------------------------------------------------------------------
# Config
# ---------------------------------------------------------------------------
IMAGES_DIR = Path(__file__).parent / "dataset" / "images"
LABELS_DIR = Path(__file__).parent / "dataset" / "labels"
ENDPOINT   = "/ocr/prescriptions/parse"

# Fuzzy name-match threshold for comparing OCR output with ground-truth label
NAME_MATCH_THRESHOLD = 0.60


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _simplify(name: str) -> str:
    """Normalise a drug name to bare alphabetic tokens for loose comparison."""
    s = name.lower()
    s = re.sub(r"\d+(?:[.,]\d+)?\s*(?:mg|g|ml|mcg|iu|ui|%)", " ", s)
    s = re.sub(r"[^a-zà-ỹ]+", " ", s)
    return re.sub(r"\s+", " ", s).strip()


def _token_overlap(a: str, b: str) -> float:
    """Jaccard token overlap of two simplified drug names."""
    ta = set(_simplify(a).split())
    tb = set(_simplify(b).split())
    if not ta or not tb:
        return 0.0
    return len(ta & tb) / len(ta | tb)


def _names_match(a: str, b: str, threshold: float = NAME_MATCH_THRESHOLD) -> bool:
    return _token_overlap(a, b) >= threshold


def _strength_norm(s: str) -> str:
    """Normalize strength string for comparison (collapse spaces, unify decimal)."""
    s = re.sub(r"\s+", "", (s or "").lower())
    return s.replace(",", ".")


def _dose_norm(s: str) -> str:
    """Normalize dose-amount string (collapse spaces, unify unit spelling)."""
    s = re.sub(r"\s+", " ", (s or "").lower().strip()).replace(",", ".")
    s = re.sub(r"vi[eê]n", "viên", s)
    s = re.sub(r"g[oó]i", "gói", s)
    s = re.sub(r"[oô]ng", "ống", s)
    return s


_DOSE_SKIP = {"theo đơn", "1 liều", ""}  # fallback values that carry no info


def _call_ocr(base_url: str, image_path: Path) -> tuple[dict, float]:
    """POST image to OCR service. Returns (response_json, wall_time_ms)."""
    url = base_url.rstrip("/") + ENDPOINT
    boundary = "----FormBoundary7MA4YWxkTrZu0gW"
    image_bytes = image_path.read_bytes()
    mime = "image/jpeg"

    body = (
        f"--{boundary}\r\n"
        f'Content-Disposition: form-data; name="file"; filename="{image_path.name}"\r\n'
        f"Content-Type: {mime}\r\n\r\n"
    ).encode() + image_bytes + f"\r\n--{boundary}--\r\n".encode()

    req = urllib.request.Request(
        url,
        data=body,
        headers={"Content-Type": f"multipart/form-data; boundary={boundary}"},
        method="POST",
    )

    t0 = time.perf_counter()
    with urllib.request.urlopen(req, timeout=120) as resp:
        raw = resp.read()
    elapsed_ms = (time.perf_counter() - t0) * 1000
    return json.loads(raw), elapsed_ms


# ---------------------------------------------------------------------------
# Evaluation
# ---------------------------------------------------------------------------

class Result:
    def __init__(
        self,
        image: str,
        got_items: list[dict],
        expected_items: list[dict],
        wall_ms: float,
        service_ms: float,
        ocr_score: float,
        pass_used: str,
        error: str | None,
    ):
        self.image        = image
        self.got          = got_items
        self.expected     = expected_items
        self.wall_ms      = wall_ms
        self.service_ms   = service_ms
        self.ocr_score    = ocr_score
        self.pass_used    = pass_used
        self.error        = error

        got_names = [i.get("name", "") for i in got_items]
        exp_names = [i.get("name", "") for i in expected_items]

        # True-positive: every expected name matched by at least one got name
        self.tp = sum(
            1 for en in exp_names
            if any(_names_match(en, gn) for gn in got_names)
        )
        # False-negative: expected names with no match in got
        self.fn = len(exp_names) - self.tp
        # False-positive: got names with no match in expected
        self.fp = sum(
            1 for gn in got_names
            if not any(_names_match(gn, en) for en in exp_names)
        )
        self.n_expected = len(exp_names)
        self.n_got      = len(got_names)

        # Field accuracy: compare strength and dose_amount for name-matched pairs.
        # Build matched pairs greedily (each got item consumed at most once).
        unmatched_got = list(got_items)
        matched_pairs: list[tuple[dict, dict]] = []
        for exp_item in expected_items:
            en = exp_item.get("name", "")
            for gi, got_item in enumerate(unmatched_got):
                if _names_match(en, got_item.get("name", "")):
                    matched_pairs.append((exp_item, got_item))
                    unmatched_got.pop(gi)
                    break

        self.strength_total   = 0
        self.strength_correct = 0
        self.dose_total       = 0
        self.dose_correct     = 0
        for exp_item, got_item in matched_pairs:
            exp_str = str(exp_item.get("strength", "") or "")
            got_str = str(got_item.get("strength", "") or "")
            if exp_str:
                self.strength_total += 1
                if _strength_norm(exp_str) == _strength_norm(got_str):
                    self.strength_correct += 1

            # Labels use "dose_amount" key; OCR output uses "dosage".
            exp_dose = str(
                exp_item.get("dose_amount") or exp_item.get("dosage", "") or ""
            )
            got_dose = str(got_item.get("dosage", "") or "")
            if _dose_norm(exp_dose) not in _DOSE_SKIP:
                self.dose_total += 1
                if _dose_norm(exp_dose) == _dose_norm(got_dose):
                    self.dose_correct += 1

    @property
    def precision(self) -> float:
        return self.tp / (self.tp + self.fp) if (self.tp + self.fp) > 0 else 0.0

    @property
    def recall(self) -> float:
        return self.tp / (self.tp + self.fn) if (self.tp + self.fn) > 0 else 0.0

    @property
    def f1(self) -> float:
        p, r = self.precision, self.recall
        return 2 * p * r / (p + r) if (p + r) > 0 else 0.0


def evaluate_image(base_url: str, image_path: Path, label_path: Path | None) -> Result:
    expected: list[dict] = []
    if label_path and label_path.exists():
        try:
            data = json.loads(label_path.read_text(encoding="utf-8"))
            expected = data.get("items", [])
        except Exception:
            pass

    try:
        resp, wall_ms = _call_ocr(base_url, image_path)
        got       = resp.get("items", [])
        meta      = resp.get("meta", {})
        svc_ms    = meta.get("processing_ms", 0.0)
        ocr_score = meta.get("ocr_score", 0.0)
        pass_used = meta.get("pass_used", "?")
        return Result(image_path.name, got, expected, wall_ms, svc_ms, ocr_score, pass_used, None)
    except Exception as exc:
        return Result(image_path.name, [], expected, 0.0, 0.0, 0.0, "?", str(exc))


# ---------------------------------------------------------------------------
# Report
# ---------------------------------------------------------------------------

CYAN   = "\033[96m"
GREEN  = "\033[92m"
YELLOW = "\033[93m"
RED    = "\033[91m"
RESET  = "\033[0m"
BOLD   = "\033[1m"


def _color_f1(f1: float) -> str:
    if f1 >= 0.85:
        return f"{GREEN}{f1:.2f}{RESET}"
    if f1 >= 0.60:
        return f"{YELLOW}{f1:.2f}{RESET}"
    return f"{RED}{f1:.2f}{RESET}"


def print_report(results: list[Result]) -> None:
    ok = [r for r in results if r.error is None]
    failed = [r for r in results if r.error is not None]

    print(f"\n{BOLD}{'='*72}{RESET}")
    print(f"{BOLD}  HealthMate OCR Benchmark — {len(results)} images{RESET}")
    print(f"{'='*72}")

    # Per-image table
    header = f"{'Image':<45} {'Exp':>3} {'Got':>3} {'TP':>3} {'FP':>3} {'FN':>3}  {'F1':>5}  {'ms':>6}  Pass"
    print(f"\n{BOLD}{header}{RESET}")
    print("-" * 90)

    for r in sorted(results, key=lambda x: x.image):
        if r.error:
            print(f"{RED}{r.image:<45}  ERROR: {r.error[:35]}{RESET}")
            continue
        row = (
            f"{r.image:<45}"
            f" {r.n_expected:>3} {r.n_got:>3}"
            f" {r.tp:>3} {r.fp:>3} {r.fn:>3}"
            f"  {_color_f1(r.f1):>5}"
            f"  {r.service_ms:>6.0f}"
            f"  {r.pass_used}"
        )
        print(row)

    # Aggregate
    print(f"\n{BOLD}{'-'*72}{RESET}")
    if ok:
        all_tp = sum(r.tp for r in ok)
        all_fp = sum(r.fp for r in ok)
        all_fn = sum(r.fn for r in ok)
        macro_p  = statistics.mean(r.precision for r in ok)
        macro_r  = statistics.mean(r.recall    for r in ok)
        macro_f1 = statistics.mean(r.f1        for r in ok)
        micro_p  = all_tp / (all_tp + all_fp) if (all_tp + all_fp) > 0 else 0
        micro_r  = all_tp / (all_tp + all_fn) if (all_tp + all_fn) > 0 else 0
        micro_f1 = 2 * micro_p * micro_r / (micro_p + micro_r) if (micro_p + micro_r) > 0 else 0

        wall_times = [r.wall_ms for r in ok]
        svc_times  = [r.service_ms for r in ok]

        print(f"\n{BOLD}ACCURACY (vs ground-truth labels){RESET}")
        print(f"  Micro  — Precision: {micro_p:.3f}  Recall: {micro_r:.3f}  F1: {_color_f1(micro_f1)}")
        print(f"  Macro  — Precision: {macro_p:.3f}  Recall: {macro_r:.3f}  F1: {_color_f1(macro_f1)}")

        all_str_total   = sum(r.strength_total   for r in ok)
        all_str_correct = sum(r.strength_correct for r in ok)
        all_dose_total  = sum(r.dose_total        for r in ok)
        all_dose_correct = sum(r.dose_correct     for r in ok)
        print(f"\n{BOLD}FIELD ACCURACY (over name-matched pairs){RESET}")
        if all_str_total > 0:
            str_acc = all_str_correct / all_str_total
            color = GREEN if str_acc >= 0.85 else (YELLOW if str_acc >= 0.60 else RED)
            print(f"  Strength  : {all_str_correct}/{all_str_total} = {color}{str_acc:.3f}{RESET}")
        else:
            print(f"  Strength  : n/a (no ground-truth strength in labels)")
        if all_dose_total > 0:
            dose_acc = all_dose_correct / all_dose_total
            color = GREEN if dose_acc >= 0.85 else (YELLOW if dose_acc >= 0.60 else RED)
            print(f"  Dose amt  : {all_dose_correct}/{all_dose_total} = {color}{dose_acc:.3f}{RESET}")
        else:
            print(f"  Dose amt  : n/a (no ground-truth dose_amount in labels)")

        print(f"\n{BOLD}LATENCY (service-reported processing_ms){RESET}")
        print(f"  Median : {statistics.median(svc_times):.0f} ms")
        print(f"  Mean   : {statistics.mean(svc_times):.0f} ms")
        print(f"  P95    : {sorted(svc_times)[int(len(svc_times)*0.95)]:.0f} ms")
        print(f"  Min/Max: {min(svc_times):.0f} / {max(svc_times):.0f} ms")

        print(f"\n{BOLD}WALL-CLOCK (including HTTP + network){RESET}")
        print(f"  Median : {statistics.median(wall_times):.0f} ms")
        print(f"  Mean   : {statistics.mean(wall_times):.0f} ms")
        print(f"  P95    : {sorted(wall_times)[int(len(wall_times)*0.95)]:.0f} ms")

        pass_counts: dict[str, int] = {}
        for r in ok:
            pass_counts[r.pass_used] = pass_counts.get(r.pass_used, 0) + 1
        print(f"\n{BOLD}OCR PASS DISTRIBUTION{RESET}")
        for p, c in sorted(pass_counts.items(), key=lambda x: -x[1]):
            print(f"  {p:<35} {c:>3} ({100*c/len(ok):.0f}%)")

        # Worst cases
        worst = sorted(ok, key=lambda r: r.f1)[:5]
        if any(r.f1 < 0.8 for r in worst):
            print(f"\n{BOLD}{YELLOW}WORST F1 CASES{RESET}")
            for r in worst:
                if r.f1 >= 0.8:
                    break
                exp_names = [i.get("name","") for i in r.expected]
                got_names = [i.get("name","") for i in r.got]
                print(f"  {r.image}")
                print(f"    Expected: {exp_names}")
                print(f"    Got:      {got_names}")
                print(f"    F1={r.f1:.2f}  pass={r.pass_used}  ocr_score={r.ocr_score:.2f}")

    if failed:
        print(f"\n{RED}{BOLD}ERRORS ({len(failed)}){RESET}")
        for r in failed:
            print(f"  {r.image}: {r.error}")

    print(f"\n{'='*72}\n")


# ---------------------------------------------------------------------------
# Export
# ---------------------------------------------------------------------------

REPORTS_DIR = Path(__file__).parent / "reports"


def save_results(results: list[Result], tag: str) -> tuple[Path, Path]:
    """Save per-image CSV + aggregate JSON to scripts/reports/. Returns (csv_path, json_path)."""
    REPORTS_DIR.mkdir(exist_ok=True)
    ts = datetime.datetime.now().strftime("%Y%m%d_%H%M%S")
    stem = f"benchmark_{tag}_{ts}" if tag else f"benchmark_{ts}"

    # ── Per-image CSV ────────────────────────────────────────────────────────
    csv_path = REPORTS_DIR / f"{stem}_per_image.csv"
    with csv_path.open("w", newline="", encoding="utf-8") as fh:
        w = csv.writer(fh)
        w.writerow([
            "image", "n_expected", "n_got", "tp", "fp", "fn",
            "precision", "recall", "f1",
            "strength_total", "strength_correct",
            "dose_total", "dose_correct",
            "wall_ms", "service_ms", "ocr_score", "pass_used", "error",
        ])
        for r in sorted(results, key=lambda x: x.image):
            w.writerow([
                r.image, r.n_expected, r.n_got, r.tp, r.fp, r.fn,
                round(r.precision, 4), round(r.recall, 4), round(r.f1, 4),
                r.strength_total, r.strength_correct,
                r.dose_total, r.dose_correct,
                round(r.wall_ms, 1), round(r.service_ms, 1),
                round(r.ocr_score, 4), r.pass_used, r.error or "",
            ])

    # ── Aggregate JSON ───────────────────────────────────────────────────────
    ok = [r for r in results if r.error is None]
    agg: dict = {"generated_at": ts, "n_total": len(results), "n_ok": len(ok), "n_error": len(results) - len(ok)}
    if ok:
        all_tp = sum(r.tp for r in ok)
        all_fp = sum(r.fp for r in ok)
        all_fn = sum(r.fn for r in ok)
        micro_p = all_tp / (all_tp + all_fp) if (all_tp + all_fp) > 0 else 0
        micro_r = all_tp / (all_tp + all_fn) if (all_tp + all_fn) > 0 else 0
        micro_f1 = 2 * micro_p * micro_r / (micro_p + micro_r) if (micro_p + micro_r) > 0 else 0
        svc = sorted(r.service_ms for r in ok)
        wall = sorted(r.wall_ms for r in ok)
        n = len(ok)
        agg["accuracy"] = {
            "micro_precision": round(micro_p, 4),
            "micro_recall":    round(micro_r, 4),
            "micro_f1":        round(micro_f1, 4),
            "macro_precision": round(statistics.mean(r.precision for r in ok), 4),
            "macro_recall":    round(statistics.mean(r.recall    for r in ok), 4),
            "macro_f1":        round(statistics.mean(r.f1        for r in ok), 4),
            "tp": all_tp, "fp": all_fp, "fn": all_fn,
        }
        str_total   = sum(r.strength_total   for r in ok)
        str_correct = sum(r.strength_correct for r in ok)
        dose_total  = sum(r.dose_total       for r in ok)
        dose_correct = sum(r.dose_correct    for r in ok)
        agg["field_accuracy"] = {
            "strength_acc":  round(str_correct  / str_total,  4) if str_total  else None,
            "strength_n":    str_total,
            "dose_acc":      round(dose_correct / dose_total, 4) if dose_total else None,
            "dose_n":        dose_total,
        }
        agg["latency_ms"] = {
            "service_mean":   round(statistics.mean(svc), 1),
            "service_median": round(statistics.median(svc), 1),
            "service_p90":    round(svc[int(n * 0.90)], 1),
            "service_p95":    round(svc[int(n * 0.95)], 1),
            "service_min":    round(svc[0], 1),
            "service_max":    round(svc[-1], 1),
            "wall_mean":      round(statistics.mean(wall), 1),
            "wall_median":    round(statistics.median(wall), 1),
            "wall_p95":       round(wall[int(n * 0.95)], 1),
        }
        pass_counts: dict[str, int] = {}
        for r in ok:
            pass_counts[r.pass_used] = pass_counts.get(r.pass_used, 0) + 1
        agg["pass_distribution"] = pass_counts

    json_path = REPORTS_DIR / f"{stem}_summary.json"
    json_path.write_text(json.dumps(agg, ensure_ascii=False, indent=2), encoding="utf-8")

    print(f"\n  Saved CSV  : {csv_path}")
    print(f"  Saved JSON : {json_path}")
    return csv_path, json_path


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main() -> None:
    parser = argparse.ArgumentParser(description="HealthMate OCR benchmark")
    parser.add_argument("--url",         default="http://localhost:8010", help="Service base URL")
    parser.add_argument("--concurrency", type=int, default=1, help="Parallel requests (keep at 1 — PaddleOCR is not thread-safe)")
    parser.add_argument("--limit",       type=int, default=0, help="Max images (0 = all)")
    parser.add_argument("--save",        action="store_true", help="Save CSV + JSON report to scripts/reports/")
    parser.add_argument("--tag",         default="", help="Optional label appended to report filename")
    args = parser.parse_args()

    images = sorted(IMAGES_DIR.glob("*.jpg")) + sorted(IMAGES_DIR.glob("*.png"))
    if args.limit:
        images = images[: args.limit]

    print(f"\n{CYAN}Found {len(images)} images — sending to {args.url}{RESET}")
    print(f"{CYAN}Concurrency: {args.concurrency}{RESET}\n")

    # Warm-up: single request so PaddleOCR loads its model before timing starts
    if images:
        print("Warming up (first request loads PaddleOCR model)...")
        evaluate_image(args.url, images[0], None)
        print("Warm-up done.\n")

    results: list[Result] = []
    with ThreadPoolExecutor(max_workers=args.concurrency) as pool:
        futures = {
            pool.submit(
                evaluate_image,
                args.url,
                img,
                LABELS_DIR / (img.stem + ".json"),
            ): img
            for img in images
        }
        done = 0
        for fut in as_completed(futures):
            r = fut.result()
            results.append(r)
            done += 1
            status = f"F1={r.f1:.2f}" if not r.error else f"ERROR"
            print(f"  [{done:>3}/{len(images)}] {r.image:<45} {status}  {r.service_ms:.0f}ms")

    print_report(results)

    if args.save:
        save_results(results, args.tag)


if __name__ == "__main__":
    main()
