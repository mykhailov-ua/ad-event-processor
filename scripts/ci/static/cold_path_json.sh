#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

echo "cold_path_json_gate: body limit regression tests..."

go test ./pkg/coldpath/ -run='TestReadLimitedBody|TestDecodeRequestOrBadRequest' -count=1
go test ./internal/controlplane/ -run='TestColdPathJSON|TestLoginBodySizeLimit' -count=1
go test ./internal/controlplane/ -run='TestColdPathJSON' -count=1
go test ./internal/payment/ -run='TestColdPathJSON' -count=1

echo "cold_path_json_gate: OK"
