# Traffic integration

Hot-path URLs use separate wire contracts: click redirects, event postbacks, OpenRTB exchange. See [PARSER.md](PARSER.md) for edge wire policy.

## Domain model

| Role | Host | Endpoints |
| :--- | :--- | :--- |
| Click / track | `trk.example.com` | `GET /click`, `POST /track` |
| OpenRTB partner | same or dedicated | `POST /openrtb/bid` |
| Telegram Mini App | edge | `GET /tg/click`, `GET /tg/impression` |
| Admin / billing | `control.example.com` | `/admin`, `/api/v1/*` (`:8188`) |

Keitaro/Voluum: Campaign URL (click) ≠ Postback URL (`/track`). Admin **Integration** tab shows both.

## Endpoints

### `GET /click`

- `302` redirect to landing URL with macros expanded.
- Required: `campaign_id` (UUID). Common: `type=click`, `click_id`, `user_id`, `sub1`…`sub30`, UTMs (`gclid`, `ttclid`, `fbclid`, …).
- Full FilterEngine path; debits click budget.
- Edge default on (`EDGE_EXPOSE_CLICK=true`). When off, nginx `:8180` returns **404** — point DNS/LB at tracker `:8181–8184`.

```http
GET /click?campaign_id=550e8400-e29b-41d4-a716-446655440000&sub1=facebook&user_id=u42 HTTP/1.1
Host: trk.example.com
```

**DMR referer hiding** (`dmr=1` / `dmr=true` or campaign `dmr_enabled`): `200` HTML meta refresh + `location.replace` instead of `302`. Coverage: `/click`, `/tg/click`, tracking-domain rotation. Plain `302` sends `Referrer-Policy: no-referrer`.

### `POST /track`

- Body: native JSON, protobuf (`ad_event_processor_native`), or OpenRTB 3 ingress (`openrtb_3`) per **Traffic format** setting.
- Edge: `Content-Length` required; chunked **rejected** ([PARSER.md](PARSER.md) §2).
- FilterEngine → Redis stream → processor.

```json
{"campaign_id":"550e8400-e29b-41d4-a716-446655440000","type":"conversion","click_id":"d5671191-236b-4e94-825e-399185a9bc8d","user_id":"u42"}
```

### `POST /openrtb/bid`

- OpenRTB 2.x bid request JSON. Chunked allowed (extensions rejected).
- In-process `RunAuction` → bid/no-bid. **No** FilterEngine debit path.
- Optional edge expose (`edge_expose_openrtb`). Runbook: [RTB.md](RTB.md).

## RTB: `/track` vs `/openrtb/bid`

Same `RunAuction` engine; different clients and settlement:

| | `POST /track` + `RTB_MODE` | `POST /openrtb/bid` |
| :--- | :--- | :--- |
| Client | SDK, server postbacks | SSP / exchange (OpenRTB 2.6) |
| Wire | native / protobuf / OpenRTB 3 ingress | OpenRTB 2.6 codec |
| Auction | Only when `RTB_MODE=shadow\|live` (default `off`) | Always (when licensed) |
| After auction | FilterEngine (budget, geo, fraud, stream) | Bid/no-bid immediately |

**`/track` lifecycle with RTB:** parse → `applyRtbAuction` (`off`/`shadow`/`live`) → `FilterEngine.Check` → `202` or reject.

| Knob | Default | Effect |
| :--- | :--- | :--- |
| `RTB_MODE` | `off` | `/track` auction gate |
| `RTB_BUDGET_AUTHORITY` | `redis` | `redis` = Lua budget; `rtb` = CAS before filters |
| `RTB_PREBID_IVT` | `false` | Reject DC/proxy IPs before auction |

Exchange partners **must** use `/openrtb/bid`, not `/track`. Profile lint: `GET /api/v1/rtb/integration-profile`.

## Macros

| Token | Set by | Use |
| :--- | :--- | :--- |
| `{campaign_id}` | Operator | Required on click URL |
| `{click_id}` | ad-event-processor | S2S correlation |
| `{user_id}` | Traffic source | Frequency / identity |
| `{sub1}`…`{sub30}` | Traffic source | Source labels |
| `gclid`, `ttclid`, `fbclid`, … | Ad network | Stored + passthrough |

Click URL template: `https://{tracking_domain}/click?campaign_id={campaign_id}&sub1={sub1}`.

## S2S and CAPI

| Direction | Caller | Admin |
| :--- | :--- | :--- |
| Inbound S2S | Affiliate → `POST /track` | Campaign → **Integration** |
| Outbound CAPI | Worker → Meta/Google/TikTok | Campaign → **CAPI & Postbacks** |

CAPI needs click IP/UA and click IDs on `/click` or zero-redirect `/track`.

**Postback token rotation** (`POSTBACK_ENCRYPTION_KEY`, 32-byte AES-GCM):

1. Pause `postback-sender`.
2. Decrypt `postback_configs.api_token_encrypted` with old key; re-encrypt with new (or re-save via admin PUT).
3. Set new key on `postback-sender` + control; restart both.
4. Smoke: test conversion → `ad_postback_dispatch_total{status="success"}`.

No dual-key window. Staging: `bash scripts/test/capi_meta_staging.sh`.

## Cost Sync / True ROI

`GET /api/v1/reports/true-roi`: **Ad Spend** (network), **True Profit** (`revenue − spend`), **True ROI %**, **True CPA**.

Join keys (calendar day + campaign): prefer UUID in ad-network campaign id; else `uuid.NewSHA1(customer_id, "fb:"+id)` etc. Pass network id as `sub2`: `…/click?campaign_id={uuid}&sub2={ad_campaign_id}`.

UI: **Integrations → Cost Sync**; **Reports → True ROI**.

## Zero-redirect tracking

1. Host `ad-event-processor-track.js` (admin bundle or Integration tab).
2. Set `TRACK_CORS_ORIGINS` on tracker.
3. Script `POST`s `/track`; auto-picks click IDs from query.

`Content-Length` required; `OPTIONS /track` → `204` when origin allowed. CI: gzip &lt; 2 KB (`scripts/ci/check_web_dist.sh`).

## Shard routing (edge)

```
slot  = crc32_castagnoli(campaign_id) & 1023
shard = slot_table[slot]
```

- `/track`, `/tg/*`: campaign id from body DFA.
- `/click`: strict query UUID.
- `/openrtb/bid`: body peek or `X-Campaign-Id`.

Verify: `go test ./internal/domain/ -run SlotMap -count=1`.

## Edge path toggles

**Platform settings → Features:** **Expose click URL on edge** (default on `single_vps`), **Expose OpenRTB bid endpoint on edge** → Save → **Apply to disk** → reload nginx.

```bash
curl -sI "https://trk.example/click?campaign_id=550e8400-e29b-41d4-a716-446655440000"
# Expect: 302 (or 4xx if unknown/filtered)
```

## Direct tracker access

Ports `:8181–8184` serve all hot paths regardless of edge flags. Match [PARSER.md](PARSER.md) policies.

## Domain health / SSL

**Settings → Domains:** reachability + TLS expiry (`DOMAIN_HEALTH_INTERVAL_MIN`, default 5 min). **Probe now**, **Setup SSL** via `POST /api/v1/domains/{hostname}/ssl/setup`. Register custom domains: `POST /api/v1/domains`.

## Smart Alerts

**Integrations → Smart Alerts:** threshold rules on CH metrics → webhook. Interval 5–60 min (`SMART_ALERTS_INTERVAL_MIN`). Complements Prometheus, not a replacement.
