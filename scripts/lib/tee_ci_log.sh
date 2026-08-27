#!/usr/bin/env bash

# Tee a command transcript under CI_ARTIFACT_DIR (default var/ci/).

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"

if (($# < 2)); then
  echo "usage: tee_ci_log.sh <log-filename> <command> [args...]" >&2
  exit 2
fi

log_name="$1"
shift
log_path="$(ci_artifact_path "$log_name")"
"$@" 2>&1 | tee "$log_path"
