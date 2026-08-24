# USDT invoice template (vendor)

Replace `{{...}}` placeholders. Send before first paid month or tier upgrade.

---

**Subject:** ad-event-processor {{TIER}} - USDT invoice - {{CUSTOMER_NAME}}

Hi {{CUSTOMER_NAME}},

Invoice for self-hosted license **{{TIER}}** (SKU `{{SKU_CODE}}`). Installation support included with the first paid month.

### Invoice details

| Field | Value |
| :--- | :--- |
| Amount | **{{AMOUNT_USDT}} USDT** |
| Period | {{PERIOD_START}} to {{PERIOD_END}} (30 days) |
| Deployment ID | `{{DEPLOYMENT_ID}}` |
| Host fingerprint | `{{HOST_FINGERPRINT}}` (required for hard-bound licenses) |

### Payment (USDT)

| Network | Address |
| :--- | :--- |
| TRC-20 | `{{WALLET_TRC20}}` |
| ERC-20 | `{{WALLET_ERC20}}` |

Memo (optional): `{{DEPLOYMENT_ID}}`

### After payment

Reply with transaction hash. JWT delivery SLA:

- Pro, Scale: 12 h
- Starter, Network, Enterprise: 24 h

### Apply license (no restart required)

1. Admin UI: Settings -> License -> paste JWT -> Apply.
2. CLI: `bash scripts/install/ad-event-processor-install.sh license-apply '<JWT>'`

---

## Example

| Field | Example |
| :--- | :--- |
| CUSTOMER_NAME | Acme Media Group |
| TIER | Pro |
| SKU_CODE | `pro` |
| AMOUNT_USDT | 329 |
| DEPLOYMENT_ID | `a1b2c3d4-e5f6-7890-abcd-ef1234567890` |
| PERIOD_START | 2026-09-01 |
| PERIOD_END | 2026-10-01 |
