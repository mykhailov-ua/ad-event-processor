# deploy/vendor

Vendor-only files. Do not ship in customer packages.

| File | Purpose |
| :--- | :--- |
| [MARKETING.md](./MARKETING.md) | Buyer-facing features — honest, current state |
| [SALES.md](./SALES.md) | Internal tier positioning and SKU table |
| [ENTERPRISE_DEPLOY.md](./ENTERPRISE_DEPLOY.md) | XDP edge and multi-region deploy, limits, verification |
| [ANTIFRAUD.md](./ANTIFRAUD.md) | Operator fraud reference |
| [sku.yaml](./sku.yaml) | License limits and feature flags |
| [KEYS.md](./KEYS.md) | Ed25519 public keys and HWID notes |
| [INVOICE.md](./INVOICE.md) | USDT invoice template |
| [migration/](./migration/) | Keitaro/Binom import maps |
| [fixtures/](./fixtures/) | Pentest lab fixtures |

Issue license: `go run ./cmd/license-issue --sku <code> ...`
