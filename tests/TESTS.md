# tests

Cross-package test tiers outside package-local `*_test.go`. Package unit tests stay beside code in `internal/` and `pkg/`.

Cross-ref: `.cursor/rules/testing.mdc`, `.cursor/rules/fault-tests.mdc`, [scripts/SCRIPTS.md](../scripts/SCRIPTS.md).

---

## Tier model

| Tier | Makefile / script | Infra | Proves |
| :--- | :--- | :--- | :--- |
| Fast | `make test-fast`, `pr_fast.sh` | none | Static gates, unit, mocks (scoped) |
| Integration | `make test-integration` | testcontainers Redis/PG/CH | Real Lua, shards, SQL |
| Fault | `make test-fault`, `fault/run.sh` | compose + injection | Budget invariant, outbox order, broker cutover |
| Alloc | `make test-alloc-gate` | none | Hot path 0 allocs/op |
| Resilience | `make test-resilience` | sentinel env | Redis failover |
| E2E | `scripts/test/admin_stack_e2e.sh` | full stack + `web/` | Browser admin (skipped when no `web/`) |

**Honesty rule:** Never claim production hot-path behavior from fast tier alone. Name the tier in PR description.

---

## `tests/integration/`

Real databases via `internal/testutil`.

**Requirements per file:**

1. `testing.Short()` guard with skip reason containing `integration:`.
2. Real infra helper (`SetupPostgres`, `SetupRedis`, etc.).
3. Behavioral `require.*` / `assert.*` — no bare `t.Skip()`, no testify/mock.
4. No scaffold-only placeholders.

```bash
make test-integration
```

**Proves:** Unified-filter Lua parity, ClickHouse store, license HWID integration, Redis cluster behavior.

---

## `tests/resilience/`

Long-running failover and multi-step proofs.

```bash
make test-resilience
bash scripts/test/sentinel.sh
```

---

## `tests/e2e/`

Playwright (or similar) admin stack tests. **Requires `web/` tree.**

CI: `.github/workflows/admin-stack-e2e.yaml` — smoke + nightly Playwright when `web/` present; no-ops when `web/scripts/build.mjs` missing.

```bash
bash scripts/test/admin_stack_e2e.sh
```

---

## Package-local test taxonomy

| Suffix | Role |
| :--- | :--- |
| `*_test.go` | Unit tests |
| `*_integration_test.go` | Integration (must follow slop gate rules) |
| `*_fault_test.go` | Fault injection |
| `*_bench_test.go` | Microbenches — name scope (`_mock` for fake Redis) |
| `*_holdout*` | Negative tests that fail if fix reverted |

**Holdouts (hot path — cite in PR when touching producers/filters):**

| Behavior | Test |
| :--- | :--- |
| Admission without reserve | `TestStreamProducerAdmissionRaceWithoutReserve` |
| Dual XADD | `TestUnifiedFilter_SetDeferStreamToProducer_DualStreamWriteFix` |
| Post-debit rollback | `TestUnifiedFilter_RollbackDebit_LocalQuanta` |
| Broker shadow cutover | `TestFault_BrokerShadowCutover_NoEventLoss` |
| Parser edge parity | `TestChaos_CrossHop_NginxGnet` (`differential_count=0`) |

---

## Mock scope (valid vs lie)

| Mock | Valid claim | Invalid claim |
| :--- | :--- | :--- |
| `mockRedisClient` in unit tests | Go branch logic | "Redis Lua verified" |
| `mockBrokerClient` | Enqueue unit | "Broker HA safe" |
| `MockEventStore` | Consumer batching | "Rows in ClickHouse" |
| `BenchmarkUnifiedFilter_Check_mock` | Wrapper bench | `/track` p99 SLA |

---

## Financial invariants

Any spend-path change must preserve:

```go
AssertBudgetInvariant(t, ...)  // current_spend <= budget_limit +/- 1 micro-unit
```

Fault tests log `fault_proof fault=<name>` on success.

---

## Running subsets

```bash
go test ./internal/ingest/ -short -count=1
go test ./internal/ingest/ -run TestFault_ -count=1
go test ./tests/integration/ -count=1
go test ./internal/controlplane/ -run TestOpenAPI_ -count=1
```

**Do not** loop `go test ./...` on shared dev hosts without operator ask (`anti-slop.mdc`).

---

## CI mapping

| Workflow job | Test tier |
| :--- | :--- |
| `ci.yaml` merge-pr-fast | fast |
| `ci.yaml` merge-integration | integration |
| `ci.yaml` resilience (main) | fault |
| `perf-gate.yaml` | alloc / load smoke |
| `parser-fuzz-nightly.yaml` | fuzz |
| `license-fuzz-nightly.yaml` | fuzz |

See [WORKFLOWS.md](../.github/workflows/WORKFLOWS.md).

---

## Adding new tests

1. Pick tier — if it needs Redis/PG/CH, use `*_integration_test.go` + testcontainers.
2. Add holdout negative when behavior is non-obvious.
3. Do not weaken holdouts to green CI — fix root cause.
4. Fault proofs for new hot-path write paths (`fault-tests.mdc`).

---

## Common mistakes

1. **Bare `t.Skip()`** — fails `anti_slop_gate.sh`.
2. **Mock-only broker/CH proof** — integration tier required for wiring claims.
3. **Tautological asserts** — `require.Equal(t, x, x)` banned.
4. **Skipping `integration:` prefix** — gate cannot classify skip reason.
