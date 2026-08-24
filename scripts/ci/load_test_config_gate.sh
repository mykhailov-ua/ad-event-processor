#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"

bash "$SCRIPTS/lib/render_load_test_config.sh" --check
echo "load_test_config_gate: ok"
