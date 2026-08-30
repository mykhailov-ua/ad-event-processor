# shellcheck shell=bash

# Role: Universal format entry for operators (Go, shell, Lua, JS/TS/configs, BPF C).
# Execution context: Sourced after paths.sh; aed_fmt runs scripts/ci/format.sh from repo ROOT.
# Verify: aed_fmt   or   bash scripts/dev/fmt.sh

aed_fmt() {
  local check=0
  case "${1:-}" in
    --check | -n | check)
      check=1
      shift
      ;;
  esac
  if [[ $# -gt 0 ]]; then
    printf 'aed_fmt: ERROR: unknown args: %s\n' "$*" >&2
    printf 'aed_fmt: usage: aed_fmt [--check|-n]\n' >&2
    return 2
  fi
  if [[ -z "${ROOT:-}" || -z "${SCRIPTS:-}" ]]; then
    printf 'aed_fmt: ERROR: source scripts/lib/paths.sh first\n' >&2
    return 2
  fi
  if [[ "$check" -eq 1 ]]; then
    bash "$SCRIPTS/ci/format.sh" --check
  else
    bash "$SCRIPTS/ci/format.sh"
  fi
}
