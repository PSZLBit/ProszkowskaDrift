# Keep this script small and idempotent. Add setup steps only when needed.
# Windows PowerShell setup for local Git hooks.
$ErrorActionPreference = "Stop"

$repoRoot = git rev-parse --show-toplevel
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
Set-Location -LiteralPath $repoRoot

git config --local core.hooksPath .githooks
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

# Hook files are tracked as 100755. Do not rewrite the index here:
# setup must not stage local hook edits.
Write-Host "Local Git hooks installed from .githooks."
Write-Host "Verify with: git config --local --get core.hooksPath"
