#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

make gen
bash "$SCRIPTS/ci/check_no_shard0_control.sh"
bash "$SCRIPTS/ci/compose_profile_check.sh"
make lint
make test-integration
make test-fault
