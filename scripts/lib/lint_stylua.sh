# shellcheck shell=bash

STYLUA_VERSION="${STYLUA_VERSION:-2.0.2}"

resolve_stylua() {
  if command -v stylua > /dev/null 2>&1; then
    command -v stylua
    return 0
  fi
  local cache="${ROOT:?}/.cache/stylua"
  local bin="$cache/stylua-${STYLUA_VERSION}-linux-x86_64"
  if [[ -x "$bin" ]]; then
    echo "$bin"
    return 0
  fi
  echo "lint_stylua: installing stylua ${STYLUA_VERSION}..." >&2
  mkdir -p "$cache"
  local zip="$cache/stylua.zip"
  curl -fsSL "https://github.com/JohnnyMorganz/StyLua/releases/download/v${STYLUA_VERSION}/stylua-linux-x86_64.zip" \
    -o "$zip"
  unzip -o -q "$zip" -d "$cache"
  if [[ -f "$cache/stylua" ]]; then
    mv "$cache/stylua" "$bin"
  fi
  chmod +x "$bin"
  if [[ ! -x "$bin" ]]; then
    echo "lint_stylua: stylua binary missing after extract: $bin" >&2
    return 1
  fi
  echo "$bin"
}
