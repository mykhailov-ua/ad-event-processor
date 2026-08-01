eSPX refactoring plan

Modular monolith for one engineer. Structure, naming, coupling — not a backlog.

See also: docs/ARCHITECTURE.md, docs/DEVELOPMENT.md, .cursor/rules/code-style.mdc (R1–R10).


Diagnosis

The repo pretends to be a microservice mesh. Auth, payment, billing, notifier, management share one Postgres, one Redis fleet, one ClickHouse. Cold path still uses gRPC and vtproto between processes that could be plain Go calls in one binary.

internal/controlplane (~150 files, ~43k LOC) imported internal/ingestion for types and stores — hot and cold glued together. Bridge/host adapters (*_bridge.go, *Host) forwarded GetPool() and added noise; most are gone, do not add them back.

Codegen (sqlc, vtproto, pb) plus LLM-generated cold path means most files were never read. Comments were stripped: fine on hot path, bad for cold onboarding.

Docs sprawl is mostly cleaned; human text in docs/, agent rules in .cursor/rules/.

Goal: fewer files, fewer packages, fewer network hops, one domain vocabulary, flat packages by filename — not Clean Architecture.


1. Repository Root

Keep it clean. No binary-specific Dockerfiles or env files at the root.

Target Root:
api/ — Protobuf and OpenAPI sources
cmd/ — Entry points (only main.go + wiring)
internal/ — Core logic (flat packages)
pkg/ — Clean utilities (no domain imports)
deploy/ — Consolidated infra manifests
scripts/ — Consolidated automation
docs/ — Human documentation (no agent-only md)
tests/ — Cross-module E2E and integration tests
Makefile, go.mod, go.sum, .env.example, REFACTORING.md

Move all Dockerfile.* to deploy/docker/.
Move all docker-compose.* to deploy/compose/.


2. Directory naming (internal/)

Product words, not abbreviations or org-chart roles.

Renames (done):
management → controlplane (`internal/controlplane`, `package controlplane`)
campaignmodel → domain (removed `internal/campaignmodel`)
internal/auth → identity (`internal/identity`, `package identity`)
internal/marginguard → ledger (`internal/ledger`, `package ledger`)
internal/fraudscoring → fraud (`internal/fraud`, `package fraud`)
adminapi → controlplane/adminapi (removed `internal/adminapi`)

Banned: ml, ai, utils, common, helpers, core, base, shared, internal2, management, mgr, svc, service. No nested theme folders (ingestion/filter/).

Allowed subdirs: db/, queries/, migrations/, pb/ (delete pb when internal gRPC gone). Non-Go assets (unified-filter.lua) in service root.


3. Directory naming (deploy/)

Consolidate the 20+ folders into functional groups.

Target deploy/:
deploy/compose/ — dev, prod, load-test compose files
deploy/docker/ — all Dockerfiles (prefixed: control.Dockerfile, tracker.Dockerfile)
deploy/config/ — nginx, redis, geoip, clickhouse config templates
deploy/monitoring/ — grafana dashboards, prometheus rules
deploy/terraform/ — infra as code (consolidated modules)
deploy/k8s/ — base manifests + minimal overlays (if required)
deploy/installer/ — standalone installer assets

Delete: fragmented per-service folders like deploy/payment/, deploy/billing/.


4. Directory naming (scripts/)

Consolidate the 14+ folders.

Target scripts/:
scripts/ci/ — CI/CD gates and checks
scripts/dev/ — local setup, stack management, profile smoke, db seeds
scripts/test/ — load, perf, fault, and e2e test runners
scripts/ops/ — deployment, tuning, maintenance
scripts/lib/ — shared shell functions

Removed alias dirs: `local-dev/`, `perf-gate/`, `edge-tuning/`, `redis/` (use `dev/`, `perf/`, `edge/`, `deploy/`).


5. Target Architecture: modular monolith

Default profile runs one control binary with in-process modules, not six gRPC peers on localhost.

internal/domain — shared structs, enums, pure logic
internal/ingestion — hot path only: gnet, filters, Redis Lua, stream consumer
internal/controlplane — admin API, outbox, workers, RBAC
internal/ledger — billing, settlement, reconciliation, margin
internal/identity — auth sessions, RBAC store
internal/notify — email, webhook delivery

Cross-module calls use Go interfaces and domain types, not gRPC, when binaries share a process.

Mapping one hop at I/O: db.Row → domain.Campaign in registry; db.Row → CampaignDTO in handler. Never db → entity → view → dto.


6. File naming

Pattern: prefix_stem_suffix.go. Tag = first segment before underscore.

Prefix roles:
api_<stem>.go — JSON handlers and types for stem
<stem>.go — orchestration for one subject (preferred cold path)
worker_<stem>.go — background loop
outbox_<stem>.go — outbox handler
store_<backend>.go — postgres, redis, clickhouse adapter
provider_<vendor>.go — external API
registry.go, serve.go — lifecycle (one file each)
*_test.go — tests mirror prefix

Size: over 800 LOC split by stem; transport + logic over 600 LOC split api_ vs stem file.


7. Comments

During this refactor: no comments in code. Do not add package godoc, exported-func godoc, or narration. Allowed only: `//go:` directives and `//nolint`.

After refactor stabilizes, optional sparse “why” lines on opaque cold-path workers (~5–10 per worker). Until then treat comments as debt — remove when touching a file.

Forbidden (always):
Package godoc, godoc on every exported func
Narration (“increment counter”, “return error”)
Changelog, ticket IDs, GAP-*, P##, milestone tags
Emoji, marketing tone, bureaucratic phrasing
Pointers to .cursor/, REFACTORING.md, backlog in code


8. Priorities (hard → easy)

1. Decouple control plane from ingestion: Move types to internal/domain.
2. Replace internal gRPC with in-process modules: billing_client.go etc. become Go structs.
3. Single control binary as default deploy: One process in compose.
4. Fold finance and identity into modules: internal/ledger, internal/identity.
5. Rename management → controlplane — done (`internal/controlplane`).
6. Delete internal-only protobuf and gRPC codegen. — done (see §9)
7. Split internal/config/env.go: done across `env_controlplane.go`, …
8. Consolidate sqlc output paths: internal/<module>/db/. — identity paths updated in sqlc.yaml
9. Merge legacy handler + service pairs. — done for module API surface; dead settlement pb convert removed
10. Rename files per naming rules. — partial: billing/payment/identity/notifier/controlplane §6 renames (handler→api, worker_*, client_*, settlement.go); controlplane clients merged to `client_integration.go`
11. Remove dead localhost clients and env vars. — done (gRPC server ports/hosts removed from config and compose)
12. Re-add sparse “why” comments on cold path (post-refactor only; see §7).
13. Repository root and deploy/scripts consolidation. — scripts alias dirs removed; smoke in `scripts/dev/`; Dockerfiles in `deploy/docker/`; compose in `deploy/compose/` (root `docker-compose.yaml` includes)


Global done

Zero internal/controlplane imports of internal/ingestion (prod + tests; `ingestion/pb` in budget delta consumer only).
Monolith: payment→settlement via `SettlementHandler.PaymentSettlement()` (`domain.PaymentSettlement`); billing/payment→notifier in-process; no localhost gRPC between control-plane modules.
`ServeOptions` uses `*AuthClient`, `*BillingClient`, `*PaymentClient`, `*NotifierClient` (not `pb.*ServiceClient`).
ivt-detector/fraud-scorer use management HTTP (`/api/v1/ops/blacklist`, `/api/v1/ops/fraud-threat`).
`control.Run` wires modules via `buildServeOptions` whenever `CONTROL_ENABLE_*` flags are set; no standalone `identity.Serve()` / `billing.Serve()` goroutines.
`OpenAPI` in identity/billing/payment/notifier always opens in-process modules; `grpc_api.go`, `handler_grpc.go`, `serve.go`, `*_service.proto`, and `buf.gen.grpc.yaml` removed.
Payment webhook HTTP (`:8187`) started from `payment.Module.StartWorkers`.
Removed gRPC-only config: `AUTH_SERVER_*`, `PAYMENT_SERVER_*`, `SETTLEMENT_SERVER_*`, `BILLING_SERVER_*`, `NOTIFIER_PORT` / `NOTIFIER_SERVER_HOST` (in-process modules only).
Scripts: profile smoke under `scripts/dev/`; deleted `local-dev/`, `perf-gate/`, `edge-tuning/`, `redis/` alias dirs.
Deploy: `deploy/docker/Dockerfile*` (platform + log workers); `deploy/compose/docker-compose.yaml` + load-test overlay; root compose stubs `include` those files.
Module API surface: `Handler` + `OpenAPI` live in `api.go`/`open.go` per module (billing, payment, identity, notifier); deleted thin `handler_types.go` and `resolve_api.go` splits.
Cold-path module APIs use domain/DB types directly; legacy protobuf round-trip helpers removed from `api.go` where gRPC transport is gone (`paymentIntentStatusString` replaces pb enum `.String()` in production).
No *_bridge.go or host adapters.
No nested domain packages under service roots.
domain is only shared type package.
New code follows naming and comment rules.
deploy/ and scripts/ follow consolidated structure.


Do not

Introduce Clean Architecture / DDD layering.
Split into more binaries to fix coupling.
Add comments to meet a quota.
Create REFACTORING_*.md children.
Move agent design docs into docs/.


9. Protobuf codegen (hot path only)

`make proto` (`scripts/ci/gen.sh --proto`) generates:

1. `api/buf.gen.nogrpc.yaml` — `protocolbuffers/go` for `api/events.proto` and `api/vast.proto`.
2. `api/buf.gen.vtproto.yaml` — `go-vtproto` for `events.proto` and `vast.proto`.
3. `safe_sync_proto_gen` → `internal/ingestion/pb`, `internal/rtb/pb`.
4. `cmd/patch-vtproto-hotpath` patches `internal/ingestion/pb/events_vtproto.pb.go`.

Removed cold-path message protos: `auth.proto`, `billing.proto`, `payment.proto`, `notifier.proto`, `settlement.proto` and `internal/{billing,payment,identity,notifier,controlplane}/pb/`.

Removed: `api/*_service.proto`, `api/buf.gen.grpc.yaml`, `*_grpc.pb.go` generation, `Register*ServiceServer` / `New*ServiceClient` in production, `*_GRPC_ENABLED` config fields, `pkg/lifecycle.ShutdownGRPC`.

In-process: module `API()` + `OpenAPI` → `OpenModule`; settlement via `SettlementHandler.PaymentSettlement()` + `PaymentModule.SetSettlementAPI`.

Payment fault/integration tests: `package payment_test` + `internal/paymenttest` (`SettlementFaultGate`) + `internal/payment/dbtest`.


10. split_control and standalone cmd/* deprecation

Status: standalone `cmd/auth` … `cmd/notifier` removed; compose and k8s use `deployment-control.yaml` (`/control` entrypoint).

Default deploy: `cmd/control` modular monolith. Compose profiles `single_vps`, `ingest_only`, `network_operator`, `resilience`, `crypto`, `fraud-scorer` run one `control` container (`CONTROL_ENABLE_*`). Local entry: `scripts/dev/stack.sh single-vps`.

| Removed | Replacement |
|---------|-------------|
| `stack.sh legacy-full` | `single_vps` or `network_operator` |
| compose/k8s auth, management, payment, billing, notifier deployments | `control` service / `deployment-control.yaml` |
| Dockerfile `/auth`, `/management`, `/payment`, `/billing`, `/notifier` binaries | `/control` only |

Monolith uses in-process `SetSettlementAPI`; no internal gRPC servers or `*_GRPC_ENABLED` env vars.
