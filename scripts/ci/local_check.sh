#!/usr/bin/env bash
# Fast local gate before push: lint, alloc gate, unit+integration tests, docker build.
set -euo pipefail

source "$(cd "$(dirname "$0")/../lib" && pwd)/paths.sh"
cd "$ROOT"

bash "$SCRIPTS/codegen/validate_configs.sh"
bash "$SCRIPTS/ci/check_comments.sh"
bash "$SCRIPTS/ci/check_compliance.sh"
bash "$SCRIPTS/ci/check_ch_direct.sh"
bash "$SCRIPTS/ci/check_openapi.sh"
make lint
make test-alloc-gate
make test
make build
