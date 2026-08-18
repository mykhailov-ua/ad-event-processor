# BENCHMARKS

**Naming:** **ad-event-processor** stack — [NAMING.md](NAMING.md).

Laptop numbers only. Not production SLA (`platform-sla.mdc`: tracker p95 &lt; 50 ms, p99 &lt; 80 ms; Redis Lua p99 &lt; 10 ms/shard).

## Hardware

| | |
| :--- | :--- |
| Host | linux/amd64, Asus TUF, Ubuntu 24.04.3 LTS |
| Kernel | 6.17.0-35-generic, cgroup v2 |
| CPU | Intel Core i5-11400H @ 2.70 GHz (base), turbo 4.5 GHz |
| Topology | 6C/12T, 1 socket, NUMA node0 |
| Cache | L1d 288 KiB, L1i 192 KiB, L2 7.5 MiB, L3 12 MiB |
| RAM | 16 GiB |
| Go | 1.25.x (`go.mod` tree) |

---

## A. Hot-path microbenchmarks (`go test -bench`)

**Date:** 2026-08-04  
**Conditions:** `GOMAXPROCS=1`, `-benchtime=300ms`, `-count=5`, `-cpu=1`; median of five runs. No live Redis unless noted.  
**Gate:** benches in `scripts/test/gate_bench.sh` → **0 B/op, 0 allocs/op** (`make test-alloc-gate`).

```bash
export GOMAXPROCS=1
go test -run='^$' -benchmem -benchtime=300ms -count=5 -cpu=1 \
  ./internal/ingestion/... ./internal/rtb/... ./internal/openrtb/... ./pkg/piihash/...
```

### A.1 HTTP wire (DFA)

| Benchmark | Corpus | ns/op | B/op | allocs/op |
| :--- | :--- | ---: | ---: | ---: |
| `HTTP1DFA_Happy` | Minimal POST `/track` + 69 B JSON | 67 | 0 | 0 |
| `HTTP1DFA_Worst` | nginx edge corpus (XFF, TLS hash, Sec-CH-UA) | 539 | 0 | 0 |
| `HTTP2DFA_Happy` | 9-byte DATA frame header | 8.6 | 0 | 0 |
| `HTTP2DFA_Worst` | h2c HEADERS+DATA `/track` | 123 | 0 | 0 |
| `HTTP3DFA_Happy` | QUIC varint | 2.2 | 0 | 0 |
| `HTTP3DFA_Worst` | H3 HEADERS+DATA `/track` | 57 | 0 | 0 |

### A.2 Body parsers

| Benchmark | ns/op | B/op | allocs/op |
| :--- | ---: | ---: | ---: |
| `TrackRequest_ParseJSONOpt` | 173 | 0 | 0 |
| `TrackRequest_ParseJSON` | 172 | 0 | 0 |
| `TrackRequest_ParseJSON_Legacy` | 205 | 0 | 0 |
| `CompositeRouting_Protobuf` | 155 | 0 | 0 |
| `CompositeRouting_JSON` | 256 | 0 | 0 |
| `ParseOpenRTB3FSM` | 306 | 0 | 0 |
| `ParseOpenRTB26` | 2567 | 0 | 0 |
| `ParseOpenRTB26Into_connReuse` | 2470 | 0 | 0 |
| `ParseOpenRTB26Split_hotOnly` | 2574 | 0 | 0 |
| `ParseClickQuery` | 209 | 0 | 0 |

### A.3 gnet handler E2E (mock registry, no Redis RTT)

| Benchmark | ns/op | B/op | allocs/op |
| :--- | ---: | ---: | ---: |
| `AdsPacketHandlerProto_NoExtra` | 294 | 0 | 0 |
| `AdsPacketHandlerProto` | 209 | 0 | 0 |
| `HotPath_AdsPacketHandlerProto_accept` | 203 | 0 | 0 |
| `AdsPacketHandlerProto_ExtraBytes` | 313 | 0 | 0 |
| `AdsPacketHandlerProto_ExtraRepeated` | 412 | 0 | 0 |
| `HotPath_AdsPacketHandlerProto_reject404` | 661 | 8 | 1 |
| `HotPath_AdsPacketHandlerProto_infra503` | 400 | 0 | 0 |
| `ClickRedirectGnet_E2E` | 577 | 0 | 0 |
| `TgClickRedirectGnet_E2E` | 699 | 0 | 0 |
| `ClickRedirectExpandMacros` | 64 | 0 | 0 |
| `OpenRTB26_exchangeGnet` | 16903 | 28122 | 24 |
| `RunOpenRTBExchangeParsed` | 2468 | 3992 | 4 |

### A.4 FilterEngine

| Benchmark | ns/op | B/op | allocs/op |
| :--- | ---: | ---: | ---: |
| `FilterLicense` | 6.6 | 0 | 0 |
| `GeoFilter` | 28 | 0 | 0 | harness `geo_mock_provider`; not SLA evidence |

### A.4a Broker producer (`BenchmarkTrackerToBroker`)

**Date:** 2026-08-09  
**Conditions:** mock broker client (no network RTT). Bench: `GOMAXPROCS=12`, `-benchmem`, `-count=3`. Latency SLA: `TestBrokerProducer_LatencySLA`, 10k samples after 2k warmup.

| Benchmark / gate | ns/op | B/op | allocs/op | p99 |
| :--- | ---: | ---: | ---: | ---: |
| `BenchmarkTrackerToBroker` | 112–113 | 0 | 0 | — |
| `TestBrokerProducer_LatencySLA` | p50 ~196 ns | 0 | 0 | **&lt; 1 µs** (lab ~483 ns) |

Included in `make test-alloc-gate` (`BrokerProducer_*` + `BenchmarkTrackerToBroker`).

```bash
go test -run=TestBrokerProducer_LatencySLA -v ./internal/ingestion/
go test -run='^$' -bench=BenchmarkTrackerToBroker -benchmem -count=3 ./internal/ingestion/
```

### A.5 FilterEngine (continued)

| Benchmark | ns/op | B/op | allocs/op |
| :--- | ---: | ---: | ---: |
| `GeoFilter_lookupError` | 35 | 0 | 0 | harness `geo_mock_provider` |
| `GeoFilter_MaxMindCountry` | — | — | — | harness `maxmind_mmdb_fixture`; skips without `.mmdb` |
| `FraudFilter_DC` | 6.0 | 0 | 0 |
| `FilterFraudBoost` | 105 | 0 | 0 | hot-path boost snapshot (`fraud_boost_snapshot_mock`); not LGBM |
| `FilterEngine_Check_fraudScoring_noSignals` | 33 | 0 | 0 | synthetic `fraudSignalsFilter`; not ML |
| `FilterEngine_Check_fraudScoring_L2Shadow` | 87 | 0 | 0 | synthetic signals; not ML |
| `FilterEngine_Check_fraudScoring_L1Reject` | 116 | 0 | 0 | synthetic signals; not ML |
| `PlacementBlacklistFilter_miss` | 175 | 96 | 2 | harness `placement_blacklist_mock_redis`; not SLA evidence |
| `PlacementBlacklistFilter_hit` | 179 | 96 | 2 | harness `placement_blacklist_mock_redis` |
| `DuplicateEventFilter_Check` | 21 | 0 | 0 |
| `UnifiedFilter_Check` (mock Redis) | 801 | 0 | 0 |
| `RedisBudgetManager_CheckAndSpend` | 116 | 0 | 0 |

Live Redis (Docker/testcontainers; fail without): `BenchmarkLuaScript_*`, `BenchmarkUnifiedFilter_Check_*Redis`, `BenchmarkUnifiedFilter_Check_QuotaMode`.

```bash
go test -run='^$' -bench='Benchmark(LuaScript|UnifiedFilter_Check_.*Redis)' -benchmem ./internal/ingestion/ -count=3
```

### A.5 Local quanta

| Benchmark | ns/op | B/op | allocs/op |
| :--- | ---: | ---: | ---: |
| `LocalQuantaSpend` | 16.6 | 0 | 0 |
| `LocalQuantaSpend_parallel` | 16.7 | 0 | 0 |
| `AcceptLocalQuantaFullSkip` | 269 | 5 | 0 |
| `LocalQuanta_FullSkip` | 648 | 0 | 0 | harness `local_quanta_noop_redis`; `LOCAL_QUOTA_MODE=live` only |

### A.5b Stream producer (micro only)

No live Redis — `benchStreamProducer()` dials dead address; measures enqueue + admission only.

| Benchmark | Notes |
| :--- | :--- |
| `StreamProducer_Process` | vtproto marshal + queue push |
| `StreamProducer_AdmissionCheck` | `tryAcquireStreamAdmission` reserve path |

```bash
go test ./internal/ingestion/ -run='^$' -bench='BenchmarkStreamProducer_' -benchmem -count=3
```

### A.6 RTB / OpenRTB encode / PII

| Benchmark | ns/op | B/op | allocs/op |
| :--- | ---: | ---: | ---: |
| `Auction` (~500 sparse) | 42.9 | 0 | 0 |
| `Auction_highDensity` | 91.2 | 0 | 0 |
| `RunAuction_MultiCreative` | 49.7 | 0 | 0 |
| `RunAuction_daypartGate` | 49.4 | 0 | 0 |
| `RunAuction_freqCapGate` | 49.4 | 0 | 0 |
| `AppendBidResponse` | 258 | 0 | 0 |
| `WriteOpenRTB26BidHTTP` | 253 | 0 | 0 |
| `PIIHash_singleIP` | 64.5 | 0 | 0 |
| `PIIHash_batch1000` | 199190 (~199 ns/IP) | 0 | 0 |

### A.7 Ingress / keys / registry / primitives

| Benchmark | ns/op | B/op | allocs/op |
| :--- | ---: | ---: | ---: |
| `IngressQuota_padded` | 7.1 | 0 | 0 |
| `IngressQuota_unpadded` | 8.7 | 0 | 0 |
| `UDPControl_ApplyPacket` | 159 | 16 | 1 |
| `UDPControl_ShardLimitRPS` | 0.8 | 0 | 0 |
| `IPRateLimiter_Check` | 98 | 16 | 1 |
| `KeyFormatting_impTSKey` | 37.9 | 0 | 0 |
| `KeyFormatting_IPRateLimiter` | 14.5 | 0 | 0 |
| `KeyFormatting_DuplicateEventFilter` | 16.2 | 0 | 0 |
| `Registry_GetCampaignWorker_hot` | 2.3 | 0 | 0 |
| `Registry_GetCampaign_mapLookup` | 10.9 | 0 | 0 |
| `HotPath_monotonicNano` | 17.0 | 0 | 0 |
| `HotPath_cachedTimeUTC` | 1.4 | 0 | 0 |
| `HotPath_filterEngineDeadlineCheck` | 21.5 | 0 | 0 |
| `HotPath_filterEngineCheck_noTimeout` | 31.4 | 0 | 0 |
| `HotPath_filterEngineCheck_withDeadline` | 63.3 | 0 | 0 |
| `HotPath_latencyRingRecord` | 25.6 | 0 | 0 |
| `HotPath_counterInc` | 4.9 | 0 | 0 |
| `HotPath_timeNow` | 30.9 | 0 | 0 |

### A.8 Audit sampling

| Benchmark | ns/op | B/op | allocs/op |
| :--- | ---: | ---: | ---: |
| `Handler_auditLog_impression_sampled` | 12 | 0 | 0 |
| `Handler_auditLog_click_always` | 467 | 15 | 0 |
| `Handler_auditLog_impression_unsampled` | 614 | 19 | 0 |

### A.10 Cold-path write (`write_path_bench_test.go`)

Mock store only — **not Postgres** (`BenchmarkPostgresStoreBatch_Mock`, harness `mock_event_store`). Postgres truth: `BenchmarkPostgresStoreBatch_integration` (testcontainers) or processor fault tests (`TestFault_ProcessorPgGate_Overflow`, `TestFault_AdsProcessorPGNetworkPartition`). CH outage path: `BenchmarkClickHouseStoreBatch_Spooled`.

| Benchmark | Harness | ns/op (lab) | Notes |
| :--- | :--- | ---: | :--- |
| `PostgresStoreBatch_Mock` | `mock_event_store` | — | Wrapper only; not PG SLA |
| `ClickHouseStoreBatch_Spooled` | `ch_spool_local` | — | Spool when CH insert fails |
| `PostgresStoreBatch_integration` | `postgres_testcontainers` | — | Skipped `-short`; needs Docker |

```bash
go test ./internal/ingestion/ -run='^$' -bench='^BenchmarkPostgresStoreBatch_Mock$' -benchmem -count=1  # mock harness only
bash scripts/test/run_bench.sh 'BenchmarkPostgresStoreBatch_integration' ./internal/ingestion
```

### A.9 Malformed / reject

| Path | Result |
| :--- | :--- |
| HTTP/1 malformed corpus (~30 cases) | typed errors only; no panic |
| JSON depth 1000 | reject &lt; 1 µs, `ErrMalformed` |
| Accept | 203 ns/op, 0 alloc |
| 404 | 661 ns/op, 1 alloc |
| 503 | 400 ns/op, 0 alloc |
| H2 incomplete preface ×3 | `gnet.Close`, `ad_h2_hostile_disconnect_total` |

### A.10 XDP (`internal/edge`)

Requires `edge_filter.o` + root/BTF. Harness `xdp_prog_test`: userspace `prog.Run` only — **not** kernel NIC RX or pinned-attach drops. Kernel proof: `edge-xdp-fault`, `scripts/test/xdp_resilience_drill.sh`, pinned attach drills ([EDGE_XDP.md](enterprise/EDGE_XDP.md)). Lab target: XDP decision p99 &lt; 10 µs on prog test path.

```bash
bash scripts/dev/bpf_setup.sh
go test -run='^$' -bench='BenchmarkXDP_' -benchmem ./internal/edge/ -count=5
```

Benches: `XDP_passSYN`, `XDP_passSYN_noFingerprint`, `XDP_dropBlocklist`, `XDP_passPPSACK`, `XDP_dropAnomaly`, `XDP_dropNonTCP`.

### A.11 Additive `/track` stack (no Redis RTT)

| Stage | ns/op |
| :--- | ---: |
| HTTP/1 nginx corpus | 539 |
| Protobuf accept | 203 |
| Geo + license + signals | ~50–150 |
| Local quanta / full-skip | 16–650 |
| StreamProducer `Process` enqueue | microbench only; no live Redis |
| StreamProducer admission `tryAcquire` | microbench only |
| RTB `Auction` (if live) | +43–91 |

Redis Lua adds one RTT (prod budget &lt; 10 ms/shard; not in unit benches).

---

## B. Purgatory (OS / cgroup / netem torture)

> **Internal engineering notes.** The data below documents OS-level stress behaviour under intentionally hostile cgroup and netem conditions. It is not a product SLA or deployment guide. Operator-facing performance targets are in `docs/ARCHITECTURE.md` (tracker p95 < 50 ms, p99 < 80 ms).

**Date:** 2026-08-07  
**SUT:** compose `bidshard-tracker-0`, `POST http://127.0.0.1:8181/track`  
**Loadgen:** `bin/wrk` 4.2.0, 4 threads, 10 000 keep-alive, 60 s  
**Probe:** `bin/bpf-collector -sample-rate 10` + `deploy/dev/bpf/loadtest_probe.o` (see `scripts/perf/purgatory/run_with_bpf.sh`)  
**Netem (`lo`):** delay 5 ms, loss 1%, dup 1%  
**Sysctl:** r/wmem max 64 KiB; syn/somaxconn 128  
**Cache pollution:** `stress-ng --cache` on CPU 1  
**Artifacts:** `var/purgatory/<ts>/` (gitignored)

```bash
PHASE=survival bash scripts/perf/purgatory/run_with_bpf.sh
```

### B.1 Profile knobs

| Knob | Hostile | Accuracy / critical |
| :--- | :--- | :--- |
| `docker update --cpus` | 0.5 | 1.0 |
| Memory | 256 MiB | 512 MiB |
| `GOMAXPROCS` | 2 (image default) | 1 |
| `LOCAL_QUOTA_MODE` | — | off (critical) |
| Lua hist sample | default | `-1` = 100% (critical) |

### B.2 wrk output (A/B)

| Run | CPUs / RAM / GOMAX | RPS | p50 | p90 | p99 | Timeouts | CFS throttle |
| :--- | :--- | ---: | ---: | ---: | ---: | ---: | ---: |
| `T092800Z` | 0.5 / 256 / 2 | 177 | 66 ms | 171 ms | 574 ms | 106 | ~50% |
| `T093448Z` | 0.5 / 256 / 2 | 160 | 74 ms | 248 ms | 510 ms | 96 | ~75% |
| `T094603Z` accuracy | 1.0 / 512 / 1 | 153 | 46 ms | 279 ms | 1.16 s | 147 | 11% (0.27 s) |
| `T095412Z` critical | 1.0 / 512 / 1 | 123.5 | 51 ms | 313 ms | 1.33 s | 91 | 8% (0.16 s) |

vs SLA: accuracy p99 **1160 ms** (~14× over 80 ms). RPS floor under 1% lo loss ≈ **10²**.

### B.3 Critical path `T095412Z` — Prometheus / Lua

| Metric | Value | Budget |
| :--- | ---: | ---: |
| Handler p95 | 61 ms | &lt; 50 ms |
| Handler p99 | 356 ms | &lt; 80 ms |
| gnet active conn | 10000 | — |
| Worker rejects/s | 0 | 0 |
| Redis Lua p99 shard 1 | 356 ms | &lt; 10 ms |
| Lua full-path n (shard 1) | 7422 | fast=0 |
| Local quanta full-skip | 0 | — |
| Fraud / ingest drops | 0 | 0 |

Lua histogram (shard 1):

| le | count |
| ---: | ---: |
| 10 ms | 1 |
| 25 ms | 4747 |
| 50 ms | 7020 |
| 100 ms | 7128 |
| 250 ms | 7298 |
| 500 ms | 7401 |
| +Inf | 7422 |

Handler p99 ≈ Lua p99 (356 ms). wrk p99 1.33 s = Lua wait + client TCP RTO.

### B.4 CFS / memory

| Snapshot | nr_periods Δ | nr_throttled Δ | throttle % | throttled_usec Δ | memory.current | oom_kill |
| :--- | ---: | ---: | ---: | ---: | ---: | ---: |
| 0.5 vCPU (cumul. after B) | 7055 | 5265 | 74.6% | 515.3 s | ~58 MiB | 0 |
| 1.0 vCPU accuracy 60 s | 637 | 70 | 11.0% | 0.27 s | ~49 MiB | 0 |

### B.5 eBPF (accuracy `T094603Z` / critical `T095412Z`)

| Signal | Value |
| :--- | :--- |
| Loadgen on-CPU | 0.2–2.4% |
| Tracker futex wall | ~42–45% |
| Redis on-CPU (critical) | 0.6% |
| Redis read/write avg | ~13 / ~41 µs |
| Peak tracker-0 FDs | ~10k |
| TCP retrans on lo | present (1% netem) |

### B.6 Causes

| # | Mechanism | Evidence |
| ---: | :--- | :--- |
| 1 | CFS throttle | 0.5 CPU 50–75% periods; 1.0 CPU ~8–11% |
| 2 | TCP RTO under netem loss | RPS stays ~150 after CPU raise; p99 ≥ 0.5–1.3 s |
| 3 | GOMAXPROCS &gt; cgroup CPUs | futex ~42–45%; default GOMAX=2 on sub-CPU quota |

CPU raise alone does not restore RPS. Microbenches (tens–hundreds ns) vs purgatory p99 (10⁸–10⁹×) → bottleneck is scheduler + TCP, not DFA.

### B.7 Artifact index

| Path | Contents |
| :--- | :--- |
| `var/purgatory/20260807T095412Z/` | critical: REPORT, wrk, bpf-report, bottleneck-report, lua_quantiles, metrics |
| `var/purgatory/20260807T094603Z/` | accuracy A/B |
| `var/purgatory/20260807T093448Z/` | hostile 0.5 vCPU + BPF |
| `scripts/perf/purgatory/run_with_bpf.sh` | orchestrator |

---

## C. Redis RAM proof (milestone phase 3)

**Script:** `scripts/perf/redis_ram_proof.sh`  
**Gate:** peak `used_memory` per Redis shard &lt; **300 MiB** under sustained `/track` load.

| Parameter | Default |
| :--- | :--- |
| `TARGET_RPS` | 100000 |
| `DURATION` | 90s |
| `RAM_MAX_BYTES` | 314572800 (300 MiB) |
| Config | `CH_INGEST_SOURCE=broker`, `STREAM_MAX_LEN=10000`, broker profile |

```bash
bash scripts/test/prepare_constrained_stack.sh   # once
TARGET_RPS=100000 bash scripts/perf/redis_ram_proof.sh
# artifacts → var/ram-proof/<ts>/report.json + fault_proof.txt
```

**Status:** lab run **2026-08-09** (`var/ram-proof/20260809T130425Z/`).

| Metric | Result |
| :--- | :--- |
| `target_rps` | 100000 |
| `duration` | 90s |
| `peak_used_memory_bytes_max_shard` | **3_622_240** (~3.5 MiB) |
| `ram_max_bytes_per_shard` | 314_572_800 (300 MiB) |
| `passed` | **true** |

Config: `CH_INGEST_SOURCE=broker`, broker profile, constrained load-test compose (`prepare_constrained_stack.sh`). Loadgen reported many dial errors at 100k RPS on lab host; RAM gate uses Redis `used_memory`, not HTTP success rate.

### C.1 Dual-path vs broker-only cutover (phase 3)

**Script:** `scripts/perf/redis_ram_cutover_compare.sh`  
**Compares:** peak `used_memory` per shard — dual-path (`CH_INGEST_SOURCE=` + `BROKER_SHADOW_MODE=1`) vs broker-only (`CH_INGEST_SOURCE=broker`).

| Parameter | Default |
| :--- | :--- |
| `TARGET_RPS` | 50000 |
| `DURATION` | 45s |

```bash
bash scripts/test/prepare_constrained_stack.sh
bash scripts/perf/redis_ram_cutover_compare.sh
# → var/ram-proof/cutover-compare-<ts>/report.json
```

**Expected (lab):** broker-only peak ≤ dual-path peak (skip main `XADD` on `_ch` reduces stream pressure). Run script locally to populate `dual_path_peak_bytes_max_shard` / `broker_only_peak_bytes_max_shard` in `report.json`.

| Metric | Broker-only (20260809) | Dual-path (same script) |
| :--- | :--- | :--- |
| `peak_used_memory_bytes_max_shard` | **3.6 MiB** (from `redis_ram_proof`) | run `cutover_compare` |
| `STREAM_MAX_LEN` | 10000 + 10s trim | same |

---

## D. Redis UDS transport (milestone phase 5)

**Script:** `scripts/perf/redis_uds_benchmark.sh`  
**Gate:** UDS dial p50 &lt; **5 µs** (stretch **2.5 µs** on governor-tuned host); UDS dial/PING beats TCP loopback.

| Parameter | Default |
| :--- | :--- |
| `UDS_DIAL_P50_BUDGET_NS` | 5000 (5 µs) |
| `UDS_BENCH_REQUESTS` | 100000 (`redis-benchmark -n`) |

```bash
GOMAXPROCS=1 taskset -c 0 bash scripts/perf/redis_uds_benchmark.sh
# artifacts → var/uds-bench/<ts>/report.json + redis-benchmark-*.txt
```

**Status:** lab run **2026-08-09** (`var/uds-bench/20260809T151336Z/`).

| Metric | Result |
| :--- | :--- |
| `uds_dial_p50_us` | **~4.8** |
| `uds_dial_p50_budget_ns` | 5000 |
| `stretch_goal_2_5us_passed` | false (lab) |
| UDS vs TCP dial | UDS faster |
| `passed` | **true** |

Compose/installer defaults: `ad_event_processor_run:/run/ad-event-processor`, `DB_DSN` with `host=/run/ad-event-processor/postgresql`, `REDIS_ADDRS=/run/ad-event-processor/redis/*.sock`.

---

## E. NOSCRIPT storm (milestone phase 8 / scenario E)

**CI unit gates** (`scripts/test/run_resilience.sh` → `TestFault_*`):

| Test | What it proves |
| :--- | :--- |
| `TestFault_NOSCRIPTStorm` | 24 workers × 50 checks; SCRIPT FLUSH; ≥80% success; `ad_redis_lua_noscript_total` &gt; 0 |
| `TestFault_ScriptFlushUnderTrackRPS` | 16×100 HTTP `/track`; p99 **&lt; 80 ms**; budget invariant |

**Compose drill:** `scripts/fault/noscript_compose_drill.sh` — **20k RPS**, SCRIPT FLUSH @ 10s, ≥80% `202`, `fault_proof fault=noscript_storm`.

| Parameter | Default |
| :--- | :--- |
| `NOSCRIPT_DRILL_RPS` | 20000 |
| `NOSCRIPT_P99_BUDGET_MS` | 80 |

**Reconnect preload:** `UnifiedFilter.AttachReconnectPreload()` (tracker) — `SCRIPT LOAD` on new pooled dial, debounced 1s/shard (`internal/ingestion/redis_lua.go`).

---

## F. Stream trimmer vs PEL (phase 3 CI smoke)

**Test:** `TestRedisStreamTrimmer_PELPendingNotInflated` (`go test ./internal/ingestion/ -run PELPending`)

Proves aggressive `XTrimMaxLenApprox` caps stream length while in-flight consumer reads remain **ackable** (no stuck PEL growth from trimmer alone).

```bash
go test -count=1 ./internal/ingestion/ -run TestRedisStreamTrimmer_PELPendingNotInflated
```

---

## G. CH spool async (milestone phase 9)

**CI unit gates:** `TestFault_CHSpoolDiskBlock`, `TestFault_CHSpoolStressNgNoDState` (`linux` + `stress-ng`; installed in CI via `run_resilience.sh`).

| Test | What it proves |
| :--- | :--- |
| `TestFault_CHSpoolDiskBlock` | 32×40 async appends; no caller blocked &gt; 25 ms |
| `TestFault_CHSpoolStressNgNoDState` | stress-ng HDD on spool dir; 0 D-state threads in process |

**Compose:** `scripts/fault/spool_compose_drill.sh` — pause ClickHouse, loadgen 5k RPS, stress-ng on processor spool volume, tracker D-state = 0.

---

## H. Parser security benches and chaos gates

**Guide:** [PARSER_SECURITY.md](PARSER_SECURITY.md)

Parser security changes must keep **0 allocs/op** on reject paths and pass `scripts/test/gate_bench.sh`. CI medians (linux/amd64, GOMAXPROCS=1) as of 2026-08-09:

| Bench | Median ns/op | B/op | allocs/op | Gate |
| :--- | ---: | ---: | ---: | :--- |
| `HTTP1DFA_Happy` | 67 | 0 | 0 | ≤ 75 |
| `HTTP1DFA_Worst` | 539 | 0 | 0 | ≤ 470 |
| `ParseOpenRTB26` | 2567 | 0 | 0 | ≤ 1600 |
| `AdsPacketHandlerProto_ExtraBytes` | 313 | 0 | 0 | ≤ 320 |
| `HTTP1ChunkedFragmented` | — | 0 | 0 | ≤ 220 |

**Chaos load (PS-G08):** in-process mix of valid protobuf and hostile wire. Pass when control-cohort p99 &lt; **80 ms** and `WorkerPoolRejectTotal` does not increase.

```bash
go test ./internal/ingestion/ -run='TestChaos_ParserLoad_CX02|TestChaos_ParserSecurity_PS_G08' -count=1 -v
bash scripts/fault/parser_chaos_load.sh --duration=8s --rps=3000
bash scripts/fault/parser_chaos_drill.sh
```

**Proof contract:** chaos tests log `fault_proof fault=<name> gap=closed gap_id=PS-Gxx ...` for CI grep.

---

## Related

- Edge cases (ListenOverflow, netem/RTO, code vs OS): [EDGE_CASES.md](EDGE_CASES.md)
- Alloc gate: `scripts/test/gate_bench.sh`, `make test-alloc-gate`
- BCE gate: `TestBCEAudit_hotSymbolsNoPanicIndexInMainBody`
- Fault corpora: `handler_http1_fsm_fault_test.go`, `ingress_security_fault_test.go`, `parser_chaos_load_test.go`
- Parser security: [PARSER_SECURITY.md](PARSER_SECURITY.md)
- Architecture: [ARCHITECTURE.md](ARCHITECTURE.md)
