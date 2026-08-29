# Vendor license reference

Internal. Not shipped to customers. Limits: [sku.yaml](./sku.yaml). Issue JWT: `go run ./cmd/license-issue --sku <code> ...`.

Self-hosted: buyer runs the appliance; license is Ed25519 JWT applied locally (Admin API or `license-apply`). No outbound license ping. No vendor-hosted tenants.

---

## Tier positioning

| Profile | SKU | Why |
| :--- | :--- | :--- |
| Solo affiliate | `starter` | Rules-only fraud; no ClickHouse ML workers required |
| Media buyer, CPA waste | `pro` | IVT detector on buyer ClickHouse |
| Network, OpenRTB + ML | `scale`+ | OpenRTB engine, ML boost, residential/moderator intel |

OpenRTB starts at **Scale**. Most buyers use click URL + S2S `/track` only.

**Pro upsell:** IVT (`ivt_ml_detector`) — works with rule registry, no model training story. ML (`ml_fraud_boost`) stays Scale+.

---

## SKU table

| SKU | USDT/mo | Hosts | Peak RPS | IVT | ML | OpenRTB | XDP |
| :--- | ---: | ---: | ---: | :---: | :---: | :---: | :---: |
| `starter` | 129 | 1 | 10k | no | no | no | no |
| `pro` | 329 | 1 | 25k | yes | no | no | no |
| `scale` | 649 | 3 | 75k | yes | yes | yes | no |
| `network` | 1199 | 10 | 150k | yes | yes | yes | no |
| `enterprise` | 2500+ | 99 | custom | yes | yes | yes | yes |
| `pilot` | 0 | 1 | 5k | no | no | no | no |

`max_active_campaigns: 0` and `max_events_per_month: 0` in schema = no license cap.

---

## Deploy profiles (operator, not SKU-gated)

| Profile | ClickHouse | IVT / ML | RAM hint |
| :--- | :---: | :---: | :---: |
| `ingest-only` | no | no | 6–8 GB |
| `full` / `single-vps` | yes | `analytics-ml` adds workers | 16+ GB |

Pro needs `full` + `analytics-ml` for IVT. Scale+ needs `FRAUD_SCORING_ENABLED` and model under `var/fraudscore/artifacts/` for ML.

---

## Pilot

1. Issue `pilot` (10 days, 5k RPS).
2. Customer applies JWT.
3. Paid: re-issue `starter` / `pro` / `scale` with matching limits.

Trial registry: `.cursor/rules/licensing.mdc`.

---

## Related files

| File | Use |
| :--- | :--- |
| [KEYS.md](./KEYS.md) | Ed25519 public keys |
| [INVOICE.md](./INVOICE.md) | USDT invoice template |
| [MARKETING.md](./MARKETING.md) | Buyer-facing feature list |
| [ANTIFRAUD.md](./ANTIFRAUD.md) | Fraud behavior (operators) |

---

## Support SLA (invoice defaults)

| Tier | JWT after USDT confirm |
| :--- | :--- |
| Pro, Scale | 12 h |
| Starter, Network, Enterprise | 24 h |

Onboarding included with first paid month ([INVOICE.md](./INVOICE.md)).
