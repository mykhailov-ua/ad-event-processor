#!/usr/bin/env bash
# Validate // Verify: commands in doc.go (lightweight: go list, go test -list, path/Makefile checks).
# Full test execution is operator-owned; use DOC_GO_VERIFY_RUN=1 to run go test -short per command.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

SCOPE="${DOC_GO_VERIFY_SCOPE:-all}"
REPORT_GAPS="${DOC_GO_VERIFY_REPORT_GAPS:-0}"
RUN_TESTS="${DOC_GO_VERIFY_RUN:-0}"

fail=0
checked_files=0
checked_cmds=0
gap_files=0

usage() {
  cat <<'EOF'
Usage: bash scripts/ci/static/doc_go_verify.sh [options]

Options (env):
  DOC_GO_VERIFY_SCOPE=all|internal|pkg|cmd   default: all
  DOC_GO_VERIFY_REPORT_GAPS=1                list doc.go missing Tradeoffs/Invariants
  DOC_GO_VERIFY_RUN=1                        execute go test -short (slow; not for pr_fast)

Validates each // Verify: line in doc.go:
  go list -e PATH          runs go list
  go test PATH ...         go test -list (or full -short when DOC_GO_VERIFY_RUN=1)
  make TARGET              target exists in Makefile
  bash SCRIPT              script file exists and is executable or readable
  go run ./cmd/...         main package path exists
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

doc_roots=()
case "$SCOPE" in
  all) doc_roots=(internal pkg cmd) ;;
  internal) doc_roots=(internal) ;;
  pkg) doc_roots=(pkg) ;;
  cmd) doc_roots=(cmd) ;;
  *)
    echo "doc_go_verify: unknown DOC_GO_VERIFY_SCOPE=$SCOPE" >&2
    exit 2
    ;;
esac

extract_verify_commands() {
  local file="$1"
  awk '
    /^\/\/ Verify/ { in_v = 1; next }
    in_v && /^package / { exit }
    in_v && /^\/\/[[:space:]]*$/ { next }
    in_v && /^\/\/[[:space:]]*(go |make |bash )/ {
      line = $0
      sub(/^\/\/[[:space:]]*/, "", line)
      print line
      next
    }
    in_v && /^\/\/[[:space:]]+/ {
      line = $0
      sub(/^\/\/[[:space:]]*/, "", line)
      if (line ~ /^(go |make |bash )/) print line
    }
  ' "$file"
}

has_section() {
  local file="$1"
  local section="$2"
  rg -q "^// ${section}:" "$file" 2> /dev/null || rg -q "^// ${section}$" "$file" 2> /dev/null
}

strip_env_prefix() {
  local cmd="$1"
  while [[ "$cmd" =~ ^[A-Za-z_][A-Za-z0-9_]*= ]]; do
    cmd="${cmd#*=}"
    cmd="${cmd#"${cmd%%[![:space:]]*}"}"
  done
  printf '%s' "$cmd"
}

validate_make() {
  local target="$1"
  if rg -q "^${target}:" Makefile 2> /dev/null; then
    return 0
  fi
  echo "doc_go_verify: make target missing in Makefile: $target" >&2
  return 1
}

validate_bash() {
  local script="$1"
  if [[ -f "$script" ]]; then
    return 0
  fi
  echo "doc_go_verify: bash script missing: $script" >&2
  return 1
}

validate_go_run() {
  local cmd="$1"
  local path
  path="$(printf '%s' "$cmd" | awk '{for (i=1;i<=NF;i++) if ($i=="./cmd/" || $i ~ /^\.\/cmd\//) {print $i; exit}}')"
  if [[ -z "$path" ]]; then
    path="$(printf '%s' "$cmd" | awk '{print $3}')"
  fi
  if go list -e "$path" > /dev/null 2>&1; then
    return 0
  fi
  echo "doc_go_verify: go run path not listable: $path (from: $cmd)" >&2
  return 1
}

validate_go_list() {
  local cmd="$1"
  local path
  path="$(printf '%s' "$cmd" | awk '{print $NF}')"
  if go list -e "$path" > /dev/null 2>&1; then
    return 0
  fi
  echo "doc_go_verify: go list failed: $cmd" >&2
  return 1
}

validate_go_test() {
  local cmd="$1"
  local list_cmd run_cmd pkg pattern

  cmd="$(strip_env_prefix "$cmd")"
  pkg="$(printf '%s' "$cmd" | awk '{
    for (i=2;i<=NF;i++) {
      if ($i ~ /^\.\//) { print $i; exit }
    }
  }')"
  if [[ -z "$pkg" ]]; then
    echo "doc_go_verify: go test missing package path: $cmd" >&2
    return 1
  fi

  pattern="$(printf '%s' "$cmd" | sed -n 's/.*-run[[:space:]]\+\([^[:space:]]\+\).*/\1/p')"
  if [[ -n "$pattern" ]]; then
    list_cmd=(go test "$pkg" -list "$pattern")
  else
    list_cmd=(go test "$pkg" -list .)
  fi

  if ! "${list_cmd[@]}" > /dev/null 2>&1; then
    echo "doc_go_verify: go test -list failed: ${list_cmd[*]} (from: $cmd)" >&2
    return 1
  fi

  if [[ "$RUN_TESTS" == "1" ]]; then
    run_cmd=(go test "$pkg" -short -count=1)
    if [[ -n "$pattern" ]]; then
      run_cmd+=(-run "$pattern")
    fi
    if ! "${run_cmd[@]}" > /dev/null 2>&1; then
      echo "doc_go_verify: go test run failed: ${run_cmd[*]}" >&2
      return 1
    fi
  fi
  return 0
}

validate_go_build() {
  local cmd="$1"
  local path
  path="$(printf '%s' "$cmd" | awk '{print $NF}')"
  if go list -e "$path" > /dev/null 2>&1; then
    return 0
  fi
  echo "doc_go_verify: go build package not listable: $path (from: $cmd)" >&2
  return 1
}

validate_command() {
  local cmd="$1"
  local file="$2"

  cmd="$(strip_env_prefix "$cmd")"
  case "$cmd" in
    go\ list*)
      validate_go_list "$cmd" || return 1
      ;;
    go\ test*)
      validate_go_test "$cmd" || return 1
      ;;
    go\ run*)
      validate_go_run "$cmd" || return 1
      ;;
    go\ build*)
      validate_go_build "$cmd" || return 1
      ;;
    make\ *)
      local target
      target="$(printf '%s' "$cmd" | awk '{print $2}')"
      validate_make "$target" || return 1
      ;;
    bash\ *)
      local script
      script="$(printf '%s' "$cmd" | awk '{print $2}')"
      validate_bash "$script" || return 1
      ;;
    *)
      echo "doc_go_verify: unsupported verify command in $file: $cmd" >&2
      return 1
      ;;
  esac
}

while IFS= read -r -d '' file; do
  checked_files=$((checked_files + 1))

  if ! rg -q '^// Package ' "$file" 2> /dev/null; then
    echo "doc_go_verify: missing // Package block: $file" >&2
    fail=1
  fi

  if ! rg -q '^// Verify' "$file" 2> /dev/null; then
    echo "doc_go_verify: missing // Verify section: $file" >&2
    fail=1
    continue
  fi

  if [[ "$REPORT_GAPS" == "1" ]]; then
    local_gap=0
    has_section "$file" "Invariants" || local_gap=1
    has_section "$file" "Tradeoffs" || local_gap=1
    if [[ "$local_gap" -eq 1 ]]; then
      gap_files=$((gap_files + 1))
      echo "doc_go_verify: gap (missing Invariants and/or Tradeoffs): $file"
    fi
  fi

  mapfile -t commands < <(extract_verify_commands "$file")
  if ((${#commands[@]} == 0)); then
    echo "doc_go_verify: empty // Verify: section: $file" >&2
    fail=1
    continue
  fi

  for cmd in "${commands[@]}"; do
    checked_cmds=$((checked_cmds + 1))
    if ! validate_command "$cmd" "$file"; then
      fail=1
    fi
  done
done < <(
  for root in "${doc_roots[@]}"; do
    [[ -d "$root" ]] || continue
    find "$root" -name doc.go -print0
  done
)

echo "doc_go_verify: checked ${checked_files} doc.go files, ${checked_cmds} verify commands (scope=$SCOPE)"
if [[ "$REPORT_GAPS" == "1" ]]; then
  echo "doc_go_verify: ${gap_files} files missing Invariants or Tradeoffs (non-obvious decisions backlog)"
fi

if [[ "$fail" -ne 0 ]]; then
  echo "doc_go_verify: FAILED" >&2
  exit 1
fi

echo "doc_go_verify: OK"
