# BidShard USDT Invoice Template (Vendor Copy/Paste)

Replace the `{{...}}` placeholders before sending. Send this invoice via Telegram or email **before** the first paid month or when processing a tier upgrade.

---

**Subject:** BidShard {{TIER}} - USDT Invoice - {{CUSTOMER_NAME}}

Hi {{CUSTOMER_NAME}},

Here is the invoice for your self-hosted BidShard license (**{{TIER}}** / SKU `{{SKU_CODE}}`). Installation assistance and onboarding support are fully included with your first paid month—there are no separate setup fees.

### Invoice Details

| Field | Value |
| :--- | :--- |
| **Amount** | **{{AMOUNT_USDT}} USDT** |
| **Billing Period** | {{PERIOD_START}} ──► {{PERIOD_END}} (30 days) |
| **Deployment ID** | `{{DEPLOYMENT_ID}}` |
| **Host Fingerprint** | `{{HOST_FINGERPRINT}}` *(required for hard-bound licenses)* |

### Payment Details (USDT Only)

| Network | Deposit Address |
| :--- | :--- |
| **TRC-20 (Preferred)** | `{{WALLET_TRC20}}` |
| **ERC-20** | `{{WALLET_ERC20}}` |

*Optional Payment Memo:* `{{DEPLOYMENT_ID}}`

---

### After Payment Instructions

Once your transaction is submitted on-chain, please reply to this thread with the **transaction hash (TXID)**. We will issue and deliver your cryptographic license key (JWT) within our standard SLA window:
- **12 Hours:** Pro and Scale Tiers
- **24 Hours:** Starter, Network, and Enterprise Tiers

### How to Apply Your License Key (Zero Downtime)

Applying your new license does **not** require a system reinstall or server reboot. You can load it instantly:

1. **Via Admin UI:** Navigate to **Settings** ──► **License** ──► Paste your JWT key ──► Click **Apply**.
2. **Via Command Line:** Run the following installation helper script:
   ```bash
   bash scripts/install/ad-event-processor-install.sh license-apply '<YOUR_JWT>'
   ```

---

## Example Invoice (Filled Reference)

| Field | Example Value |
| :--- | :--- |
| **CUSTOMER_NAME** | Acme Media Group |
| **TIER** | Pro Self-Hosted |
| **SKU_CODE** | `pro` |
| **AMOUNT_USDT** | 329 |
| **DEPLOYMENT_ID** | `a1b2c3d4-e5f6-7890-abcd-ef1234567890` |
| **HOST_FINGERPRINT** | `sha256:4f3c7d...` |
| **PERIOD_START** | 2026-09-01 |
| **PERIOD_END** | 2026-10-01 |
