# USDT invoice template (vendor — copy/paste)

Replace `{{…}}` placeholders. Send via Telegram or email **before** first paid month or tier upgrade.

---

**Subject:** BidShard {{TIER}} — USDT invoice · {{CUSTOMER_NAME}}

Hi {{CUSTOMER_NAME}},

Invoice for your on-prem BidShard license (**{{TIER}}** / SKU `{{SKU_CODE}}`). Onboarding is included with the first paid month — no separate setup fee.

| Field | Value |
| :--- | :--- |
| **Amount** | **{{AMOUNT_USDT}} USDT** |
| **Billing period** | {{PERIOD_START}} → {{PERIOD_END}} (30 days) |
| **Deployment ID** | `{{DEPLOYMENT_ID}}` |
| **Host fingerprint** | `{{HOST_FINGERPRINT}}` (if hard bind) |

**Pay to (USDT):**

| Network | Address |
| :--- | :--- |
| TRC-20 (preferred) | `{{WALLET_TRC20}}` |
| ERC-20 | `{{WALLET_ERC20}}` |

**Memo / note (optional):** `{{DEPLOYMENT_ID}}`

**After payment:** reply with transaction hash (txid). We issue a signed JWT within:

- **24 hours** — Starter / Network / Enterprise
- **12 hours** — Pro / Scale

**Apply JWT (no reinstall):**

1. Admin UI → **Settings** → License → paste JWT → **Apply**, or  
2. `bash scripts/install/ad-event-processor-install.sh license-apply '<JWT>'`

Docs: [PILOT_LICENSE.md](../../docs/PILOT_LICENSE.md) · [QUICKSTART § upgrade](../../docs/QUICKSTART.md#license-tier-upgrade-no-reinstall)

Tier limits reference: [sku.yaml](./sku.yaml) · internal ladder: [SALES_KIT.md](./SALES_KIT.md)

— BidShard vendor support (Telegram: {{SUPPORT_TELEGRAM}})

---

## Example (filled)

| Field | Example |
| :--- | :--- |
| CUSTOMER_NAME | Acme Media |
| TIER | Pro on-prem |
| SKU_CODE | `pro` |
| AMOUNT_USDT | 329 |
| DEPLOYMENT_ID | `a1b2c3d4-e5f6-7890-abcd-ef1234567890` |
| HOST_FINGERPRINT | `sha256:abc…` |
| PERIOD_START | 2026-09-01 |
| PERIOD_END | 2026-10-01 |
