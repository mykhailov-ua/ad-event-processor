#!/usr/bin/env bash

aed_go_bin() {
  if [[ -n "${AD_EVENT_PROCESSOR_GO_BIN:-}" && -x "${AD_EVENT_PROCESSOR_GO_BIN}" ]]; then
    printf '%s' "${AD_EVENT_PROCESSOR_GO_BIN}"
    return 0
  fi
  local go_bin=""
  go_bin="$(command -v go 2> /dev/null || true)"
  if [[ -z "$go_bin" && -x /usr/local/go/bin/go ]]; then
    go_bin=/usr/local/go/bin/go
  fi
  if [[ -z "$go_bin" && -x "${HOME}/go/bin/go" ]]; then
    go_bin="${HOME}/go/bin/go"
  fi
  if [[ -z "$go_bin" && -x /usr/lib/go/bin/go ]]; then
    go_bin=/usr/lib/go/bin/go
  fi
  if [[ -z "$go_bin" ]]; then
    return 1
  fi
  printf '%s' "$go_bin"
}

aed_go_run() {
  local go_bin
  if ! go_bin="$(aed_go_bin)"; then
    printf 'aed-go: ERROR: go not found (set AD_EVENT_PROCESSOR_GO_BIN or install Go)\n' >&2
    return 127
  fi
  "$go_bin" run "$@"
}

aed_go_build() {
  local go_bin
  if ! go_bin="$(aed_go_bin)"; then
    printf 'aed-go: ERROR: go not found (set AD_EVENT_PROCESSOR_GO_BIN or install Go)\n' >&2
    return 127
  fi
  "$go_bin" build "$@"
}
