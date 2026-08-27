#!/usr/bin/env bash

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
