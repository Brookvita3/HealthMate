#!/usr/bin/env bash
# Unit tests for report capture. Usage: ./scripts/run_unit_tests.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export FORCE_COLOR=1

banner() {
  local title="$1"
  printf '\n%s\n' "$(printf '=%.0s' {1..72})"
  printf '  %s\n' "$title"
  printf '%s\n' "$(printf '=%.0s' {1..72})"
}

banner "HealthMate OCR — Unit tests (prescription parser)"
echo "  Repo     : $ROOT"
echo "  Date     : $(date '+%Y-%m-%d %H:%M:%S')"
echo "  Python   : $(python --version 2>&1)"
echo ""
echo "  Collecting tests..."
python -m pytest tests/test_prescription_parser.py --collect-only -q | sed 's/^/  /'
echo ""
echo "  Running tests..."
python -m pytest tests/test_prescription_parser.py -v --tb=line -ra --durations=5 --color=yes
echo ""
banner "RESULT: PASSED (all unit tests)"
