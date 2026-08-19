# Billing

Vendor-only: USDT license sales, tier ladder, invoices. Customer appliance does not process license payment — it verifies offline JWT.

**See also:** [LICENSE.md](LICENSE.md), [TRIAL.md](TRIAL.md), `deploy/vendor/sku.yaml`.

## Tier ladder (USDT / month)

| Tier | SKU | USDT/mo | Hosts | Peak RPS |
| :--- | :--- | ---: | ---: | ---: |
| **Starter** | `starter` | **$129** | 1 | 10k |
| **Pro** | `pro` | **$329** | 1 | 25k |
| **Scale** | `scale` | **$649** | 3 | 75k |
| **Network** | `network` | **$1,199** | 10 | 150k |
| **Enterprise** | `enterprise` | **$2,500+** | 99 | custom |
| **Pilot** | `pilot` | $0 | 1 | 5k (10 days) |

**Enforcement:** hosts + peak RPS + feature gates (OpenRTB, ML, eBPF). **Unlimited campaigns and event volume** on self-hosted — quote RPS and hosts only. No setup fee; first month = license only.

Issue JWTs: `go run ./cmd/license-issue --sku <code> …`. Runtime limits: `sku.yaml`.

## Positioning

- **Starter** — redirect + `/track`, Cost Sync; OpenRTB blocked. Default pitch: ingest-only stack (~6–8 GB RAM, no ClickHouse required).
- **Pro / Scale** — OpenRTB live; host cap via `max_activations`.
- **Enterprise** — eBPF/XDP edge, custom SLA.

Pilot → paid at day 10: convert or revoke; no auto 30-day extension. Tier upgrade = new JWT only — [START.md § License tier upgrade](START.md#license-tier-upgrade).

**SLA:** USDT confirmed → renewal JWT in **24 h** (Pro/Scale **12 h**). Pilot extension case-by-case +7 days max.

## Manual USDT invoice

Send before first paid month or tier upgrade (Telegram/email). Template fields:

| Field | Value |
| :--- | :--- |
| Amount | `{{AMOUNT_USDT}} USDT` |
| Period | 30 days |
| Deployment ID | `{{DEPLOYMENT_ID}}` |
| Host fingerprint | if hard bind |
| Pay (TRC-20 preferred) | `{{WALLET_TRC20}}` |
| Memo | optional: deployment ID |

After payment: buyer sends txid → vendor runs `license-issue` → buyer applies JWT (Settings → License or `license-apply`).

## Invoice template

```text
ad-event-processor license — {{TIER}} (30 days)
Amount: {{AMOUNT_USDT}} USDT (TRC-20 preferred)
Wallet: {{WALLET_TRC20}}
Deployment ID: {{DEPLOYMENT_ID}}
Host fingerprint: {{HOST_FINGERPRINT}} (if hard bind)
After payment send txid. Vendor delivers JWT within 24h (Pro/Scale 12h).
Docs: docs/LICENSE.md · tier upgrade: docs/START.md#license-tier-upgrade
Limits: deploy/vendor/sku.yaml
```

## Cryptomus (automated)

Merchant of record for USDT TRC-20. Buyer pays via Trust Wallet or any wallet; appliance never handles license checkout.

```text
Buyer → Cryptomus invoice → webhook (vendor VPS) → license-issue (offline key host) → JWT to buyer
```

**Setup:** Cryptomus merchant account, USDT/TRON enabled, webhook at `https://billing.<domain>/cryptomus/webhook` on **vendor server only** (never stores `license_private.key`).

**Invoice fields:** `order_id` = `deployment_id` (or `dep_<uuid>_YYYY-MM`); `amount` = SKU price; `currency=USDT`, `network=TRON`.

**Webhook rules:** verify `sign` (MD5 body + API key); allowlist IP `91.227.144.54`; HTTPS; idempotency on `uuid`. Issue JWT only on `paid` / `paid_over` — not on `confirm_check`, `cancel`, `fail`.

**Not on appliance:** `POST /api/v1/selfserve/payment-intents` is ledger top-up for embedded admins, not offline license renewal.

**Manual fallback:** static TRC-20 address + Tronscan verify + `license-issue` ($0 SaaS).

## Risk controls

- **KYT** on `payer_address` before JWT (optional Crystal/TRM); deny + refund on sanctions/mixer exposure.
- **Trial wallet anchor:** `license-issue --usdt-tx` → blocks repeat pilot on same wallet ([TRIAL.md](TRIAL.md)).
- **Manual approve** on startup: webhook queues; operator runs `license-issue`.
- **Refund after JWT issued:** revocation JWT (`license-issue --revoke`) or wait for `valid_until`.
- **Wrong amount:** new invoice for delta or refund; full JWT only after full payment.

## Issue flow

```bash
export AD_EVENT_PROCESSOR_LICENSE_PRIVATE_KEY_FILE=deploy/vendor/license_private.key
go run ./cmd/license-issue \
  --sku pro --customer "Acme Media" \
  --deployment-id "<uuid>" --fingerprint "<host-fp>" \
  --out /tmp/acme-pro.jwt
```

Send invoice first; after on-chain confirm, deliver JWT + [LICENSE.md](LICENSE.md) renewal steps.
