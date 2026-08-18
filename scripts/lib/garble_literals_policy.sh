#!/usr/bin/env bash
# V2-D.1: per-binary garble -literals defaults for release_garble.sh.
# Source from release_garble.sh and garble_literals_policy_gate.sh — do not execute directly.

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

# Resolve -literals for one cmd: GARBLE_LITERALS overrides all; else GARBLE_LITERALS_<CMD>; else default.
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
