# shellcheck shell=bash

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
