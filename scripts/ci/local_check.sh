#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

bash "$SCRIPTS/ci/validate_configs.sh"
bash "$SCRIPTS/ci/check_scripts_layout.sh"
bash "$SCRIPTS/ci/comments.sh"
bash "$SCRIPTS/ci/compliance.sh"
bash "$SCRIPTS/ci/ch_direct.sh"
bash "$SCRIPTS/ci/openapi.sh"
make lint
make test-alloc-gate
make test
make build
