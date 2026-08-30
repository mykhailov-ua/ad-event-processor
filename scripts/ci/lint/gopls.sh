#!/usr/bin/env bash
set -euo pipefail

# Role: Lint gate: gopls check at warning level.
# Execution context: CI merge-lint via lint.sh.
# Invariants/contracts enforced: Child linter failure propagates to exit 1.
# Verify: bash scripts/ci/lint/gopls.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

GOPLS_JOBS="${GOPLS_JOBS:-8}"
export GOOS="${GOPLS_GOOS:-linux}"
export GOARCH="${GOPLS_GOARCH:-amd64}"

ensure_gopls() {
  if command -v gopls > /dev/null 2>&1; then
    return 0
  fi
  echo "lint_gopls_gate: installing gopls..."
  go install golang.org/x/tools/gopls@v0.22.0
  GOPATH="$(go env GOPATH)"
  if [[ -z "$GOPATH" ]]; then
    GOPATH="$HOME/go"
  fi
  export PATH="$GOPATH/bin:$PATH"
}

should_skip_dir() {
  local dir="$1"
  case "$dir" in
    */pb | */pb/* | */db | */db/* | */api/gen/*) return 0 ;;
  esac
  return 1
}

ensure_gopls
echo "lint_gopls_gate: gopls check (GOOS=$GOOS GOARCH=$GOARCH, jobs=$GOPLS_JOBS)..."

list_file="$(mktemp)"
trap 'rm -f "$list_file"' EXIT
while IFS= read -r dir; do
  should_skip_dir "$dir" && continue
  find "$dir" -maxdepth 1 -name '*.go' -print 2> /dev/null | rg -q '.' || continue
  printf '%s\0' "$dir"
done < <(go list -f '{{.Dir}}' ./... 2> /dev/null) > "$list_file"

out_file="$(mktemp)"
trap 'rm -f "$list_file" "$out_file"' EXIT

check_one_dir() {
  local dir="$1"
  local files=()
  local f
  while IFS= read -r f; do
    files+=("$f")
  done < <(find "$dir" -maxdepth 1 -name '*.go' -print 2> /dev/null)
  ((${#files[@]})) || return 0
  gopls check -severity=warning "${files[@]}"
}

export -f check_one_dir
export GOOS GOARCH

if ! xargs -0 -P "$GOPLS_JOBS" -I{} bash -c 'check_one_dir "$1"' _ {} < "$list_file" > "$out_file" 2>&1; then
  :
fi

if [[ -s "$out_file" ]]; then
  cat "$out_file"
  echo "lint_gopls_gate: gopls reported warnings" >&2
  exit 1
fi

echo "lint_gopls_gate: OK"
