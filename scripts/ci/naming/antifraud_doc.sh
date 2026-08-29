#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

DOC="$ROOT/deploy/vendor/ANTIFRAUD.md"
FILTER_LAYER="$ROOT/internal/ingest/filter_layer.go"
FILTERS_GO="$ROOT/internal/ingest/filters.go"

if [[ ! -f "$DOC" ]]; then
  echo "antifraud_doc_gate: missing $DOC" >&2
  exit 1
fi

if rg -qi 'eliminated.{0,40}SISMEMBER|SISMEMBER.{0,40}eliminated' "$DOC"; then
  if rg -q 'SIsMember' "$FILTER_LAYER"; then
    echo "antifraud_doc_gate: ANTIFRAUD.md claims eliminated SISMEMBER but FraudBlacklistFilter still calls SIsMember" >&2
    exit 1
  fi
fi

if rg -qi 'eliminated.{0,40}HEXISTS|HEXISTS.{0,40}eliminated' "$DOC"; then
  if rg -q 'HExists' "$FILTERS_GO"; then
    echo "antifraud_doc_gate: ANTIFRAUD.md claims eliminated HEXISTS but PlacementBlacklistFilter still calls HExists" >&2
    exit 1
  fi
fi

if rg -q 'Hot path does not read' "$DOC"; then
  echo "antifraud_doc_gate: ANTIFRAUD.md stale silent_reject hot-path claim" >&2
  exit 1
fi

if rg -q 'local atomic.*ingress|ingress.*local atomic' "$DOC"; then
  if ! rg -q 'SetIngressRPDHandledExternally' "$DOC"; then
    echo "antifraud_doc_gate: ANTIFRAUD.md must document EntitlementsFilter + SetIngressRPDHandledExternally when mentioning ingress RPD" >&2
    exit 1
  fi
fi

echo "antifraud_doc_gate: ok"
