#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

LUA_LINT_IMAGE="${LUA_LINT_IMAGE:-openresty/openresty:alpine}"
LUA_LS_VERSION="${LUA_LS_VERSION:-3.19.1}"

run_luacheck() {
  echo "lint_lua_gate: luacheck (deploy/nginx/lua, internal/ingestion)..."
  if command -v luacheck > /dev/null 2>&1; then
    luacheck deploy/nginx/lua internal/ingestion --config .luacheckrc
    return 0
  fi
  if command -v docker > /dev/null 2>&1; then
    docker run --rm -v "$ROOT:/work" -w /work "$LUA_LINT_IMAGE" \
      sh -c 'apk add --no-cache luacheck >/dev/null 2>&1 && luacheck deploy/nginx/lua internal/ingestion --config .luacheckrc'
    return 0
  fi
  echo "lint_lua_gate: need luacheck or docker" >&2
  exit 1
}

resolve_lua_language_server() {
  if command -v lua-language-server > /dev/null 2>&1; then
    command -v lua-language-server
    return 0
  fi
  local ext
  for ext in \
    "$HOME/.cursor/extensions"/sumneko.lua-*/server/bin/lua-language-server \
    "$HOME/.vscode/extensions"/sumneko.lua-*/server/bin/lua-language-server; do
    if [[ -x "$ext" ]]; then
      echo "$ext"
      return 0
    fi
  done
  local cache="$ROOT/.cache/lua-language-server"
  local bin="$cache/lua-language-server-${LUA_LS_VERSION}-linux-x64/bin/lua-language-server"
  if [[ -x "$bin" ]]; then
    echo "$bin"
    return 0
  fi
  echo "lint_lua_gate: installing lua-language-server ${LUA_LS_VERSION}..." >&2
  mkdir -p "$cache"
  local tarball="lua-language-server-${LUA_LS_VERSION}-linux-x64.tar.gz"
  curl -fsSL "https://github.com/LuaLS/lua-language-server/releases/download/${LUA_LS_VERSION}/${tarball}" \
    -o "$cache/$tarball"
  tar -xzf "$cache/$tarball" -C "$cache"
  if [[ ! -x "$bin" ]]; then
    echo "lint_lua_gate: lua-language-server binary missing after extract: $bin" >&2
    exit 1
  fi
  echo "$bin"
}

run_lua_language_server() {
  local luals
  luals="$(resolve_lua_language_server)"
  echo "lint_lua_gate: lua-language-server (--check=. --checklevel=Warning)..."
  local out
  out="$("$luals" --check=. --checklevel=Warning 2>&1)" || true
  printf '%s\n' "$out"
  if printf '%s\n' "$out" | grep -qE 'Diagnosis complete, [1-9][0-9]* problems found'; then
    echo "lint_lua_gate: LuaLS warnings present" >&2
    exit 1
  fi
  if ! printf '%s\n' "$out" | grep -q 'Diagnosis completed, no problems found'; then
    echo "lint_lua_gate: unexpected lua-language-server output" >&2
    exit 1
  fi
}

run_luacheck
run_lua_language_server
echo "lint_lua_gate: OK"
