# Vendor license reference

Internal. Not shipped in customer packages. Runtime limits: [sku.yaml](./sku.yaml). JWT issue: `go run ./cmd/license-issue --sku <code> ...`.

Self-hosted only for on-prem SKUs: buyer runs the appliance; license is an Ed25519 JWT applied locally (Admin Settings -> License or `license-apply`). No outbound license ping.

Optional **managed SaaS** (vendor-hosted isolated cells) uses SKU `managed_saas` and JWT `deployment_mode: managed_saas`. See `docs/MANAGED_SAAS.md`. Not the same as workspace `customers` inside one deployment.

---

## Managed SaaS vs on-prem (vendor quote)

| Dimension | On-prem SKUs (`starter`…`enterprise`) | Managed SaaS (`managed_saas`) |
| :--- | :--- | :--- |
| Who runs infra | Buyer VPS / metal | Vendor per-buyer compose cell |
| JWT `deployment_mode` | `on_prem` (default) | `managed_saas` |
| Data location | Buyer disk | Vendor cell volumes; export on offboarding |
| Pricing | `sku.yaml` USDT/mo table below | Custom vendor contract (SKU `price_usd_monthly: 0`) |
| HWID bind | `hard` on paid tiers | `soft` (vendor reissues cell JWT) |

Do not list on-prem SKUs as "cloud hosted". Do not quote managed SaaS RPS without a cell-specific JWT.

---

## Positioning (self-hosted buyers)

| Buyer profile | Typical tier | Why |
| :--- | :--- | :--- |
| Solo affiliate, redirect + postback | `starter` | Rules-only fraud on tracker; no ClickHouse ML workers required |
| Media buyer / small team, CPA waste is the pain | `pro` | **IVT detector** on buyer's own ClickHouse: bot rules, auto-blacklist, silent reject |
| Network or high volume, needs programmatic + advanced ML | `scale`+ | OpenRTB engine + **ML fraud boost** + residential / moderator intel feeds |

OpenRTB (`/openrtb/bid`, in-process auction) starts at **Scale**. Audience is narrow (SSP/exchange integrations); most self-hosted buyers run click URL + S2S `/track` only.

**Pro tier choice: IVT, not ML.** Both are cold-path sidecars (`cmd/ivt-detector`, `cmd/fraud-scorer`). For license upsell without vendor data:

| | IVT (`ivt_ml_detector`) | ML (`ml_fraud_boost`) |
| :--- | :--- | :--- |
| Data | Buyer ClickHouse `ml_features_1m` | Same + optional local model artifact |
| Setup | Compose `analytics-ml` + `full` profile (CH required) | Above + `FRAUD_SCORING_ENABLED`, model path |
| Buyer-visible outcome | Blacklist / silent reject / boost via outbox | Campaign-level score boost (needs existing fraud signals) |
| Works day one | Yes (rule registry, no training) | Bootstrap model ok; prod fit is buyer-operated |
| Upsell story | "Stops bot farms on your traffic" | "Tightens scoring on your aggregates" |

IVT wins Pro because it enforces without a model training story; ML stays the Scale upsell paired with OpenRTB and intel feeds.

---

## Billing dimensions

License JWT enforces:

- `max_activations` (host count)
- `max_rps` (peak ingest RPS)
- Feature flags in `sku.yaml` (`ivt_ml_detector`, `openrtb_engine`, `ml_fraud_boost`, `ebpf_xdp_edge`, etc.)

`max_active_campaigns: 0` and `max_events_per_month: 0` in SKU schema mean no license cap on those fields.

---

## SKU table (from sku.yaml)

| SKU | USDT/mo | Hosts | Peak RPS | IVT detector | ML boost | OpenRTB | eBPF XDP |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| `starter` | 129 | 1 | 10k | no | no | no | no |
| `pro` | 329 | 1 | 25k | **yes** | no | no | no |
| `scale` | 649 | 3 | 75k | yes | yes | yes | no |
| `network` | 1199 | 10 | 150k | yes | yes | yes | no |
| `enterprise` | 2500+ | 99 | custom | yes | yes | yes | yes |
| `pilot` | 0 | 1 | 5k | no | no | no | no |

Full fields: `deploy/vendor/sku.yaml`.

### Upgrade ladder (quote to buyer)

1. **Starter -> Pro ($+200/mo):** +15k peak RPS, +7 tenants, **IVT detector** (requires `full` / `single-vps` stack with ClickHouse).
2. **Pro -> Scale ($+320/mo):** +2 hosts, +50k RPS, **OpenRTB engine**, **ML fraud boost**, residential + moderator intel feeds.
3. **Scale -> Network:** multi-region, slot migration, more hosts/RPS.

Do not quote OpenRTB on Pro; issue JWT with SKU `pro` from `sku.yaml` only.

---

## Deploy profiles (operator choice, not SKU-gated)

| Profile | ClickHouse | IVT / ML workers | Typical RAM |
| :--- | :---: | :---: | :---: |
| `ingest-only` | no | no (license flags no-op without CH) | 6-8 GB |
| `single-vps` / `full` | yes | `analytics-ml` profile: `ivt-detector`, `fraud-scorer` | 16+ GB |

Campaign state and settlement use Postgres. Tracker hot path does not block on ClickHouse.

Pro buyers need `full` + `analytics-ml` for IVT to run; Scale+ for ML microbatch (`FRAUD_SCORING_ENABLED=true`, `ml_fraud_boost` license, model under `var/fraudscore/artifacts/`). Processor defaults: microbatch flush 50ms, CH scorer scan 60s, tracker boost resync 10s.

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
| [competitive_backlog.md](./competitive_backlog.md) | Parity gaps vs Keitaro/Binom/BeMob |

---

## Support SLA (invoice template defaults)

| Tier | JWT after USDT confirm |
| :--- | :--- |
| Pro, Scale | 12 h |
| Starter, Network, Enterprise | 24 h |

Onboarding: included with first paid month (see INVOICE.md).
