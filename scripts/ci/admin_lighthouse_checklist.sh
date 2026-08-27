#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

bash "$SCRIPTS/ci/admin_bundle_gate.sh"

ARTIFACT_DIR="${ADMIN_LIGHTHOUSE_ARTIFACT_DIR:-$CI_ARTIFACT_DIR}"
mkdir -p "$ARTIFACT_DIR"
CHECKLIST="$ARTIFACT_DIR/lighthouse-inp-checklist.txt"

cat > "$CHECKLIST" << 'EOF'
Lighthouse INP checklist (admin UI release)

1. Build production bundle: node web/scripts/build.mjs
2. Start control or preview with fresh dist embedded
3. Log in as admin; open /campaigns and /reports/placements
4. Run Lighthouse (mobile) on /campaigns and /reports/placements
5. Target: INP p95 < 200 ms on staging or local preview

Record in release PR:
- INP p95 value
- URL tested (staging host)
- Lighthouse report attachment or screenshot
EOF

echo "Lighthouse INP checklist written to $CHECKLIST"
echo "Manual: run Lighthouse on /campaigns after login; target INP p95 < 200 ms."
