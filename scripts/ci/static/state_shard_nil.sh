#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

echo "state_shard_nil_gate: behavioral regression (no Docker)..."
go test -count=1 -run 'TestShard0Nil_|TestPingConnectedRedisShards' ./internal/controlplane/ ./internal/ingest/
echo "state_shard_nil_gate: OK"
