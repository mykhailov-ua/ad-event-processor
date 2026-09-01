# scripts

Shell and Python automation. **Path scopes the domain** — inside `scripts/ci/license/` do not prefix filenames with `license_`.

Cross-ref: [.github/workflows/WORKFLOWS.md](../.github/workflows/WORKFLOWS.md), [deploy/DEPLOY.md](../deploy/DEPLOY.md).

---

## Layout

| Directory | Role | Entry |
| :--- | :--- | :--- |
| `ci/` | Merge gates, static analysis | `pr_fast.sh`, `lint.sh` |
| `dev/` | Local stack, codegen helpers | `stack/stack.sh`, `admin_ui.sh` |
| `test/` | Integration, load, edge, BPF, license | domain subdirs |
| `fault/` | Chaos and resilience proofs | `run.sh` |
| `security/` | License red-team, pentest | `license_red_team.sh` |
| `perf/` | Benchmarks, regression | `redis_uds_benchmark.sh` |
| `ops/` | Redis topology, CPU isolation, sysctl | `verify_redis_topology.sh` |
| `install/` | Appliance bootstrap | `appliance_bootstrap.sh` |
| `lab/` | HWID collect, binary patch lab | `binary_patch_lab.sh` |
| `lib/` | Shared `aed_*` shell functions | sourced by other scripts |

**Ceiling:** ≤ 20 leaf `.sh`/`.py` at any flat `scripts/<dir>/` root. Orchestrators compose leaves once — no matryoshka re-running the same gate.

---

## CI (`scripts/ci/`)

### `pr_fast.sh`

Primary PR gate. Runs naming, anti-slop, unit tests (`-short`), admin web stub checks, OpenAPI subset.

```bash
bash scripts/ci/pr_fast.sh
```

### `lint.sh`

Fan-in over `ci/lint/*` — language linters plus config validation.

```bash
bash scripts/ci/lint.sh
make lint    # runs gen + fmt + lint.sh
```

| Script | Language | Tool | Scope |
| :--- | :--- | :--- | :--- |
| `lint/go.sh` | Go | golangci-lint (hot/cold split) | `internal/ingest`, `filter`, `track`, `stream`, `rtb`, `cmd/tracker`, `pkg/broker` + cold rest |
| `lint/gopls.sh` | Go | gopls check (warning+) | all non-generated packages |
| `lint/go_modernize.sh` | Go | rangeint static gate | ingest hot path |
| `lint/lua.sh` | Lua | luacheck, LuaLS | `deploy/nginx/lua`, `internal/ingest`, `internal/filter`, `internal/stream`, `scripts/test/load` |
| `lint/shell.sh` | Shell | shellcheck | all `scripts/**` and `deploy/**` `*.sh` (`.shellcheckrc`) |
| `lint/python.sh` | Python | ruff | `model/`, `scripts/dev/gen_ingest_gnet.py` |
| `lint/proto.sh` | Protobuf | buf lint | `api/` |
| `lint/workflows.sh` | YAML (CI) | actionlint | `.github/workflows/` |
| `lint/ts.sh` | TS/JS | tsc, `node --check` | `web/` when present |
| `lint/configs.sh` | Infra | compose config, nginx `-t`, OpenAPI | deploy + admin contracts |

**Env knobs:**

| Variable | Effect |
| :--- | :--- |
| `LINT_STRICT=1` | golangci full repo (no `--new-from-rev`) |
| `LINT_INCREMENTAL=1` | golangci `--new-from-rev=origin/main` |
| `LINT_SHELL_SEVERITY` | override shellcheck severity (default `error`) |
| `LINT_SHELL_WARN=1` | shellcheck `-S warning` (local deep pass) |
| `SKIP_LINT=1` | `pr_fast.sh` skips `lint.sh` (CI runs lint in `merge-lint` first) |

Configs: `.golangci-hot.yaml`, `.golangci-cold.yaml`, `.luacheckrc`, `.luarc.json`, `.stylua.toml`, `.shellcheckrc`, `model/pyproject.toml`, `api/buf.yaml`.

**Not separate linters (covered elsewhere):**

| Surface | Gate |
| :--- | :--- |
| SQL in Go | `scripts/ci/static/sql_safety.sh` in `pr_fast.sh` |
| BPF C | `clang-format` in `format.sh` |
| Lua formatting | stylua in `format.sh` (`make fmt`) |
| YAML / JSON / CSS | prettier in `format.sh` |
| Markdown / rules | prettier + naming gates |

### Subdirs

| Subdir | Examples |
| :--- | :--- |
| `license/` | `verify_tier.sh`, red-team gates, fuzz nightly |
| `admin/` | `web.sh`, `openapi.sh`, `ui_surface.sh` |
| `static/` | `hot_path_static.sh`, `escape_heap.sh`, `anti_slop.sh`, `cold_path_static.sh` |
| `naming/` | `legacy_naming.sh`, `antifraud_doc.sh` |
| `bpf/` | `resource.sh` |
| `lint/` | golangci wrapper |

**Rule:** One leaf = one contract. Do not add `*_gate.sh` that only wraps existing gates with `exit 0` on failure.

---

## Dev (`scripts/dev/`)

### `stack/stack.sh`

Canonical compose orchestrator.

```bash
bash scripts/dev/stack/stack.sh ingest-only   # default laptop
bash scripts/dev/stack/stack.sh build
bash scripts/dev/stack/stack.sh down
bash scripts/dev/stack/preflight.sh
bash scripts/dev/stack/seed_admin.sh
```

Reads `deploy/compose/` overlays, `.env`, `REDIS_SHARD_COUNT`, `COMPOSE_MEMORY_PROFILE`.

**Pitfalls:**

- Do not run `full` + ClickHouse on 16 GB RAM without `CH_ENABLED` discipline.
- After sysctl apply, recreate listeners (stack warns).
- `verify_redis_topology.sh` must pass before debugging shard errors.

### `codegen/`

Model venv and format helpers for `model/`.

---

## Test (`scripts/test/`)

| Subdir | Role |
| :--- | :--- |
| `edge/` | Lua tests, XDP drills, reverse proxy smoke |
| `load/` | `gate_run.sh`, malformed load, parser chaos |
| `bpf/` | `nightly_job.sh` hot/cold/export soak |
| `license/` | Red-team extended, garble alloc gate |
| `capi/` | CAPI compliance smoke |
| `telegram/` | Telegram webhook smoke |
| `release/` | Release QA smoke |
| `redis/` | Topology smoke |

**Operate load gate:**

```bash
bash scripts/test/load/gate_run.sh
PERF_GATE_STRICT=1 bash scripts/test/load/gate_run.sh   # self-hosted
```

---

## Fault (`scripts/fault/`)

```bash
bash scripts/fault/run.sh
bash scripts/fault/compose_fault_drill.sh all
bash scripts/fault/sentinel_failover_env.sh
```

Success logs `fault_proof fault=<name>`. Required on `main` CI resilience job.

---

## Security (`scripts/security/`)

```bash
make license-red-team
make license-verify
bash scripts/security/license_pentest.sh
```

Manual drills: `deploy/vendor/fixtures/hwid_spoof/`, `deploy/vendor/fixtures/binary_patch/`.

---

## Ops (`scripts/ops/`)

| Script | Role |
| :--- | :--- |
| `verify_redis_topology.sh` | `.env` shard count vs addresses |
| `cpu_isolation.sh` | cpuset verify for tracker |
| `sysctl.sh` | Host tuning from `deploy/sysctl/` |

---

## Install (`scripts/install/`)

```bash
bash scripts/install/appliance_bootstrap.sh
bash scripts/install/appliance_bootstrap.sh --profile full
```

---

## Shared lib (`scripts/lib/`)

| Helper | Role |
| :--- | :--- |
| `paths.sh` | `ROOT`, `SCRIPTS` resolution |
| `installer_env.sh` | Read `.env` for compose |
| `dev_bind_mounts.sh` | Dev volume mounts |
| `redis_topology.sh` | Shard helpers |

Use `aed_` prefix for new internal shell functions (`naming.mdc`).

---

## Makefile targets (scripts callers)

| Make target | Script tier |
| :--- | :--- |
| `make test-fast` | unit + pr_fast subset |
| `make test-integration` | testcontainers |
| `make test-fault` | fault matrix |
| `make test-alloc-gate` | hot path benches |
| `make license-verify` | `ci/license/verify_tier.sh` |
| `make gen` | sqlc, templates |

---

## Development rules

1. **Update callers in same commit** when moving a script — Makefile, Taskfile, `.github/workflows/`, docs.
2. **No hardcoded script inventories** in gates — discover via Makefile/workflows.
3. **No `exit 0` on failure** in named gates.
4. **Integration skip reason** must contain `integration:` prefix (`anti_slop_gate.sh`).
5. **Echo/log** — no Unicode dashes or emoji in `scripts/` (`naming.mdc`).

---

## Verification

| Goal | Command |
| :--- | :--- |
| Pre-push | `bash scripts/ci/pr_fast.sh` |
| Hot path change | `make test-alloc-gate` |
| OpenAPI change | `bash scripts/ci/admin/openapi.sh` |
| License change | `make license-verify` |
| Full main parity | see `.github/workflows/ci.yaml` |
