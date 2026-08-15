#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

make gen
go test -race -short -count=1 -timeout=25m ./internal/... ./pkg/...
