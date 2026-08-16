# EDGE_CASES

**Naming:** **ad-event-processor** stack — [NAMING.md](NAMING.md).

Developer notes: failure modes that look like product bugs but sit at **OS / cgroup / TCP / Redis**, plus how we proved them. Numbers are laptop compose runs (2026-08-07), not production SLA. SLA budgets: `platform-sla.mdc` (tracker p95 &lt; 50 ms, p99 &lt; 80 ms; Redis Lua p99 &lt; 10 ms/shard).

Related: [BENCHMARKS.md](BENCHMARKS.md) (micro + purgatory tables), [ARCHITECTURE.md](ARCHITECTURE.md), [PARSER_SECURITY.md](PARSER_SECURITY.md), fault catalog `.cursor/rules/fault-resilience.mdc`.

Repo posture: **source-only** — clone is not ready-to-use; runtime/WAL/var artifacts stay gitignored (`.cursor/rules/source-repo.mdc`).

---

## Verdict (read this first)

| Finding | Code bug? | What it is |
| :--- | :---: | :--- |
| Purgatory p99 hundreds of ms–seconds under netem | **No** | TCP RTO + CFS + path delay; handler p99 ≈ Lua p99 |
| `ListenOverflows` / silent drops under close-churn | **No** | Host `somaxconn`/accept backlog + connection storm |
| SCRIPT FLUSH mid-load | **No** (soft) | Brief NOSCRIPT; tracker reloads; not a herd collapse in our run |
| `FILTER_TIMEOUT_MS=100` under 0.3% loss | **No** | `filter_timeout` reason stayed ~0; timeouts elsewhere |
| Microbench DFA/JSON/filter (ns–µs) | N/A | Healthy; not the floor under torture |

**Do not open P0/P1 on ingest parsers from these runs.** Fix ops defaults, runbooks, and alerts. Code changes only if production sysctl/listen are wrong or accept drain regresses under ET epoll.

---

## Hardware (all runs)

| | |
| :--- | :--- |
| Host | linux/amd64, Ubuntu 24.04.3, kernel 6.17.0-35-generic, cgroup v2 |
| CPU | Intel Core i5-11400H @ 2.70 GHz (6C/12T), L3 12 MiB |
| RAM | 16 GiB |
| Go | 1.25.x |
| SUT | compose `bidshard-tracker-0`, direct `POST http://127.0.0.1:8181/track` |

---

## 1. Causal model (three stacked floors)

Independent mechanisms. Raising CPU alone does **not** restore RPS when loss is present.

| # | Mechanism | Symptom | Evidence |
| ---: | :--- | :--- | :--- |
| 1 | **CFS throttle** | Kernel freezes cgroup for rest of period | 0.5 vCPU: 50–75% periods throttled; 1.0 vCPU + `GOMAXPROCS=1`: ~8–11% |
| 2 | **TCP RTO under netem loss** | Tail latency jumps in 200/400/800 ms steps | RPS stays ~10² after CPU raise; wrk p99 ≥ 0.5–1.3 s |
| 3 | **GOMAXPROCS vs quota** | Extra Ps → futex storms on thaw | BPF futex wall ~42–45%; fractional CPU + Go 1.25 still floors GOMAX at ≥2 unless pinned |

```text
wrk / churn ──► lo netem ──► TCP RTO ────────────┐
                 CFS freeze ──► backlog / futex ──┼──► p99 ≫ SLA, RPS ~10²
                 tiny somaxconn ──► ListenDrop ───┘
```

Go 1.25 is cgroup-aware but **will not set `GOMAXPROCS` &lt; 2** when the host has ≥2 CPUs. On `docker update --cpus=0.5`, pin `GOMAXPROCS=1` explicitly for tests.

---

## 2. Purgatory (steady hostile path)

Harness: `scripts/perf/purgatory/run_with_bpf.sh`  
Artifacts: `var/purgatory/<ts>/` (gitignored)

**Conditions (critical `T095412Z`):** 1.0 vCPU, 512 MiB, `GOMAXPROCS=1`, `LOCAL_QUOTA_MODE=off`, Lua histogram 100% sample, netem 5 ms / 1% loss / 1% dup, tiny TCP buffers, stress-ng L3, wrk 10k KA / 60 s, eBPF probe.

| Signal | Value |
| :--- | ---: |
| wrk RPS | 123.5 |
| wrk p50 / p99 | 51 ms / **1.33 s** |
| Handler p99 (Prom) | **356 ms** |
| Redis Lua p99 shard 1 | **356 ms** (n=7422 full-path) |
| CFS throttle (60 s) | ~8% / 0.16 s |
| Redis on-CPU | ~0.6% |

**Reading:** under lo netem, EVALSHA RTT dominates in-process time (handler ≈ Lua). wrk p99 adds client RTO/queueing. Microbenches (tens–hundreds ns) are **10⁸–10⁹×** smaller — bottleneck left the DFA.

Accuracy A/B (same netem, raise CPU): p50 improved, **RPS did not**; residual floor = loss/RTO.

---

## 3. Edge cascade (accept blackhole)

Harness: `scripts/perf/purgatory/run_edge_cascade.sh`  
Primary artifacts: `var/purgatory/edge-20260807T143839Z/`

**Intent:** combo hypothesized as product-breaking — mild loss + prod `FILTER_TIMEOUT` + SCRIPT FLUSH + accept pressure — with eBPF attached.

**Conditions:**

| Knob | Value |
| :--- | :--- |
| CPU / RAM / GOMAX | 1.0 / 512 MiB / 1 |
| `LOCAL_QUOTA_MODE` | off |
| `FILTER_TIMEOUT_MS` | 100 (`EDGE_FILTER_TIMEOUT_MS`) |
| netem | 5 ms, **0.3%** loss, 0% dup |
| listen backlog | **128** (set `somaxconn` **before** recreate so bind picks it up) |
| Load | wrk 10k KA 60 s |
| Inject @15 s | wrk 4k `Connection: close` / 12 s |
| Inject @18 s | `SCRIPT FLUSH SYNC` on `bidshard-redis-1-1` |
| Probe | `bin/bpf-collector` + `loadtest_probe.o` |

**Output (final pass):**

| Signal | Value |
| :--- | ---: |
| **TcpExtListenOverflows / ListenDrops Δ** | **+8874** (cumul. 18903) |
| Primary wrk RPS | 97 |
| Primary wrk p99 | 795 ms |
| Timeouts / non-2xx | 8 / 3 |
| Handler p99 (Prom) | 154 ms |
| Lua p99 shard 1 | 154 ms |
| NOSCRIPT/s (Prom) | ~0.05 after flush |
| `filter_timeout` reason | **0** |
| `infra_unavailable` | trace |
| CFS throttle | ~36% periods / ~1.5 s |
| Tracker OOM | no |

**That edge case:** with backlog 128 and a burst of *new* connections, the kernel **silently drops** accepts while the process is healthy, CPU is not pegged, and Prometheus still shows “some” RPS. Classic SYN/accept overflow literature (CPU/mem/net green, clients fail).

Backlog is fixed **at `listen`/`bind`**. Changing `somaxconn` without recreating the tracker leaves a large backlog (we saw 16384 until recreate).

**Doctor:** `ad-event-processor doctor --only listen` or `GET /api/v1/ops/doctor` → `listen` probe reads `TcpExtListenOverflows` / `ListenDrops` from `/proc/net/netstat` and warns on delta or low `somaxconn`.

SCRIPT FLUSH returned `OK`; NOSCRIPT rate stayed small — reload path held; not a thundering-herd outage in this profile. `DEBUG SLEEP` disabled on Redis image (expected).

---

## 4. Candidate matrix (ideas vs status)

| ID | Scenario | Status | Notes |
| :--- | :--- | :--- | :--- |
| A | Accept overflow + Lua stall | **Confirmed** | §3; OS/sysctl + churn |
| B | FILTER_TIMEOUT ↔ TCP RTO alignment | Exercised; weak | timeout reason 0 at 100 ms |
| C | SCRIPT FLUSH @ 10k conn | Soft | NOSCRIPT ~0.05/s |
| D | Redis BUSY / unkillable Lua | Not run | needs slow write script |
| E | nginx↔gnet keepalive race | Not run | idle timeout mismatch |
| F | Half-open Redis + tiny buffers | Partial (netem+buffers) | no dedicated TCP-deadlock proof |
| G | Fractional CPU + GOMAX≥2 | Confirmed in purgatory | pin GOMAX=1 |
| H | Local quanta × Redis kill | **Confirmed** | `TestFault_LocalQuantaRedisSIGKILL_BudgetInvariant`; budget invariant after stop/start |
| I | Short-conn TIME_WAIT / FD | Related to A | close burst |
| J | Wall-clock step vs TTC | Unit only | compose NTP step TBD |

**Next product-relevant experiment:** **H** (local quanta + Redis SIGKILL) — only class that can break **money**, not just latency.

---

## 5. Code vs OS checklist

Ask in order:

1. Did we shrink `somaxconn` / `tcp_max_syn_backlog` / rmem intentionally? → **OS**
2. Is `ListenOverflows` rising while process is up? → **OS accept path**, not JSON parser
3. Does handler p99 track Lua p99 under loss? → **network/Redis RTT**, not DFA
4. Did CPU raise fix RPS with loss still on? → if no, **RTO floor**, not missing `GOMAXPROCS` alone
5. Is budget Redis↔PG drifted after crash? → then **code/invariant** (local quanta, idempotency)

Hot-path alloc/BCE gates (`make test-alloc-gate`) stay orthogonal: they do not catch listen overflow or netem.

---

## 6. Reproduce

```bash
# Stack (existing images; --build may fail on realpath in some builders)
docker compose -f docker-compose.yaml -f docker-compose.load-test.yaml \
  up -d --no-build db redis-0 redis-1 redis-2 redis-3 redis-4 redis-5 \
  clickhouse processor prometheus grafana tracker-0 tracker-1 nginx
bash scripts/test/sync_tracker_registry.sh

# Steady purgatory + eBPF
PHASE=survival bash scripts/perf/purgatory/run_with_bpf.sh

# Edge cascade (ListenOverflow + FLUSH + close burst)
EDGE_FILTER_TIMEOUT_MS=100 EDGE_NETEM_LOSS=0.3% \
  bash scripts/perf/purgatory/run_edge_cascade.sh
# → var/purgatory/edge-<ts>/{REPORT.txt,bpf-report.md,bottleneck-report.md}
```

Requires: `bin/wrk`, `bin/stress-ng`, `bin/bpf-collector`, `deploy/dev/bpf/loadtest_probe.o`, Docker, privileged helper for tc/sysctl/BPF (script starts it).

Redis password: from root `.env` (`REDIS_PASSWORD`). Wrong password → silent no-op FLUSH (first cascade attempt).

---

## 7. What to watch in prod / staging

| Signal | Why |
| :--- | :--- |
| `nstat` / `TcpExtListenOverflows`, `ListenDrops` | Silent accept blackhole |
| `ss -ltn` backlog on tracker ports | Confirm listen vs `somaxconn` |
| `ad_http_request_duration` p99 vs `ad_redis_lua_duration` p99 | Same → wait on Redis/path |
| `ad_redis_lua_noscript_total` | SCRIPT FLUSH / restart herd |
| cgroup `nr_throttled` / `throttled_usec` | CFS freeze amplifier |
| `GOMAXPROCS` vs `cpu.max` | Fractional CPU mismatch |
| Filter reason `filter_timeout` / `infra_unavailable` | App-level vs kernel drop |

Alert idea: page on rising `ListenOverflows` even when tracker “healthy” and CPU low.

---

## 8. Ops hygiene (not hot-path PRs)

- Production: keep `net.core.somaxconn` and tracker listen backlog in the **thousands**; document in deploy runbook.
- Recreate/restart listeners after sysctl backlog changes.
- Run `ad-event-processor doctor --only listen` (or Ops → Doctor) after sysctl or edge churn; see §3 for silent accept drops.
- Match `GOMAXPROCS` to cgroup CPUs on fractional quotas (explicit pin in load-test / purgatory).
- Prefer keepalive / pooled clients at the edge; avoid short-conn storms into gnet under tight backlog.
- Treat 1% lo loss + tiny buffers as **torture**, not a baseline for ranking code changes.

---

## 9. Parser security vs infrastructure failures

Parser hardening (**P0–P3**: **PS-G01–G13**, **PS-H01–H06**) and infrastructure stress (purgatory, listen overflow, netem) answer different questions. Use this table before filing a hot-path parser regression.

**Canonical docs:** [PARSER_SECURITY.md](PARSER_SECURITY.md), [MILESTONES.md](MILESTONES.md) §2. **Verification:** `bash scripts/fault/parser_chaos_drill.sh`; slow-body isolation: `bash scripts/fault/parser_slow_body_drill.sh`.

| Symptom | Likely layer | First checks |
| :--- | :--- | :--- |
| Handler p99 ≈ Redis Lua p99 under netem loss | Network / Redis RTT | `ad_redis_lua_duration_seconds`, path loss, not DFA |
| `ListenOverflows` rising, CPU low | OS accept backlog | `somaxconn`, edge connection churn — see §3; `listen` doctor probe |
| Single connection drips 1 B/s, slot never frees | Parser / connection policy | `ad_http1_incomplete_close_total`, `HTTP1_INCOMPLETE_MAX`, `HTTP1_BODY_IDLE_MS` — [PARSER_SECURITY.md](PARSER_SECURITY.md) (PS-G01 **closed**) |
| Pool RSS spike after one 1 MiB reject | sync.Pool cap guard | `requestBufferPool` drops `cap > 64 KiB` — PS-H01 |
| Millions of `{`/`}` pairs in one JSON body | Key-pair budget | `MaxJSONKeyPairs`, `ad_json_key_pair_reject_total` — PS-H02 |
| nginx returns 400, tracker accepts same wire | Edge ↔ tracker differential | `TestChaos_CrossHop_NginxGnet` — PS-G04 (**zero** differentials) |
| 1 MB quote-dense OpenRTB pegs one core | Parser scan budget | `ad_ortb_scan_truncated_total`, `ORTB_SCAN_MAX_BYTES` — PS-G03 |
| Mixed chaos load raises pool rejects | Worker pool saturation | `WorkerPoolRejectTotal`, `CHAOS_LOAD_*` drill — PS-G08 |
| Microbenches green, chaos proofs green, p99 bad under wrk | Torture profile | Not a parser bug — see verdict table at top |

**Do not** treat purgatory or edge-cascade numbers as parser regression signals. **Do** run `bash scripts/fault/parser_chaos_drill.sh` after any change to `handler_http*`, `openrtb_*`, or ingress policy.

**Out of scope** (cold-path JSON, fraud ML, XDP, TCP backlog): [PARSER_SECURITY.md](PARSER_SECURITY.md) §9, [COLD_PATH_JSON.md](COLD_PATH_JSON.md), [enterprise/EDGE_XDP.md](enterprise/EDGE_XDP.md).

---

## 10. Sentinel promotion: `503 shard_unavailable` vs `202 Accepted`

When a Redis master is down or the **adaptive per-shard breaker** is open, the tracker isolates blast radius by campaign shard (`crc32(campaign_id) → slot → shard`).

| Request | Shard state | HTTP | Body |
| :--- | :--- | :---: | :--- |
| Campaign on failed shard (e.g. shard 0 during `redis-0` pause) | Breaker open / Redis unreachable | **503** | `shard_unavailable` |
| Campaign on healthy shard (shards 1–3) | Breaker closed | **202** | protobuf or JSON accepted event |
| Unknown campaign while registry stale | Pub/sub / PG sync lag | **503** | `registry_stale` |

**Code:** `internal/ingestion/handler.go` (`respShardUnavailable`, `202 Accepted` writers). **Proof:** `tests/resilience/shard_outage_fault_test.go` (`TestFault_Shard0Outage` — shard 0 → 503 + `shard_unavailable`; shards 1–3 → 202 under latency budget).

**Compose scenario G** (`scripts/test/sentinel.sh` + `deploy/compose/docker-compose.sentinel.yaml`): background Redis budget GET load at **30k RPS** (`SENTINEL_LOAD_TARGET_RPS`) via Sentinel while `redis-0` is paused; healthy shards keep `other_ok` traffic, shard 0 records errors, budget keys stay consistent after promotion. HTTP 503/202 mapping is unit-tested in resilience; the compose drill proves **Redis path isolation** under promotion load.

```bash
SENTINEL_LOAD_TARGET_RPS=30000 bash scripts/test/sentinel.sh
# fault_proof fault=sentinel_promotion_isolation target_rps=30000 ...
```

---

## 11. Artifact index

| Path | Contents |
| :--- | :--- |
| `var/purgatory/20260807T095412Z/` | Critical purgatory (1% loss, Lua=handler p99) |
| `var/purgatory/20260807T094603Z/` | Accuracy CPU/GOMAX A/B |
| `var/purgatory/edge-20260807T143251Z/` | Cascade (ListenOverflow confirmed; FILTER from .env leak) |
| `var/purgatory/edge-20260807T143839Z/` | Cascade with `FILTER_TIMEOUT_MS=100` |
| `scripts/perf/purgatory/run_with_bpf.sh` | Steady torture + eBPF |
| `scripts/perf/purgatory/run_edge_cascade.sh` | Overflow + FLUSH + close burst |
| `scripts/perf/tcp_syn_drop_gate.sh` | Phase 6: ListenOverflow gate under loadgen (tuned sysctl) |
| `deploy/compose/docker-compose.sentinel.yaml` | Sentinel + replica overlay for scenario G |
| `docs/BENCHMARKS.md` | Full number tables |
