# Self-hosted deployment model

eSPX ships as **binaries the customer runs on their own bare metal** (or VM). The vendor (eSPX) sells a **deployment license**; the customer operates an **ad network** for their advertisers. This document is the canonical policy for layers, services, billing, licensing, and UI.

Related: [ARCHITECTURE.md](./ARCHITECTURE.md), [DEVELOPMENT.md](./DEVELOPMENT.md), [MULTI_REGION.md](./MULTI_REGION.md).

---

## Three layers

```text
┌──────────────────────────────────────────────────────────────┐
│ Layer V — Vendor (eSPX)                                      │
│   license.jwt → billing.license_status                       │
│   Feature flags, install-wide caps (RPS, events/month, MR)   │
│   No Stripe/crypto keys, no vendor invoices inside the stack │
└────────────────────────────┬─────────────────────────────────┘
                             │ module gates + install caps
┌────────────────────────────▼─────────────────────────────────┐
│ Layer O — Operator (license holder, bare-metal owner)        │
│   management — control plane, campaigns, outbox, settlement    │
│   auth — users, RBAC, API keys                                 │
│   payment — Stripe + crypto → advertiser wallet TOPUP          │
│   billing — invoices and internal plans for advertisers        │
│   notifier — operator alerts (Telegram, SMTP, webhooks)        │
└────────────────────────────┬─────────────────────────────────┘
                             │
┌────────────────────────────▼─────────────────────────────────┐
│ Layer A — Advertisers (rows in public.customers)             │
│   Campaigns, balance_ledger spend, self-serve API (optional) │
└──────────────────────────────────────────────────────────────┘
```

**Do not mix layers.** Vendor revenue (license renewal) is **outside** the product (contract, license file). In-product money flows are **operator ↔ advertiser** only.

---

## Licensing (must-have)

| Mechanism | Role |
| :--- | :--- |
| `ESPX_LICENSE_MODE=file` | **Primary** for air-gapped / bare metal (`license.jwt` on disk) |
| `ESPX_LICENSE_MODE=online` | Optional heartbeat to vendor license server |
| `billing.license_status` | Cold-path mirror of JWT claims for workers and registry |
| `licensing.DeploymentSnapshot` | Module gates (`rtb_live`, `ebpf_xdp_edge`, `ivt_ml_detector`, `multi_region`, …) |

**Install-wide enforcement** (Layer V): expired/revoked license → hot path rejects with `license_expired`; optional modules exit at startup.

**Not vendor licensing:** `billing.subscription_plans` and `billing.customer_subscriptions` belong to **Layer O** — internal tiers the operator assigns to advertisers, not SaaS plans sold by eSPX.

### Entitlements merge

```text
effective_limit   = min(deployment_license.limits[X], advertiser_subscription.limits[X])
effective_feature = deployment_license.features[X] AND advertiser_subscription.features[X]
```

- `deployment_license` — from `license_status` / JWT (vendor).
- `advertiser_subscription` — from `customer_subscriptions` (operator).

Hot path reads the merged snapshot via `Registry.SyncEntitlements`. Vendor caps always bound the install; operator caps bound individual advertisers inside that install.

### Deprecated for self-hosted

| Item | Policy |
| :--- | :--- |
| Weighted PU / dedup-reject metering | Not billed; dedup remains a **delivery** control only |
| CH → `usage_meters` (`VolumeMeterWorker`) | Replace with **PG** rollup (`campaign_stats` / `events`) for operator metering |
| Vendor overage invoice | Out of scope; operator invoices advertisers via `cmd/billing` |
| PU pricing tables in `licensing/volume.go` | Sales reference only; not runtime vendor billing |

---

## Administrative microservices (re-profiled, not removed)

Services stay separate **binaries** for blast radius and the legacy `split_control` compose profile. On a single VPS, run **`cmd/control`** — one process that embeds management, auth, payment, billing, and notifier with env-gated components (`CONTROL_ENABLE_*`).

Cold-path workers (margin-guard, cost-sync, volume meter, recon, ledger invariant) run **only** in control/management — never in tracker replicas.

### `cmd/control` — modular monolith (recommended for single VPS)

- Compose profiles: `single_vps`, `ingest_only`, `network_operator`.
- `scripts/dev/stack.sh single-vps` — tracker + processor + control.
- `ingest_only` disables payment/billing/notifier and margin-guard/cost-sync via `CONTROL_ENABLE_*=0`.
- Health: `http://127.0.0.1:8188/health` (management HTTP inside control).

### `cmd/management` — operator control plane (split deploy)

- Campaign CRUD, outbox, settlement gRPC, recon, workers.
- `/api/v1/*` JSON API (+ `internal/adminapi` when wired).
- `/api/v1/selfserve/*` — **advertiser API** (optional); not vendor onboarding.
- ClickHouse: **analytics only** (charts, forecast, IVT); never billing authority.

### `cmd/auth`

Sessions, PASETO, API keys for operator staff and advertiser automation. Unchanged role.

### `cmd/payment` — operator wallet rail

| Concern | Self-hosted policy |
| :--- | :--- |
| Stripe / crypto keys | **Operator's** env on their host |
| Webhooks | Operator domain (`PAYMENT_WEBHOOK_HOST`) |
| Settlement | Outbox → management → `balance_ledger` TOPUP (unchanged) |
| HTMX checkout fragments | **Legacy / dev-only** — see [UI](#ui-no-server-side-htmx) |

Keep: gRPC intents, crypto hold worker, financial recon, chargeback paths.

### `cmd/billing` — operator invoicing

- `GenerateInvoice` — operator bills **advertisers** from `balance_ledger` + optional internal plan fee/overage.
- `usage_meters` — fed from **PG** (accepted events), closed periods, idempotent hourly keys.
- Invoice worker, notifier delivery, ledger drift alerter — remain operator-facing.

### `cmd/notifier`

Operator ops alerts. Credentials belong to the install owner.

---

## Billing and settlement authority

| Concern | Source of truth |
| :--- | :--- |
| Ad spend (CPM/CPC) | PG `balance_ledger` + `campaigns.current_spend` |
| Hot budget stop | Redis Lua + local quanta (hybrid); **not** post-factum delivery |
| Advertiser event quotas | PG `usage_meters` (operator metering), not ClickHouse |
| Analytics / charts | ClickHouse MVs (`stale` flag in API) |
| Vendor license caps | JWT / `license_status` (`max_events_per_month`, `max_rps`, …) |

**Post-factum** applies to **closed settlement** (monthly invoice, closed-hour metering), **not** to stopping delivery when budget is exhausted.

---

## `billing.*` schema ownership

| Table | Owner | Purpose |
| :--- | :--- | :--- |
| `license_status` | Vendor (Layer V) | JWT mirror, module gates |
| `subscription_plans` | Operator (Layer O) | Advertiser tier templates |
| `customer_subscriptions` | Operator | Per-advertiser plan |
| `usage_meters` / `usage_daily` | Operator | Quota / overage for advertisers |
| `invoices` | Operator → advertiser | Not vendor → customer |

Default migration seeds for `subscription_plans` are **examples** for the operator's network, not eSPX SaaS pricing.

---

## Deploy profiles

Typical bare-metal layout (see also [RESTRUCTURE_PLAN §16](../.cursor/RESTRUCTURE_PLAN.md#16-модульный-монолит-для-self-hosted-и-multi-region)):

| Unit | Binaries | Required |
| :--- | :--- | :--- |
| Hot | `tracker` (+ edge nginx/xdp) | Yes |
| Settlement | `processor` | Yes |
| Control | `management`, `auth` | Yes |
| Treasury | `payment`, `billing` | Optional (enable when operator wallet/invoicing used) |
| Alerts | `notifier` | Optional |
| Analytics / ML | `ivt-detector`, `fraud-scorer` | License-gated |

One fat image with compose **profiles** is supported; disabling `payment`/`billing` does not remove binaries from the repo.

---

## UI: no server-side HTMX

Self-hosted installs do **not** use server-rendered HTMX admin. The supported operator surface is:

1. **JSON** `/api/v1/*` (and OpenAPI in `docs/openapi/openapi.yaml`).
2. **External SPA** (operator-built or packaged separately) against that API.
3. Optional `//go:embed` static in `management` for a bundled SPA (GAP-PROD-02) — not SSR fragments.

### Removed in GAP-HYG-04

| Before | After |
| :--- | :--- |
| `/admin/*` HTMX routes | `410 Gone` JSON error; use `/api/v1` |
| `handler_billing.go` HTML | Deleted; billing via `/api/v1/billing/*` and gRPC |
| `internal/payment/http_htmx.go`, `htmx_*.go` | Deleted; `/ui/payment/*` returns `410 Gone` |
| `pkg/httpresponse/htmx_error.go` | Deleted; errors use `pkg/httpresponse` JSON envelope |

`GET /` returns JSON `404` with a link to this section until the bundled SPA ships.

Payment and billing **gRPC/JSON APIs** remain; only HTML fragment UIs were removed.

---

## Multi-region

Licensed enterprise option (`features.multi_region`), not the default single-cell bare-metal story. Global PostgreSQL for finance in MR is documented in [MULTI_REGION.md](./MULTI_REGION.md).

---

## Checklist for operators

1. Install `license.jwt`; set `ESPX_LICENSE_MODE=file` (or online if allowed).
2. Configure Stripe/crypto **operator** keys if wallet rail is enabled.
3. Do not expect vendor billing inside `usage_meters` or CH rollups.
4. Use JSON API or your own SPA; `/admin/*` returns `410 Gone`.
5. Treat `subscription_plans` as **your** advertiser tiers, not eSPX SaaS SKUs.
6. Harden data on your hardware: [runbooks/DATA_SECURITY.md](./runbooks/DATA_SECURITY.md) (LUKS, TLS, secrets, retention).
7. Review protection model (vendor IP, your data, egress trust): [PROTECTION.md](./PROTECTION.md).

---

## Product policy (decisions)

Canonical product choices for self-hosted sales and engineering backlog.

### UI

| Decision | Detail |
| :--- | :--- |
| **Bundled SPA** | Single operator UI via `//go:embed` in `cmd/management` (GAP-PROD-02) |
| **No HTMX SSR** | Legacy HTML removal: GAP-HYG-04 |

### Vendor license (Layer V)

| Decision | Detail |
| :--- | :--- |
| **Pricing** | Vendor defines SKUs in **vendor-only** YAML or vendor admin UI; output is **Ed25519-signed JWT** |
| **Client cannot edit** | No UI on install to change features/price; only `license.jwt` from vendor server |
| **No usage reporting to vendor** | Monthly license = time + feature set; **no mandatory** event telemetering. **Optional** opt-in aggregates: [TELEMETRY_AND_TRUST.md](./TELEMETRY_AND_TRUST.md) |
| **License period** | **Monthly** renewal via JWT `valid_until`; pre-renewal SPA warning; offline grace **Y** days (GAP-PROD-04) |
| **No vendor event caps** | `max_events_per_month` in JWT deprioritized for enforcement; unlimited traffic on client hardware within licensed **features** |
| **Heartbeat** | Online refresh every **X** hours (`ESPX_LICENSE_REFRESH_INTERVAL`); if license server unreachable, cached JWT + **Y** days offline grace with operator warning (GAP-PROD-04 — extend beyond JWT `grace_days` expiry semantics) |
| **Revocation** | Server can push revoked JWT / short-lived tokens; `vendor.licenses.revoked` on license server DB |

See [LICENSE_COMMERCE.md](./LICENSE_COMMERCE.md) (vendor-side SKU constructor). Trust and opt-in telemetry: [TELEMETRY_AND_TRUST.md](./TELEMETRY_AND_TRUST.md).

### Operator commercial (Layer O)

| Decision | Detail |
| :--- | :--- |
| **Subscription constructor** | Operator configures advertiser tiers (YAML or SPA); separate from vendor SKU |
| **Runbooks** | [OPERATOR_SUBSCRIPTION_TIERS.md](./runbooks/OPERATOR_SUBSCRIPTION_TIERS.md), [CUSTOMER_ENTITY_MODEL.md](./runbooks/CUSTOMER_ENTITY_MODEL.md) |

### Deploy profiles

Compose profiles gate which containers start. Use `scripts/dev/stack.sh` or `docker compose --profile <name>`.

| Profile | Command | Binaries / services | Audience |
| :--- | :--- | :--- | :--- |
| `single_vps` | `stack.sh single-vps` | `tracker`, `processor`, `control` (all cold path in one process) | Default bare-metal / 1–2 CPU |
| `ingest_only` | `stack.sh ingest-only` | Same; `CONTROL_ENABLE_PAYMENT/BILLING/NOTIFIER/MARGIN_GUARD/COST_SYNC=0` | Arbitrage / buy-side |
| `network_operator` | `stack.sh network-operator` | `control` with payment + billing + notifier enabled | Ad network with wallet |
| `analytics_ml` | `stack.sh analytics-ml` | + `fraud-scorer`, `ivt-detector` (ClickHouse required) | Optional ML cold path |
| `split_control` | `stack.sh full` | Separate `auth`, `management`, `payment`, `billing`, `notifier` containers | Legacy / multi-container dev |

ClickHouse stays in the default infra profile (processor analytics); optional for ingest-only when ML modules are off (GAP-PROD-05).

CI validates profile wiring: `scripts/ci/compose_profile_check.sh`.

---

## Commercial FAQ

**Does eSPX phone home with traffic volumes by default?** No. License heartbeat carries deployment identity and JWT refresh only. Optional hourly pulse requires explicit opt-in ([TELEMETRY_AND_TRUST.md](./TELEMETRY_AND_TRUST.md)).

**Can the client run unlimited RPS after buying a license?** They run on their hardware without vendor metering; license gates **features** (RTB, ebpf, MR) and deployment fingerprint.

**Who invoices whom?** Vendor invoices operator **outside** the stack (monthly). Operator invoices advertisers **inside** via `billing` + `payment` when enabled.

**Will the vendor steal связки (traffic sources)?** Heartbeat and telemetry schemas exclude campaign IDs, domains, and URLs. Operators can allowlist the license host and disable telemetry. Threat intel opt-in sends only install-wide reject aggregates and signal hashes.

**Public GitHub vs commercial binary?** See [TELEMETRY_AND_TRUST.md § GitHub](./TELEMETRY_AND_TRUST.md#github-public-repo-vs-future-commercial-product) — portfolio today; Pro binary split when product sales start.

