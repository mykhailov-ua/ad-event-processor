#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

OPENAPI_DIR="$ROOT/api/openapi"
MAIN_SPEC="$OPENAPI_DIR/openapi.yaml"
RULESET="$OPENAPI_DIR/spectral.yaml"

echo "openapi_gate: export route stubs and bundle..."
go run ./cmd/openapi-export

echo "openapi_gate: catalog parity test..."
go test ./internal/openapi/ -count=1

if command -v npx > /dev/null 2>&1; then
  echo "openapi_gate: spectral lint..."
  npx --yes @stoplight/spectral-cli@6.14.3 lint "$MAIN_SPEC" --ruleset "$RULESET"
else
  echo "openapi_gate: skip spectral (npx not available)"
fi

bash "$SCRIPTS/ci/openapi_breaking_gate.sh"

echo "openapi_gate: OK"
