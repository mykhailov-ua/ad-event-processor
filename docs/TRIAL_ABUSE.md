# Trial abuse protection (no CRM)

Policy for the on-prem **pilot** SKU (`deploy/vendor/sku.yaml`). Target buyers: solo media buyers and arbitrage teams (cold traffic, Telegram, disposable VPS). **No CRM** is planned.

**See also:** [LICENSE.md](LICENSE.md), [QUICKSTART.md](QUICKSTART.md), [deploy/vendor/SALES_KIT.md](../deploy/vendor/SALES_KIT.md).

Implementation tasks (phases, DoD, tests): [DEVELOPMENT.md](DEVELOPMENT.md) section 14.

---

## Can we operate without CRM?

Yes. Abuse control for this motion is **vendor-side issue policy + JWT limits**, not a sales database.

| Need | Replacement |
| :--- | :--- |
| Identify repeat trial seekers | Vendor registry of anchors (Telegram ID, USDT tx, `deployment_id`, HWID after first bundle) |
| Block re-issue | Check registry in `license-issue` before signing pilot JWT |
| Conversion | Existing paid flow: USDT invoice -> renewal JWT with same `deployment_id` ([LICENSE.md](LICENSE.md)) |
| False positive | Manual re-issue with same `deployment_id`; log reason in vendor notes (file or DB) |

Skip: HubSpot, corporate-email gates, 12-month cooldowns, SMS verification.

---

## Two planes (read this first)

BidShard pilot is **offline JWT on the buyer's VPS**. Each install has its own Postgres, including `vendor.*` tables from `internal/ledger/migrations/`.

```text
VENDOR PLANE (your laptop / vendor DB)          CUSTOMER PLANE (buyer VPS)
----------------------------------------          -------------------------
license-issue CLI                                 tracker + control + local PG
vendor trial registry (PROPOSED)                  vendor.licenses (per install)
private signing key                               license.jwt on disk
Telegram / USDT invoice                           TELEMETRY_ENABLED=false (default)
```

**Implication:** tables on the buyer's `vendor.licenses` / `vendor.license_activations` **do not** stop cross-customer abuse. A new VPS is a fresh database. Repeat-trial defense must run **when you sign the JWT**, on the vendor plane.

Closed-contour installs ([QUICKSTART.md](QUICKSTART.md)) do not phone home. Do not rely on customer-runtime denylist or installer telemetry for abuse control.

---

## Threat model

Typical abuser loop:

1. New Hetzner/Contabo VPS.
2. Install stack; receive pilot JWT from vendor.
3. Run traffic until `valid_until` (10 days on pilot SKU).
4. Keep campaign knowledge; delete VM.
5. New identity channel; request another pilot.

**Mitigated today**

| Control | Where |
| :--- | :--- |
| JWT copy to another host blocked | `internal/licensing/bind.go` (`VerifyDeploymentBind`, HWID v2 or legacy fingerprint in JWT) |
| Pilot expiry stops ingest | License watcher + `valid_until` / grace ([LICENSE.md](LICENSE.md)) |
| Same `deployment_id` on conversion | Vendor process in [LICENSE.md](LICENSE.md) renewal checklist |

**Not mitigated today**

| Gap | Why |
| :--- | :--- |
| Second pilot for new VPS | `cmd/license-issue` has no eligibility check without `--telegram-id` / HWID anchors |
| No vendor-wide anchor store without env config | Registry file is opt-in via `BIDSHARD_VENDOR_TRIAL_REGISTRY` |

**Mitigated (Phase 1-2)**

| Control | Where |
| :--- | :--- |
| Repeat pilot on same Telegram/HWID | `internal/trialregistry` + `license-issue` |
| Pilot scoped below Starter | `deploy/vendor/sku.yaml` (`max_rps: 5000`, 10 days, OpenRTB off) |
| Runtime RTB gate for pilot | `SanitizeFeaturesForSKU` in `tier_policy.go` |

---

## Design goal

From [SALES_KIT.md](../deploy/vendor/SALES_KIT.md): pilot is for latency, stability, Cost Sync / CAPI **smoke** -- not a free month of media buying.

Raise abuse cost above paying Starter ($129/mo) without enterprise onboarding friction:

- Pilot entitlements capped in `sku.yaml` (5k RPS, 10 days, OpenRTB off).
- Record anchors on the **vendor** side at issue and re-issue time.
- Optional USDT pre-pay before first JWT (commercial experiment; no code path today).

---

## Vendor-side registry (PROPOSED)

Store on vendor infrastructure (dedicated Postgres, or SQLite/file until volume warrants). **Not** in the buyer appliance migrations.

```sql
-- PROPOSED: vendor tooling DB (not customer install)
CREATE TABLE trial_anchors (
    anchor_type    TEXT NOT NULL
                   CHECK (anchor_type IN ('telegram','deployment_id','hwid','usdt_tx')),
    anchor_value   TEXT NOT NULL,
    deployment_id  UUID NOT NULL,
    issued_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_until    TIMESTAMPTZ NOT NULL,
    status         TEXT NOT NULL DEFAULT 'active'
                   CHECK (status IN ('active','expired','converted','revoked')),
    notes          TEXT,
    PRIMARY KEY (anchor_type, anchor_value)
);

CREATE INDEX idx_trial_anchors_deployment ON trial_anchors (deployment_id);
```

### Anchors and when they are known

| Anchor | Collected | Use |
| :--- | :--- | :--- |
| `telegram` | Before first JWT (buyer messages vendor bot / support) | Primary identity for this audience |
| `deployment_id` | Mint once per buyer journey; reuse on paid renewal | Links pilot -> starter JWT |
| `hwid` | After install (`hwid_v2` from support bundle) | Block second pilot on same metal |
| `usdt_tx` | On first payment (invoice or optional deposit) | One wallet -> one buyer line |

Email alone is weak (disposable). Use as a note only, not a hard key.

### Issue rules (PROPOSED)

Run inside `license-issue` (or a thin wrapper) **before** `licensing.SignJWT`:

```text
REJECT sku=pilot WHEN:
  EXISTS trial_anchors(telegram, :id) WITH status IN (active, expired)
  OR EXISTS trial_anchors(hwid, :hwid) WITH status IN (active, expired)
     AND issued_at > now() - :hwid_cooldown     -- duration: TBD (see SALES_KIT extension policy)
  OR EXISTS trial_anchors(usdt_tx, :wallet) WITH trials on pilot
```

**First pilot JWT** may ship without `hwid_hash` (buyer has not installed yet). `VerifyDeploymentBind` allows empty bind until renewal ([bind.go](../internal/licensing/bind.go)). Record HWID when re-issuing or when buyer sends support bundle **before** any second pilot request.

Do not hard-block on IP /24 alone (colo, VPN, shared hosters).

---

## Pilot SKU (shipped)

Current `pilot` row in [sku.yaml](../deploy/vendor/sku.yaml):

| Field | Value |
| :--- | ---: |
| `valid_days` | 10 |
| `max_rps` | 5000 |
| `max_api_keys` | 3 |
| `max_tenants` | 1 |
| `max_export_chunk_bytes` | 0 |
| `features.rtb_live` | false |
| `features.openrtb_engine` | false |

Runtime: `SanitizeFeaturesForSKU` mirrors Starter OpenRTB off for pilot even if JWT features drift.

Per-feature "smoke only" gates (Cost Sync cap, single CAPI test) are **not implemented**.

---

## Customer-plane runtime (today)

What actually runs on the buyer stack:

```text
license apply / watcher
        |
        v
CheckHostActivation (activation_enforce.go)
        |
        +-- bind.mode hard (pilot) --> VerifyDeploymentBind only; return
        |
        +-- bind.mode multi (pro+) --> recordActivation -> vendor.license_activations
```

Tracker ingest calls `CheckHostActivation` via [registry_license.go](../internal/ingestion/registry_license.go). For pilot, denial is **fingerprint/HWID mismatch**, not a cross-trial registry.

Expiry: JWT `valid_until` + grace. `vendor.license_revoke_queue` is consumed by the control-plane worker (`worker_license_revoke_queue.go`), which sets `vendor.licenses.revoked` and reloads the license watcher. Do not depend on it for pilot cutoff (JWT expiry is the primary gate).

---

## End-to-end flow

### Today (manual, no CRM)

```text
1. Buyer -> vendor Telegram
2. Vendor: new deployment_id (record in vendor notes / spreadsheet / PROPOSED registry)
3. Vendor: go run ./cmd/license-issue --sku pilot --customer "..." --deployment-id <uuid> --out pilot.jwt
4. Buyer: install + license-apply
5. Buyer: support bundle with hwid_v2 -> vendor records HWID in registry
6. Day ~10: convert or stop (SALES_KIT); no auto extension
7. Paid: USDT -> starter JWT, same deployment_id + --hwid-v2 (LICENSE.md)
```

### Target (still no CRM; partial automation shipped)

```text
1. Landing -> vendor-trial-bot captures telegram_user_id into pending queue
2. Vendor: trial-registry list-pending
3. license-issue --approve-pending <id> (registry eligibility check + pilot JWT)
4. [Optional] USDT payment recorded -> usdt_tx anchor (see below)
5. After bundle: store hwid anchor (license-issue --record-hwid)
6. license_banner nudge before expiry (shipped Phase 4)
```

`POST /api/v1/telegram/validate` today validates WebApp `initData` on the **appliance** ([tg_handlers.go](../internal/controlplane/tg_handlers.go)). It is not a vendor signup OAuth flow. `cmd/vendor-trial-bot` is a separate vendor-plane long-poll bot; it never holds the Ed25519 signing key.

**Bot token storage:** set `BIDSHARD_VENDOR_TRIAL_BOT_TOKEN` in the vendor shell or a local `.env` on the operator workstation. Do not commit tokens; do not reuse the appliance `TELEGRAM_BOT_TOKEN` unless you accept shared blast radius.

### USDT deposit anchor (design only; not wired)

Commercial experiment: require a small USDT deposit before the first pilot JWT to raise abuse cost. This is **vendor-plane bookkeeping** only until an explicit bridge exists.

```text
1. Buyer sends USDT to vendor invoice address (manual or future watcher)
2. Vendor records tx id in registry via license-issue --usdt-tx on issue
3. Repeat pilot on same wallet is rejected (ErrTrialWalletUsed)
```

**Not wired today:** `POST /api/v1/selfserve/payment-intents` and `billing.customer_subscriptions` live on the buyer appliance and require an existing `customer_id`. On-prem pilot buyers typically have no self-serve customer row. Do not couple trial registry to appliance billing schema without a reviewed bridge doc.

When a bridge is implemented, update `usdt_tx` collection to either (a) vendor CLI only, or (b) a vendor-side deposit watcher that calls `EnqueuePending` + operator approve -- never direct JWT signing from payment webhooks without human or registry gate.

---

## Manual override

When HWID changes after disk replace (documented in [LICENSE.md](LICENSE.md)):

- Re-issue pilot or starter JWT with **same** `deployment_id` and new `--hwid-v2`.
- Append a line to vendor audit log (file column `notes` or `trial_override_audit` PROPOSED table).
- Do not delete anchor rows globally.

`--force` / `VENDOR_ADMIN_TOKEN` flags do **not** exist in `cmd/license-issue` today.

---

## Explicitly out of scope

| Item | Reason |
| :--- | :--- |
| CRM (HubSpot, Pipedrive) | No team; wrong motion for $129 buyers |
| Customer-plane trial denylist | Fresh PG per install; useless for repeat abusers |
| `license_revoke_queue` worker | Shipped in control-plane; customer-plane activation abuse only |
| IP-only block | High false positives for VPS buyers |
| Campaign-count caps | SALES_KIT: unlimited on self-hosted |
| Invented Prometheus metrics | Add with implementation, not in policy doc |

---

## Metrics (vendor plane only)

| Metric | Source |
| :--- | :--- |
| Pilots issued / week | Count rows in vendor `trial_anchors` or `license-issue` audit log |
| Pilot -> paid | `status = converted` / (`converted` + `expired`) per `deployment_id` |
| Repeat blocks | `license-issue` rejections logged with reason |
| Time to paid | `converted.issued_at - first pilot issued_at` |

Customer install metrics require opt-in telemetry (`TELEMETRY_ENABLED`); default closed contour has none.

---

## Summary

CRM is not required. Effective controls for this product shape:

1. **Vendor-side anchor registry** at JWT issue time (Telegram, `deployment_id`, HWID after bundle, USDT tx).
2. **Tighter pilot SKU** in `sku.yaml` so trial value stays below Starter.
3. **Manual vendor discipline** today; automate via `license-issue` checks (PROPOSED).
4. **Same `deployment_id` JWT swap** on conversion (already documented).

Customer-plane `vendor.*` tables and `recordActivation` are not the cross-trial enforcement layer for `bind.mode: hard` pilot.
