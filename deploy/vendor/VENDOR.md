# deploy/vendor

Vendor-only files. Do not ship in customer packages.

| File | Purpose |
| :--- | :--- |
| [PUBLIC_OFFER.md](./PUBLIC_OFFER.md) | Public offer agreement (EN + UK) — customer-facing template |
| [OFFER_IMPLEMENTATION_GUIDE.md](./OFFER_IMPLEMENTATION_GUIDE.md) | Internal sales/ops: FAQ scripts, acceptance, license issuance, deploy checklist |
| [MARKETING.md](./MARKETING.md) | Product specification (buyer-facing, neutral) |
| [SALES.md](./SALES.md) | Internal tier positioning and SKU table |
| [ENTERPRISE_DEPLOY.md](./ENTERPRISE_DEPLOY.md) | XDP edge and multi-region deploy, limits, verification |
| [ANTIFRAUD.md](./ANTIFRAUD.md) | Operator fraud reference |
| [sku.yaml](./sku.yaml) | License limits and feature flags |
| [KEYS.md](./KEYS.md) | Ed25519 public keys and HWID notes |
| [INVOICE.md](./INVOICE.md) | USDT invoice template |
| [migration/](./migration/) | Keitaro/Binom import maps |
| [fixtures/](./fixtures/) | Pentest lab fixtures |

Issue license: `go run ./cmd/license-issue --sku <code> ...`
