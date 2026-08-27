#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

COMPOSE=(docker compose --profile ingest_only)
DB_PORT="${DB_PORT:-5432}"

log() { printf 'seed-demo-workspaces: %s\n' "$*"; }

if ! "${COMPOSE[@]}" ps --status running db 2> /dev/null | grep -q db; then
  log "starting db"
  CH_ENABLED=0 "${COMPOSE[@]}" up -d db
fi

log "seeding demo workspaces and campaigns"
"${COMPOSE[@]}" exec -T db psql -h localhost -p "$DB_PORT" -U ad_event_processor_user -d ad_event_processor << 'EOF'
INSERT INTO customers (id, name, balance, currency, allowed_overdraft)
VALUES
  ('00000000-0000-0000-0000-000000000001', 'Horizon Media Group', 25000000000, 'USD', 0),
  ('00000000-0000-0000-0000-000000000002', 'Pacific Ads Studio', 18000000000, 'USD', 0),
  ('00000000-0000-0000-0000-000000000003', 'Nordic Performance Co', 12000000000, 'USD', 0),
  ('00000000-0000-0000-0000-000000000004', 'Atlas Buying Desk', 9000000000, 'USD', 0),
  ('00000000-0000-0000-0000-00000000000c', 'Summit Traffiq', 6500000000, 'USD', 0)
ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name,
  balance = GREATEST(customers.balance, EXCLUDED.balance);

INSERT INTO campaigns (id, name, budget_limit, status, customer_id, pacing_mode, daily_budget, timezone, freq_limit, freq_window)
VALUES
  ('00000000-0000-0000-0000-000000000001', 'US Summer Surge', 5000000000, 'ACTIVE', '00000000-0000-0000-0000-000000000001', 'ASAP', 0, 'UTC', 0, 0),
  ('00000000-0000-0000-0000-000000000002', 'EU Retargeting V2', 3200000000, 'ACTIVE', '00000000-0000-0000-0000-000000000001', 'ASAP', 0, 'UTC', 0, 0),
  ('00000000-0000-0000-0000-000000000003', 'LATAM Mobile App', 2800000000, 'ACTIVE', '00000000-0000-0000-0000-000000000002', 'ASAP', 0, 'UTC', 0, 0),
  ('00000000-0000-0000-0000-000000000004', 'Global Video Reach', 4100000000, 'PAUSED', '00000000-0000-0000-0000-000000000002', 'ASAP', 0, 'UTC', 0, 0),
  ('00000000-0000-0000-0000-000000000005', 'APAC Crypto Swap', 1500000000, 'ACTIVE', '00000000-0000-0000-0000-00000000000c', 'ASAP', 0, 'UTC', 0, 0)
ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name,
  status = EXCLUDED.status,
  customer_id = EXCLUDED.customer_id;
EOF

cat << EOF

seed-demo-workspaces: done
  Workspaces: Horizon Media Group, Pacific Ads Studio, Nordic Performance Co, Atlas Buying Desk, Summit Traffiq
  Open Team or Billing and pick a workspace from the list (no UUID typing).

EOF
