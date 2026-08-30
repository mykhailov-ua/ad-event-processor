#!/usr/bin/env bash

# Role: Library: Garble literals policy lib.
# Execution context: Sourced by CI, fault, and dev scripts; not a standalone gate.
# Invariants/contracts enforced: Helpers must not exit 0 on error paths when used as gate prerequisites.
# Verify: source scripts/lib/garble_literals_policy.sh
garble_literals_cmd_env_key() {
  local cmd="$1"
  printf 'GARBLE_LITERALS_%s' "$(echo "$cmd" | tr '[:lower:]-' '[:upper:]_')"
}

garble_literals_default_for_cmd() {
  local cmd="$1"
  case "$cmd" in
    control | processor) echo 1 ;;
    *) echo 0 ;;
  esac
}

garble_literals_for_cmd() {
  local cmd="$1"
  local key val
  if [[ -v GARBLE_LITERALS ]]; then
    echo "$GARBLE_LITERALS"
    return
  fi
  key="$(garble_literals_cmd_env_key "$cmd")"
  if [[ -v "$key" ]]; then
    val="${!key}"
    echo "$val"
    return
  fi
  garble_literals_default_for_cmd "$cmd"
}
