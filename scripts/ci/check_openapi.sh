#!/usr/bin/env bash
# GAP-PROD-03: OpenAPI drift gate — contract tests + spec regeneration check.
set -euo pipefail

source "$(cd "$(dirname "$0")/../lib" && pwd)/paths.sh"
cd "$ROOT"

go test ./tests/contract/... -count=1
go test ./internal/openapi/... -count=1

before_hash=$(sha256sum docs/openapi/openapi.yaml | awk '{print $1}')
go run ./scripts/openapi >/dev/null
after_hash=$(sha256sum docs/openapi/openapi.yaml | awk '{print $1}')
if [ "$before_hash" != "$after_hash" ]; then
	echo "ERROR: docs/openapi/openapi.yaml is stale; run: go run ./scripts/openapi" >&2
	exit 1
fi

echo "openapi: contract + drift check OK"
