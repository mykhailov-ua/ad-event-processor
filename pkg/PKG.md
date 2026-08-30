# pkg

Shared libraries **importable without `internal/`**. `pkg/*` must **not** import `internal/*`.

Use when two or more binaries or domain packages need the same stable, transport-level helper. Do not create `pkg/` wrappers around single-use domain logic.

Cross-ref: `.cursor/rules/boundaries.mdc`, `.cursor/rules/code-style.mdc`, per-package `doc.go`.

---

## Packages

| Package | Role | Hot/cold |
| :--- | :--- | :--- |
| `broker` | Consumer offset files; wire in `protocol/`, `log/`, `client/` | Hot ingest (client/consumer) |
| `coldpath` | Limited body read, JSON decode, UUID parse for admin HTTP | Cold only |
| `clientip` | Client IP from X-Forwarded-For when peer trusted | Cold (control, identity) |
| `dedupkey` | Dedup key formatting for streams and spend sync | Hot |
| `domainhealth` | Domain health check helpers | Cold |
| `faultproof` | Fault test telemetry (`fault_proof` lines) | Test harness |
| `gnetutil` | gnet conn idle / max lifetime on accept | Hot (broker, region-proxy servers) |
| `httpresponse` | JSON success/error envelopes for admin HTTP | Cold only |
| `iogate` | Disk write gate, group commit, writev coalesce | Hot (broker WAL, logger, region WAL) |
| `landerhost` | Hosted lander store, preview tokens, zip publish | Cold (flow + filter routing snapshot) |
| `legal` | Embedded EULA text and acceptance JSON | Cold |
| `lifecycle` | Graceful shutdown, health probes, metrics sidecar | Both |
| `logger` | MPSC ring log shard, disk evacuator | Hot (tracker) + sidecars |
| `moderatorintel` | Signed moderator IP prefix feed (JSON object) | Hot read (filter, ingest) |
| `money` | Money micros formatting | Cold |
| `naming` | Shared product naming constants | Both |
| `netaddr` | gnet listen URI, Redis URL, unix socket detect | Both |
| `piihash` | HighwayHash PII columns (ip, ua, user id, subnet) | Hot (tracker, filter, stream) |
| `platformconfig` | install.yaml schema, Redis render | Cold |
| `proxyupstream` | SSRF-safe click proxy upstream URL validation | Hot (ingest click proxy) + cold (campaign editor) |
| `regionproxy` | Multi-region WAL, quorum, uplink (Enterprise) | Cold (regional cell) |
| `runtimepaths` | Runtime path resolution (`var/`, `bin/`, broker socket) | Both |
| `supportbundle` | Support bundle tar layout and redaction | Cold |
| `branding` | Product/vendor strings and safe-view HTTP headers | Hot (ingest decoy bytes) + cold (admin) |

Hot/cold reflects **production** import paths. Fault tests may import additional packages.

Module-level contracts live in each package `doc.go` (Role, Verify, Go importers).

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

| Subpackage | Role |
| :--- | :--- |
| `protocol` | Frame codec, produce/fetch/offset wire |
| `log` | mmap WAL segments, durability modes |
| `client` | TCP/unix client; optional Redis leader discovery |
| `consumer` | Fetch loop with file-backed offsets |
| root | `ConsumerOffsetTracker` persistence |

Ring-buffer producer lives in `internal/stream/broker/broker_producer.go`, not in `pkg/broker` root.

**Test:**

```bash
go test ./pkg/broker/... -count=1
go test ./internal/ingest/ -short -run TestFault_Broker -count=1
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

Files used only on cold path (`coldpath`, `money`, `httpresponse`) have normal Go style freedom.

---

## When to add vs extend

| Situation | Action |
| :--- | :--- |
| One domain needs helper | Keep in `internal/<domain>/` |
| Tracker + processor need same hash/key bytes | `pkg/dedupkey`, `pkg/piihash` |
| Admin-only JSON helper | `pkg/coldpath`, `pkg/httpresponse` |
| New `internal/common` or `util` | **Banned** — use domain or `pkg/coldpath` |

---

## Verification

```bash
go test ./pkg/<package>/ -short -count=1
bash scripts/ci/static/pkg_boundary.sh
bash scripts/ci/static/hot_path_static.sh    # if hot consumer touched
```

Import lint: `pkg/*` must not import `internal/*`.

---

## Common mistakes

1. **Putting business rules in `pkg/`** — belongs in `internal/<domain>/`.
2. **Importing `internal/` from `pkg/`** — compile error by policy; fix design.
3. **Duplicating `coldpath` in handlers** — use shared helper for body limits.
4. **Treating `httpresponse` as hot** — ingest uses prebuilt bytes and branding headers, not JSON writers.
