#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

CHECK=0
if [[ "${1:-}" == "--check" ]]; then
  CHECK=1
fi

GOFUMPT_PKG="mvdan.cc/gofumpt@v0.7.0"
GOIMPORTS_PKG="golang.org/x/tools/cmd/goimports@v0.30.0"
SHFMT_PKG="mvdan.cc/sh/v3/cmd/shfmt@v3.10.0"
PRETTIER_VERSION="3.5.3"

go_dirs() {
  go list -f '{{.Dir}}' \
    ./cmd/... \
    ./internal/... \
    ./pkg/... \
    ./tests/... \
    ./api/... \
    ./scripts/... \
    2> /dev/null | sort -u
}

run_check() {
  local label="$1"
  shift
  local out
  if ! out="$("$@" 2>&1)"; then
    printf 'format: %s failed\n%s\n' "$label" "$out" >&2
    return 1
  fi
  if [[ -n "$out" ]]; then
    printf 'format: %s drift\n%s\n' "$label" "$out" >&2
    return 1
  fi
}

format_go() {
  local -a dirs=()
  mapfile -t dirs < <(go_dirs)
  if [[ ${#dirs[@]} -eq 0 ]]; then
    echo "format: skip go (no packages)" >&2
    return 0
  fi
  echo "format: go (${#dirs[@]} packages)..."
  if [[ "$CHECK" -eq 1 ]]; then
    run_check gofumpt go run "$GOFUMPT_PKG" -d "${dirs[@]}"
    run_check goimports go run "$GOIMPORTS_PKG" -d "${dirs[@]}"
  else
    go run "$GOFUMPT_PKG" -w "${dirs[@]}"
    go run "$GOIMPORTS_PKG" -w "${dirs[@]}"
  fi
}

shell_files() {
  find scripts deploy \
    -type f -name '*.sh' \
    ! -path '*/node_modules/*' \
    ! -path '*/vendor/*' \
    2> /dev/null | sort -u
}

format_shell() {
  local -a files=()
  mapfile -t files < <(shell_files)
  if [[ ${#files[@]} -eq 0 ]]; then
    echo "format: skip shell (no scripts)" >&2
    return 0
  fi
  echo "format: shell (${#files[@]} files)..."
  local -a shfmt_args=(-i 2 -ci -bn -sr)
  if [[ "$CHECK" -eq 1 ]]; then
    run_check shfmt go run "$SHFMT_PKG" "${shfmt_args[@]}" -d "${files[@]}"
  else
    go run "$SHFMT_PKG" "${shfmt_args[@]}" -w "${files[@]}"
  fi
}

prettier_files() {
  find web/src web/scripts web/e2e .github deploy \
    -type f \( \
    -name '*.ts' -o -name '*.tsx' -o -name '*.js' -o -name '*.mjs' \
    -o -name '*.css' -o -name '*.json' -o -name '*.yml' -o -name '*.yaml' \
    \) \
    ! -path '*/node_modules/*' \
    ! -path '*/dist/*' \
    ! -path '*/vendor/*' \
    2> /dev/null | sort -u
}

format_prettier() {
  if ! command -v npx > /dev/null 2>&1; then
    echo "format: skip prettier (npx not found)" >&2
    return 0
  fi
  local -a files=()
  mapfile -t files < <(prettier_files)
  if [[ ${#files[@]} -eq 0 ]]; then
    echo "format: skip prettier (no files)" >&2
    return 0
  fi
  echo "format: prettier (${#files[@]} files)..."
  if [[ "$CHECK" -eq 1 ]]; then
    npx --yes "prettier@${PRETTIER_VERSION}" --check "${files[@]}"
  else
    npx --yes "prettier@${PRETTIER_VERSION}" --write "${files[@]}"
  fi
}

bpf_files() {
  find deploy -type f \( -name '*.c' -o -name '*.h' \) 2> /dev/null | sort -u
}

format_clang() {
  if ! command -v clang-format > /dev/null 2>&1; then
    echo "format: skip clang-format (not installed)" >&2
    return 0
  fi
  local -a files=()
  mapfile -t files < <(bpf_files)
  if [[ ${#files[@]} -eq 0 ]]; then
    echo "format: skip clang-format (no bpf sources)" >&2
    return 0
  fi
  echo "format: clang-format (${#files[@]} files)..."
  if [[ "$CHECK" -eq 1 ]]; then
    local f
    for f in "${files[@]}"; do
      if ! clang-format --dry-run -Werror "$f" > /dev/null 2>&1; then
        echo "format: clang-format drift in $f" >&2
        clang-format "$f" | diff -u "$f" - >&2 || true
        return 1
      fi
    done
  else
    clang-format -i "${files[@]}"
  fi
}

format_go
format_shell
format_prettier
format_clang

if [[ "$CHECK" -eq 1 ]]; then
  echo "format: check passed"
else
  echo "format: done"
fi
