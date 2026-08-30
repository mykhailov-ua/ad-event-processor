#!/usr/bin/env bash

# Role: CI artifact directory default (var/ci) and ci_artifact_path helper.
# Execution context: Sourced via paths.sh on CI runners and local gate runs.
# Invariants/contracts enforced: Creates CI_ARTIFACT_DIR if missing; respects ROOT when set.
# Verify: echo "${CI_ARTIFACT_DIR:-}" after sourcing paths.sh
if [[ -z "${CI_ARTIFACT_DIR:-}" ]]; then
  if [[ -n "${ROOT:-}" ]]; then
    CI_ARTIFACT_DIR="$ROOT/var/ci"
  else
    CI_ARTIFACT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/var/ci"
  fi
fi

mkdir -p "$CI_ARTIFACT_DIR"
export CI_ARTIFACT_DIR

ci_artifact_path() {
  local name="$1"
  mkdir -p "$CI_ARTIFACT_DIR"
  printf '%s/%s\n' "$CI_ARTIFACT_DIR" "$name"
}
