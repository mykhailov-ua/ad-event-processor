# RTB production runbook (OpenRTB 2.6 SMB)

Shadow → live checklist for `POST /openrtb/bid` on tracker. Scope: display + video, single-imp (default), PMP deals, reconcile export — not OpenRTB 3.0 / CTV pods / DOOH.

## Prerequisites

- Tracker with `RTB_MODE=shadow` or `live`, `CH_DSN` set, processor running (CH migrations applied).
- Control plane `/api/v1` with `rtb:read` / `rtb:write` for operator.
- Partner SSP endpoint points at `https://<tracking-domain>/openrtb/bid`.

Env reference: `deploy/rtb/env.example`, `.env.example` (RTB exchange + CH retention).

## 1. Validate integration profile

```bash
curl -sS -H "Authorization: Bearer $TOKEN" \
  https://control.example/api/v1/rtb/integration-profile | jq .
```

Confirm supported objects match partner contract (§0.2 in `OPENRTB-FULL.md`).

## 2. Lint sample bid requests

```bash
curl -sS -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  --data-binary @internal/openrtb/testdata/bid_request_min.json \
  https://control.example/api/v1/rtb/validate-bid-request | jq .
```

Fix validation errors before sending traffic to tracker.

## 3. Shadow mode soak

1. Set `RTB_MODE=shadow` on tracker; reload.
2. Send partner traffic (or replay fixtures) to `/openrtb/bid`.
3. Watch metrics: `ad_rtb_exchange_request_total`, `ad_rtb_exchange_duration_seconds`, `ad_rtb_shadow_winner_mismatch_total`.
4. Compare shadow vs live gate:

```bash
curl -sS -H "Authorization: Bearer $TOKEN" \
  'https://control.example/api/v1/rtb/shadow-diff?window=1h' | jq .
```

Gate criteria (tune per partner): low `mismatch_rate`, stable `parity_rate`, no budget invariant violations in Postgres.

## 4. Configure deals and floors

- Create PMP deals: `POST /api/v1/rtb/deals` (audited).
- Optional floor optimizer: `POST /api/v1/rtb/floors/apply` (uses `rtb_deal_outcomes` in CH).

## 5. Enable live exchange

1. Set `RTB_MODE=live`, `RTB_BUDGET_AUTHORITY` per your deployment (`redis` default).
2. Set `RTB_EXCHANGE_MAX_QPS` to partner contract (non-zero in production).
3. Set `RTB_EXCHANGE_NO_BID_MODE` (`204` or `nbr`) per SSP.
4. Optional: `RTB_PREBID_IVT=true` rejects datacenter/proxy IPs before auction (same gate as `/track` RTB).
5. Roll one tracker node; verify bids return `x-openrtb-version: 2.6`.
6. Run E2E budget tests: `go test ./tests/e2e/... -run 'RtbLive|OpenRTB26'` (needs Postgres + Redis).

## 6. Reconcile with partner

Export window stats (all requests or single `request.id`):

```bash
curl -sS -H "Authorization: Bearer $TOKEN" \
  'https://control.example/api/v1/rtb/reconcile/export?window=24h&request_id=PARTNER-BID-ID' | jq .
```

Fields: `bids`, `wins`, `spend_micro` from `rtb_exchange_log` (CH). Compare to partner dashboard for the same window.

## 7. Observability and retention

| Check | Command / metric |
|-------|------------------|
| Doctor RTB knobs | `GET /api/v1/ops/doctor` → `rtb_config` |
| Exchange errors | `ad_rtb_exchange_validate_errors_total` |
| QPS throttle | `ad_rtb_exchange_throttle_total` |
| CH janitor | `CH_JANITOR_ENABLED=true`, `CH_JANITOR_INTERVAL_H`, retention days for `rtb_deal_outcomes` / `rtb_exchange_log` |

## Rollback

1. Set `RTB_MODE=shadow` or `off` on tracker.
2. Drain in-flight; partner receives no-bid (`204` or `nbr`).
3. Investigate via reconcile export + shadow-diff before re-enabling live.

## Deferred

OpenRTB 3.0, multi-imp >1, gzip responses, async `lurl` worker — see `OPENRTB-FULL.md`.
