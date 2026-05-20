# Chay unit test parser - output gon, phu hop chup man hinh bao cao / luan van.
# Usage (PowerShell):  .\scripts\run_unit_tests.ps1
# Ghi log:            .\scripts\run_unit_tests.ps1 -SaveLog

param([switch]$SaveLog)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $Root

$env:FORCE_COLOR = "1"
$env:PYTEST_CURRENT_TEST = ""

function Write-Banner([string]$Title) {
    $line = ("=" * 72)
    Write-Host ""
    Write-Host $line -ForegroundColor DarkCyan
    Write-Host "  $Title" -ForegroundColor Cyan
    Write-Host $line -ForegroundColor DarkCyan
}

Write-Banner "HealthMate OCR - Unit tests (prescription parser)"
Write-Host ("  Repo     : {0}" -f $Root) -ForegroundColor DarkGray
Write-Host ("  Date     : {0}" -f (Get-Date -Format "yyyy-MM-dd HH:mm:ss")) -ForegroundColor DarkGray
try {
    $pyVer = (python --version 2>&1).ToString().Trim()
    Write-Host ("  Python   : {0}" -f $pyVer) -ForegroundColor DarkGray
} catch {
    Write-Host "  Python   : (not found on PATH)" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "  Collecting tests..." -ForegroundColor Yellow
python -m pytest tests/test_prescription_parser.py --collect-only -q 2>&1 | ForEach-Object { Write-Host "  $_" }

Write-Host ""
Write-Host "  Running tests..." -ForegroundColor Yellow
$pytestArgs = @(
    "tests/test_prescription_parser.py",
    "-v",
    "--tb=line",
    "-ra",
    "--durations=5",
    "--color=yes"
)

if ($SaveLog) {
    $reportDir = Join-Path $Root "scripts\reports"
    New-Item -ItemType Directory -Force -Path $reportDir | Out-Null
    $logPath = Join-Path $reportDir ("pytest_{0:yyyyMMdd_HHmmss}.log" -f (Get-Date))
    Write-Host ("  Log file : {0}" -f $logPath) -ForegroundColor DarkGray
    Write-Host ""
    python -m pytest @pytestArgs 2>&1 | Tee-Object -FilePath $logPath
    $exitCode = $LASTEXITCODE
} else {
    Write-Host ""
    python -m pytest @pytestArgs
    $exitCode = $LASTEXITCODE
}

Write-Host ""
if ($exitCode -eq 0) {
    Write-Banner "RESULT: PASSED (all unit tests)"
    exit 0
} else {
    Write-Banner "RESULT: FAILED - see traceback above"
    exit $exitCode
}
