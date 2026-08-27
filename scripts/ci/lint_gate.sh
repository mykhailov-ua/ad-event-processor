#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [[ "${CI:-}" == "true" && "${LINT_INCREMENTAL:-}" != "1" ]]; then
  export LINT_STRICT=1
fi

echo "lint_gate: Go (golangci-lint, LINT_STRICT=${LINT_STRICT:-0})..."
bash "$SCRIPTS/ci/lint_go_gate.sh" all

echo "lint_gate: Go (go vet ./...)..."
go vet ./...

echo "lint_gate: Go (gopls check, warning+)..."
bash "$SCRIPTS/ci/lint_gopls_gate.sh"

echo "lint_gate: Lua (luacheck + lua-language-server)..."
bash "$SCRIPTS/ci/lint_lua_gate.sh"

echo "lint_gate: TypeScript/JavaScript (tsc + node --check)..."
bash "$SCRIPTS/ci/lint_ts_gate.sh"

echo "lint_gate: configs (compose + nginx)..."
bash "$SCRIPTS/ci/lint_configs_gate.sh"

echo "lint_gate: OK"
