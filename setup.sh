#!/bin/sh
# Keep this script small and idempotent. Add setup steps only when needed.
set -eu

cd "$(git rev-parse --show-toplevel)"

git config --local core.hooksPath .githooks
chmod +x .githooks/*

echo "Local Git hooks installed from .githooks."
echo "Verify with: git config --local --get core.hooksPath"
