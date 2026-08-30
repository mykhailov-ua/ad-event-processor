#!/usr/bin/env bash
set -euo pipefail

# Role: Lint gate: buf lint on protobuf.
# Execution context: CI merge-lint via lint.sh.
# Invariants/contracts enforced: Child linter failure propagates to exit 1.
# Verify: bash scripts/ci/lint/proto.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [[ ! -f "$ROOT/api/buf.yaml" ]]; then
  echo "lint_proto_gate: skip (api/buf.yaml missing)"
  exit 0
fi

echo "lint_proto_gate: buf lint (api/)..."
(cd "$ROOT/api" && go run github.com/bufbuild/buf/cmd/buf@latest lint)
echo "lint_proto_gate: OK"
