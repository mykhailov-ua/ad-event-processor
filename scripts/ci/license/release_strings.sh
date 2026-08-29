#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/release_strings_patterns.sh"

if [[ $# -lt 1 ]]; then
  echo "usage: release_strings_gate.sh <binary> [binary...]" >&2
  exit 1
fi

for bin in "$@"; do
  release_strings_scan_binary "$bin"
  echo "release_strings_gate: OK ($bin)"
done
