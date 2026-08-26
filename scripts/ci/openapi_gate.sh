#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

OPENAPI_DIR="$ROOT/api/openapi"
MAIN_SPEC="$OPENAPI_DIR/openapi.yaml"
BUNDLE="$OPENAPI_DIR/openapi.bundle.yaml"
RULESET="$OPENAPI_DIR/spectral.yaml"
WEB_DIR="$ROOT/web"

echo "openapi_gate: export route stubs and bundle..."
go run ./cmd/openapi-export

echo "openapi_gate: catalog parity test..."
go test ./internal/openapi/ -count=1

if [[ ! -f "$WEB_DIR/package.json" ]]; then
  echo "openapi_gate: missing $WEB_DIR/package.json" >&2
  exit 1
fi

if [[ ! -d "$WEB_DIR/node_modules/@stoplight/spectral-cli" ]]; then
  echo "openapi_gate: npm ci (web)..."
  (cd "$WEB_DIR" && npm ci)
fi

echo "openapi_gate: spectral lint on pilot spec..."
(cd "$WEB_DIR" && npx spectral lint "$MAIN_SPEC" --ruleset "$RULESET")

echo "openapi_gate: generated TS types match spec..."
(cd "$WEB_DIR" && npm run openapi:types)
if ! git diff --quiet -- web/src/types/generated/openapi.d.ts 2> /dev/null; then
  echo "openapi_gate: web/src/types/generated/openapi.d.ts is stale; run npm run openapi:types" >&2
  git diff -- web/src/types/generated/openapi.d.ts | head -40 >&2 || true
  exit 1
fi

bash "$SCRIPTS/ci/openapi_breaking_gate.sh"

echo "openapi_gate: OK"
