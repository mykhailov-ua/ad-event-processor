# Runbook: operator subscription tiers (Layer O → A)

How the **license holder** configures internal plans for **advertisers** (`public.customers`) on a self-hosted install. This is **not** the vendor license (Layer V).

Related: [SELF_HOSTED.md](../SELF_HOSTED.md), [CUSTOMER_ENTITY_MODEL.md](./CUSTOMER_ENTITY_MODEL.md).

---

## Scope

| Layer | Who configures | Tables |
| :--- | :--- | :--- |
| **V — Vendor** | eSPX (you) | `license.jwt`, `billing.license_status` |
| **O — Operator** | Install owner | `billing.subscription_plans`, `billing.customer_subscriptions` |
| **A — Advertiser** | End user of the network | `public.customers`, campaigns, ledger |

Advertisers **cannot** change `subscription_plans` unless they have operator admin RBAC.

---

## Data model

```text
billing.subscription_plans     — template (limits_json, features_json, base_fee_micro)
billing.customer_subscriptions — one row per advertiser (plan_code, overrides_json)
licensing.Effective(dep, adv)  — hot path: min(vendor license, advertiser plan)
```

Seeded plans (`basic`, `pro`, `enterprise`) in migrations are **examples**. Replace or extend for your network.

---

## Option A: SQL bootstrap (small install)

```sql
-- 1. New plan template
INSERT INTO billing.subscription_plans (code, display_name, limits_json, features_json, base_fee_micro)
VALUES (
  'arbitrage_std',
  'Standard advertiser',
  '{"max_active_campaigns": 20, "max_rps": 5000, "max_requests_per_day": 1000000, "max_events_per_month": 0, "max_api_keys": 2}'::jsonb,
  '{"rtb_live": false}'::jsonb,
  0
);

-- 2. Assign to advertiser
INSERT INTO billing.customer_subscriptions (customer_id, plan_code, status, period_start)
VALUES ('<advertiser-uuid>', 'arbitrage_std', 'active', CURRENT_DATE)
ON CONFLICT (customer_id) DO UPDATE SET plan_code = EXCLUDED.plan_code, updated_at = NOW();
```

`max_events_per_month: 0` in operator JSON means **no operator-side monthly event cap** for that tier (use `0` as unlimited in merge helpers where documented).

---

## Option B: declarative YAML (target — GAP-PROD-03)

Planned operator file, e.g. `deploy/operator/plans.yaml`:

```yaml
# Operator-owned; applied by management on startup or via POST /api/v1/ops/plans/reload
plans:
  - code: network_pro
    display_name: Pro advertiser
    base_fee_micro: 500000000
    limits:
      max_active_campaigns: 200
      max_rps: 50000
      max_requests_per_day: 10000000
    features:
      rtb_live: true
assignments:
  - customer_id: "uuid-here"
    plan_code: network_pro
    overrides:
      limits:
        max_rps: 100000
```

Until GAP-PROD-03 ships, use SQL or management JSON API (`adminapi` licensing handlers when wired).

---

## Option C: bundled SPA (GAP-PROD-02)

Operator admin UI (`//go:embed`) will expose plan CRUD and advertiser assignment. API backing: `/api/v1/...` (plans + subscriptions).

---

## Entitlements after change

1. Update `subscription_plans` / `customer_subscriptions`.
2. Registry reload: `campaigns:update` fan-out or wait for `SyncEntitlements` interval on tracker.
3. Verify: `GET /api/v1/license/status` (deployment) vs `GET /api/v1/customers/{id}/quota-status` (advertiser).

---

## Billing linkage (operator → advertiser)

| Mechanism | When |
| :--- | :--- |
| `base_fee_micro` | Monthly invoice line via `cmd/billing` `GenerateInvoice` |
| `usage_meters` | Overage on accepted events (PG rollup, not CH) |
| `balance_ledger` | Prepaid spend; self-serve top-up via `payment` |

Disable billing stack in compose profile `ingest_only` — tiers still apply to RPS/RPD caps without invoices.

---

## Checklist

- [ ] Vendor license active (`license_status.state = ACTIVE`)
- [ ] Plan codes referenced by subscriptions exist
- [ ] `limits.max_rps` ≤ vendor license `max_rps` (Effective will min)
- [ ] RTB features in plan require `rtb_live` in **vendor** license
- [ ] Tracker registry picked up new entitlements

---

## Troubleshooting

| Symptom | Check |
| :--- | :--- |
| HTTP 403 `license_expired` | Layer V — vendor JWT, not operator plan |
| HTTP 429 ingress | Layer O/A — RPS/RPD; adjust plan or subscription override |
| Invoice missing overage | `usage_meters` worker (PG path); closed month |
| RTB ignored | `RTB_MODE=live` **and** vendor + advertiser `rtb_live` |
