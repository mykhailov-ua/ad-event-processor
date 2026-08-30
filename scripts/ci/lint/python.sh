#!/usr/bin/env bash
set -euo pipefail

# Role: Lint gate: ruff on Python sources.
# Execution context: CI merge-lint via lint.sh.
# Invariants/contracts enforced: Child linter failure propagates to exit 1.
# Verify: bash scripts/ci/lint/python.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

MODEL_DIR="$ROOT/model"
RUFF_VERSION="${RUFF_VERSION:-0.11.12}"

ensure_ruff() {
  if command -v ruff > /dev/null 2>&1; then
    return 0
  fi
  if [[ -x "$MODEL_DIR/.venv/bin/ruff" ]]; then
    export PATH="$MODEL_DIR/.venv/bin:$PATH"
    return 0
  fi
  echo "lint_python_gate: installing ruff ${RUFF_VERSION}..." >&2
  python3 -m pip install --user "ruff==${RUFF_VERSION}" > /dev/null
  export PATH="${HOME}/.local/bin:${PATH}"
}

ensure_ruff

if [[ -d "$MODEL_DIR" ]]; then
  echo "lint_python_gate: ruff check (model/)..."
  (cd "$MODEL_DIR" && ruff check .)
fi

if [[ -f "$ROOT/scripts/dev/gen_ingest_gnet.py" ]]; then
  echo "lint_python_gate: ruff check (scripts/dev/gen_ingest_gnet.py)..."
  ruff check "$ROOT/scripts/dev/gen_ingest_gnet.py"
fi

if [[ ! -d "$MODEL_DIR" && ! -f "$ROOT/scripts/dev/gen_ingest_gnet.py" ]]; then
  echo "lint_python_gate: no Python targets" >&2
  exit 1
fi

echo "lint_python_gate: OK"
