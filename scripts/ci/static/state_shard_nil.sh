#!/usr/bin/env bash
set -euo pipefail

# Role: Static gate: Shard-0 nil optional client proofs.
# Execution context: CI merge-pr-fast via pr_fast unless noted.
# Invariants/contracts enforced: Non-zero exit on contract violation; no silent pass on failure.
# Verify: bash scripts/ci/static/state_shard_nil.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

echo "state_shard_nil_gate: behavioral regression (no Docker)..."
go test -count=1 -run 'TestShard0Nil_|TestPingConnectedRedisShards' ./internal/controlplane/ ./internal/ingest/
echo "state_shard_nil_gate: OK"
