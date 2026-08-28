# LOAD_TEST_WRK_LOCAL_BPF_MILESTONE

Local laptop load test with **wrk** as the HTTP generator, **eBPF** session probes, and two stack profiles: **ingest-only** (no ClickHouse) and **constrained + ClickHouse** (cgroup + server memory caps).

**Status:** DRAFT  
**Slug:** `load_test_wrk_local_bpf`  
**Depends on:** `make gen`, `make bpf-dev`, valid `var/license.jwt`, seeded campaigns for load-test UUIDs  
**Blocks:** perf regression baselines under `.ci-baselines/bpf/hot`, operator sign-off before perf claims in PRs  
**Domain rules:** `core.mdc`, `hot-path.mdc`, `load-test-bpf.mdc`, `development.mdc`, `anti-slop.mdc`

---

## 1. AI honesty, slop, and laziness (mandatory)

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| wrk path exists | Assumed `apt install wrk` without repo pin | `test -x bin/wrk` or document install step in section 7 |
| BPF session ran | `AD_EVENT_PROCESSOR_BPF_PROBE` unset; empty `bpf/maps/summary.json` | `test -f var/load-test/<ts>/bpf/maps/summary.json` |
| ingest-only = no CH | Started `full` or `clickhouse` profile by mistake | `docker compose ps` has no `clickhouse`; `CH_ENABLED=0` in tracker env |
| CH memory capped | Used default compose without `docker-compose.load-test.yaml` | `deploy/clickhouse/config.load-test.yaml` mounted; cgroup limit 1152M on `clickhouse` |
| SLA met | Cited wrk stdout latency only | `go run ./cmd/load-report bpf-gate` + Prometheus `ad_http_request_duration_seconds` p99 |
| Production RPS | Laptop wrk saturated loadgen CPU before tracker | Compare `bpf_loadgen_on_cpu_pct` in `bpf-gate.md`; state machine RAM tier |
| Sink wired | 202 on `/track` with deferred stream and nil producer | `AssertBudgetInvariant`; processor stream lag; CH row count after constrained profile |
| Mock bench = SLA | `BenchmarkUnifiedFilter_Check_mock` ns/op in PR | Load tier only; paste `malformed.sh` or wrk session + `load-report` |

### 1.2 Possible AI slop (this milestone)

| Slop pattern | What it looks like | What we require instead |
| :--- | :--- | :--- |
| wrk latency as prod SLA | Single wrk run `Latency Distribution` pasted | Prometheus handler p99 + `bpf-gate.md` rows |
| loadgen vs wrk confusion | `malformed.sh` run claimed as wrk milestone | Section 7 names generator (`wrk` vs `loadgen`) per profile row |
| CH on laptop default | `stack.sh full` for routine wrk | Profile A = `ingest-only`; Profile B = explicit `prepare_constrained_stack.sh` |
| Root wrk session | `sudo wrk` breaks compose file ownership | Non-root operator; `bpf_probe_session.sh` invokes sudo internally |
| Missing BPF object | `bpf-collector` start without `make bpf-dev` | `deploy/dev/bpf/loadtest_probe.o` present before session start |
| Invented RPS target | "Must hit 50k RPS" without license/SKU | Pilot cap 5k RPS (`sku.yaml`); lab target = sustain SLA at chosen wrk `-c`/`-t` |
| Void sink under load | Debit OK, no stream/CH rows | Post-run: Redis `XLEN` or processor metric; CH `count()` on constrained profile |
| Lab gate as CI gate | `BPF_GATE_PROFILE=lab` skips handler p99 | Document tier: lab = regression signal; `PERF_RUNNER_LABEL` = strict gate |

### 1.3 AI laziness (shortcuts to refuse)

| Shortcut | Agent motive | Refuse by |
| :--- | :--- | :--- |
| Reuse `malformed.sh` only | Already in tree | wrk path in section 5 step 1; `malformed.sh` = optional cross-check |
| Skip CH profile | OOM fear without trying | Profile B only after Profile A green; tear down CH after |
| `-short` unit tests | Avoid compose | Section 7 paste full wrk + bpf-gate commands |
| Purgatory as default | `run_with_bpf.sh` is torture cgroup | Purgatory = appendix; lab wrk = section 5 canonical |
| No budget check | Only HTTP latency | `seed_load_test_limits.sh` + `AssertBudgetInvariant` or load-report budget row |
| Fake `var/load-test/` | Invented timestamps | Paste real `OUT=` dir from script stdout |

### 1.4 Forbidden claims until verified

- "Tracker p99 < 50 ms" without Prometheus scrape during sustained load window
- "CH ingest keeps up" on ingest-only profile (no CH running)
- "Memory safe" without `runtime-pre` / `runtime-post` snapshots or docker OOM events
- "BPF proves 0 allocs" (BPF measures scheduling/syscalls; alloc proof = `make test-alloc-gate`)
- wrk RPS on 4 GB host compared to `scale` SKU 75k RPS ceiling
- "Green" without pasted exit code from section 7 commands

### 1.5 Doc-only delivery

This milestone is **spec + procedure**. Implementation of `scripts/test/wrk_load_bpf.sh` starts only after operator moves status to REVIEW and explicitly requests the script.

---

## 2. Scope

### In scope

- Repeatable local procedure: stack up, seed data, BPF session, wrk POST `/track`, collect artifacts under `var/load-test/<utc>/` or `var/bpf-session/<utc>/`
- Profile **A** `ingest-only`: tracker, processor, control, Postgres, Redis (no ClickHouse)
- Profile **B** `constrained-ch`: load-test compose overlay, ClickHouse `max_server_memory_usage=1 GiB`, cgroup cap 1152M, 2 trackers, Prometheus for SLA gates
- eBPF: `AD_EVENT_PROCESSOR_BPF_PROBE=1`, targets `tracker`, `redis`, `processor`, optional `nginx`
- SLA evaluation via `go run ./cmd/load-report all` and `bpf-gate`
- wrk tuning matrix (threads, connections, duration) documented per profile
- Host prerequisites: Linux amd64, Docker, BTF, passwordless sudo for BPF collector

### Out of scope

- `scripts/perf/purgatory/*` survival/torture cgroup tests (appendix reference only)
- Production `edge-xdp` compliance maps and XDP drop proofs (`edge.mdc`)
- CI `PERF_RUNNER_LABEL` strict gate (document comparison; run on self-hosted runner separately)
- `web/` admin UI performance
- Multi-region, broker-primary cutover fault matrix (`make test-fault`)

### Stop triggers (revert slice; do not compensate)

- Operator: "не код" before REVIEW (doc-only turns stay doc-only)
- OOM kills on host: stop Profile B; do not widen CH memory without updating section 4 caps table
- Sustained handler p99 > 80 ms for 30 s: abort wrk (same as `LOAD_SLA_GATE` / `core.mdc`)

---

## 3. Contract and inputs

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| Tracker ingest SLA ceilings | `core.mdc` | Section 6 pass/fail |
| BPF interpretation | `load-test-bpf.mdc` | Sample rate, artifacts, abort rules |
| ingest-only compose profile | `scripts/dev/stack.sh ingest-only`, `development.mdc` | Profile A |
| Constrained compose + ports | `.env.load-test`, `scripts/lib/load_test_env.sh`, `deploy/compose/docker-compose.load-test.yaml` | Profile B |
| CH server memory cap | `deploy/clickhouse/config.load-test.yaml` (`max_server_memory_usage: 1073741824`) | Profile B |
| Dev cgroup overlay | `deploy/compose/docker-compose.memory-dev.yaml` via `COMPOSE_MEMORY_PROFILE=dev` | Profile A optional |
| Campaign seed UUIDs | `scripts/test/seed_load_test_limits.sh` | wrk JSON body `campaign_id` |
| License / ingest gate | `var/license.jwt`, pilot 5k RPS cap | wrk rate ceiling for honest claims |
| BPF object + collector | `make bpf-dev`, `cmd/bpf-collector`, `deploy/dev/bpf/loadtest_probe.o` | Session start |
| wrk binary | `bin/wrk` (build or copy per section 7) | HTTP load |
| Prometheus (Profile B) | `LOAD_TEST_PROMETHEUS_URL` from rendered load-test env | `bpf-gate` handler + Lua p99 |
| Existing wrk+BPF reference | `scripts/perf/purgatory/run_with_bpf.sh` | Lua script pattern for POST `/track` |

---

## 4. Design spec (concrete, not intent)

### 4.1 Stack profiles

| Element | Profile A: ingest-only | Profile B: constrained + CH |
| :--- | :--- | :--- |
| Compose entry | `bash scripts/dev/stack.sh ingest-only` | `PREPARE=1 bash scripts/test/prepare_constrained_stack.sh` or `CONSTRAINED=1` path in `malformed.sh` prep |
| ClickHouse | **Absent** (`CH_ENABLED=0`) | **Present**; memory limit file mounted |
| CH memory | N/A | cgroup `1152M`; server `max_server_memory_usage=1 GiB` |
| Redis shards | 2 (`INGEST_REDIS_SHARD_COUNT` default) | 6 with UDS (`LOAD_TEST_REDIS_USE_UDS=1`) |
| Trackers | 1 (`tracker-0`, port 8181 typical) | 2 constrained (`8100`, `8200` ingest per `.env.load-test`) |
| Prometheus | Optional (not required for ingest-only smoke) | **Required** for SLA gate rows |
| Min host RAM | ~4 GB comfortable (`development.mdc`) | ~8 GB; 16 GB safer for wrk + compose |
| Authoritative sink | Redis stream / processor path (no CH rows) | Redis stream + processor + CH analytics tables |

### 4.2 wrk load shape

| Parameter | Profile A default | Profile B default | Notes |
| :--- | :--- | :--- | :--- |
| Method | `POST` | `POST` | `/track` JSON body |
| Path | `http://127.0.0.1:8181/track` | `http://127.0.0.1:${LOAD_TEST_TRACKER_INGEST_BASE}/track` (round-robin 2 bases) | Match seeded campaign |
| `-t` threads | `4` | `4` | <= physical cores reserved for wrk |
| `-c` connections | `200` (A), ramp `500` | `500` | Raise until p99 flattens or SLA fails |
| `-d` duration | `120s` | `300s` | Minimum 2 min after warm-up for Prometheus `[5m]` quantile |
| Body | JSON: `campaign_id`, `event`, `request_id` (unique per request) | Same | IDs from `seed_load_test_limits.sh` corpus |
| Headers | `Content-Type: application/json`, `Connection: keep-alive` | Same | Mirror `run_with_bpf.sh` Lua |
| Rate cap | <= 5000 RPS aggregate (pilot license) unless operator overrides JWT | Same | wrk is open-loop; tune `-c` to approach cap |

### 4.3 eBPF session

| Element | Spec | Owner artifact |
| :--- | :--- | :--- |
| Enable flag | `AD_EVENT_PROCESSOR_BPF_PROBE=1` | Operator env |
| Laptop sample rate | `AD_EVENT_PROCESSOR_BPF_SAMPLE_RATE=10` | Reduces overhead |
| Slow path threshold | `AD_EVENT_PROCESSOR_BPF_SLOW_US` default from `load-test-bpf.mdc` | `bpf-report.md` |
| Targets | `tracker,processor,redis` (+ `nginx` Profile B) | `bpf_probe_session.sh` |
| Loadgen comm | `AD_EVENT_PROCESSOR_BPF_LOADGEN_COMM=wrk` | Collector attributes CPU to wrk |
| Session root | `var/load-test/<utc>/` (integrated) or `var/bpf-session/<utc>/` (standalone) | `bpf_session.sh` |
| Non-root operator | wrk and compose as user; collector uses sudo | `malformed.sh` guard vs root |
| Artifacts | `bpf/maps/summary.json`, `bpf/events.ndjson`, `bpf-report.md`, `bottleneck-report.md` | `cmd/load-report` |

### 4.4 Abort and fail-closed invariants

| Invariant | Condition | Action |
| :--- | :--- | :--- |
| Handler SLA abort | Control cohort `ad_http_request_duration_seconds` p99 > 80 ms for 30 s | Stop wrk; mark run FAILED |
| Budget invariant | `current_spend > budget_limit` in Postgres | Stop wrk; FAILED (`AssertBudgetInvariant`) |
| Post-debit reject | `ad_stream_producer_post_debit_rejected_total` increase during run | FAILED; tune admission |
| CH OOM | ClickHouse container restart or OOM kill in `runtime-post` | FAILED Profile B; do not raise cap in same run |
| Tracker outbound connect | `tracker_outbound_connect` > 0 during gate | FAILED (`bpf-gate`) |
| Deferred stream without publisher | Accept with `fcap:ignored` and nil producer | Must not occur; 503 fail-closed before debit |

---

## 5. Implementation plan (ordered)

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | `scripts/test/wrk_load_bpf.sh` | Add lab orchestrator: parse profile `ingest-only` \| `constrained-ch`, start/stop BPF, invoke wrk with shared Lua body, write `var/load-test/<utc>/` | `bash scripts/test/wrk_load_bpf.sh --help` lists profiles; dry-run prints wrk command |
| 2 | `scripts/test/wrk_track.lua` | wrk Lua: POST JSON template, per-request UUID `request_id`, configurable `campaign_id` env | wrk completes 30s smoke against healthy tracker |
| 3 | `bin/wrk` | Document build/install in `docs/DEVELOPMENT.md` load-test subsection (or `scripts/test/install_wrk.sh`) | `test -x bin/wrk` |
| 4 | Profile A docs in section 7 | Wire `stack.sh ingest-only` + optional `COMPOSE_MEMORY_PROFILE=dev` | Tracker `/health` 200; no `clickhouse` container |
| 5 | Profile B docs in section 7 | Call `prepare_constrained_stack.sh`; export `LOAD_TEST_PROMETHEUS_URL` | Prometheus targets `up`; CH `system.settings` shows memory limit |
| 6 | `scripts/test/seed_load_test_limits.sh` | Run before wrk (both profiles) | 100 campaigns; budgets seeded |
| 7 | BPF integration | Reuse `bpf_probe_session.sh` from step 1; set `AD_EVENT_PROCESSOR_BPF_LOADGEN_COMM=wrk` | `bpf/maps/summary.json` non-empty after run |
| 8 | `go run ./cmd/load-report` | `all`, `bpf-gate` on session dir; `LOAD_SLA_GATE=1` for Profile B | `bpf-gate.md` written; exit 0 or documented SKIP rows for lab profile |
| 9 | Baseline optional | `go run ./cmd/load-report bpf-gate-compare .ci-baselines/bpf/hot <session>` | Compare markdown archived when operator requests regression |
| 10 | Cross-check | Optional: `AD_EVENT_PROCESSOR_BPF_PROBE=1 bash scripts/test/malformed.sh smoke` same host | Mixed-traffic loadgen run for parity; not a substitute for wrk milestone sign-off |

---

## 6. SLA and performance

Global ceilings from `core.mdc`. Lab wrk runs **prove compliance at declared RPS**, not unlimited capacity.

### 6.1 Application SLO (tracker `/track`)

| Surface / path | Metric | Ceiling | How measured |
| :--- | :--- | :--- | :--- |
| Tracker ingest | `ad_http_request_duration_seconds` p95 | < 50 ms | Prometheus `histogram_quantile(0.95, ...)` during load window |
| Tracker ingest | `ad_http_request_duration_seconds` p99 | < 80 ms (hard 100 ms) | Same at 0.99; `bpf-gate` row `tracker_handler_p99_ms` |
| Load-test abort | Handler p99 | > 80 ms for 30 s sustained | Abort wrk (`LOAD_SLA_GATE`, `malformed.sh` parity) |
| Filter deadline | `FILTER_TIMEOUT_MS` (prod) | <= 100 ms | Tracker env in prod-like Profile B |
| Redis unified-filter Lua | `ad_redis_lua_duration_seconds` p99 per shard | < 10 ms | `redis_lua_p99_max_ms` in `bpf-gate` |
| Geo filter (sampled) | Local geo check | < 10 us | Not gated in wrk lab; microbench tier only |
| Stream post-debit | `ad_stream_producer_post_debit_rejected_total` | ~0 during run | Prometheus counter delta |
| Budget | Postgres `current_spend <= budget_limit` | +/-1 micro-unit | `AssertBudgetInvariant` / load-report budget check |

### 6.2 BPF resource gate (strict / CI parity)

From `load-test-bpf.mdc` and `internal/loadreport/bpf_gate.go`. On laptop lab, set `BPF_GATE_PROFILE=lab` only when documenting known skips; otherwise treat as FAIL.

| Check | FAIL threshold | Source |
| :--- | ---: | :--- |
| `tracker_handler_p99_ms` | >= 80 | Prometheus |
| `redis_lua_p99_max_ms` | >= 10 | Prometheus per-shard max |
| `tracker_outbound_connect` | > 0 | Hot path must not dial outbound on `/track` |
| `tracker_rss_delta_kb` | > 5120 | Memory growth during session |
| `filter_check_uprobe_p99_us` | >= 1000 | BPF summary (when uprobes present) |
| `process_track_uprobe_p99_us` | >= 5000 | BPF summary |
| `bpf_loadgen_on_cpu_pct` | > 25 | wrk saturating CPU invalidates RPS claims |

### 6.3 Memory constraints (compose)

| Service | Profile A (`memory-dev`) | Profile B (`load-test` overlay) |
| :--- | :--- | :--- |
| ClickHouse | not running | 1152M cgroup; 1 GiB `max_server_memory_usage` |
| tracker-0 | 512M typical (`memory-dev`) | 768M |
| processor | 384M | per `docker-compose.load-test.yaml` |
| Postgres | 512M | 896M |
| Redis shard | 128M each | 256M each |
| Host guideline | 4 GB+ free | 8 GB+ free; wrk on CPU not shared with tracker pin |

### 6.4 wrk-specific honesty

| Claim | Valid | Invalid |
| :--- | :--- | :--- |
| wrk reported latency | Qualitative vs prior baseline on same machine | Production SLA without Prometheus |
| Achieved RPS | Lab sustain rate at passing p99 | SKU `scale` 75k RPS headline |
| ingest-only throughput | Hot path + Redis + processor sans CH | End-to-end analytics latency |
| Profile B | CH insert lag under memory cap | "CH unlimited" ingest |

---

## 7. Verification (paste in PR)

### 7.1 Prerequisites

```bash
make gen bpf-dev
go build -o bin/bpf-collector ./cmd/bpf-collector
test -x bin/wrk || (mkdir -p bin && cp "$(command -v wrk)" bin/wrk)
test -f var/license.jwt
bash scripts/dev/preflight.sh
```

### 7.2 Profile A: ingest-only + wrk + BPF

```bash
export COMPOSE_MEMORY_PROFILE=dev
export CH_ENABLED=0
bash scripts/dev/stack.sh down
bash scripts/dev/stack.sh build
bash scripts/dev/stack.sh ingest-only
bash scripts/test/seed_load_test_limits.sh

export AD_EVENT_PROCESSOR_BPF_PROBE=1
export AD_EVENT_PROCESSOR_BPF_SAMPLE_RATE=10
export AD_EVENT_PROCESSOR_BPF_LOADGEN_COMM=wrk
export CAMPAIGN_ID=00000000-0000-0000-0000-000000000001

OUT="var/load-test/$(date -u +%Y%m%dT%H%M%SZ)-ingest-only-wrk"
mkdir -p "$OUT"
bash scripts/test/bpf_probe_session.sh start "$OUT"

wrk -t4 -c200 -d120s -s scripts/test/wrk_track.lua \
  "http://127.0.0.1:8181/track" 2>&1 | tee "$OUT/wrk.log"

bash scripts/test/bpf_probe_session.sh stop "$OUT" "$(cat "$OUT/bpf/collector.pid")"
bash scripts/test/snapshot_runtime.sh "$OUT/runtime-post" 10

export LOAD_SLA_GATE=0
go run ./cmd/load-report all "$OUT"
go run ./cmd/load-report bpf-gate "$OUT" --prom "${PROMETHEUS_URL:-http://127.0.0.1:7000}"
```

### 7.3 Profile B: constrained + ClickHouse + wrk + BPF

```bash
PREPARE=1 bash scripts/test/prepare_constrained_stack.sh
source deploy/compose/.env.load-test.runtime
export AD_EVENT_PROCESSOR_BPF_PROBE=1
export AD_EVENT_PROCESSOR_BPF_SAMPLE_RATE=10
export AD_EVENT_PROCESSOR_BPF_LOADGEN_COMM=wrk
export PROMETHEUS_URL="${LOAD_TEST_PROMETHEUS_URL}"

OUT="var/load-test/$(date -u +%Y%m%dT%H%M%SZ)-constrained-ch-wrk"
mkdir -p "$OUT"
bash scripts/test/bpf_probe_session.sh start "$OUT"

wrk -t4 -c500 -d300s -s scripts/test/wrk_track.lua \
  "http://127.0.0.1:${LOAD_TEST_TRACKER_INGEST_BASE}/track" 2>&1 | tee "$OUT/wrk-primary.log"

bash scripts/test/bpf_probe_session.sh stop "$OUT" "$(cat "$OUT/bpf/collector.pid")"
bash scripts/test/snapshot_runtime.sh "$OUT/runtime-post" 10

export LOAD_SLA_GATE=1
go run ./cmd/load-report all "$OUT"
go run ./cmd/load-report bpf-gate "$OUT" --prom "$PROMETHEUS_URL"
```

Optional CH row check after Profile B:

```bash
docker compose -f deploy/compose/docker-compose.yaml \
  -f deploy/compose/docker-compose.load-test.yaml \
  --env-file .env --env-file .env.load-test \
  exec -T clickhouse clickhouse-client -q \
  "SELECT count() FROM ad_event_processor.impressions WHERE event_time > now() - INTERVAL 10 MINUTE"
```

### 7.4 Pass criteria table

| Check | Command / procedure | Pass criteria |
| :--- | :--- | :--- |
| BPF artifacts | `test -s "$OUT/bpf/maps/summary.json"` | File exists, non-zero size |
| Handler p99 | `bpf-gate.md` row `tracker_handler_p99_ms` | Value < 80 ms (FAIL if lab profile not used and over limit) |
| Lua p99 | `redis_lua_p99_max_ms` | < 10 ms per shard max |
| No outbound dial | `tracker_outbound_connect` | 0 |
| RSS delta | `tracker_rss_delta_kb` | <= 5120 |
| wrk errors | `grep -E 'Socket errors|non-2xx' "$OUT/wrk*.log"` | No sustained error growth; HTTP 2xx dominant |
| Budget | load-report budget section or manual `AssertBudgetInvariant` | No violation |
| CH memory | `docker inspect clickhouse` + CH logs | No OOM kill during Profile B |
| Holdout parity | `go test ./internal/ingestion/ -short -run 'TestStreamProducerAdmissionRaceWithoutReserve\|TestUnifiedFilter_SetDeferStreamToProducer_DualStreamWriteFix' -count=1` | Pass (unit; not a substitute for load tier) |

### 7.5 Appendix: purgatory wrk (torture, not lab default)

```bash
sudo bash scripts/perf/purgatory/run_with_bpf.sh
```

Use only for cgroup/memory survival experiments. Do not cite as Profile A/B SLA sign-off.

---

## 8. Definition of done

- [ ] Sections 1.1-1.4, 4, 5, 6, 7 complete (no template placeholders)
- [ ] `scripts/test/wrk_load_bpf.sh` and `scripts/test/wrk_track.lua` exist (step 1-2)
- [ ] Profile A and Profile B verification pasted with exit codes
- [ ] `bpf-gate.md` and `bottleneck-report.md` archived under `var/load-test/<utc>/`
- [ ] PR/commit title names surface: `Add wrk load-test script for local BPF sessions`
- [ ] No perf claim in docs/PR without session path cited

---

## 9. Rollback

- `bash scripts/dev/stack.sh down` and remove `var/load-test/<failed-utc>/`
- If step 1 script misbehaves: delete `scripts/test/wrk_load_bpf.sh` slice; keep using `malformed.sh` + manual wrk from section 7 until fixed
- Do not leave ClickHouse up on laptop after Profile B (`development.mdc` CH guidance)
