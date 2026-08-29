# pkg

Shared libraries **importable without `internal/`**. `pkg/*` must **not** import `internal/*`.

Use when two or more binaries or domain packages need the same stable, transport-level helper. Do not create `pkg/` wrappers around single-use domain logic.

Cross-ref: `.cursor/rules/boundaries.mdc`, `.cursor/rules/code-style.mdc`.

---

## Packages

| Package | Role | Hot/cold |
| :--- | :--- | :--- |
| `broker` | Broker wire protocol — client/server framing, ring buffer types | Hot producer path |
| `coldpath` | Limited body read, JSON decode, UUID parse for admin HTTP | Cold only |
| `clientip` | Extract client IP from headers/socket | Hot |
| `dedupkey` | Dedup key formatting for streams | Hot |
| `domainhealth` | Domain health check helpers | Cold |
| `faultproof` | Fault test telemetry (`fault_proof` lines) | Test |
| `gnetutil` | gnet server utilities | Hot |
| `httpresponse` | Prebuilt response byte slices | Hot |
| `iogate` | I/O backpressure gate | Hot |
| `landerhost` | Lander host resolution | Hot/cold |
| `legal` | Legal snippet strings | Cold |
| `lifecycle` | Graceful shutdown registration | Both |
| `logger` | Structured logging setup | Both |
| `moderatorintel` | Moderator intel snapshot types | Hot read |
| `money` | Money micros formatting | Cold |
| `naming` | Shared product naming constants | Both |
| `netaddr` | Network address parsing | Hot |
| `piihash` | PII hashing for exports | Cold |
| `platformconfig` | Platform config key constants | Cold |
| `proxyupstream` | Upstream proxy configuration | Edge |
| `regionproxy` | Region proxy wire types | Cold |
| `runtimepaths` | Runtime path resolution (`var/`, `bin/`) | Both |
| `supportbundle` | Support bundle tar layout | Cold |
| `branding` | Product branding strings | Cold |

---

## `coldpath` (most used on admin surface)

| API | Role |
| :--- | :--- |
| `ReadLimitedBody(w, r, max)` | Cap request bodies — default 64 KiB |
| `DecodeRequestOrBadRequest[T]` | Typed decode + 400 envelope |
| `DefaultMaxBody` | 65536 |

**Gate:** `bash scripts/ci/static/cold_path_json.sh`

Handlers must not use raw `io.ReadAll` on admin routes.

---

## `broker` (hot ingest)

Ring buffer producer types used by `internal/stream` and `cmd/broker`.

**Test:**

```bash
go test ./pkg/broker/... -count=1
go test ./internal/ingest/ -run TestFault_Broker -count=1
```

**Pitfalls:**

- Claiming broker-primary cutover safe from unit tests only — requires fault tier.
- Blocking `Produce` on tracker request thread — async enqueue only.

---

## Hot path constraints in `pkg/`

Even in `pkg/`, code called from `/track` must obey hot-path rules:

- No `fmt.Sprintf` in per-event loops.
- No `interface{}` boxing on ingest path.
- No allocations in bench-critical helpers — verify with `make test-alloc-gate` when touched.

Files used only on cold path (`coldpath`, `money`) have normal Go style freedom.

---

## When to add vs extend

| Situation | Action |
| :--- | :--- |
| One domain needs helper | Keep in `internal/<domain>/` |
| Tracker + processor need same bytes | `pkg/httpresponse` or `pkg/dedupkey` |
| Admin-only JSON helper | `pkg/coldpath` |
| New `internal/common` or `util` | **Banned** — use domain or `pkg/coldpath` |

---

## Verification

```bash
go test ./pkg/<package>/ -short -count=1
bash scripts/ci/static/hot_path_static.sh    # if hot consumer touched
```

Import lint: `pkg/*` must not appear in `internal/` import graph as importer of domain — only as imported by `internal/`.

---

## Common mistakes

1. **Putting business rules in `pkg/`** — belongs in `internal/<domain>/`.
2. **Importing `internal/` from `pkg/`** — compile error by policy; fix design.
3. **Duplicating `coldpath` in handlers** — use shared helper for body limits.
