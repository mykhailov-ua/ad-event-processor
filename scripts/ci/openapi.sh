#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

go test ./tests/contract/... -count=1
go test ./internal/openapi/... -count=1
bash "$SCRIPTS/ci/check_no_html_success.sh"

before_hash=$(sha256sum docs/openapi/openapi.yaml | awk '{print $1}')
go run ./cmd/openapi-gen >/dev/null
after_hash=$(sha256sum docs/openapi/openapi.yaml | awk '{print $1}')
if [ "$before_hash" != "$after_hash" ]; then
	echo "ERROR: docs/openapi/openapi.yaml is stale; run: go run ./cmd/openapi-gen" >&2
	exit 1
fi

echo "openapi: contract + drift check OK"
