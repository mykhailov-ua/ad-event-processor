# <AREA>_<SLUG>_MILESTONE

Abstract milestone spec. Copy to `deploy/vendor/<AREA>_<SLUG>_MILESTONE.md` before implementation.

**Filename:** UPPERCASE only. Pattern: `<AREA>_<SLUG>_MILESTONE.md`  
Examples: `ADMIN_SHELL_MILESTONE.md`, `INGEST_PARSER_MILESTONE.md`, `LICENSE_GUARD_MILESTONE.md`

**Status:** DRAFT | IN_PROGRESS | REVIEW | SHIPPED  
**Slug:** (semantic slug; cross-reference in PRs and rules)  
**Depends on:** (prior slugs)  
**Blocks:** (downstream slugs)  
**Domain rules:** (e.g. `ui.mdc`, `hot-path.mdc` - link, do not duplicate)

---

## 1. AI honesty, slop, and laziness (mandatory)

Fill first. Empty tables = milestone not startable. Catalogs: `anti-slop.mdc` (**LLM dishonesty patterns**, **Hot path dishonesty**), `boundaries.mdc` (**Test dishonesty catalog**, **Coupling limits**).

### 1.1 Known hallucination risks (possible lies)

| Risk | Why agents lie | How to falsify |
| :--- | :--- | :--- |
| Artifact exists | Assumed path from old session or git history | `test -f <path>` or `rg` |
| Behavior wired | Unit mock passed; prod sink different | Integration/fault test or metric in section 7 |
| CI green | `pr_fast` / `-short` only | Paste exact command tier from section 7 |
| Hot path sink | Broker/stream/CH claimed from mock only | Trace hop to real sink (`anti-slop.mdc` **Writes-to-nowhere**) |
| | | |

### 1.2 Possible AI slop (this milestone)

| Slop pattern | What it looks like | What we require instead |
| :--- | :--- | :--- |
| Abstract plan | "Implement X", "wire Y" | Section 5: artifacts + done-when |
| Symptom patch | Local fix; root cause open | Section 4: invariants |
| Phantom reference | "See prior contract", "as before" | Text in doc or linked `.mdc` |
| Parallel system | New layer on broken base | One owner per concern |
| Doc ahead of code | SHIPPED in doc; tree empty | DRAFT until section 7 pasted |
| Stale cross-doc | Rules contradict each other | Same PR fixes linked `.mdc` or notes drift |
| Domain template bloat | Repeats detail from `.mdc` | `Domain rules:` link only |
| Mock-as-prod | `mockRedis` / `mockBroker` / in-memory store green | Mock boundary stated; integration in section 7 |
| Skip-as-coverage | `t.Skip("fault integration test")` without `integration:` | Prefix fixed or test run listed in section 7 |
| Void sink | Accept on ingress; nil producer / shadow-only / deferred stream with no writer | Section 4: authoritative sink per hop |
| Bench-as-SLA | Microbench ns/op in PR or section 6 | Load test / Prometheus for prod claims |
| Tautology test | `require.Equal(t, x, x)` or mock-only happy path | `*_holdout` negative; revert-fix proof |
| Mutation survivor | Tests green if production branch deleted | Holdout fails on revert; mutation runner not in CI |
| Black-box fiction | "External encrypted suite will catch it" | Name tier: `test-integration` / `test-fault` in section 7 |

### 1.3 AI laziness (shortcuts to refuse)

| Shortcut | Agent motive | Refuse by |
| :--- | :--- | :--- |
| Copy from git history | Faster than spec | Section 5 from current tree only |
| Smallest diff | Patch symptom | Section 4 root cause |
| Skip verification | Close task | Section 7 pasted commands |
| Doc ask -> code | "While I'm here" | Doc-only: zero impl diff |
| `-short` only | Green CI narrative | Declare tier; run integration/fault if wiring claimed |
| Example-only test | One happy table row | Fuzz/chaos row or holdout negative (`anti-slop.mdc`) |
| Extend mock returns | Force unit green | Real infra or document mock limit |
| Weaken holdout | "Flaky" | Fix cause; never drop admission/rollback/dual-stream tests |
| Piecemeal migration | Avoid revert | Atomic section 5; rollback section 9 |
| Invent gate/script | Looks thorough | `anti-slop.mdc` CI table |

### 1.4 Forbidden claims until verified

- "Done", "fixed", "green CI" without pasted command and exit code
- "Implemented X" without artifact path or holdout test
- Benchmark or latency numbers not in section 6 or 7
- Invented env vars, flags, routes, or files
- Doc-only / rules-only ask answered with code in the same turn
- "Server untouched" while diff includes embed, Docker, CI, handlers (list in section 5 if intentional)
- "Wired" / "stream/broker/CH path works" from unit mocks only (`mockRedisClient`, `mockBrokerClient`, `MockEventStore`)

### 1.5 Doc-only delivery

Operator asked for spec or rules only: **no implementation diff** unless this doc is REVIEW and operator explicitly requested code.

---

## 2. Scope

### In scope

- (concrete deliverables: binaries, routes, files, behaviors)

### Out of scope

- (defer to another `<AREA>_<SLUG>_MILESTONE.md`)

### Stop triggers (revert slice; do not compensate)

- Operator rejects approach (doc-only ask, scope stop, or "do not touch server" without listed server steps in section 5)
- Fix adds parallel system instead of correcting root cause in section 4

---

## 3. Contract and inputs

What this milestone must respect. Domain-specific detail lives in linked rules, not here.

| Input / contract | Source | This milestone uses |
| :--- | :--- | :--- |
| | (path, schema, rule file) | (how) |

---

## 4. Design spec (concrete, not intent)

No abstract goals ("make it robust", "polish UI"). State **observable structure**.

| Element | Spec | Owner artifact |
| :--- | :--- | :--- |
| | (dimensions, boundaries, invariants) | (file or component) |

Forbidden in this section: mood words; unnamed "later"; duplicate ownership of same concern.

---

## 5. Implementation plan (ordered)

Rejected: bullet "implement feature X". Required: ordered steps with artifacts.

| Step | Artifact(s) | Action | Done when |
| :--- | :--- | :--- | :--- |
| 1 | | | (test, gate, or observable check) |
| 2 | | | |

Each step must name **where** change lands and **what** proves it.

---

## 6. SLA and performance

Cite `core.mdc` for global ceilings. Add milestone-specific rows only when this slug touches that surface.

| Surface / path | Metric | Ceiling | How measured |
| :--- | :--- | :--- | :--- |
| | | | (bench name, Prometheus, load test, profiler - not guess) |

If not applicable: write `N/A` with one-line reason.

---

## 7. Verification (paste in PR)

```bash
# Milestone-specific commands (no placeholders in shipped doc)
```

| Check | Command / procedure | Pass criteria |
| :--- | :--- | :--- |
| Holdout | | Test fails if behavior regresses |
| Gate | | CI script name |
| Manual | | Observable outcome |

PR body must include commands **actually run**, not "should pass".

---

## 8. Definition of done

- [ ] Sections 1.1-1.4, 4, 5, 6, 7 complete (no template placeholders)
- [ ] Section 1.2 slop rows addressed or N/A with reason (include mock-as-prod / void-sink if hot path)
- [ ] Implementation plan rows checked
- [ ] New lie/lazy mode added to section 1.1 if discovered
- [ ] Verification output pasted
- [ ] Commit title names concrete surface (`core.mdc`)

---

## 9. Rollback

If operator stops work: revert milestone slice; no half-migrated parallel structure.
