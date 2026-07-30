# Runbook: `customers` entity model (arbitrage vs ad network)

Who is `public.customers` in PostgreSQL, and how to bootstrap Layer A for different operator business models.

Related: [OPERATOR_SUBSCRIPTION_TIERS.md](./OPERATOR_SUBSCRIPTION_TIERS.md), [SELF_HOSTED.md](../SELF_HOSTED.md).

---

## What `customer` means in eSPX

`public.customers` is an **advertiser account** in the operator's books:

- Owns campaigns and brands (optional)
- Has `balance_ledger` (prepaid wallet)
- May have `billing.customer_subscriptions` (operator tier)
- Appears in RBAC as tenant boundary for self-serve API keys

It is **not** the vendor's license buyer. The license buyer is the **operator** (Layer O, deployment owner).

---

## Model 1: Solo arbitrage / buy-side (one logical advertiser)

**Profile:** `ingest_only` or minimal control stack. One team, no external advertisers.

```text
Operator staff (auth users)
    └── single synthetic customer "house"
            └── campaigns (many)
            └── balance_ledger (internal accounting only)
```

### Bootstrap

```sql
INSERT INTO customers (id, name, balance, currency)
VALUES ('<fixed-uuid>', 'House account', 0, 'USD');

-- Optional: skip customer_subscriptions if vendor license alone is enough
-- Or assign a permissive internal plan with max_events_per_month: 0
```

### Configuration

| Setting | Value |
| :--- | :--- |
| `RTB_MODE` | `off` (direct `campaign_id` on `/track`) |
| `payment` / `billing` | Usually **disabled** in compose |
| Self-serve API | Not exposed |
| `customers` count | 1 |

### Traffic flow

`/track` → campaign under house customer → Redis budget → stream → PG settlement. No Stripe, no advertiser invoices.

---

## Model 2: Ad network (many advertisers)

**Profile:** `network_operator`. External advertisers top up and run campaigns.

```text
Operator admin
    ├── customer: Advertiser A  → subscription plan "pro"
    ├── customer: Advertiser B  → subscription plan "basic"
    └── customer: House (optional internal tests)
```

### Bootstrap per advertiser

1. `POST /api/v1/customers` (or SQL) — create customer + initial balance.
2. Assign `billing.customer_subscriptions.plan_code`.
3. Issue API key via self-serve or operator UI (`/api/v1/selfserve/api-keys`).
4. Advertiser creates campaigns via self-serve or operator creates on their behalf.

### Configuration

| Setting | Value |
| :--- | :--- |
| `RTB_MODE` | `shadow` or `live` if programmatic lane needed |
| `payment` / `billing` | **Enabled** — Stripe/crypto keys are **operator's** |
| Self-serve | Enabled with RBAC |
| RBAC | Operator users vs advertiser-scoped API keys |

---

## Model 3: Hybrid (network + internal arbitrage lane)

Separate customers:

| Customer | Purpose | RTB |
| :--- | :--- | :--- |
| `house_arb` | Internal media buying | `off` |
| External UUIDs | Paying advertisers | `live` if licensed |

Use campaign-level routing; do not mix budgets across customers without explicit transfers.

---

## Identity mapping

| Concept | Storage |
| :--- | :--- |
| Operator employee | `auth` users, management RBAC |
| Advertiser org | `public.customers` |
| Campaign | `campaigns.customer_id` → FK |
| License buyer (vendor) | **Not in PG** — `license.jwt` / `deployment_id` |

---

## Self-serve vs operator-managed

| Flow | API |
| :--- | :--- |
| Operator creates campaign for client | `/api/v1/campaigns` |
| Advertiser self-service | `/api/v1/selfserve/campaigns` |
| Wallet top-up | `/api/v1/selfserve/payment-intents` → `payment` |

Self-serve routes enforce `customer_id` from session/API key — advertiser cannot see other tenants' campaigns (`ensureCampaignAccess`).

---

## Checklist by profile

### ingest_only (arbitrage)

- [ ] One house `customer`
- [ ] Campaigns created via operator API or CLI
- [ ] Payment/billing services stopped or not configured
- [ ] CH optional if ML not needed (see SELF_HOSTED § deploy profiles)

### network_operator

- [ ] Plan templates in `subscription_plans`
- [ ] Per-advertiser `customer_subscriptions`
- [ ] Payment webhooks on operator domain
- [ ] Bundled SPA or external UI on `/api/v1`

---

## Common mistakes

| Mistake | Fix |
| :--- | :--- |
| Treating `customers` as vendor license tenants | License is deployment-wide JWT |
| One customer for all advertisers in a network | Breaks ledger, RBAC, invoices |
| Enabling Stripe without `payment` service | Top-up intents fail; use compose profile |
| Expecting vendor to meter events | Vendor license is not usage-based (see SELF_HOSTED commercial policy) |
