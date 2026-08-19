# Trial

Pilot SKU abuse policy (`deploy/vendor/sku.yaml`). Target: solo media buyers, arbitrage teams. **No CRM.**

**See also:** [LICENSE.md](LICENSE.md), [START.md](START.md), [BILLING.md](BILLING.md). Implementation: [DEVELOPMENT.md](DEVELOPMENT.md) §14.

## Two planes

```text
VENDOR PLANE                          CUSTOMER PLANE (buyer VPS)
license-issue, signing key            tracker + control + local PG
vendor trial registry                 var/license.jwt on disk (local dev)
Telegram / USDT                       TELEMETRY_ENABLED=false (default)
```

Repeat-trial defense runs **when you sign the JWT** on the vendor plane. Buyer `vendor.*` tables do not stop cross-customer abuse (fresh PG per VPS). Closed-contour installs do not phone home.

## Without CRM

| Need | Replacement |
| :--- | :--- |
| Identify repeat seekers | Vendor registry: Telegram ID, USDT tx, `deployment_id`, HWID |
| Block re-issue | Check registry in `license-issue` before pilot JWT |
| Conversion | USDT invoice → renewal JWT, same `deployment_id` |
| False positive | Manual re-issue, same `deployment_id`; log in vendor notes |

## Threat model

Abuser loop: new VPS → pilot JWT → traffic until expiry → delete VM → new identity → repeat pilot.

**Mitigated:** JWT copy blocked (`VerifyDeploymentBind`); pilot expiry; same `deployment_id` on conversion; repeat pilot on same Telegram/HWID (`internal/trialregistry`).

**Gaps without registry config:** second pilot on new VPS; no vendor anchor store unless `BIDSHARD_VENDOR_TRIAL_REGISTRY` set.

## Pilot SKU (`sku.yaml`)

| Field | Value |
| :--- | ---: |
| `valid_days` | 10 |
| `max_rps` | 5000 |
| OpenRTB | off |
| Goal | Latency, Cost Sync / CAPI smoke — not free media buying |

## Vendor registry anchors

| Anchor | When | Use |
| :--- | :--- | :--- |
| `telegram` | Before first JWT | Primary identity |
| `deployment_id` | Mint once per journey | Links pilot → paid |
| `hwid` | After install / support bundle | Block second pilot on same metal |
| `usdt_tx` | On payment | One wallet → one buyer line |

**Reject pilot when:** prior `telegram` or `hwid` anchor with status `active|expired`; same wallet on prior pilot. Do not hard-block IP /24 alone.

## Flows

**Manual today:** Telegram → `license-issue --sku pilot` → install + `license-apply` → bundle HWID → day 10 convert or stop → paid USDT + starter JWT same `deployment_id`.

**Target:** `vendor-trial-bot` → `trial-registry list-pending` → `license-issue --approve-pending`. Bot token: `BIDSHARD_VENDOR_TRIAL_BOT_TOKEN` on vendor workstation only — never commit; do not reuse appliance `TELEGRAM_BOT_TOKEN`.

**HWID override:** re-issue with same `deployment_id`, new `--hwid-v2`; audit in vendor notes.

## Out of scope

CRM, customer-plane trial denylist, IP-only block, campaign-count caps, appliance billing schema for trial registry (vendor CLI only until reviewed bridge).
