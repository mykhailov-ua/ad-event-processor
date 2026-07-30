#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

make gen
bash "$SCRIPTS/ci/comments.sh"
make lint
make test-integration
make test-fault
