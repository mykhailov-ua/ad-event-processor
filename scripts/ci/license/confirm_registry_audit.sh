#!/usr/bin/env bash
set -euo pipefail

# Role: License gate: Trial registry audit confirm.
# Execution context: CI license-verify tier or release QA.
# Invariants/contracts enforced: Required rows fail closed; optional rows use skip_gate with env flags.
# Verify: bash scripts/ci/license/confirm_registry_audit.sh
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"

if [[ ! -d "$ROOT/web/src" ]]; then
  echo "confirm_registry_audit: skipped (web/ absent)"
  exit 0
fi

cd "$ROOT/web"

node --experimental-strip-types --import ./scripts/register_ts.mjs --input-type=module -e "
import { registry } from './src/helpers/confirm_registry.js';
import { REQUIRED_CONFIRM_KEYS } from './src/helpers/confirm_catalog.js';

const missing = REQUIRED_CONFIRM_KEYS.filter((k) => !registry.has(k));
if (missing.length) {
  console.error('Missing confirm_registry entries:', missing.join(', '));
  process.exit(1);
}
console.log('confirm_registry audit: OK (' + REQUIRED_CONFIRM_KEYS.length + ' keys)');
"
