# Vendor license reference

Internal. Not shipped in customer packages. Runtime limits: [sku.yaml](./sku.yaml). JWT issue: `go run ./cmd/license-issue --sku <code> ...`.

---

## Billing dimensions

License JWT enforces:

- `max_activations` (host count)
- `max_rps` (peak ingest RPS)
- Feature flags in `sku.yaml` (`openrtb_engine`, `ml_fraud_boost`, `ebpf_xdp_edge`, etc.)

`max_active_campaigns: 0` and `max_events_per_month: 0` in SKU schema mean no license cap on those fields.

---

## SKU table (from sku.yaml)

| SKU | USDT/mo | Hosts | Peak RPS | OpenRTB | ML boost | IVT detector | eBPF XDP |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| `starter` | 129 | 1 | 10k | no | no | no | no |
| `pro` | 329 | 1 | 25k | yes | no | no | no |
| `scale` | 649 | 3 | 75k | yes | yes | yes | no |
| `network` | 1199 | 10 | 150k | yes | yes | yes | no |
| `enterprise` | 2500+ | 99 | custom | yes | yes | yes | yes |
| `pilot` | 0 | 1 | 5k | no | no | no | no |

Full fields: `deploy/vendor/sku.yaml`.

---

## Deploy profiles (operator choice, not SKU-gated)

| Profile | ClickHouse | Typical RAM |
| :--- | :---: | :---: |
| `ingest-only` | no | 6-8 GB |
| `single-vps` / `full` | yes | 16+ GB |

Campaign state and settlement use Postgres. Tracker hot path does not block on ClickHouse.

---

## Pilot workflow

1. Issue SKU `pilot` (10 days, 5k RPS, hard bind optional).
2. Customer applies JWT via Admin Settings -> License or `license-apply`.
3. Paid tier: new JWT for `starter` / `pro` / `scale` with matching `sku.yaml` limits.

Trial registry and repeat-pilot rules: `.cursor/rules/licensing.mdc`.

---

## Related vendor files

| File | Use |
| :--- | :--- |
| [KEYS.md](./KEYS.md) | Ed25519 public keys |
| [INVOICE.md](./INVOICE.md) | USDT invoice template |
| [ANTIFRAUD.md](./ANTIFRAUD.md) | Fraud behavior reference |
| [antifraud_backlog.md](./antifraud_backlog.md) | Open antifraud work items |

---

## Support SLA (invoice template defaults)

| Tier | JWT after USDT confirm |
| :--- | :--- |
| Pro, Scale | 12 h |
| Starter, Network, Enterprise | 24 h |

Onboarding: included with first paid month (see INVOICE.md).
