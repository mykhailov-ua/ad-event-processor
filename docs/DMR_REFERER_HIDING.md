# Meta Refresh + JS Replace Referer Suppression

**Status:** Shipped (tracker hot path, admin API + UI toggle, migration `00086`).  
**Scope:** `internal/ingestion/dmr_redirect.go`, `click_redirect.go`, `tg_click.go`, `domain_pool_rotation.go`, `internal/controlplane` (`dmr_enabled`), `web/` campaign config  
**Verification:** `go test ./internal/ingestion/ -run 'TestBuildDmr|TestClickRedirectGnet_DMR_|TestClickRedirect_ProxySkippedWhenDMR|TestClickRedirect_DomainRotation_DMR|TestTgClickRedirectGnet_DMR|TestParseDmr' -count=1`

---

## 1. Summary

Traditional `302 Found` redirects can leak the `Referer` header to landing pages. The tracker can instead return HTTP `200 OK` HTML with:

1. `<meta http-equiv="refresh" content="0;url=...">` (no-JS fallback)
2. `<script>window.location.replace("...")</script>` (primary path on JS clients)

Enable via:

- Query: `dmr=1` or `dmr=true` (case-insensitive; `dmr=10` is rejected)
- Campaign: `dmr_enabled` on PATCH `/api/v1/campaigns/{id}` (admin UI checkbox on campaign detail → Config; replicated to tracker registry)

Plain `302` redirects also send `Referrer-Policy: no-referrer`.

**Coverage:** `/click`, `/tg/click`, tracking-domain rotation. Click reverse-proxy is skipped when DMR is active.

**Limits:** No-JS meta-only clients may still send a referrer (browser-dependent). Safe-page in-place HTML and proxy streaming without DMR are out of scope.

Macro values (`{click_id}`, `{sub*}`, etc.) are percent-encoded before landing URL assembly (302 and DMR).

---

## 2. Request flow

```text
  GET /click?campaign_id=...&dmr=1
             │
             ▼
  parseClickQuery → FilterEngine (if configured) → buildRedirectLocation
             │
             ▼
  clickDmrActive(query || campaign.dmr_enabled)
             │
      ┌──────┴──────┐
      │ true        │ false
      ▼             ▼
  BuildDmrResponse   302 + Referrer-Policy: no-referrer
  HTTP 200 HTML
```

---

## 3. Implementation

### 3.1 Buffer assembly

`BuildDmrResponse(dst, url)` writes the full HTTP/1.1 response into `dst` in one pass. Length is computed before write; `dmrGrow` may heap-allocate when `cap(dst)` is insufficient.

`writeGnetClickDmrRedirect` pre-sizes `ctx.bufSlice` to `dmrResponseWireLen(location)` (minimum cap 4096) before calling `BuildDmrResponse`, so steady-state clicks on a warm connection avoid grow on typical landing URLs.

**Harness note:** microbenchmarks use a stack `var buf [4096]byte` or `[8192]byte`. That proves **0 allocs/op for `BuildDmrResponse` only** when `cap(dst)` fits the response. The first DMR response on a connection whose `bufSlice` is smaller than the wire length can still allocate once.

### 3.2 Escaping

HTML attribute (`meta content`) and JS string (`location.replace`) paths escape independently:

| Input | HTML | JS |
| --- | --- | --- |
| `"&'<>` | entities | `\x3c` / `\x3e` / quotes / `/` |
| control chars `< 0x20` | `&#NN;` | `\n` / `\r` |
| U+2028 / U+2029 | `&#8232;` / `&#8233;` | `\u2028` / `\u2029` |

Held-out tests: `TestBuildDmrResponse_scriptBreakout`, `_scriptBreakout_uppercase`, `_htmlControlChars`, `_jsLineSeparators` in `dmr_redirect_test.go`.

Handler wiring (gnet harness, not curl):

| Test | Asserts |
| --- | --- |
| `TestClickRedirectGnet_DMR_queryFlag` | `dmr=1` → 200 HTML + landing in body |
| `TestClickRedirectGnet_DMR_campaignEnabled` | `dmr_enabled` without query |
| `TestClickRedirectGnet_302` | 302 + `Referrer-Policy: no-referrer` on wire |
| `TestClickRedirect_ProxySkippedWhenDMR` | proxy campaign + DMR → HTML, not upstream |
| `TestClickRedirect_DomainRotation_DMR` | banned host rotation + DMR |
| `TestTgClickRedirectGnet_DMR` | `/tg/click` + `dmr=1` |
| `TestWriteGnetClickDmrRedirect_PreSizesConnBuf` | `bufSlice` pre-grow when wire > 4096 B |

Macro encoding: `TestBuildRedirectLocation_encodesMacroValues`, `TestBuildTgRedirectLocation_encodesMacroValues`.

---

## 4. Benchmarks (microbench harness)

Command (re-run after changes; do not trust stale numbers):

```bash
go test -run='^$' -bench='Benchmark(BuildDmrResponse|WriteGnetClickDmrRedirect)' -benchmem -count=3 ./internal/ingestion/
```

Example output (linux/amd64, 2026-08-17, i5-11400H):

```text
BenchmarkBuildDmrResponse_ZeroAlloc-12              ~900-930 ns/op    0 B/op    0 allocs/op
BenchmarkBuildDmrResponse_LongURL-12                ~3100-3300 ns/op   0 B/op    0 allocs/op
BenchmarkWriteGnetClickDmrRedirect_ConnBufCap4096  ~same order; 0 allocs/op after first grow on warm ctx.bufSlice
```

`BenchmarkBuildDmrResponse` and `BenchmarkWriteGnetClickDmrRedirect_ConnBufCap4096` are listed in `scripts/test/gate_bench.sh`. They are **not** a substitute for `make test-alloc-gate` on the full `/click` path.

---

## 5. Local verification

```bash
# Unit + handler harness
go test -v ./internal/ingestion/ -run 'TestBuildDmr|TestClickRedirectGnet_DMR_|TestClickRedirect_ProxySkippedWhenDMR|TestClickRedirect_DomainRotation_DMR|TestTgClickRedirectGnet_DMR|TestParseDmr|TestBuildRedirectLocation_encodes|TestBuildTgRedirectLocation_encodes' -count=1

# Microbench (optional)
go test -run='^$' -bench='Benchmark(BuildDmrResponse|WriteGnetClickDmrRedirect)' -benchmem ./internal/ingestion/

# Admin web (dmr_enabled toggle in campaign detail)
bash scripts/ci/admin_web.sh

# Staging / manual (requires running tracker + seeded campaign)
curl -i "http://localhost:8181/click?campaign_id=<uuid>&dmr=1"
```

Expected: `HTTP/1.1 200 OK`, `Content-Type: text/html`, body contains `meta http-equiv="refresh"` and `window.location.replace`, with landing URL present in the body (not header-only).

Apply migration `internal/ingestion/migrations/00086_campaign_dmr.sql` before using `dmr_enabled` from Postgres.
