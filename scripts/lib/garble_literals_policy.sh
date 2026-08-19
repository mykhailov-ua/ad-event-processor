#!/usr/bin/env bash

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
