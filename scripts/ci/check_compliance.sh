#!/usr/bin/env bash
# CMP-FORB-01..04: Regulatory compliance checks
set -euo pipefail

source "$(cd "$(dirname "$0")/../lib" && pwd)/paths.sh"
cd "$ROOT"

echo "Checking regulatory compliance..."

# 1. CMP-FORB-04: No cilium/ebpf in management or tracker
echo "Checking CMP-FORB-04: No ebpf in management/tracker..."
if go list -f '{{.Imports}}' ./internal/management/... ./cmd/management/... ./cmd/tracker/... 2>/dev/null | grep -q "github.com/cilium/ebpf"; then
    echo "COMPLIANCE FAILURE: github.com/cilium/ebpf imported in management or tracker!"
    exit 1
fi
echo "CMP-FORB-04: OK"

# 2. CMP-FORB-01: No DOM / Canvas / WebGL fingerprint JS
echo "Checking CMP-FORB-01: No DOM/Canvas/WebGL fingerprinting..."
# Search for canvas/webgl fingerprinting patterns in js/ts/html files
if grep -rnEi "toDataURL|getImageData|getChannelData|font-family|canvas-fingerprint" . --include="*.js" --include="*.ts" --include="*.html" --exclude-dir="node_modules" 2>/dev/null; then
    echo "COMPLIANCE FAILURE: Found potential device fingerprinting pattern!"
    exit 1
fi
echo "CMP-FORB-01: OK"

# 3. CMP-FORB-02: No hack back (reverse DDoS, flood origin)
echo "Checking CMP-FORB-02: No outbound attack or hack-back helpers..."
# Search for flood or attack patterns or reverse DDoS helpers
if grep -rnEi "\bsyn_flood\b|\budp_flood\b|\bhack_back\b|\breverse_ddos\b" . --exclude-dir="scripts" --exclude-dir="node_modules" --exclude-dir="docs" --exclude-dir=".cursor" 2>/dev/null; then
    echo "COMPLIANCE FAILURE: Found potential hack-back or attack pattern!"
    exit 1
fi
echo "CMP-FORB-02: OK"

# 4. CMP-FORB-03: No port scan / nmap / active probe
echo "Checking CMP-FORB-03: No port scanning or active probing..."
# Search for nmap or portscan dependencies or calls
if grep -rnEi "\bnmap\b|\bportscan\b|\bport_scan\b|\bactive_probe\b" . --exclude-dir="scripts" --exclude-dir="node_modules" --exclude-dir="docs" --exclude-dir=".cursor" 2>/dev/null; then
    echo "COMPLIANCE FAILURE: Found potential port scan or active probe pattern!"
    exit 1
fi
echo "CMP-FORB-03: OK"

# 5. CMP-DEF-04: No outbound-to-source-IP from management (test or lint)
echo "Checking CMP-DEF-04: No outbound connections to visitor/source IPs from management..."
# Search for outbound dials or HTTP requests to visitor/source IPs in management
if grep -rnEi "dial.*visitor_ip|http.*Get.*visitor_ip|dial.*blocked_ip" ./internal/management/ ./cmd/management/ 2>/dev/null; then
    echo "COMPLIANCE FAILURE: Found potential outbound dial to visitor/blocked IP from management!"
    exit 1
fi
echo "CMP-DEF-04: OK"

# 6. M10-C3: fingerprint must not be sole L4 drop cause
echo "Checking M10-C3: no fingerprint-only XDP_DROP..."
if grep -E 'if.*tcp_hash.*XDP_DROP|fingerprint_block' deploy/edge/xdp/bpf/edge_filter.c 2>/dev/null; then
    echo "COMPLIANCE FAILURE: fingerprint may gate XDP_DROP (M10-C3)"
    exit 1
fi
echo "M10-C3: OK"

# 7. M9-07: JumpHash SelectAndShard must not appear in production tracker paths
echo "Checking M9-07: no SelectAndShard outside jumphash-tagged sources..."
if rg 'SelectAndShard' --glob '*.go' --glob '!*_test.go' --glob '!*jumphash*' . 2>/dev/null; then
    echo "COMPLIANCE FAILURE: HybridBalancer.SelectAndShard present in production paths (GAP-HOT-03)"
    exit 1
fi
echo "M9-07: OK"

# 8. GAP-CMP-01: compliance matrix maintained
echo "Checking GAP-CMP-01: COMPLIANCE_MATRIX.md..."
if [[ ! -f docs/COMPLIANCE_MATRIX.md ]]; then
    echo "COMPLIANCE FAILURE: docs/COMPLIANCE_MATRIX.md missing"
    exit 1
fi
echo "GAP-CMP-01 matrix: OK"

# 9. CMP-DEF-03: tarpit hard cap <= 15s
echo "Checking CMP-DEF-03: tarpit max delay cap..."
if ! grep -q 'MAX_SEC > 15' deploy/nginx/lua/edge-tarpit.lua; then
    echo "COMPLIANCE FAILURE: edge-tarpit.lua missing 15s hard cap"
    exit 1
fi
echo "CMP-DEF-03 cap: OK"

# 10. Offline tarpit Lua unit test (when luajit available)
if command -v luajit >/dev/null 2>&1; then
    echo "Running tarpit_test.lua..."
    luajit deploy/nginx/lua/tests/tarpit_test.lua deploy/nginx/lua
    echo "tarpit_test.lua: OK"
fi

echo "COMPLIANCE CHECK SUCCESSFUL: All defensive perimeter rules are met!"
