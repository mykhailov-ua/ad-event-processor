#!/usr/bin/env bash
# Role: Click ingress hot-path static gate (no blocking outbound HTTP on landing_bundle).
# Verify: bash scripts/ci/static/click_ingress_hotpath_gate.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

TARGET="internal/ingest/landing_bundle.go"
if rg -n 'http\.(Get|Post|Head|Do)\(' "$TARGET" 2> /dev/null; then
  echo "click_ingress_hotpath_gate: blocking http client call in $TARGET" >&2
  exit 1
fi
if rg -n '\bcurl\b' "$TARGET" 2> /dev/null; then
  echo "click_ingress_hotpath_gate: curl invocation in $TARGET" >&2
  exit 1
fi

echo "click ingress hot-path gate OK"
