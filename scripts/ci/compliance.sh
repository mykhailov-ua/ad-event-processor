#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

echo "Checking regulatory compliance..."

echo "No ebpf in management/tracker..."
if go list -f '{{.Imports}}' ./internal/controlplane/... ./cmd/control/... ./cmd/tracker/... 2> /dev/null | grep -q "github.com/cilium/ebpf"; then
  echo "COMPLIANCE FAILURE: github.com/cilium/ebpf imported in management or tracker!"
  exit 1
fi
echo "ebpf import ban: OK"

echo "No DOM/Canvas/WebGL fingerprinting..."
if rg -n 'toDataURL|getImageData|getChannelData|canvas-fingerprint' \
  -g '*.js' -g '*.ts' -g '*.html' \
  --glob '!node_modules/**' \
  --glob '!internal/ingestion/safe_page_hydrator.js' \
  --glob '!web/src/components/safe_page_panel.ts' \
  --glob '!web/src/safe_page_hydrator_entry.ts' \
  . > /dev/null 2>&1; then
  echo "COMPLIANCE FAILURE: Found potential device fingerprinting pattern!"
  rg -n 'toDataURL|getImageData|getChannelData|canvas-fingerprint' \
    -g '*.js' -g '*.ts' -g '*.html' \
    --glob '!node_modules/**' \
    --glob '!internal/ingestion/safe_page_hydrator.js' \
    --glob '!web/src/components/safe_page_panel.ts' \
    --glob '!web/src/safe_page_hydrator_entry.ts' \
    . || true
  exit 1
fi
echo "fingerprint SDK ban: OK"

echo "No outbound attack or hack-back helpers..."
if grep -rnEi "\bsyn_flood\b|\budp_flood\b|\bhack_back\b|\breverse_ddos\b" . --exclude-dir="scripts" --exclude-dir="node_modules" --exclude-dir="docs" --exclude-dir=".cursor" --exclude-dir="bin" 2> /dev/null; then
  echo "COMPLIANCE FAILURE: Found potential hack-back or attack pattern!"
  exit 1
fi
echo "hack-back ban: OK"

echo "No port scanning or active probing..."
if grep -rnEi "\bnmap\b|\bportscan\b|\bport_scan\b|\bactive_probe\b" . --exclude-dir="scripts" --exclude-dir="node_modules" --exclude-dir="docs" --exclude-dir=".cursor" --exclude-dir="bin" --exclude-dir=".cache" --exclude-dir=".venv" 2> /dev/null; then
  echo "COMPLIANCE FAILURE: Found potential port scan or active probe pattern!"
  exit 1
fi
echo "port scan ban: OK"

echo "No outbound connections to visitor/source IPs from management..."
if grep -rnEi "dial.*visitor_ip|http.*Get.*visitor_ip|dial.*blocked_ip" ./internal/controlplane/ ./cmd/control/ 2> /dev/null; then
  echo "COMPLIANCE FAILURE: Found potential outbound dial to visitor/blocked IP from management!"
  exit 1
fi
echo "management outbound ban: OK"

echo "No fingerprint-only XDP_DROP..."
if grep -E 'if.*tcp_hash.*XDP_DROP|fingerprint_block' deploy/edge/xdp/bpf/edge_filter.c 2> /dev/null; then
  echo "COMPLIANCE FAILURE: fingerprint may gate XDP_DROP"
  exit 1
fi
echo "XDP fingerprint gate: OK"

echo "static_slot_only: no SelectAndShard outside jumphash-tagged sources..."
if rg 'SelectAndShard' --glob '*.go' --glob '!*_test.go' --glob '!*jumphash*' . 2> /dev/null; then
  echo "COMPLIANCE FAILURE: HybridBalancer.SelectAndShard present in production paths"
  exit 1
fi
echo "static_slot_only: OK"

echo "Tarpit max delay cap..."
if ! grep -q 'MAX_SEC > 15' deploy/nginx/lua/edge-tarpit.lua; then
  echo "COMPLIANCE FAILURE: edge-tarpit.lua missing 15s hard cap"
  exit 1
fi
echo "tarpit cap: OK"

if command -v luajit > /dev/null 2>&1; then
  bash "$SCRIPTS/test/nginx_lua_tests.sh" compliance
fi

echo "COMPLIANCE CHECK SUCCESSFUL: All defensive perimeter rules are met!"
