#!/usr/bin/env bash
# Resolve Go toolchain for scripts (sudo/minimal PATH often lacks `go`).

ad_event_processor_go_bin() {
  if [[ -n "${AD_EVENT_PROCESSOR_GO_BIN:-}" && -x "${AD_EVENT_PROCESSOR_GO_BIN}" ]]; then
    printf '%s' "${AD_EVENT_PROCESSOR_GO_BIN}"
    return 0
  fi
  if [[ -n "${ESPX_GO_BIN:-}" && -x "${ESPX_GO_BIN}" ]]; then
    printf 'ad-event-processor-go: WARN: ESPX_GO_BIN is deprecated; use AD_EVENT_PROCESSOR_GO_BIN\n' >&2
    printf '%s' "${ESPX_GO_BIN}"
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

ad_event_processor_go_run() {
  local go_bin
  if ! go_bin="$(ad_event_processor_go_bin)"; then
    printf 'ad-event-processor-go: ERROR: go not found (set AD_EVENT_PROCESSOR_GO_BIN or install Go)\n' >&2
    return 127
  fi
  "$go_bin" run "$@"
}

ad_event_processor_go_build() {
  local go_bin
  if ! go_bin="$(ad_event_processor_go_bin)"; then
    printf 'ad-event-processor-go: ERROR: go not found (set AD_EVENT_PROCESSOR_GO_BIN or install Go)\n' >&2
    return 127
  fi
  "$go_bin" build "$@"
}

# Deprecated aliases (one release).
espx_go_bin() { ad_event_processor_go_bin; }
espx_go_run() { ad_event_processor_go_run "$@"; }
espx_go_build() { ad_event_processor_go_build "$@"; }
