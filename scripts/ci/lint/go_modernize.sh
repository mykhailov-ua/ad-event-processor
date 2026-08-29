#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

echo "lint_go_modernize_gate: rangeint static scan..."
bash "$SCRIPTS/ci/static/go_rangeint.sh"
