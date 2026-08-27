# Browser pixel and CAPI backlog (internal)

**Status:** Closed (2026-08-26). All slugs shipped in tree; verification via `go test ./internal/postback/...`, `cd web && npm test`, and optional `bash scripts/test/pixel_live_smoke.sh` on a running stack.

Shippable work to close the gap between **server-side outbound CAPI** (shipped) and **operator-ready browser tracking** (lander pixel + optional ad-network pixels with deduplication). Derived from Integration tab audit, `traffic.mdc`, and outbound postback code in `internal/postback/`.

**Not in scope:** Managed OAuth for Meta/Google/TikTok; loading `fbq`/`gtag`/`ttq` from tracker hot path; sync Graph API calls on `POST /track`; sub-5-minute Cost Sync; claims that browser pixel alone satisfies Meta/Google attribution without click ids; product-prefixed static embed filenames (`ad-event-processor-track.js` per `anti-slop.mdc`).

**Canonical implementation truth:** `.cursor/rules/traffic.mdc`, `docs/INTEGRATIONS.md`, `internal/postback/provider_*.go`, `web/src/static/track.js`. Outbound adapter wave: `capi_outbound_platform_wave` in [competitive_backlog.md](./competitive_backlog.md) (shipped).

Cross-reference slugs by name in PRs and docs. Do not close a slug until every applicable gate below is checked.

---

## Priority legend

| Label | Meaning |
| :--- | :--- |
| `pixel_p0` | Blocks correct browser conversions or CAPI dedup; ship first |
| `pixel_p1` | Operator UX (snippets, validation, lander editor); high support load without it |
| `pixel_p2` | Funnel events, extra networks, polish |

---

## Global close checklist (every slug)

Rule: `anti-slop.mdc` agent checklist

- [x] Every new symbol resolves (`go build` / `go test -c` on touched packages)
- [x] Hot path does not import cold fraud scoring or `internal/postback` (`boundaries.mdc`, `hot-path.mdc`)
- [x] No sync Postgres / ClickHouse / external HTTP on `/track` accept path (`architecture.mdc`)
- [x] At most one sync Redis `EVALSHA` per accepted event; no extra round-trips (`hot-path.mdc`)
- [x] Verification commands pasted in PR with package path (no unrun claims - `quality.mdc`)
- [x] Holdout or integration test when behavior is non-obvious (`testing.mdc`)
- [x] Doc claims match code; no microbench cited as prod SLA (`anti-slop.mdc`)
- [x] `bash scripts/ci/pr_fast.sh` scoped to touched packages (`ci.mdc`)
- [x] `bash scripts/ci/check_no_legacy_naming.sh` clean on touched vendor docs (`naming.mdc`)

Rule: `ui.mdc` (when touching `web/**`)

- [x] Read Go handler DTO before new form fields
- [x] `apiConfirmed` on mutations; `renderErrorBlock` on errors
- [x] `cd web && npm run typecheck` and `bash scripts/ci/admin_web.sh`
- [x] JSDoc on every function and block-scoped arrow in touched files (`quality.mdc`)

Rule: `core.mdc` commit policy (when landing code)

- [x] Imperative commit title names concrete surface (route, static asset, adapter, page)
- [x] Docs-only pixel/CAPI claims ship in the same commit as code

---

## Architecture: two pixel roles

| Role | Runs on | Endpoint | Purpose |
| :--- | :--- | :--- | :--- |
| **Tracker lander pixel** | Buyer LP | `POST /track` (same host as click) | Zero-redirect conversion ingest; capture `click_id`, `fbclid`, `gclid`, `ttclid` |
| **Ad-network browser pixel** | Buyer LP | Meta / Google / TikTok directly | Optimization signals, audience, browser-side Enhanced Conversions |
| **Outbound CAPI** | `postback-sender` worker | Provider Graph/API | Server-side conversion after settlement; already shipped for facebook, google, tiktok, taboola, outbrain, microsoft_ads, webhook |

Native networks **Taboola / Outbrain** do not need a browser pixel in v1; they require click ids (`tblci`, `ob_click_id`) on the conversion payload only.

```text
Ad click -> GET /click -> LP (optional network JS + track.js)
                |
                v
         POST /track (202, async settle)
                |
                v
    ConversionPostbackEnqueuer -> outbox -> PostbackWorker -> CAPI/S2S
```

---

## Hot / cold / browser boundary

| Surface | Allowed | Forbidden |
| :--- | :--- | :--- |
| `/track`, `/click` hot | Parse body, FilterEngine, one `EVALSHA`, async stream, CORS `OPTIONS` 204 | PG/CH, outbound HTTP to Meta/Google, `fmt.Sprintf` in inner loops |
| `track.js` on LP | `fetch()` to tracker with `keepalive`; read query params | Blocking redirect through tracker; secrets in snippet |
| Processor settlement | Batch PG/CH, conversion reject, payout applier | Blocking tracker ingest |
| `internal/postback` worker | HTTP to ad networks; 10 s client timeout per request today | Per-event work on tracker thread |
| Admin snippet generator | Copy-paste HTML/JS only | Storing network access tokens in generated LP HTML |

SLA reference (`core.mdc`): tracker `ad_http_request_duration_seconds` p95 < 50 ms, p99 < 80 ms (hard 100 ms); filter deadline `FILTER_TIMEOUT_MS` <= 100 ms production.

---

## SLA and latency budget (this backlog)

| Surface | Metric / knob | Ceiling | Slugs |
| :--- | :--- | :--- | :--- |
| Tracker `POST /track` (lander pixel) | `ad_http_request_duration_seconds` p95 | < 50 ms | `pixel_track_js_prod_asset`, `track_js_event_id_contract` |
| Tracker `POST /track` | p99 | < 80 ms (hard 100 ms) | Same |
| Tracker CORS preflight | `OPTIONS /track` | < 10 ms p99 (no body parse) | `pixel_track_js_prod_asset` |
| Filter chain | `FILTER_TIMEOUT_MS` prod | <= 100 ms | No new filters for pixel work |
| Outbound CAPI worker | Per HTTP dispatch | < 10 s today (`postback_sender_worker.go`); target < 90 s only if batching changes documented | `capi_meta_event_id_dedup` |
| Outbound CAPI | Async only | Zero tracker blocking | All CAPI slugs |
| Admin snippet API | Handler wall time | < 500 ms p99 (`competitive_backlog.md` admin CRUD) | `pixel_platform_snippet_kit` |
| Static `track.js` asset | CDN/browser cache | Immutable hash filename; no per-request admin round-trip | `pixel_track_js_prod_asset` |

Do not cite `BenchmarkUnifiedFilter_Check_mock` or `BenchmarkFilterFraudBoost` as tracker ingest SLA.

---

## Per-network requirements matrix

| Network | Click URL must carry | Inbound conversion | Outbound (shipped) | Browser pixel (v1 backlog) | Dedup key |
| :--- | :--- | :--- | :--- | :--- | :--- |
| Meta (Facebook/IG) | `fbclid` via template `meta-facebook` | `track.js` or affiliate S2S | `facebook` CAPI | Optional `fbq` snippet | `event_id` browser + server (`capi_meta_event_id_dedup`) |
| Google Ads | `gclid` | `track.js` or S2S | `google` offline | Optional `gtag` snippet | `transaction_id` (`capi_google_transaction_id`) |
| TikTok | `ttclid` | `track.js` or S2S | `tiktok` Events API | Optional TikTok Pixel snippet | `event_id` (server uses `click_id` today) |
| Taboola | `tblci` | `track.js` or S2S | `taboola` S2S GET | Not required | N/A |
| Outbrain | `ob_click_id` | `track.js` or S2S | `outbrain` S2S GET | Not required | N/A |
| Microsoft Ads | `msclkid` | `track.js` or S2S | `microsoft_ads` offline | Optional UET tag (`account|customer|goal|UET_TAG_ID`) | `conversion_time` + `msclkid` + shared `event_id` |
| Affiliate / CRM | `click_id` | Inbound S2S `POST /track` | `webhook` | Not required | Partner-defined |

---

## Summary

| Slug | Priority | Outcome | Rough surface | Status |
| :--- | :--- | :--- | :--- | :--- |
| `pixel_track_js_prod_asset` | pixel_p0 | Production `track.js` URL in snippets (not `/src/static/`) | `web/`, admin static embed, Integration tab | Shipped |
| `track_js_event_id_contract` | pixel_p0 | Stable `event_id` from LP through `/track` to CAPI | `track.js`, ingestion parser, postback payload | Shipped |
| `capi_meta_event_id_dedup` | pixel_p0 | Meta CAPI sends `event_id` for browser/server dedup | `provider_facebook.go` | Shipped |
| `pixel_click_id_validation_ui` | pixel_p0 | Dry-run / test dispatch warns on missing network click ids | postback test handler, admin UI | Shipped |
| `pixel_platform_snippet_kit` | pixel_p1 | Copy-paste Meta/Google/TikTok JS with shared `event_id` | Integration tab, helper module | Shipped |
| `pixel_lander_editor_embed` | pixel_p1 | Lander editor blocks for tracker + optional network pixels | `lander_editor_page.tsx` | Shipped |
| `pixel_capi_operator_runbook` | pixel_p1 | Per-network setup steps in `docs/INTEGRATIONS.md` | docs only + cross-links | Shipped |
| `capi_google_transaction_id` | pixel_p2 | Google offline `transaction_id` for dedup | `provider_google.go` | Shipped |
| `pixel_pageview_funnel_optional` | pixel_p2 | Documented PageView via `type: impression` + optional snippet | docs + Integration examples | Shipped |

---

## `pixel_track_js_prod_asset`

**Priority:** pixel_p0

**Gap:** Integration tab snippet imports `/src/static/track.js` (dev path). Production landers need a stable URL served from admin/tracker static embed (`traffic.mdc`: host `src/static/track.js` in admin dist).

**Current state:** `buildDirectTrackSnippet` in `web/src/static/track.js` hardcodes module import path unsuitable for buyer LPs. `TRACK_CORS_ORIGINS` documented but not validated in UI.

**Target:** Snippet references `/static/track.js` (or hashed build output) on tracker or admin CDN origin; module or IIFE build that does not require Vite dev server.

### Implementation

1. **Build:** add `track.js` to admin `npm run build` static output (same pattern as other shipped static assets). Export name stays `trackEvent` (`anti-slop.mdc` product-prefixed embed ban).

2. **Tracker static route:** verify `TestAdminStaticRoutes` or tracker static handler serves built file; document canonical URL in Integration tab.

3. **Snippet generator:** update `buildDirectTrackSnippet(trackURL, campaignId)` to emit production import or script tag (no `/src/` path).

4. **CORS:** Integration tab checklist field: operator must add LP origin to `TRACK_CORS_ORIGINS` (comma-separated per `.env.example`).

5. **Tests:**
   - `web/src/static/track.test.ts` - snippet does not contain `/src/static/`
   - `internal/controlplane` static route test if new mount path

### SLA and latency

| Surface | Budget |
| :--- | :--- |
| `POST /track` from `fetch(keepalive)` | Same as ingest SLA; no extra server work |
| Static asset | Cacheable; zero tracker compute on serve |

### Anti-slop gates

- [x] No filename `ad-event-processor-track.js` in shipped snippets
- [x] No claim that CORS is optional when LP origin differs from tracker

### Done gates

- [x] Snippet copy in UI uses production URL
- [x] `cd web && npm run typecheck`
- [x] `bash scripts/ci/admin_web.sh`
- [x] `docs/INTEGRATIONS.md` zero-redirect section updated

---

## `track_js_event_id_contract`

**Priority:** pixel_p0

**Gap:** Meta and TikTok deduplicate browser + server events via `event_id`. TikTok adapter sets `EventID` from `click_id` (`provider_tiktok.go`); `/track` and `track.js` do not accept a dedicated `event_id`; Meta CAPI omits `event_id` (`provider_facebook.go`).

**Current state:** Operators cannot align browser pixel `eventID` with server CAPI without manual coordination.

**Target:** Single id generated on LP (or supplied by affiliate S2S), stored on conversion payload, forwarded to all outbound providers that support dedup.

### Implementation

1. **`track.js`:** optional `eventId` in `trackEvent(opts)`; default `crypto.randomUUID()` when conversion fires and no id passed.

2. **Ingest:** accept `event_id` in native JSON `/track` body (DFA field or payload map); persist in event payload for settlement.

3. **`PostbackPayload`:** add `EventID string` `json:"event_id,omitempty"`; merge in `mergeEventPayloadInto` (`conversion_outbox.go`).

4. **Fallback chain:** `event_id` -> `tx_id` -> `click_id` (document order; never empty for conversion postbacks when click_id present).

5. **Tests:**
   - `internal/ingestion/subids_test.go` or new test: `event_id` roundtrip in JSON body
   - `internal/postback/conversion_outbox_test.go`: payload carries `EventID`
   - `web/src/static/track.test.ts`: default UUID when omitted

### SLA and latency

| Surface | Budget |
| :--- | :--- |
| Hot `/track` parse | One extra string field; 0 alloc regression - run `make test-alloc-gate` if parser touched |

### Anti-slop gates

- [x] No sync external call to generate id on server hot path
- [x] Do not reuse `click_id` as sole dedup key when browser pixel fires before click_id is known (document UUID default)

### Done gates

- [x] `go test ./internal/postback/... ./internal/ingestion/... -count=1 -short`
- [x] OpenAPI/postback DTO updated if test dispatch accepts `event_id`

---

## `capi_meta_event_id_dedup`

**Priority:** pixel_p0

**Gap:** Meta Conversions API supports `event_id` for deduplication with browser pixel. `FacebookEvent` struct has no field; duplicates inflate reporting when both browser and server fire.

**Current state:** CAPI sends `event_name`, `user_data.fbc`, `custom_data` only.

**Target:** Populate `event_id` on Meta payload from `PostbackPayload.EventID` (after `track_js_event_id_contract`).

### Implementation

1. **`provider_facebook.go`:** add `EventID string` `json:"event_id,omitempty"` to `FacebookEvent`; set from payload fallback chain.

2. **Dry-run / test:** `provider_capi_payload_test.go` asserts `event_id` present when payload has it.

3. **httptest:** extend `TestFacebookCAPI_Payload` and `scripts/test/capi_meta_bootstrap.sh` smoke.

4. **Docs:** Meta setup step: same `eventID` in `fbq('track', ..., {eventID})` and server CAPI.

### SLA and latency

| Surface | Budget |
| :--- | :--- |
| Postback worker | JSON marshal only; no extra HTTP round-trip |

### Anti-slop gates

- [x] Do not document "guaranteed dedup" without test_event_code / Events Manager verification steps
- [x] `test_event_code` path unchanged (`supportsTestEventCode` in admin UI)

### Done gates

- [x] `go test ./internal/postback/... -run Facebook -count=1`
- [x] Holdout: payload with `event_id` -> Graph body contains same id

---

## `pixel_click_id_validation_ui`

**Priority:** pixel_p0

**Gap:** CAPI to Meta/Google/TikTok fails silently or at network when `fbclid`/`gclid`/`ttclid` missing (`postback_provider_ui.ts` blurbs state requirement; no preflight).

**Current state:** `POST /api/v1/postbacks/config/{id}/test` dry-run exists; fraud integrations health lists configured providers.

**Target:** Test dispatch and Integration tab show **actionable warnings** when provider needs a click id not present in dry-run fixture.

### Implementation

1. **`internal/postback/dry_run.go`:** extend `DryRunResult` with `warnings []string` (e.g. `missing_fbclid_for_meta`).

2. **Provider guards:** facebook requires `FBCLID` or `fbc`-derivable input; google requires `GCLID`; tiktok requires `TTCLID`; taboola `TBLCI`; outbrain `OBClickID`; microsoft `MSCLKID`.

3. **Admin UI** (`campaign_postback_section.tsx`): render warnings below test result; link to Integration click URL builder.

4. **Fraud integrations** (`service_fraud_integrations.go`): optional `misconfigured_reason` when token set but traffic template lacks required macro (read-only check against campaign traffic template id).

5. **Tests:** `postbacks_handlers_test.go` dry-run returns warning when fbclid empty for facebook.

### SLA and latency

| Surface | Budget |
| :--- | :--- |
| Test dispatch | < 500 ms p99; no CH |

### Anti-slop gates

- [x] Warnings are not errors when operator uses S2S-only flow with ids in POST body
- [x] No hardcoded demo KPIs in UI

### Done gates

- [x] `go test ./internal/controlplane/... -run Postback -count=1`
- [x] `bash scripts/ci/admin_web.sh`

---

## `pixel_platform_snippet_kit`

**Priority:** pixel_p1

**Gap:** Operators paste tracker snippet but must hand-write Meta `fbq`, Google `gtag`, TikTok pixel with matching `event_id`. Competitors ship combined tag managers; we ship CAPI only.

**Current state:** Integration tab has lander pixel section only. Postback tab has CAPI credentials separately.

**Target:** Optional second copy block per configured outbound provider: minimal browser tag + shared `event_id` variable tied to `trackEvent` call.

### Implementation

1. **Helper** `web/src/helpers/platform_pixel_snippets.ts`:
   - `buildMetaPixelSnippet(pixelId, eventName, eventIdVar)`
   - `buildGoogleGtagSnippet(conversionId, eventIdVar)` (conversion label from postback template when not URL)
   - `buildTikTokPixelSnippet(pixelCode, eventIdVar)`
   - ASCII-only comments; no secrets (pixel id is public; never emit CAPI token).

2. **Integration tab** (`campaign_tracking_section.tsx`): collapsible "Optional ad network browser tags" when postback config exists for provider.

3. **event_id sync:** generated snippet declares `const conversionEventId = crypto.randomUUID()` once; passes to `trackEvent({ eventId: conversionEventId })` and `fbq('track', ..., { eventID: conversionEventId })`.

4. **Taboola/Outbrain:** show "browser pixel not required" note instead of snippet.

5. **Microsoft Ads:** UET tag id in fourth pipe field (`account|customer|goal|UET_TAG_ID`); `buildMicrosoftUETSnippet` with shared `conversionEventId`.

6. **Tests:** unit tests for snippet strings; no live network calls.

### SLA and latency

| Surface | Budget |
| :--- | :--- |
| Snippet generation | Client-side only; no API |

### Anti-slop gates

- [x] Snippets are copy-paste examples, not auto-injected into hosted landers without operator action
- [x] No `live: true` route without backend if persisting snippet prefs added later

### Done gates

- [x] `cd web && npm run typecheck`
- [x] `bash scripts/ci/check_ui_slop.sh` clean on touched views

---

## `pixel_lander_editor_embed`

**Priority:** pixel_p1

**Gap:** Hosted lander editor has no first-class embed for conversion tracking; buyers using hosted LP must paste snippets manually.

**Current state:** `lander_editor_page.tsx` has no track/pixel blocks.

**Target:** Lander editor "Tracking" panel: insert tracker pixel block; optional toggles for Meta/Google/TikTok blocks when campaign postback config exists.

### Implementation

1. **UI panel:** read campaign postback config via existing API; show `StubBanner` when no write access.

2. **Insert action:** append script block to lander HTML (or designated injection zone if editor model supports it).

3. **Reuse** `platform_pixel_snippets.ts` and `buildDirectTrackSnippet`.

4. **Preview:** disclaimer that CORS and live `TRACK_CORS_ORIGINS` must be set before publish.

5. **Tests:** e2e optional; minimum component test for insert markup.

### SLA and latency

| Surface | Budget |
| :--- | :--- |
| Lander save | Unchanged; snippet is static HTML in lander body |

### Anti-slop gates

- [x] Do not store CAPI tokens in lander HTML
- [x] Saved lander must not show "Saved" before 2xx

### Done gates

- [x] `bash scripts/ci/admin_web.sh`
- [x] Holdout: `lander_tracking_snippet.test.ts` insert-before-`</body>`; optional stack smoke `bash scripts/test/pixel_live_smoke.sh`

---

## `pixel_capi_operator_runbook`

**Priority:** pixel_p1

**Gap:** `docs/INTEGRATIONS.md` lists providers but not end-to-end pixel + CAPI setup per network.

**Target:** Operator runbook section with checklists (no marketing prose per `naming.mdc`).

### Implementation

1. **`docs/INTEGRATIONS.md`** new section "Browser pixel and CAPI setup":
   - Flow diagram (text)
   - Per-network: Click URL macros, LP snippet, CAPI fields, test_event_code, Events Manager verification
   - `TRACK_CORS_ORIGINS` and `OPTIONS /track`
   - Explicit: Taboola/Outbrain = S2S only

2. **Cross-link** from Integration tab guide to doc anchor.

3. **Vendor doc gate:** only if claiming hot-path behavior; otherwise no `antifraud_doc_gate.sh` requirement.

### Done gates

- [x] Doc paths match code (`provider_facebook.go` v19.0 base URL, etc.)
- [x] No `guarantee` / `eliminated` claims (`naming.mdc`)

---

## `capi_google_transaction_id`

**Priority:** pixel_p2

**Gap:** Google offline conversions support `transaction_id` for deduplication with gtag. `GoogleOfflineConversion` sends `gclid` only.

**Target:** Map `PostbackPayload.EventID` (or `tx_id`) to `transaction_id` when set.

### Implementation

1. **`provider_google.go`:** add field to struct; populate from payload.

2. **Tests:** `provider_capi_test.go` or new table test.

### SLA and latency

| Surface | Budget |
| :--- | :--- |
| Postback worker | Marshal only |

### Done gates

- [x] `go test ./internal/postback/... -run Google -count=1`

---

## `pixel_pageview_funnel_optional`

**Priority:** pixel_p2

**Gap:** Buyers running full funnel (PageView -> Lead -> Purchase) may need impression events on LP load; Integration tab shows impression JSON example but no combined snippet.

**Target:** Optional `trackEvent({ type: 'impression' })` on `DOMContentLoaded` in snippet kit; document that outbound CAPI typically maps `click` -> ViewContent (Meta) only when postback `target_event` matches.

### Implementation

1. **`buildPageViewTrackSnippet`** in `track.js` or helper.

2. **Docs:** clarify impression does not debit conversion budget; filter chain same as other events.

3. **Out of scope:** auto-configure Meta PageView CAPI on every impression (cost/noise).

### SLA and latency

| Surface | Budget |
| :--- | :--- |
| Extra `/track` per page load | Same ingest SLA; warn operators on high-volume LPs |

### Done gates

- [x] Example in Integration tab behind "Advanced" disclosure
- [x] No new hot-path filter by default

---

## Explicit non-goals (this backlog)

| Item | Reason |
| :--- | :--- |
| Tracker-hosted Tag Manager | Cold-path scope creep; operators use GTM if needed |
| Auto-install Meta Pixel without operator paste | Policy/consent and LP ownership |
| Browser pixel loaded from tracker domain as proxy | Third-party cookie / bloat / antifraud pixel conflict (`ANTIFRAUD_MARKET_ANALYSIS.md` 7.2) |
| Deduplication without `event_id` contract | False confidence in reporting |
| Embedding CAPI token in LP | Secret leak |

---

## Verification matrix (release)

| Check | Command |
| :--- | :--- |
| Hot path unchanged | `bash scripts/ci/hot_path_static_gate.sh` when ingestion touched |
| Alloc gate | `make test-alloc-gate` when `handler*.go` or parser touched |
| Postback adapters | `go test ./internal/postback/... -count=1` |
| Admin UI | `cd web && npm run typecheck && bash scripts/ci/admin_web.sh` |
| Meta bootstrap smoke | `bash scripts/test/capi_meta_bootstrap.sh` (when Meta adapter or env docs change) |
| UI slop | `bash scripts/ci/check_ui_slop.sh` |
| Legacy naming | `bash scripts/ci/check_no_legacy_naming.sh` |

---

## Verification commands

```bash
go test ./internal/postback/... -count=1 -short
go test ./internal/ingestion/ -run 'Track|event_id' -count=1 -short
cd web && npm test
bash scripts/test/pixel_live_smoke.sh   # optional; requires running tracker stack
```

---

## Suggested ship order

1. `pixel_track_js_prod_asset` + `track_js_event_id_contract` (unblock real LPs)
2. `capi_meta_event_id_dedup` + `pixel_click_id_validation_ui` (Meta path end-to-end)
3. `pixel_platform_snippet_kit` + `pixel_capi_operator_runbook` (operator self-serve)
4. `pixel_lander_editor_embed` (hosted lander UX)
5. `capi_google_transaction_id` + `pixel_pageview_funnel_optional` (polish)
