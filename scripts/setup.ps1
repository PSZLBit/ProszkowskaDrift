# Keep this script small and idempotent. Add setup steps only when needed.
# Windows PowerShell twin of setup.sh (Linux/macOS/Git Bash).
$ErrorActionPreference = "Stop"

Set-Location (git rev-parse --show-toplevel)
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

git config --local core.hooksPath .githooks
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

# Keep hooks executable in the Git index so Linux/macOS users can run them.
# This is a no-op while the index entry is already 100755.
foreach ($hook in (git ls-files .githooks)) {
  git update-index --chmod=+x $hook
  if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

Write-Host "Local Git hooks installed from .githooks."
Write-Host "Verify with: git config --local --get core.hooksPath"
