#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [[ "${CI:-}" == "true" && "${LINT_INCREMENTAL:-}" != "1" ]]; then
  export LINT_STRICT=1
fi

echo "lint_gate: Go (golangci-lint, LINT_STRICT=${LINT_STRICT:-0})..."
bash "$SCRIPTS/ci/lint/go.sh" all

echo "lint_gate: Go (go vet ./...)..."
go vet ./...

echo "lint_gate: Go (rangeint modernize static gate)..."
bash "$SCRIPTS/ci/lint/go_modernize.sh"

echo "lint_gate: Go (gopls check, warning+)..."
bash "$SCRIPTS/ci/lint/gopls.sh"

echo "lint_gate: Lua (luacheck + lua-language-server)..."
bash "$SCRIPTS/ci/lint/lua.sh"

echo "lint_gate: Shell (shellcheck)..."
bash "$SCRIPTS/ci/lint/shell.sh"

echo "lint_gate: Python (ruff)..."
bash "$SCRIPTS/ci/lint/python.sh"

echo "lint_gate: Protobuf (buf lint)..."
bash "$SCRIPTS/ci/lint/proto.sh"

echo "lint_gate: GitHub workflows (actionlint)..."
bash "$SCRIPTS/ci/lint/workflows.sh"

echo "lint_gate: TypeScript/JavaScript (tsc + node --check)..."
bash "$SCRIPTS/ci/lint/ts.sh"

echo "lint_gate: configs (compose + nginx + OpenAPI)..."
bash "$SCRIPTS/ci/lint/configs.sh"

echo "lint_gate: OK"
