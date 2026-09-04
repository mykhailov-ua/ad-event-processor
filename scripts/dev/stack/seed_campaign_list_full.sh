#!/usr/bin/env bash
# Role: Full campaign list UX dataset (PG stats + list facets + ClickHouse economics).
# Execution context: ingest-only stack; runs buyer dashboard seed with list UX enrichment.
# Env knobs: SEED_BUYER_COUNT (default 120).
# Verify: bash scripts/dev/stack/seed_campaign_list_full.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

export SEED_BUYER_COUNT="${SEED_BUYER_COUNT:-120}"
export SEED_INGEST_CAMPAIGN_COUNT="${SEED_INGEST_CAMPAIGN_COUNT:-$SEED_BUYER_COUNT}"
export SEED_UI_DEMO_COUNT="${SEED_UI_DEMO_COUNT:-$SEED_BUYER_COUNT}"

bash "$ROOT/scripts/dev/stack/seed_admin.sh" --no-up || true
bash "$ROOT/scripts/dev/stack/seed_buyer_dashboard.sh"

cat << EOF

seed-campaign-list-full: done
  Campaigns: ${SEED_BUYER_COUNT} (ACTIVE/PAUSED/EXHAUSTED/ARCHIVED mix)
  Customers: 3 portfolios for customer filter (seq 1 primary)
  Owners:    8 team users for owner filter
  Countries: target_countries on every campaign
  Margin:    balance_ledger breach rows on seq % 29
  ClickHouse: buyer economics enabled

Open Campaigns in admin UI (admin@test.local / Password123!)

EOF
