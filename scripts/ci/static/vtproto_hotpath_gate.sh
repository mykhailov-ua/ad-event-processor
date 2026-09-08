#!/usr/bin/env bash
set -euo pipefail

# Role: events_vtproto.pb.go must stay vtproto-hotpath patched when present in tree.
# Verify: bash scripts/ci/static/vtproto_hotpath_gate.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

path="internal/ingest/pb/events_vtproto.pb.go"
if [[ ! -f "$path" ]]; then
  echo "vtproto_hotpath_gate: SKIP (no $path; run make gen)"
  exit 0
fi

failed=0

if ! rg -q 'appendReuseBytes\(m\.ExtraKeys' "$path"; then
  echo "vtproto_hotpath_gate: FAIL missing appendReuseBytes patch in $path"
  failed=1
fi

if rg -n 'if m\.[A-Za-z0-9]+ == nil \{\s*m\.[A-Za-z0-9]+ = \[\]byte\{\}' "$path" 2> /dev/null; then
  echo "vtproto_hotpath_gate: FAIL nil-slice guard still present in $path"
  failed=1
fi

if [[ "$failed" -ne 0 ]]; then
  echo "vtproto_hotpath_gate: run go run ./cmd/patch-vtproto-hotpath"
  exit 1
fi

echo "vtproto_hotpath_gate: OK"
