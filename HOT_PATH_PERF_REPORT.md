# Hot path performance report

Generated: 2026-08-24T21:55:00Z (UTC)  
Host: linux amd64, Go 1.25.12, CPU Intel i5-11400H @ 2.70GHz  
Settings: `GOMAXPROCS=1`, `benchtime=200ms`, `count=10` (`PERF_GATE_STRICT=true`)

Raw artifacts: `var/hot-path-perf/`

## Executive summary

| Area | Result |
| :--- | :--- |
| Gate benchmarks (`scripts/test/gate_bench.sh`) | PASS (582 lines, ingestion + rtb) |
| Zero-alloc gate tests (`make test-alloc-gate` subset) | PASS |
| Escape heap gate | PASS (`431` lines, baseline `431`) |
| Filter / Lua benches | PASS |
| eBPF 5-minute probe | **RUN** (598.8 s; fallback edge load; see below) |

Microbench numbers below are isolated unit scope. They are **not** production `/track` p95/p99 (use load test + Prometheus for E2E SLO).

---

## Zero-alloc and heap gates

```bash
go test -short -count=1 -run 'ZeroAlloc|zeroAlloc_fraudScoring|FraudScoring_LatencySLA|BrokerProducer|ApplyRtbAuction_shadow_zeroAlloc|RecordRtbShadow|HTTP1Parse|OpenRTB26_Exchange|Check_zeroAlloc_localQuantaFullSkip' ./internal/ingestion/...
# ok ad-event-processor/internal/ingestion 0.145s

bash scripts/ci/escape_heap_gate.sh
# escape-heap-gate: hot_path_heap_escape_lines=431 baseline=431 (50 files)
# escape-heap-gate: OK
```

---

## Core hot-path benchmarks (ns/op, B/op, allocs/op)

SLA references from `core.mdc` / `hot-path.mdc` shown where applicable.

### `/track` ingest and filters

| Benchmark | ns/op | B/op | allocs/op | SLA note |
| :--- | ---: | ---: | ---: | :--- |
| `AdsPacketHandlerProto` | 966 | 712 | 2 | handler microbench only |
| `AdsPacketHandlerProto_NoExtra` | 886 | 712 | 2 | |
| `AdsPacketHandlerProto_ExtraBytes` | 888 | 704 | 1 | |
| `TrackRequest_ParseJSONOpt` | 227 | 0 | 0 | |
| `HTTP1Parse` | 489 | 0 | 0 | |
| `HotPath_filterEngineCheck_noTimeout` | 28.1 | 0 | 0 | |
| `HotPath_filterEngineCheck_withDeadline` | 60.0 | 0 | 0 | |
| `FilterFraudBoost` | 93.1 | 0 | 0 | bench ~90 ns (`core.mdc`) |
| `UnifiedFilter_Check_mock` | 1,205 | 192 | 3 | mock Redis wrapper only |
| `BenchmarkLuaScript_Happy` | 59,640 | 0 | 0 | real Redis testcontainers in CI; mock bench here |

### `/click` and redirect

| Benchmark | ns/op | B/op | allocs/op |
| :--- | ---: | ---: | ---: |
| `ParseClickQuery` | 223 | 0 | 0 |
| `ParseClickQuery30Params` | 1,038 | 0 | 0 |
| `BuildDmrResponse_ZeroAlloc` | 824 | 0 | 0 |
| `TgClickRedirectGnet_E2E` | 766 | 0 | 0 |
| `WriteGnetClickDmrRedirect_ConnBufCap4096` | 51,390 | 8,295 | 1 |

### Geo / TLS / proxy signals

| Benchmark | ns/op | B/op | allocs/op | SLA note |
| :--- | ---: | ---: | ---: | :--- |
| `CIDR_LPM_Lookup_IPv4` | 35.6 | 0 | 0 | geo p99 < 10 us (sampled prod) |
| `CIDR_LPM_Lookup_IPv6` | 5.3 | 0 | 0 | |
| `TLS_Fingerprint_Lookup` | 45.8 | 0 | 0 | |
| `ProxyVPN_Extended_Lookup` | 77.9 | 0 | 0 | |

### RTB

| Benchmark | ns/op | B/op | allocs/op | SLA note |
| :--- | ---: | ---: | ---: | :--- |
| `Auction` (rtb) | 39.8 | 0 | 0 | `RunAuction` p99 < 15 us |
| `RunOpenRTBExchangeParsed` | 2,018 | 3,992 | 4 | |
| `ParseOpenRTB26Split_hotOnly` | 2,242 | 0 | 0 | |
| `OpenRTB26_exchangeGnet` | 14,890 | 26,348 | 24 | gnet E2E fixture |

### Broker / routing / misc

| Benchmark | ns/op | B/op | allocs/op |
| :--- | ---: | ---: | ---: |
| `TrackerToBroker` | 152.7 | 0 | 0 |
| `CompositeRouting_Protobuf` | 139.2 | 0 | 0 |
| `FlowRouter_BanditSelect` | 46.1 | 0 | 0 |
| `HotPath_cachedTimeUTC` | 1.29 | 0 | 0 |

### Cold-adjacent on hot surface (allocating)

| Benchmark | ns/op | B/op | allocs/op |
| :--- | ---: | ---: | ---: |
| `ClickProxy_Stream` | 51,320 | 41,700 | 103 |
| `ClickProxy_BuildUpstreamURL` | 2,272 | 1,184 | 23 |
| `Attestation_VerifyCookie` | 541 | 16 | 1 |

---

## HTTP parser DFA (zero alloc)

| Benchmark | ns/op | B/op | allocs/op |
| :--- | ---: | ---: | ---: |
| `HTTP1DFA_Happy` | 89.6 | 0 | 0 |
| `HTTP2DFA_Happy` | 8.31 | 0 | 0 |
| `HTTP3DFA_Happy` | 2.03 | 0 | 0 |
| `HTTP2DecodeFrame` | 8.19 | 0 | 0 |
| `HTTP3VarintDecode` | 2.22 | 0 | 0 |

---

## eBPF probe (5 minutes)

**Status: completed** (session `var/hot-path-perf/bpf-5min-20260824T220100Z/`).

| Field | Value |
| :--- | :--- |
| Duration | 598.8 s |
| Started | 2026-08-24T22:00:24Z |
| Targets | tracker-0, tracker-1, nginx, redis-0..3 |
| Load | Fallback curl loop to nginx edge `:8000` (~24,216 requests); `loadgen` could not reach trackers on host `8181/8182` |
| Full report | `var/hot-path-perf/bpf-5min-20260824T220100Z/bpf-report.md` |

**Caveats:** Edge `/track` returned 503 during much of the probe; `processor` was restarting. Tracker-0 showed 0% on-CPU in BPF (idle). Numbers reflect idle/partial stack + edge noise, not a clean business-load SLO run. Re-run with `PREPARE=1 bash scripts/test/malformed.sh business` when stack is healthy for Prometheus + uprobe p99.

### Scheduler / cgroup (highlights)

| Process | ctx/s | on-CPU % | peak RAM (MiB) | CPU throttle % |
| :--- | ---: | ---: | ---: | ---: |
| tracker-1 | 18,478 | 10.7 | 104 | 0.0 |
| tracker-0 | 0 | 0.0 | 100 | 0.0 |
| redis (each shard) | ~22-24 | ~0.5 | ~7-8 | 0.0 |
| nginx | - | - | 204 | 0.0 |

No cgroup memory.max events; no CPU throttling on trackers/redis.

### Hot syscalls (tracker, wall %)

| Syscall | avg (us) | p99 (us) | wall % | Note |
| :--- | ---: | ---: | ---: | :--- |
| `futex` | 23.9 | 0.3 | ~41 | goroutine park/wake; long max = idle sleep |
| `epoll_wait` | 1,667,880 | 0.3 | ~4.1 | gnet poll wait (idle-dominated) |
| `read` | 3.3 | 0.5 | ~0 | Redis RTT path |
| `write` | 9.6 | 0.3 | ~0 | |
| `fsync` | 8,535-11,643 | 1,049 | ~0 | 10 calls each tracker |

### Disk durability (group-commit gate)

| Metric | Value |
| :--- | :--- |
| Combined write+writev | 23,473 |
| Durability sync (fsync+fdatasync) | 92 |
| Sync reduction vs 1:1 baseline | **99.6%** (target >= 70%) |
| Group-commit coalescing | **PASS** |

### Network

| Process | TCP retrans | connect avg (us) |
| :--- | ---: | ---: |
| tracker-0 | 59 | 19.2 |
| tracker-1 | 36 | 18.6 |

Retrans > 0: check Redis/network during real load; may be benign on idle reconnects.

### FD / threads (stable)

Peak tracker FDs ~569 (557 sockets); thread count flat (39-40). No FD leak signal (`fd_delta` +1).

### Reproduce (operator)

```bash
make bpf-dev
export AD_EVENT_PROCESSOR_BPF_SUDO_PASS='...'   # local .env only; never commit
bash scripts/dev/bpf_session.sh start var/hot-path-perf/bpf-$(date -u +%Y%m%dT%H%M%SZ)

AD_EVENT_PROCESSOR_BPF_PROBE=1 AD_EVENT_PROCESSOR_BPF_SAMPLE_RATE=10 \
  DURATION=300 PREPARE=1 bash scripts/test/malformed.sh business

bash scripts/dev/bpf_session.sh stop
bash scripts/dev/bpf_session.sh report var/hot-path-perf/bpf-<timestamp>
```

Artifacts: `bpf-report.md`, `bpf/maps/summary.json`, `events.ndjson`, `load_loop.log`.

---

## Commands executed (verification honesty)

```bash
PERF_GATE_STRICT=true bash scripts/test/gate_bench.sh
go test -short -count=1 -run 'ZeroAlloc|...' ./internal/ingestion/...
bash scripts/ci/escape_heap_gate.sh
go test -run='^$' -bench='Benchmark(FilterFraudBoost|UnifiedFilter_Check_mock|LuaScript_Happy)' -benchmem -benchtime=200ms -count=10 -cpu=1 ./internal/ingestion/...
make bpf-dev
bash scripts/test/bpf_probe_session.sh start var/hot-path-perf/bpf-20260824T215500Z  # failed: no root
AD_EVENT_PROCESSOR_BPF_SUDO_PASS=... bash scripts/test/bpf_probe_session.sh start var/hot-path-perf/bpf-5min-20260824T220100Z
# 5 min curl fallback load to nginx :8000 (~24216 req)
bash scripts/test/bpf_probe_session.sh stop && bpf_session.sh report ...
benchstat var/hot-path-perf/gate_bench.txt
benchstat var/hot-path-perf/filter_lua_bench.txt
```

---

## Files

| Path | Content |
| :--- | :--- |
| `var/hot-path-perf/gate_bench.txt` | Full `gate_bench.sh` output |
| `var/hot-path-perf/filter_lua_bench.txt` | Filter + Lua benchmarks |
| `var/hot-path-perf/escape_heap_gate.txt` | `go build -gcflags=-m` escape scan |
| `var/hot-path-perf/bpf-20260824T215500Z/bpf_start.log` | BPF start failure log (no sudo) |
| `var/hot-path-perf/bpf-5min-20260824T220100Z/` | 5-min BPF session (report, summary.json, load_loop.log) |
