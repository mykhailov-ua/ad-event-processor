# RTB production runbook (OpenRTB 2.6)

Shadow → live for `POST /openrtb/bid`. Display + video, single-imp, PMP deals, reconcile. Not OpenRTB 3.0 / CTV / DOOH.

**Prereqs:** tracker `RTB_MODE=shadow|live`, `CH_DSN`, processor + CH migrations; control `/api/v1` with `rtb:read`/`rtb:write`; partner → `https://<tracking-domain>/openrtb/bid`. Env: `deploy/rtb/env.example`.

## Steps

**1. Profile**

```bash
curl -sS -H "Authorization: Bearer $TOKEN" \
  https://control.example/api/v1/rtb/integration-profile | jq .
```

**2. Lint**

```bash
curl -sS -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  --data-binary @internal/openrtb/testdata/bid_request_min.json \
  https://control.example/api/v1/rtb/validate-bid-request | jq .
```

**3. Shadow soak** — `RTB_MODE=shadow`; traffic to `/openrtb/bid`. Watch `ad_rtb_exchange_request_total`, `ad_rtb_shadow_winner_mismatch_total`.

```bash
curl -sS -H "Authorization: Bearer $TOKEN" \
  'https://control.example/api/v1/rtb/shadow-diff?window=1h' | jq .
```

Gate: low mismatch rate, stable parity, no PG budget violations.

**4. Deals** — `POST /api/v1/rtb/deals`; floors: `POST /api/v1/rtb/floors/apply`.

**5. Live** — `RTB_MODE=live`; `RTB_BUDGET_AUTHORITY=redis`; set `RTB_EXCHANGE_MAX_QPS`, `RTB_EXCHANGE_NO_BID_MODE` (`204`|`nbr`); optional `RTB_PREBID_IVT=true`. Verify `x-openrtb-version: 2.6`. E2E: `go test ./tests/e2e/... -run 'RtbLive|OpenRTB26'`.

**6. Reconcile**

```bash
curl -sS -H "Authorization: Bearer $TOKEN" \
  'https://control.example/api/v1/rtb/reconcile/export?window=24h&request_id=PARTNER-BID-ID' | jq .
```

**7. Ops** — `GET /api/v1/ops/doctor` → `rtb_config`; `ad_rtb_exchange_validate_errors_total`; `CH_JANITOR_ENABLED=true`.

## Rollback

`RTB_MODE=shadow|off` → drain → no-bid. Investigate reconcile + shadow-diff before re-enable.

Deferred: OpenRTB 3.0, multi-imp &gt;1, gzip, async `lurl` — `OPENRTB-FULL.md`.
