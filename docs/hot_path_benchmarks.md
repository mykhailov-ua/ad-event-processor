# Hot Path Benchmarks

Measured latency and allocation profile for tracker ingest: wire parsers, body codecs, filters, RTB, redirects, and adversarial inputs. Numbers are from local `go test -benchmem` runs (not production SLA).

**Run date:** 2026-08-04  
**Host:** linux/amd64, Intel Core i5-11400H @ 2.70 GHz  
**Go:** see `go.mod` (1.25+ tree)  
**Settings:** `GOMAXPROCS=1`, `-benchtime=300ms`, `-count=5`, `-cpu=1`  
**Median** of five runs unless noted.

Reproduce:

```bash
export GOMAXPROCS=1
go test -run='^$' -benchmem -benchtime=300ms -count=5 -cpu=1 \
  ./internal/ingestion/... ./internal/rtb/... ./internal/openrtb/... ./pkg/piihash/...
```

CI alloc gate (subset): `bash scripts/test/gate_bench.sh` (`make test-alloc-gate`).

**Gate rule:** hot-path benchmarks in `scripts/test/gate_bench.sh` must report **0 B/op, 0 allocs/op**.

---

## 1. HTTP wire parsers (DFA)

Simulates bytes as they arrive on the gnet socket before business logic.

| Benchmark | Corpus | ns/op | B/op | allocs/op | Notes |
| :--- | :--- | ---: | ---: | ---: | :--- |
| `HTTP1DFA_Happy` | Minimal POST `/track` + 69 B JSON | **67** | 0 | 0 | Smallest valid request |
| `HTTP1DFA_Worst` | **nginx edge corpus** — full X-Forwarded-For, TLS hash, Sec-CH-UA, keep-alive | **539** | 0 | 0 | Typical production `/track` through OpenResty |
| `HTTP2DFA_Happy` | 9-byte DATA frame header | **8.6** | 0 | 0 | Frame header only |
| `HTTP2DFA_Worst` | Full h2c `/track` (HEADERS + DATA) | **123** | 0 | 0 | H2 ingress evaluation path |
| `HTTP3DFA_Happy` | QUIC varint decode | **2.2** | 0 | 0 | Varint primitive |
| `HTTP3DFA_Worst` | H3 HEADERS + DATA `/track` | **57** | 0 | 0 | H3 ingress evaluation path |

`HTTP1DFA_Worst` uses `nginxTrackCorpus` from `handler_http1_fsm_test.go` (same headers edge nginx forwards).

---

## 2. Body parsers (JSON / protobuf / OpenRTB)

| Benchmark | ns/op | B/op | allocs/op | Notes |
| :--- | ---: | ---: | ---: | :--- |
| `TrackRequest_ParseJSONOpt` | **173** | 0 | 0 | Production JSON DFA (`requests_parse_opt.go`) |
| `TrackRequest_ParseJSON` | **172** | 0 | 0 | Standard JSON path |
| `TrackRequest_ParseJSON_Legacy` | **205** | 0 | 0 | Legacy parser (reference) |
| `CompositeRouting_Protobuf` | **155** | 0 | 0 | vtproto `AdEvent` + routing |
| `CompositeRouting_JSON` | **256** | 0 | 0 | JSON composite routing |
| `ParseOpenRTB3FSM` | **306** | 0 | 0 | OpenRTB 3 ingress FSM |
| `ParseOpenRTB26` | **2567** | 0 | 0 | Full bid request decode |
| `ParseOpenRTB26Into_connReuse` | **2470** | 0 | 0 | Decode with conn scratch reuse |
| `ParseOpenRTB26Split_hotOnly` | **2574** | 0 | 0 | Split parse hot slice only |
| `ParseClickQuery` | **209** | 0 | 0 | `GET /click` query DFA |

---

## 3. End-to-end handler (gnet `React`, mock registry)

No Redis on accept path; measures parse + dispatch + pre-built response.

| Benchmark | Scenario | ns/op | B/op | allocs/op |
| :--- | :--- | ---: | ---: | ---: |
| `AdsPacketHandlerProto_NoExtra` | Protobuf accept | **294** | 0 | 0 |
| `AdsPacketHandlerProto` | Protobuf accept (handler_test) | **209** | 0 | 0 |
| `HotPath_AdsPacketHandlerProto_accept` | Same as gated E2E accept | **203** | 0 | 0 |
| `AdsPacketHandlerProto_ExtraBytes` | Protobuf + `extra_bytes` | **313** | 0 | 0 |
| `AdsPacketHandlerProto_ExtraRepeated` | Protobuf + repeated extra keys | **412** | 0 | 0 |
| `HotPath_AdsPacketHandlerProto_reject404` | Campaign not found → 404 | **661** | 8 | 1 |
| `HotPath_AdsPacketHandlerProto_infra503` | Redis/infra error → 503 | **400** | 0 | 0 |
| `ClickRedirectGnet_E2E` | `GET /click` through handler | **577** | 0 | 0 |
| `TgClickRedirectGnet_E2E` | `GET /tg/click` E2E | **699** | 0 | 0 |
| `ClickRedirectExpandMacros` | Macro expansion only | **64** | 0 | 0 |
| `OpenRTB26_exchangeGnet` | Full exchange HTTP path | **16903** | 28122 | 24 |
| `RunOpenRTBExchangeParsed` | Auction core only (no wire) | **2468** | 3992 | 4 |

OpenRTB exchange gnet path still allocates (response buffer growth); auction parse core is 4 allocs/op.

---

## 4. FilterEngine and filters

| Benchmark | ns/op | B/op | allocs/op | Notes |
| :--- | ---: | ---: | ---: | :--- |
| `FilterLicense` | **6.6** | 0 | 0 | License snapshot |
| `GeoFilter` | **28** | 0 | 0 | In-memory country set |
| `GeoFilter_lookupOK` | **40** | 0 | 0 | MaxMind path (mock table) |
| `GeoFilter_lookupError` | **35** | 0 | 0 | Fail-open on lookup error |
| `FraudFilter_DC` | **6.0** | 0 | 0 | Datacenter signal (no block) |
| `FilterFraudBoost` | **105** | 0 | 0 | ML boost snapshot apply |
| `FilterEngine_Check_fraudScoring_noSignals` | **33** | 0 | 0 | Fraud layer idle |
| `FilterEngine_Check_fraudScoring_L2Shadow` | **87** | 0 | 0 | L2 shadow scoring |
| `FilterEngine_Check_fraudScoring_L1Reject` | **116** | 0 | 0 | L1 fraud short-circuit |
| `PlacementBlacklistFilter_miss` | **175** | 96 | 2 | Placement hash miss |
| `PlacementBlacklistFilter_hit` | **179** | 96 | 2 | Placement blocked |
| `DuplicateEventFilter_Check` | **21** | 0 | 0 | Local dedup check |
| `UnifiedFilter_Check` | **801** | 0 | 0 | **mock Redis** — Lua path not exercised |
| `RedisBudgetManager_CheckAndSpend` | **116** | 0 | 0 | Mock client budget math |

### Redis Lua (requires live Redis)

These benchmarks use `testcontainers` Redis (`setupTestRedis`). They **fail without Docker**:

- `BenchmarkLuaScript_Happy` — `budget-fast` / impression fast path
- `BenchmarkLuaScript_Worst` — `unified-filter` with TTC + fcap + pacing
- `BenchmarkUnifiedFilter_Check_RealRedis`
- `BenchmarkUnifiedFilter_Check_FastPath_RealRedis`
- `BenchmarkUnifiedFilter_Check_QuotaMode`

Run with Docker available:

```bash
go test -run='^$' -bench='Benchmark(LuaScript|UnifiedFilter_Check_.*Redis)' -benchmem ./internal/ingestion/ -count=3
```

---

## 5. Local quanta (zero-RTT budget path)

| Benchmark | ns/op | B/op | allocs/op |
| :--- | ---: | ---: | ---: |
| `LocalQuantaSpend` | **16.6** | 0 | 0 |
| `LocalQuantaSpend_parallel` | **16.7** | 0 | 0 |
| `AcceptLocalQuantaFullSkip` | **269** | 5 | 0 |
| `LocalQuanta_FullSkip` | **648** | 0 | 0 |

`LocalQuantaSpend` ~13 ns/op is the design target for CAS debit (`TrySpendDebit`).

---

## 6. In-process RTB (`internal/rtb`)

| Benchmark | ns/op | B/op | allocs/op | Catalog |
| :--- | ---: | ---: | ---: | :--- |
| `Auction` | **42.9** | 0 | 0 | ~500 candidates, sparse |
| `Auction_highDensity` | **91.2** | 0 | 0 | Dense bucket |
| `RunAuction_MultiCreative` | **49.7** | 0 | 0 | Multi-creative winner |
| `RunAuction_daypartGate` | **49.4** | 0 | 0 | Daypart gate in scan |
| `RunAuction_freqCapGate` | **49.4** | 0 | 0 | Freq cap gate in scan |

---

## 7. OpenRTB encode (`internal/openrtb`)

| Benchmark | ns/op | B/op | allocs/op |
| :--- | ---: | ---: | ---: |
| `AppendBidResponse` | **258** | 0 | 0 |
| `WriteOpenRTB26BidHTTP` | **253** | 0 | 0 |

---

## 8. PII hashing (`pkg/piihash`)

| Benchmark | ns/op | B/op | allocs/op |
| :--- | ---: | ---: | ---: |
| `PIIHash_singleIP` | **64.5** | 0 | 0 |
| `PIIHash_batch1000` | **199190** (~199 ns/IP) | 0 | 0 |

---

## 9. Ingress control & key formatting

| Benchmark | ns/op | B/op | allocs/op | Notes |
| :--- | ---: | ---: | ---: | :--- |
| `IngressQuota_padded` | **7.1** | 0 | 0 | Padded quota cell |
| `IngressQuota_unpadded` | **8.7** | 0 | 0 | False-sharing reference |
| `UDPControl_ApplyPacket` | **159** | 16 | 1 | UDP ingress gate |
| `UDPControl_ShardLimitRPS` | **0.8** | 0 | 0 | Per-shard RPS check |
| `IPRateLimiter_Check` | **98** | 16 | 1 | Lua script bench helper (not prod FilterEngine) |
| `KeyFormatting_impTSKey` | **37.9** | 0 | 0 | TTC Redis key |
| `KeyFormatting_IPRateLimiter` | **14.5** | 0 | 0 | Rate-limit key stack buffer |
| `KeyFormatting_DuplicateEventFilter` | **16.2** | 0 | 0 | Dedup key |

---

## 10. Registry & hot-path primitives

| Benchmark | ns/op | B/op | allocs/op |
| :--- | ---: | ---: | ---: |
| `Registry_GetCampaignWorker_hot` | **2.3** | 0 | 0 |
| `Registry_GetCampaign_mapLookup` | **10.9** | 0 | 0 |
| `HotPath_monotonicNano` | **17.0** | 0 | 0 |
| `HotPath_cachedTimeUTC` | **1.4** | 0 | 0 |
| `HotPath_filterEngineDeadlineCheck` | **21.5** | 0 | 0 |
| `HotPath_filterEngineCheck_noTimeout` | **31.4** | 0 | 0 |
| `HotPath_filterEngineCheck_withDeadline` | **63.3** | 0 | 0 |
| `HotPath_latencyRingRecord` | **25.6** | 0 | 0 |
| `HotPath_counterInc` | **4.9** | 0 | 0 |
| `HotPath_timeNow` | **30.9** | 0 | 0 | Wall clock (avoid on hot path) |

---

## 11. Audit log sampling (side path)

| Benchmark | ns/op | B/op | allocs/op |
| :--- | ---: | ---: | ---: |
| `Handler_auditLog_impression_sampled` | **12** | 0 | 0 |
| `Handler_auditLog_click_always` | **467** | 15 | 0 |
| `Handler_auditLog_impression_unsampled` | **614** | 19 | 0 |

---

## 12. Malformed wire & adversarial traffic

No heap growth on parser reject paths; fault tests assert **zero panics** across the corpus.

### 12.1 HTTP/1 malformed corpus (`TestFault_HTTP1_MalformedCorpus`)

`parseHTTP1` on ~30 cases: binary garbage, null bytes in method/headers, invalid Content-Length, pipelined valid+garbage, chunked empty body, method overflow, CRLF injection in headers, etc. All return typed errors (`errInvalidRequest`, `errIncompleteRequest`, `errPayloadTooLarge`) — **no panic**.

| Pattern | Expected error |
| :--- | :--- |
| Empty / binary garbage / null in wire | `errInvalidRequest` |
| Body shorter than CL | `errIncompleteRequest` |
| CL > maxBody (1024 in test) | `errPayloadTooLarge` |
| Valid nginx corpus + 64 B garbage tail | Parses first request OK (`pipelined_valid_then_garbage`) |

### 12.2 JSON depth bomb

| Test | Result |
| :--- | :--- |
| `ParseTrackRequestJSON` depth 1000 nested | Reject `< 1 µs`, `ErrMalformed` |
| `MaxJSONDepth` = 16 (tracker), `OrtbMaxJSONDepth` = 32 | Deeper nesting → 400, 0 allocs on reject path |

### 12.3 Handler reject latency (simulated bad traffic)

| Path | ns/op | vs accept | allocs/op |
| :--- | ---: | :--- | ---: |
| Accept (`HotPath_AdsPacketHandlerProto_accept`) | **203** | 1× | 0 |
| 404 campaign not found | **661** | ~3.3× | 1 |
| 503 infra / breaker | **400** | ~2× | 0 |

Reject paths use pre-built `filterRejectSpecs` bodies; 404 has one small alloc for response assembly.

### 12.4 HTTP/2 hostile client

`TestFault_H2HostileIncompleteDisconnect`: partial preface repeated `H2_INCOMPLETE_MAX` (3) times → `gnet.Close`, metric `ad_h2_hostile_disconnect_total`. Prevents slowloris-style preface spin.

### 12.5 DDoS / flood simulation (unit scope)

| Component | Benchmark | ns/op | Behavior under flood |
| :--- | :--- | ---: | :--- |
| Edge XDP pass SYN | `BenchmarkXDP_passSYN` | requires BPF | Drop/blacklist at NIC (see §13) |
| UDP ingress gate | `UDPControl_ApplyPacket` | **159** | Per-packet shard limit |
| Ingress quota | `IngressQuota_padded` | **7.1** | Per-worker op budget |
| IP rate limit (Lua) | `IPRateLimiter_Check` | **98** | Edge/nginx primary RL; script exists for parity tests |

Full-stack flood validation: `ESPX_BPF_PROBE=1 bash scripts/test/malformed.sh` (loadgen + optional BPF session). Not covered by micro-benchmarks alone.

---

## 13. eBPF/XDP (`internal/edge/bpf`)

Requires compiled `edge_filter.o` and BPF loader. Skipped when BPF unavailable (`go test` without root/BTF).

```bash
bash scripts/dev/bpf_setup.sh
go test -run='^$' -bench='BenchmarkXDP_' -benchmem ./internal/edge/bpf/ -count=5
```

| Benchmark | Scenario |
| :--- | :--- |
| `XDP_passSYN` | Legitimate SYN to tracker port |
| `XDP_passSYN_noFingerprint` | SYN, fingerprint disabled |
| `XDP_dropBlocklist` | SYN from blocklisted IP |
| `XDP_passPPSACK` | ACK after handshake |
| `XDP_dropAnomaly` | TCP flag anomaly |
| `XDP_dropNonTCP` | UDP to tracker port |

Target: XDP decision p99 &lt; 10 µs vs tracker SLA (see `edge.mdc`).

---

## 14. Summary — typical `/track` budget stack (no Redis RTT)

Approximate additive cost for nginx-shaped JSON protobuf accept (mock registry, no Lua):

| Stage | ns/op (median) |
| :--- | ---: |
| HTTP/1 parse (nginx corpus) | 539 |
| Protobuf body + handler accept | 203 |
| Geo + license + signals (order varies) | ~50–150 |
| Local quanta or full-skip | 16–650 |
| RTB `Auction` (if `RTB_MODE=live`) | +43–91 |

**Redis Lua** adds one network RTT (production p99 target &lt; 10 ms per shard, not measured in these unit benches).

---

## 15. Related

- Alloc gate: `scripts/test/gate_bench.sh`, `make test-alloc-gate`
- BCE objdump gate: `TestBCEAudit_hotSymbolsNoPanicIndexInMainBody` in `bce_audit_test.go`
- Fault corpora: `handler_http1_fsm_fault_test.go`, `ingress_security_fault_test.go`
- Architecture: [ARCHITECTURE.md](ARCHITECTURE.md)
