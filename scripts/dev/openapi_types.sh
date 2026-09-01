#!/usr/bin/env bash
set -euo pipefail

# Role: Export OpenAPI bundle and regenerate admin TypeScript types.
# Execution context: Local dev via make openapi-types; requires go and npx.
# Verify: bash scripts/dev/openapi_types.sh
#          make openapi-types
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

OPENAPI_DIR="$ROOT/api/openapi"
BUNDLE="$OPENAPI_DIR/openapi.bundle.yaml"
MAIN_SPEC="$OPENAPI_DIR/openapi.yaml"
GENERATED_DIR="$ROOT/web/src/types/generated"
OUTPUT="$GENERATED_DIR/openapi.d.ts"
OPENAPI_TS_VERSION="7.6.1"

need_export=0
if [[ ! -f "$BUNDLE" ]]; then
  need_export=1
else
  while IFS= read -r -d '' spec_file; do
    if [[ "$spec_file" -nt "$BUNDLE" ]]; then
      need_export=1
      break
    fi
  done < <(find "$OPENAPI_DIR" -type f \( -name '*.yaml' -o -name '*.yml' \) -print0)
fi

if [[ "$need_export" -eq 1 ]]; then
  echo "openapi_types: exporting bundle..."
  go run ./cmd/openapi-export
else
  echo "openapi_types: bundle up to date ($BUNDLE)"
fi

mkdir -p "$GENERATED_DIR"

echo "openapi_types: generating $OUTPUT..."
npx --yes "openapi-typescript@${OPENAPI_TS_VERSION}" "$BUNDLE" -o "$OUTPUT"

echo "openapi_types: OK"
