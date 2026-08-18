#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
# shellcheck source=../lib/go.sh
source "$ROOT/scripts/lib/go.sh"

echo "edge_pin_dir_smoke: static alignment"
grep -q 'BPF_PIN_DIR' "$ROOT/deploy/edge/xdp/entrypoint.sh"
grep -q '/sys/fs/bpf/ad-event-processor' "$ROOT/deploy/edge/xdp/entrypoint.sh"
if grep -q '/sys/fs/bpf/espx' "$ROOT/deploy/edge/xdp/entrypoint.sh"; then
  echo "edge_pin_dir_smoke: stale espx pin path in entrypoint.sh" >&2
  exit 1
fi

GO_BIN="$(ad_event_processor_go_bin)"
echo "edge_pin_dir_smoke: unit tests"
"$GO_BIN" test ./internal/edge -run 'TestPinned|TestEntrypoint' -count=1
"$GO_BIN" test ./internal/edge -run TestPinnedBlocklistOpensAfterPin -count=1

echo "edge_pin_dir_smoke: ok"
