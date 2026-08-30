# shellcheck shell=bash

# Role: Library: Lua path lint helper.
# Execution context: Sourced by CI, fault, and dev scripts; not a standalone gate.
# Invariants/contracts enforced: Helpers must not exit 0 on error paths when used as gate prerequisites.
# Verify: source scripts/lib/lint_lua_paths.sh
lint_lua_dirs=(
  deploy/nginx/lua
  internal/ingest
  internal/filter
  internal/stream
  scripts/test/load
)

lint_lua_find_args() {
  local dir
  for dir in "${lint_lua_dirs[@]}"; do
    [[ -d "$ROOT/$dir" ]] || continue
    printf '%s\0' "$dir"
  done
}
