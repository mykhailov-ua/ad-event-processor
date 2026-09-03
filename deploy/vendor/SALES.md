# Vendor license reference

Internal. Not shipped to customers. Canonical limits and feature matrix: [sku.yaml](./sku.yaml). Issue JWT: `go run ./cmd/license-issue --sku <code> ...`.

Self-hosted appliance: buyer runs the stack on their VPS; license is an Ed25519 JWT applied locally (Admin Settings, `license-apply`, or `POST /api/v1/license/apply`). No outbound license ping. No vendor-hosted tenants.

---

## License model

| Property | Detail |
| :--- | :--- |
| Format | Monthly Ed25519 JWT on disk (`var/license.jwt` default) |
| Product id | `ad-event-processor` |
| Enforcement package | `internal/licensing` |
| Hot path | Inline license snapshot; tracker does not call Postgres per request |
| Ingest gate | `IngestAllowed` false only for `EXPIRED` or `REVOKED` (grace and offline warn still accept) |
| Feature gate | `SanitizeFeaturesForSKU` clamps JWT claims to SKU matrix before `Effective` entitlements |
| Renewal | Re-issue JWT with same `deployment_id`; paste new token in Admin |
| Telemetry default | `TELEMETRY_ENABLED=false` on appliance installs |

Billable dimensions in SKU schema:

| Limit | Meaning |
| :--- | :--- |
| `max_rps` | Peak ingest RPS ceiling (tracker filter) |
| `max_activations` | Distinct host fingerprints allowed (`bind.mode: multi`) |
| `max_regions` | Multi-region compose profile cap |
| `max_tenants` | Workspace / team tenant cap |
| `max_api_keys` | API key count cap |
| `max_export_chunk_bytes` | Report export chunk size cap |
| `max_active_campaigns: 0` | No license cap on campaigns |
| `max_events_per_month: 0` | No license cap on event volume |
| `max_requests_per_day: 0` | No daily request cap in schema |

Quote buyers on **peak RPS** and **host count** (activations). Campaign and event volume are not SKU-gated when limits are zero.

---

## Tier positioning

| Buyer profile | SKU | Rationale |
| :--- | :--- | :--- |
| Solo affiliate, rules-only fraud | `starter` | No ClickHouse ML workers; `/track` + S2S postbacks |
| Media buyer, CPA waste reduction | `pro` | IVT detector on buyer ClickHouse (`ivt_ml_detector`) |
| Network, OpenRTB + ML antifraud | `scale` | OpenRTB engine, ML boost, residential/moderator intel |
| Multi-region footprint | `network` | `multi_region`, `slot_migration`, 10 hosts |
| Edge XDP + platform API sync | `enterprise` | `ebpf_xdp_edge`, `ad_platform_campaign_api` |

OpenRTB starts at **Scale**. Most buyers use click URL + S2S `/track` only.

**Pro upsell:** IVT (`ivt_ml_detector`) works with rule registry; no model training story on buyer side. ML (`ml_fraud_boost`) stays Scale+.

---

## SKU table (price and capacity)

| SKU | USDT/mo | Valid days | Grace days | Hosts (`max_activations`) | Peak RPS | Regions | Tenants | API keys |
| :--- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `starter` | 129 | 30 | 7 | 1 | 10k | 1 | 3 | 5 |
| `pro` | 329 | 30 | 7 | 1 | 25k | 1 | 10 | 10 |
| `scale` | 649 | 30 | 7 | 3 | 75k | 1 | 25 | 25 |
| `network` | 1,199 | 30 | 7 | 10 | 150k | 3 | 50 | 50 |
| `enterprise` | 2,500+ | 30 | 7 | 99 | custom | 99 | 999999 | 999 |
| `pilot` | 0 | 14 | 7 | 1 | 5k | 1 | 1 | 3 |
| `license` | internal | 30 | 7 | 999 | unlimited | 99 | 999999 | 999 |

Export chunk limits (`max_export_chunk_bytes`):

| SKU | Chunk cap |
| :--- | ---: |
| `starter` | 5 MiB |
| `pro` | 10 MiB |
| `scale` | 50 MiB |
| `network` | 100 MiB |
| `enterprise` / `license` | 1 GiB |
| `pilot` | 0 (exports disabled) |

Bind mode per SKU (`sku.yaml` `bind.mode`):

| SKU | Mode | Behavior |
| :--- | :--- | :--- |
| `starter`, `pilot` | `hard` | Single host fingerprint or HWID v2 hash |
| `pro`, `scale`, `network` | `multi` | Up to `max_activations` distinct fingerprints in `vendor.license_activations` |
| `enterprise`, `license` | `fingerprint` | Hard bind (same code path as `hard`) |

HWID v2 (Argon2id over DMI/disk/MAC telemetry) preferred over legacy host fingerprint when JWT carries `hwid_hash`. Issue with `--hwid-v2 <hash>`.

---

## Feature matrix

Boolean flags in JWT `features` block. `SanitizeFeaturesForSKU` forces off flags not included in tier even if manually set at issue time.

| Feature key | Starter | Pro | Scale | Network | Enterprise | Pilot | What it gates |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: | :--- |
| `margin_guard` | yes | yes | yes | yes | yes | yes | Margin guard policies and ledger worker (`CONTROL_ENABLE_MARGIN_GUARD`) |
| `ivt_ml_detector` | no | yes | yes | yes | yes | no | `cmd/ivt-detector`, IVT reports, compose `analytics-ml` profile |
| `ml_fraud_boost` | no | no | yes | yes | yes | no | `cmd/fraud-scorer`, `FRAUD_SCORING_ENABLED`, boost snapshot on hot path |
| `rtb_live` | no | no | yes | yes | yes | no | Live RTB auction path (paired with `openrtb_engine`) |
| `openrtb_engine` | no | no | yes | yes | yes | no | `/openrtb/bid`, RTB admin, command palette RTB entries |
| `external_residential_intel` | no | no | yes | yes | yes | no | Residential proxy hot-read in tracker (`ApplyResidentialIntelHotReadWhenEntitled`) |
| `moderator_intel_feed` | no | no | yes | yes | yes | no | Moderator intel feed loader on tracker gnet path |
| `fraud_dispute_evidence` | no | no | yes | yes | yes | no | Fraud dispute evidence export routes |
| `multi_region` | no | no | no | yes | yes | no | Compose `multi-region` profile, region-proxy |
| `slot_migration` | no | no | no | yes | yes | no | Redis slot migration dual-write and admin migration jobs |
| `ebpf_xdp_edge` | no | no | no | no | yes | no | `cmd/edge-xdp`, `cmd/edge-bpf-sync`, XDP maps on tracker edge |
| `ad_platform_campaign_api` | no | no | no | no | yes | no | Platform campaign sync worker (`CONTROL_ENABLE_PLATFORM_CAMPAIGN_SYNC`) |

Normalized helpers (`internal/licensing/entitlements/features.go`):

| Helper | True when |
| :--- | :--- |
| `OpenRTBEnabled()` | `openrtb_engine` or `rtb_live` |
| `MlFraudBoostEnabled()` | `ml_fraud_boost` |
| `IvtMLEnabled()` | `ivt_ml_detector` |
| `EbpfEdgeEnabled()` | `ebpf_xdp_edge` |
| `MultiRegionEnabled()` | `multi_region` |
| `AdPlatformCampaignAPIEnabled()` | `ad_platform_campaign_api` |
| `FraudDisputeEvidenceEnabled()` | `fraud_dispute_evidence` |

Admin command palette and integration hub hide entries when the matching feature is off (`GET /api/v1/command-palette`).

---

## License states

| State | Tracking | Ingest | Admin UI |
| :--- | :--- | :--- | :--- |
| `ACTIVE` | JWT valid | Allowed | Normal |
| `GRACE` | JWT expired within `grace_days` (7 default) | Allowed | Renewal banner |
| `EXPIRED` | Past grace | **Blocked** | Block banner |
| `REVOKED` | Vendor revoke row or `claims.revoked` | **Blocked** | Block banner |
| `OFFLINE_WARN` | Optional heartbeat offline, before renew tail | Allowed | `banner_severity: warn` |
| `OFFLINE_GRACE` | Heartbeat offline near `offline_grace_days` | Allowed | `banner_severity: grace` |

Pilot SKU sets `offline_grace_days: 0` in catalog (no extended offline tail beyond JWT grace).

Diagnostics: `GET /api/v1/license/status` returns `deployment_id`, `host_fingerprint`, `hwid_v2`, `hwid_match`, `days_to_expiry`, `plan_code`, `max_rps`, `tier_warnings`.

Env knobs (`internal/licensing/entitlements/heartbeat_policy.go`):

| Env | Default | Effect |
| :--- | :--- | :--- |
| `AD_EVENT_PROCESSOR_LICENSE_OFFLINE_GRACE_DAYS` | 14 | Days without heartbeat before effective `EXPIRED` |
| `AD_EVENT_PROCESSOR_LICENSE_RENEW_BEFORE_DAYS` | 7 | Pre-expiry warning window |

Lab garbled builds: `AD_EVENT_PROCESSOR_LICENSE_GUARD=0` disables ptrace watchdog probes (see `.cursor/rules/licensing.mdc`).

---

## Deploy profiles (operator, not SKU-gated)

Compose profiles are operator choice; SKU gates **features inside** the profile.

| Profile | ClickHouse | IVT / ML workers | RAM hint |
| :--- | :---: | :---: | :---: |
| `ingest-only` | no | no | 6-8 GB |
| `minimal` | yes | optional `analytics-ml` | ~6 GB |
| `full` / `single-vps` | yes | `analytics-ml` adds workers | 16+ GB |
| `multi-region` | yes | per region | Enterprise / Network license required |
| `analytics-ml` | yes (required) | `fraud-scorer`, `ivt-detector` | +2-4 GB |

SKU to profile mapping (minimum for entitled features):

| SKU feature | Required profile / env |
| :--- | :--- |
| `ivt_ml_detector` | `full` + `analytics-ml`; buyer ClickHouse |
| `ml_fraud_boost` | above + `FRAUD_SCORING_ENABLED=1` + model under `var/fraudscore/artifacts/` |
| `openrtb_engine` | `full`; tracker OpenRTB routes |
| `ebpf_xdp_edge` | edge compose + `CAP_BPF`; separate `edge-xdp` binary |
| `multi_region` | compose profile `multi-region` |
| `ad_platform_campaign_api` | `CONTROL_ENABLE_PLATFORM_CAMPAIGN_SYNC=1` |

Bootstrap: `bash scripts/install/appliance_bootstrap.sh` (default `ingest-only`); `--profile full` for analytics tiers.

---

## Pilot workflow

1. Collect buyer identity: Telegram id (primary), optional USDT tx for paid conversion tracking.
2. Run trial registry check (see **Trial registry**); reject repeat pilot on same `telegram` or `hwid`.
3. Issue pilot JWT:

```bash
export AD_EVENT_PROCESSOR_LICENSE_PRIVATE_KEY_FILE=deploy/vendor/license_private.key
go run ./cmd/license-issue \
  --sku pilot \
  --customer "Acme Media" \
  --deployment-id "<uuid>" \
  --hwid-v2 "<hwid>" \
  --telegram-id "<telegram_user_id>" \
  --out /tmp/acme-pilot.jwt
```

4. Buyer applies JWT (Admin Settings or `license-apply`). No restart; entitlements reload immediately.
5. Pilot limits: 14 days, 5k RPS, 1 host, rules-only fraud (`margin_guard` only among ML/RTB flags).
6. Conversion: on USDT payment, re-issue paid SKU with **same** `deployment_id`:

```bash
go run ./cmd/license-issue \
  --sku pro \
  --customer "Acme Media" \
  --deployment-id "<same-uuid>" \
  --mark-converted
```

7. Record HWID after first appliance boot if needed: `license-issue --record-hwid --deployment-id <uuid>`.

Approve Telegram self-serve pending row: `license-issue --approve-pending <pending_id>`.

---

## Trial registry

Repeat-trial defense at **JWT issue time** on vendor machine, not on buyer VPS.

| Anchor | Use |
| :--- | :--- |
| `telegram` | Primary buyer identity |
| `deployment_id` | Links pilot to paid renewal |
| `hwid` | Block second pilot on same metal |
| `usdt_tx` | One wallet address per buyer line |

File: `deploy/vendor/trial_registry.json` (`VENDOR_TRIAL_REGISTRY` env).

Reject pilot when prior `telegram` or `hwid` with status `active` or `expired`; same USDT wallet on prior pilot. Do not hard-block /24 IP alone.

| CLI | Purpose |
| :--- | :--- |
| `go run ./cmd/trial-registry list-pending` | Open self-serve requests |
| `license-issue --approve-pending <id>` | Issue pilot from pending |
| `license-issue --record-hwid` | Bind HWID after first boot |
| `license-issue --mark-converted` | Close pilot row on paid conversion |
| `VENDOR_TRIAL_FORCE=1 --force --force-reason "..."` | Operator override (audited) |

---

## Issue and renew (vendor)

Prerequisites: Ed25519 private key (`deploy/vendor/license_private.key` or `KEYS.md` rotation). Public key ships in release images.

```bash
export AD_EVENT_PROCESSOR_LICENSE_PRIVATE_KEY_FILE=deploy/vendor/license_private.key

# New customer (paid)
go run ./cmd/license-issue \
  --sku scale \
  --customer "Network Buyer Ltd" \
  --deployment-id "$(uuidgen)" \
  --hwid-v2 "<optional-hwid>" \
  --out /tmp/buyer.jwt

# Renewal / tier change (reuse deployment_id)
go run ./cmd/license-issue \
  --sku scale \
  --customer "Network Buyer Ltd" \
  --deployment-id "<existing-uuid>" \
  --out /tmp/buyer-renewal.jwt

# Revoke (sets revoked claim; buyer ingest stops after reload)
go run ./cmd/license-issue \
  --sku scale \
  --customer "Network Buyer Ltd" \
  --deployment-id "<uuid>" \
  --revoke \
  --out /tmp/buyer-revoked.jwt
```

Flags reference (`cmd/license-issue`):

| Flag | Purpose |
| :--- | :--- |
| `--sku-file` | Catalog path (default `deploy/vendor/sku.yaml`) |
| `--sku` | Tier code |
| `--valid-days` | Override catalog validity |
| `--kid` | Signing key id for rotation |
| `--telegram-id`, `--usdt-tx` | Trial registry anchors |

Verification gates: `make license-verify`, `make license-red-team`.

---

## Customer apply

| Path | Command / route |
| :--- | :--- |
| Admin UI | Settings -> License -> paste JWT |
| CLI | `license-apply` (ships with appliance) |
| API | `POST /api/v1/license/apply` body `{ "token": "<jwt>" }` |
| First owner signup | `POST /api/v1/auth/activate` with `license_token` |

Install env:

| Env | Value |
| :--- | :--- |
| `AD_EVENT_PROCESSOR_LICENSE_MODE` | `file` |
| `AD_EVENT_PROCESSOR_LICENSE_FILE` | `var/license.jwt` (default) |
| `TELEMETRY_ENABLED` | `false` (default on appliance) |

No process restart required after apply; `internal/licensing` watcher reloads snapshot.

---

## Upgrade and downgrade paths

| Change | Procedure |
| :--- | :--- |
| Tier upgrade | Re-issue JWT same `deployment_id`, higher SKU; buyer pastes token |
| Tier downgrade | Re-issue lower SKU; `SanitizeFeaturesForSKU` disables features on next reload |
| Host add (multi bind) | Within `max_activations`; new host apply triggers `CheckHostActivation` |
| Host add over cap | Upgrade SKU or retire old fingerprint in vendor DB |
| OpenRTB enable | Minimum `scale`; set `rtb_live` + `openrtb_engine` in catalog |
| XDP enable | `enterprise` only; deploy edge binaries separately |
| Pilot to paid | New JWT, same `deployment_id`, `--mark-converted` |

Deployment ceiling invariant (P-C4-03): customer JWT limits and features never exceed deployment-level ceiling when both snapshots present.

---

## Payment workflow (vendor)

USDT monthly via Cryptomus webhook on **vendor VPS only** (not appliance).

| Step | Detail |
| :--- | :--- |
| Invoice | Template: [INVOICE.md](./INVOICE.md) |
| Webhook | Verify `sign`; idempotency on payment `uuid` |
| JWT delivery | Issue only on `paid` or `paid_over` status |
| Appliance `POST /api/v1/selfserve/payment-intents` | Ledger top-up for media-buyer balance, **not** license renewal |

Support SLA (JWT delivery after USDT confirm):

| Tier | SLA |
| :--- | :--- |
| Pro, Scale | 12 h |
| Starter, Network, Enterprise | 24 h |

Onboarding call included with first paid month per [INVOICE.md](./INVOICE.md).

---

## Volume bands (internal metering)

Optional `volume_band` on JWT: `S`, `M`, `L`. Used for internal PU metering (`internal/licensing/entitlements/volume.go`), not buyer-facing caps when `max_events_per_month: 0`.

| Band | Included events (weighted) | Base PU |
| :--- | ---: | ---: |
| S | 10B | 100 |
| M | 50B | 250 |
| L | 100B | 500 |

Module PU add-ons (when feature enabled): OpenRTB, eBPF XDP, IVT, ML boost coefficients per band. Billable weight: accepted events 1.0, dedup rejects 0.1, eBPF drops 0.0.

---

## Enforcement surfaces (quick map)

| Surface | Check |
| :--- | :--- |
| Tracker ingest | RPS limit, `IngestAllowed`, residential/moderator intel flags |
| OpenRTB `/openrtb/bid` | `OpenRTBAllowed(state, ent)` |
| Edge XDP | `EbpfEdgeAllowed` at daemon start |
| Processor ML | `MlFraudBoostEnabled` on deployment snapshot |
| Admin routes | Per-handler `ModuleAllowed` or licensing bridge |
| Command palette | Feature-filtered nav |
| Report export | `max_export_chunk_bytes` |
| API keys / tenants | Limit counters on create |

---

## Related files

| File | Use |
| :--- | :--- |
| [sku.yaml](./sku.yaml) | Canonical tier limits and features |
| [KEYS.md](./KEYS.md) | Ed25519 public keys and rotation |
| [INVOICE.md](./INVOICE.md) | USDT invoice template |
| [MARKETING.md](./MARKETING.md) | Buyer-facing feature list (no SKU math) |
| [ANTIFRAUD.md](./ANTIFRAUD.md) | Fraud behavior reference for operators |
| [VENDOR.md](./VENDOR.md) | Vendor docs index |
| `.cursor/rules/licensing.mdc` | Engineering invariants and verification catalog |
