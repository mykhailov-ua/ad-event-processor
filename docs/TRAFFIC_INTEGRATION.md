# BidShard traffic integration guide

Operators and buyers integrate BidShard using **separate hot-path URLs** — click redirects, event postbacks, and OpenRTB exchange traffic use different wire contracts and parser budgets. This guide maps domains, endpoints, and macros without reading engineering architecture docs.

**Related:** [ARCHITECTURE.md](ARCHITECTURE.md) §1.1 (request lifecycle), [PARSER_SECURITY.md](PARSER_SECURITY.md) (edge wire policy).

---

## 1. Domain model

| Role | Typical hostname | Endpoints | Notes |
| :--- | :--- | :--- | :--- |
| **Click / track domain** | `trk.example.com` | `GET /click`, `POST /track` | Buyer-facing traffic and postbacks |
| **OpenRTB partner** | Same or dedicated host | `POST /openrtb/bid` | SSP exchange; chunked body allowed |
| **Telegram Mini App** | Same edge | `GET /tg/click`, `GET /tg/impression` | Always on edge when `/tg/*` is proxied |
| **Admin / billing** | `control.example.com` | `/admin`, `/api/v1/*` | Control plane `:8188` — not for ad traffic |

**Keitaro / Voluum pattern:** Campaign URL (click) ≠ Postback URL (`/track`). The admin **Integration** tab on each campaign surfaces both.

---

## 2. Endpoints

### `GET /click` — click redirect

- **Response:** `302 Found` to campaign landing URL with macros expanded.
- **Query (required):** `campaign_id` (UUID).
- **Query (common):** `type=click`, `click_id`, `user_id`, `sub1`…`sub30`, passthrough UTMs (`gclid`, `ttclid`, `fbclid`, …).
- **Settlement:** Full FilterEngine path; debits click budget.
- **Edge:** Appliance default **on** (`edge_expose_click` / `EDGE_EXPOSE_CLICK=true`). Platform settings can turn it off for backward compat; when off, nginx `:8180` returns **404** and you must point DNS/LB at tracker ports `:8181–8184`.

Example:

```http
GET /click?campaign_id=550e8400-e29b-41d4-a716-446655440000&sub1=facebook&user_id=u42 HTTP/1.1
Host: trk.example.com
```

### `POST /track` — impressions, conversions, server events

- **Body:** BidShard native JSON, protobuf (`ad_event_processor_native`), or OpenRTB 3 ingress JSON (`openrtb_3`) per platform **Traffic format** setting.
- **Edge policy:** `Content-Length` required; `Transfer-Encoding: chunked` **rejected** on `/track` (see [PARSER_SECURITY.md](PARSER_SECURITY.md) §2).
- **Settlement:** FilterEngine → Redis stream → processor.

Example (native JSON):

```json
{
  "campaign_id": "550e8400-e29b-41d4-a716-446655440000",
  "type": "conversion",
  "click_id": "d5671191-236b-4e94-825e-399185a9bc8d",
  "user_id": "u42"
}
```

Impression pixel (server-side or tag manager; same `/track` URL):

```json
{
  "campaign_id": "550e8400-e29b-41d4-a716-446655440000",
  "type": "impression",
  "user_id": "{user_id}"
}
```

### `POST /openrtb/bid` — SSP exchange (OpenRTB 2.6)

- **Body:** OpenRTB 2.x bid request JSON.
- **Edge policy:** Chunked transfer encoding **allowed** (extensions rejected). See [PARSER_SECURITY.md](PARSER_SECURITY.md) §4.
- **Settlement:** In-process `RunAuction` → bid/no-bid HTTP response. **Does not** run the full FilterEngine debit path used by `/track` events.
- **Edge:** Optional — enable **Expose OpenRTB bid endpoint on edge** (`edge_expose_openrtb`).
- **Runbook:** [RTB_PRODUCTION_RUNBOOK.md](RTB_PRODUCTION_RUNBOOK.md) (shadow soak, deals, reconcile). See [RTB.md](RTB.md) for full feature list.

---

## 2.1 RTB on `/track` vs `POST /openrtb/bid` (do not conflate)

Both paths call the same in-process `RunAuction` engine, but they serve **different clients** and **different settlement paths**:

| | `POST /track` + `RTB_MODE` | `POST /openrtb/bid` |
| :--- | :--- | :--- |
| **Client** | Mobile/web SDK, server postbacks, single-endpoint integrations | SSP / exchange partners (OpenRTB 2.6) |
| **Wire** | BidShard native JSON, protobuf (`ad_event_processor_native`), or OpenRTB 3 **ingress** on `/track` | OpenRTB 2.6 bid request/response codec |
| **When auction runs** | Only when `RTB_MODE=shadow` or `live` (default `off`) | Always (when RTB licensed and catalog loaded) |
| **After auction** | Continues through **FilterEngine** (budget, geo, fraud, stream) | Returns bid/no-bid immediately; no FilterEngine |
| **Typical use** | Header-bidding SDK that posts one URL for both auction + event | Programmatic exchange seat |

**Request lifecycle on `/track` when `RTB_MODE` is enabled:**

1. Parse body into pooled `Event`.
2. **`applyRtbAuction`** (before filters):
   - `off` — skip auction; use `campaign_id` from body.
   - `shadow` — run `RunAuction`, record metrics and CH deal outcomes; **do not** rewrite `campaign_id` or reject.
   - `live` — run `RunAuction`; winner rewrites `campaign_id` and clearing price; no-bid → HTTP reject (never reaches Lua).
3. **FilterEngine.Check** — same as non-RTB `/track`.
4. Respond `202` or filter reject.

**`RTB_MODE` env / system setting** (`rtb_mode` in Redis `config:values`): `off` | `shadow` | `live`. Default `off`. Exchange partners **must not** point at `/track`; they use `/openrtb/bid` regardless of `RTB_MODE`.

**Related env (tracker):**

| Knob | Default | Effect |
| :--- | :--- | :--- |
| `RTB_MODE` | `off` | `/track` auction gate |
| `RTB_BUDGET_AUTHORITY` | `redis` | `redis` = Lua budget; `rtb` = CAS in-process before filters |
| `RTB_PREBID_IVT` | `false` | Reject datacenter/proxy IPs before auction on both paths |

Shadow promotion checklist: [RTB_PRODUCTION_RUNBOOK.md](RTB_PRODUCTION_RUNBOOK.md) §3–5. Integration profile lint: `GET /api/v1/rtb/integration-profile`.

---

## 3. Macro table

| Token | Set by | Use |
| :--- | :--- | :--- |
| `{campaign_id}` | Operator / template | Required on click URL |
| `{click_id}` | BidShard on redirect | S2S postback correlation |
| `{user_id}` | Traffic source | Frequency / identity |
| `{sub1}` … `{sub30}` | Traffic source | Source / placement labels |
| `{subid1}` … `{subid30}` | Partner templates | Aliases in outbound webhook macros |
| `gclid`, `ttclid`, `fbclid`, … | Ad network | Stored on event + passthrough to landing URL |

Platform **Click URL template** (Settings): `https://{tracking_domain}/click?campaign_id={campaign_id}&sub1={sub1}`.

---

## 3.1 Inbound affiliate S2S vs outbound CAPI

| Direction | Who calls whom | Where in admin |
| :--- | :--- | :--- |
| **Inbound S2S** | Affiliate / CRM → BidShard `POST /track` | Campaign → **Integration** → “Affiliate inbound S2S postback” |
| **Outbound CAPI** | BidShard worker → Meta / Google / TikTok | Campaign → **CAPI & Postbacks** |

Inbound body is BidShard native JSON (`campaign_id`, `type`, `click_id`, …). Edge requires `Content-Length` (no chunked). Outbound CAPI credentials never appear on the Integration tab.

Outbound CAPI attribution requires:

- Click **IP** and **User-Agent** (captured on the hot path for every `/track` / `/click`).
- Click IDs (`fbclid`, `gclid`, `ttclid`) on **GET `/click`** redirect or zero-redirect **POST `/track`**.

Configure destinations under Campaign → **CAPI & Postbacks**. Optional `test_event_code` routes Meta/TikTok events into the provider test stream.

### Postback token encryption key rotation

Tokens are AES-GCM encrypted with `POSTBACK_ENCRYPTION_KEY` (32-byte key; shorter keys are zero-padded). Rotation is a cold-path ops task:

1. Pause or scale down `postback-sender` so no decrypt runs mid-rewrite.
2. Decrypt each `postback_configs.api_token_encrypted` with the **old** key; re-encrypt with the **new** key (admin PUT with plaintext token also re-encrypts when you re-save).
3. Set `POSTBACK_ENCRYPTION_KEY` to the new value on `postback-sender` and control (admin encrypt path); restart both.
4. Smoke: save a campaign CAPI config, fire a test conversion (or DLQ retry), confirm `ad_postback_dispatch_total{status="success"}` increments.

Staging automation: `bash scripts/test/capi_meta_staging.sh` (set `TRACK_URL`, `CAMPAIGN_ID`, optional `META_TEST_EVENT_CODE`; dry-run with `CAPI_STAGING_DRY_RUN=1`).

There is no dual-key decrypt window in-process today — complete the rewrite before flipping the env.

---

## 3.2 Cost Sync join keys + True ROI

Cost Sync pulls ad-network spend into Postgres `campaign_costs` and ClickHouse `cost_snapshots` (rolled into `placement_stats_hourly.spend_micro`). The buyer report **True ROI** (`GET /api/v1/reports/true-roi`) exposes:

| Column | Meaning |
| :--- | :--- |
| **Ad Spend** | Network spend (`ad_spend_micro`) from Cost Sync |
| **True Profit** | `revenue_micro − ad_spend_micro` |
| **True ROI %** | profit / Ad Spend × 100 |
| **True CPA** | Ad Spend / conversions (tracker conversion count) |

**Join keys (calendar day + campaign):**

1. Prefer putting the BidShard `campaign_id` UUID into the ad-network campaign identifier when the network allows (Cost Sync accepts a UUID as-is).
2. Otherwise Cost Sync maps `network_prefix + external_id` → deterministic UUID (`uuid.NewSHA1(customer_id, "fb:"+id)` for Meta, similarly `google:`, …). Create/link the BidShard campaign to that UUID before spend lands.
3. On click URLs, pass the network campaign id as `sub2` / `ad_campaign_id` for placement attribution:  
   `…/click?campaign_id={uuid}&sub2={ad_campaign_id}` — same host as `/track` ([§2](#2-click-url)).

UI: **Integrations → Cost Sync** for credentials + run; **Reports → True ROI** for the columns above.

---

## 3.3 Zero-redirect tracking (lander pixel)

For landers you control, fire conversions without a full redirect chain:

1. Host `bidshard-track.js` from the admin static bundle (or copy from Campaign → **Integration** → “Zero-redirect lander pixel”).
2. Set `TRACK_CORS_ORIGINS` on the tracker to include your LP origin.
3. The script `POST`s to the same `/track` URL as inbound S2S; auto-picks `fbclid` / `gclid` / `ttclid` from the page query; optional `sub1`…`sub30`.

Edge and tracker require `Content-Length` on `POST /track` (chunked rejected). CORS preflight: `OPTIONS /track` → `204` when origin is allowed.

**CI:** gzip size of `bidshard-track.js` &lt; 2 KB (`scripts/ci/check_web_dist.sh`).

---

## 4. Shard routing on the edge

When traffic passes through nginx (`:8180` / `:443`), the edge picks a tracker peer using the same CRC32 slot table as Go:

```
slot  = crc32_castagnoli(campaign_id) & 1023
shard = slot_table[slot]
```

- `/track`, `/tg/*`: campaign id from body DFA scan.
- `/click`: campaign id from strict query parse (`campaign_id` UUID).
- `/openrtb/bid`: campaign id from body peek (or `X-Campaign-Id` header).

Verify parity: `go test ./internal/domain/ -run SlotMap -count=1` and edge slot-map sync metrics.

---

## 5. Enabling optional edge paths

1. Open **Platform settings** → **Features**.
2. Enable **Expose click URL on edge** (default on for new `single_vps` installs) and/or **Expose OpenRTB bid endpoint on edge**.
3. **Save**, then **Apply to disk** (writes `EDGE_EXPOSE_*` to compose env).
4. Reload nginx if env changed; Redis `config:values` sync applies gate flags without config edit.

**Acceptance check:**

```bash
curl -sI "https://trk.example/click?campaign_id=550e8400-e29b-41d4-a716-446655440000"
# Expect: HTTP/1.1 302 (or 4xx if campaign unknown / filtered)
```

When flags are off, `GET /click` on `:8180` returns **404** from the edge gate (tracker ports still serve `/click`).

---

## 6. Direct tracker access (dev / custom TLS)

Tracker listeners `:8181–8184` accept all hot-path URLs regardless of edge flags. Use this for:

- Local development without nginx
- Custom click domain terminating TLS on a separate load balancer

Match parser policies documented in [PARSER_SECURITY.md](PARSER_SECURITY.md) so nginx and gnet stay aligned.

---

## 7. Domain health and SSL (operator)

**Settings → Domains** (`/settings/domains`) lists tracking, admin, and custom hostnames:

- HTTP reachability + TLS expiry (worker probe every 5 min, `DOMAIN_HEALTH_INTERVAL_MIN`)
- **Probe now** and **Setup SSL** (certbot / Caddy hook via `POST /api/v1/domains/{hostname}/ssl/setup`)

Use this after pointing DNS A-records at the appliance. Custom domains register with `POST /api/v1/domains`.

---

## 8. Smart Alerts (operator webhook rules)

**Integrations → Smart Alerts** (`/integrations/smart-alerts`): threshold rules on ClickHouse metrics (`clicks`, `cr`, `roi_pct`, `bot_clicks`) → JSON webhook (Slack / Discord / custom). Evaluation interval 5–60 min (`SMART_ALERTS_INTERVAL_MIN`). History tab + per-event ack.

Not a substitute for Prometheus/Alertmanager on the host — complements buyer-visible KPI drift without opening the ops stack.
