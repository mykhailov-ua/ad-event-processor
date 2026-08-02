# BCE & unsafe audit — `handler_http1_fsm.go`, `tg_click.go`

Audit date: 2026-08-02. Method: `go tool objdump` on `go test -c ./internal/ingestion`, plus
`TestBCEAudit_*` gates in `internal/ingestion/bce_audit_test.go`.

## BCE methodology

Go emits `runtime.panicIndex` / `runtime.panicSlice*` on bounds failures. Those calls live in a
**trailing panic slab** at the end of each function; the hot path branches to them only on fault.

What matters for `/track` and `/tg/click` latency:

1. **Main body** (instructions before the first `CALL runtime.panicIndex`) must not contain
   `panicIndex` — verified by `TestBCEAudit_hotSymbolsNoPanicIndexInMainBody`.
2. **Per-iteration CMP** against `len(buf)` in inner loops should be minimized via:
   - `_ = buf[n-1]` before `for i < n`
   - `_ = buf[i+window-1]` after `i+window <= n` window checks (macro dispatch)

### Changes applied

| Location | Hint / refactor | Effect |
|----------|-------------------|--------|
| `foldKeyU32` / `foldKeyU64` | `_ = key[off+3]`, `_ = key[off+7]` | Removes 8× `panicIndex` from fold helpers; callers already gate `kl` |
| `teValueHasChunked` | `vn := len(val); _ = val[vn-1]` | Single bound for TE token scan |
| `httpPathHasPrefix` | `_ = path[pn-1]` before `path[pl]` | BCE for suffix delimiter check |
| `parseTgQuery` | `_ = path[pn-1]`, `_ = query[qn-1]` | Query/path scan without per-index len reload |
| `matchTgQueryKey` | `_ = key[kl-1]` | BCE for fixed-width key compares |
| `dispatchTgRedirectMacro` | macro length consts + `[N]byte` unsafe compare + window hint | CMPQ BX 29→11 (amd64); 0 `panicIndex` in main body |

### `parseHTTP1` (existing)

Already had `_ = data[n-1]` at entry (line 49). Objdump still lists `panicIndex` in the **panic slab**
for slice expressions (`data[i+1]`, sub-slices) — expected; happy-path header loop uses unchecked
`MOVZX` after explicit `i+1 >= n` guards.

Chunked body parsing (`handler_http1_chunked.go`) shares the `parseHTTP1` symbol tree; slice-cap
panics there are cold-path only.

### Runtime calls — expected on hot path

| Call | Where | Verdict |
|------|-------|---------|
| `runtime.morestack` | Large functions (`parseHTTP1`, `parseTgQuery`) | Cold; grow stack on deep frames |
| `runtime.panicIndex` | Trailing slab only (audited symbols) | OK |
| `runtime.mallocgc` / `newobject` | Not in `foldKey*`, `matchTgQueryKey`, `unsafeString` | OK |

### Re-run audit locally

```bash
go test -c -o /tmp/ingestion.test ./internal/ingestion/
TEST_BINARY=/tmp/ingestion.test go test ./internal/ingestion/ -run BCEAudit -v

# Manual spot-check
go tool objdump -s 'espx/internal/ingestion.dispatchTgRedirectMacro' /tmp/ingestion.test | rg panicIndex
go tool objdump -s 'espx/internal/ingestion.foldKeyU32' /tmp/ingestion.test | rg panicIndex
```

## unsafe.String lifetime audit

### `unsafeString` / `UnsafeString`

```go
// filters.go — zero-copy view; backing []byte must outlive all uses in the current handler frame.
func unsafeString(b []byte) string
```

### `handler_http1_fsm.go` → `parsedHTTPRequest`

| Field | Backing | Lifetime |
|-------|---------|----------|
| `Method`, `Path`, `Body`, headers | gnet ring `data []byte` | ≤ `OnTraffic` for connection |
| Sub-slices passed to filters | same buffer | Must not retain in goroutines without `StringBuffer` copy |

### `tg_click.go`

| Use | Backing | Rule |
|-----|---------|------|
| `parseTgQuery` → `out.clickIDStr`, `bridgeToken`, `subs`, `placementID` | `ctx.wCamp.buf` scratch | **Do not append to scratch** until redirect URL built (`reactTgClick` order enforces this) |
| `fillTgEventFromParsed` → `evt.IP`, `evt.UA`, TLS/CH headers | `req.*` from HTTP parse | gnet frame lifetime |
| `evt.Payload` | `marshalTgBridgePayload` copies into `evt.Payload` | Owned by `domain.Event` |
| `expandTgRedirectMacros` | `UnsafeBytes(string)` | Strings must stay alive during macro expansion only |

### Violations to avoid

1. Storing `unsafeString` views in package-level caches or Redis payloads.
2. `go func()` capturing `evt` before copying strings into `evt.StringBuffer` (settlement path).
3. Reusing `ctx.wCamp.buf` between `parseTgQuery` and `buildTgRedirectLocation` without keeping parsed strings in registers/stack (they alias scratch).

`TestUnsafeAudit_tgClickScratchLifetime` documents the scratch-aliasing invariant.

## Gate integration

- `TestBCEAudit_*` runs with `go test ./internal/ingestion/...`
- Optional CI: set `TEST_BINARY` when objdump-ing a release `tracker` build for parity checks
