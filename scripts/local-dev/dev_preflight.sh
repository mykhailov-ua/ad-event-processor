#!/usr/bin/env bash
set -euo pipefail
export PREFLIGHT_SMOKE=1
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/dev/preflight.sh" "$@"
