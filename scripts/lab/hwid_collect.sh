#!/usr/bin/env bash

set -euo pipefail

# Role: Lab-only HWID fingerprint collection print test for license appliance binding.
# Execution context: Operator lab host; HWID_LAB_COLLECT=1 enables verbose collect path.
# Invariants/contracts enforced: TestHWID_LabCollectPrint completes without panic; output is deterministic shape.
# Verify: bash scripts/lab/hwid_collect.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

export HWID_LAB_COLLECT=1
go test ./internal/licensing/ -run '^TestHWID_LabCollectPrint$' -count=1 -v
