#!/usr/bin/env bash
set -euo pipefail

# Role: Lint gate: shellcheck on scripts.
# Execution context: CI merge-lint via lint.sh.
# Invariants/contracts enforced: Child linter failure propagates to exit 1.
# Verify: bash scripts/ci/lint/shell.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

SHELLCHECK_IMAGE="${SHELLCHECK_IMAGE:-koalaman/shellcheck:stable}"
LINT_SHELL_SEVERITY="${LINT_SHELL_SEVERITY:-error}"
if [[ "${LINT_SHELL_WARN:-}" == "1" ]]; then
  LINT_SHELL_SEVERITY=warning
fi

shellcheck_targets() {
  find scripts deploy \
    -type f -name '*.sh' \
    ! -path '*/node_modules/*' \
    ! -path '*/vendor/*' \
    -print0
}

run_shellcheck() {
  local -a files=()
  while IFS= read -r -d '' file; do
    files+=("$file")
  done < <(shellcheck_targets)

  if ((${#files[@]} == 0)); then
    echo "lint_shell_gate: no shell entrypoints found" >&2
    exit 1
  fi

  echo "lint_shell_gate: shellcheck (-x -S ${LINT_SHELL_SEVERITY}, ${#files[@]} files)..."
  local batch=()
  local file
  local status=0
  for file in "${files[@]}"; do
    batch+=("$file")
    if ((${#batch[@]} >= 40)); then
      if ! shellcheck_batch "${batch[@]}"; then
        status=1
      fi
      batch=()
    fi
  done
  if ((${#batch[@]} > 0)); then
    if ! shellcheck_batch "${batch[@]}"; then
      status=1
    fi
  fi
  return "$status"
}

shellcheck_batch() {
  if command -v shellcheck > /dev/null 2>&1; then
    shellcheck -x -S "$LINT_SHELL_SEVERITY" "$@"
    return $?
  fi
  if command -v docker > /dev/null 2>&1; then
    docker run --rm -v "$ROOT:/work" -w /work "$SHELLCHECK_IMAGE" \
      shellcheck -x -S "$LINT_SHELL_SEVERITY" "$@"
    return $?
  fi
  echo "lint_shell_gate: need shellcheck or docker" >&2
  return 1
}

run_shellcheck
echo "lint_shell_gate: OK"
