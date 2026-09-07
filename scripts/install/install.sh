#!/usr/bin/env bash
# Role: Unified installer entry (curl | bash). Delegates to ad-event-processor-install.sh.
# Verify: bash scripts/install/install.sh --help
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "${SCRIPT_DIR}/scripts/install/ad-event-processor-install.sh" ]]; then
  ROOT="$SCRIPT_DIR"
elif [[ -f "${SCRIPT_DIR}/ad-event-processor-install.sh" ]]; then
  ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
else
  ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
fi
exec bash "$ROOT/scripts/install/ad-event-processor-install.sh" "$@"
