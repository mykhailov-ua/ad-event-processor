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
scripts/dev/ — local setup, stack management, db seeds
scripts/test/ — load, perf, fault, and e2e test runners
scripts/ops/ — deployment, tuning, maintenance
scripts/lib/ — shared shell functions

Delete: scripts/local-dev/, scripts/perf-gate/, scripts/edge-tuning/, scripts/redis/.


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
6. Delete internal-only protobuf and gRPC codegen. — in progress (see §9)
7. Split internal/config/env.go: config.go, ingest.go, database.go, etc.
8. Consolidate sqlc output paths: internal/<module>/db/. — identity paths updated in sqlc.yaml
9. Merge legacy handler + service pairs.
10. Rename files per naming rules.
11. Remove dead localhost clients and env vars.
12. Re-add sparse “why” comments on cold path (post-refactor only; see §7).
13. Repository root and deploy/scripts consolidation.


Global done

Zero internal/controlplane imports of internal/ingestion (prod + tests; `ingestion/pb` in budget delta consumer only).
Monolith: payment→settlement via `SettlementHandler.PaymentSettlement()` (`domain.PaymentSettlement`); billing/payment→notifier in-process; no localhost dials for those hops.
`ServeOptions` uses `*AuthClient`, `*BillingClient`, `*PaymentClient`, `*NotifierClient` (not `pb.*ServiceClient`).
ivt-detector/fraud-scorer use management HTTP (`/api/v1/ops/blacklist`, `/api/v1/ops/fraud-threat`); settlement gRPC optional via `SETTLEMENT_GRPC_ENABLED=0`.
Split_control profile deprecated; still uses network gRPC until profile removal.
Monolith profile (`cmd/control`): `SETTLEMENT_GRPC_ENABLED=0`; no localhost gRPC between control-plane modules.
Standalone `Serve()` gRPC listeners gated by `*_GRPC_ENABLED` (default on; `0` skips TCP while keeping workers/HTTP).
Settlement gRPC server in `controlplane.ServeWithOptions` gated by `SettlementGRPCEnabled` (compose `control` sets `SETTLEMENT_GRPC_ENABLED=0`).
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


9. Protobuf codegen split (internal gRPC removal)

Default `make proto` (`scripts/ci/gen.sh --proto`):

1. `api/buf.gen.nogrpc.yaml` — `protocolbuffers/go` for all `api/*.proto` (messages only).
2. `api/buf.gen.vtproto.yaml` — `go-vtproto` for `events.proto` and `vast.proto` only.
3. `safe_sync_proto_gen` → `internal/*/pb/`.
4. `safe_prune_service_vtproto` drops stale `*_vtproto.pb.go` under identity, billing, payment, notifier, controlplane.
5. `cmd/patch-vtproto-hotpath` patches `internal/ingestion/pb/events_vtproto.pb.go`.

Transitional gRPC (split_control / localhost clients until in-process migration finishes):

`make proto-grpc` or `scripts/ci/gen.sh --proto --proto-with-grpc` adds `api/buf.gen.grpc.yaml` for auth, billing, payment, notifier, settlement.

Templates:

| File | Plugins | Scope |
|------|---------|-------|
| `buf.gen.nogrpc.yaml` | protobuf/go | all protos |
| `buf.gen.vtproto.yaml` | go-vtproto | events, vast |
| `buf.gen.grpc.yaml` | grpc/go | five service protos (opt-in) |
| `buf.gen.yaml` | protobuf/go | alias of nogrpc (direct `buf generate`) |

Keep: `events.proto`, `vast.proto` with full vtproto (hot path).

Remove next: `*_grpc.pb.go` and `service` blocks in auth, billing, payment, notifier, settlement protos once callers use Go interfaces.

gRPC server gating (done)

| Env | Config field | Effect |
|-----|--------------|--------|
| `AUTH_GRPC_ENABLED` | `AuthGRPCEnabled` | Standalone `identity.Serve()` TCP listener (default on; `0` = workers/metrics only) |
| `BILLING_GRPC_ENABLED` | `BillingGRPCEnabled` | Standalone `billing.Serve()` TCP listener |
| `PAYMENT_GRPC_ENABLED` | `PaymentGRPCEnabled` | Standalone `payment.Serve()` TCP listener |
| `NOTIFIER_GRPC_ENABLED` | `NotifierGRPCEnabled` | Standalone `notifier.Serve()` TCP listener |
| `SETTLEMENT_GRPC_ENABLED` | `SettlementGRPCEnabled` | `controlplane.ServeWithOptions` settlement TCP listener |

All flags: `os.Getenv(...) != "0"` (unset = enabled). Monolith compose `control` sets `SETTLEMENT_GRPC_ENABLED=0` and `NOTIFIER_GRPC_ENABLED=0`; `.env.example` defaults settlement off.

In-process module `API()` calls handler core methods directly; gRPC `handler_grpc.go` / `handler.go` are thin pb wrappers only. Settlement in-process via `SettlementHandler.PaymentSettlement()` + `PaymentModule.SetSettlementAPI`; no localhost dial when `SETTLEMENT_GRPC_ENABLED=0`.

Split deploy: standalone `cmd/payment` + `cmd/management` keep settlement gRPC (`SETTLEMENT_GRPC_ENABLED` unset or `1` on management). `payment/outbox_worker.go` and `settlement_ledger_client.go` dial `SettlementServiceClient` when `SetSettlementAPI` is not injected.

`local_client.go` / `Module.Client()` / `Module.GRPC()` removed from identity, billing, payment, notifier — monolith and split `Serve()` use `mod.API()` + gRPC adapters at dial boundary only.

Blockers for deleting `api/auth.proto` (and siblings)

`*_grpc.pb.go` is still required until split-deploy and handler layers drop gRPC types:

| Generated type | Status | Remaining import sites |
|----------------|--------|------------------------|
| `AuthServiceClient` / `AuthServiceServer` | Monolith in-process via `identity.AuthAPI`; `Login` returns `LoginResult` (service `LoginDTO`) | `identity/{handler,handler_core,handler_grpc,auth_convert,serve,grpc_api,resolve_api}.go`; `controlplane/auth_client.go` |
| `BillingServiceClient` / `BillingServiceServer` | Monolith in-process via `billing.BillingAPI` (`Invoice`, not `pb.Invoice`) | `billing/{handler,handler_core,handler_grpc,handler_validate,invoice_types,invoice_convert,serve,grpc_api,resolve_api}.go`; `controlplane/billing_client.go` |
| `PaymentServiceClient` / `PaymentServiceServer` | Monolith in-process via `payment.PaymentAPI`; handler uses `PaymentIntent` (service still `db` internally) | `payment/{handler,handler_core,handler_grpc,intent_convert,serve,grpc_api,resolve_api}.go`; `controlplane/payment_client.go` |
| `NotifierServiceClient` / `NotifierServiceServer` | Monolith in-process via `notifier.NotifierAPI`; service returns `Notification` (not `pb.Notification`) | `notifier/{handler,handler_grpc,notification_types,notification_convert,api,service_input,serve,grpc_api}.go`; `controlplane/notifier_client.go` |
| `SettlementServiceClient` / `SettlementServiceServer` | Monolith in-process via `SettlementHandler.PaymentSettlement()` (`domain.PaymentSettlement`) | `controlplane/{settlement_handler,settlement_handler_grpc,serve}.go`; `payment/{settlement_grpc_client,resolve_settlement,settlement_ledger_client,outbox_worker}.go` |

Message types (`*.pb.go`) remain in use for handler request/response structs and outbox payloads — delete protos only after those call sites use `internal/domain` types.

Fresh clone / CI: `make proto` alone does not emit `*_grpc.pb.go`. Use `make proto-grpc` until blockers above are cleared (or keep committed/stale grpc outputs during transition).

Done when: no `Register*ServiceServer` / `New*ServiceClient` in production tree; `api/auth.proto` … `api/settlement.proto` removed or reduced to domain-only messages; `buf.gen.grpc.yaml` deleted.

Payment fault/integration tests: `package payment_test` + `internal/paymenttest` (settlement fixture) + `internal/payment/dbtest` (migrations); no `controlplane` import from `package payment`.


10. split_control and standalone cmd/* deprecation

Status: deprecation notices only — compose profile and standalone binaries remain until callers migrate.

Default deploy: `cmd/control` modular monolith. Compose profiles `single_vps`, `ingest_only`, `network_operator` run one `control` container with in-process management, identity (auth), payment, billing, and notifier (`CONTROL_ENABLE_*`). Local entry: `scripts/dev/stack.sh single-vps`.

Deprecated:

| Item | Replacement |
|------|-------------|
| Compose profile `split_control` (`scripts/dev/stack.sh full`) | `single_vps` or `network_operator` |
| `cmd/auth`, `cmd/management`, `cmd/payment`, `cmd/billing`, `cmd/notifier` | `cmd/control` with matching `CONTROL_ENABLE_*` |

Monolith env: set `SETTLEMENT_GRPC_ENABLED=0` so payment→settlement uses in-process `domain.PaymentSettlement` instead of localhost settlement gRPC. Compose `control` service sets this; bare-metal installs should set it in `.env` when running `cmd/control`.

Standalone `Serve()` gRPC listeners (deprecated `cmd/auth`, `cmd/billing`, `cmd/payment`, `cmd/notifier`) are gated by `AUTH_GRPC_ENABLED`, `BILLING_GRPC_ENABLED`, `PAYMENT_GRPC_ENABLED`, `NOTIFIER_GRPC_ENABLED` (default on; set `0` to run HTTP sidecars and workers only). Monolith (`cmd/control`) uses `OpenModule` only — it never calls `Serve()` for those modules.

Split_control keeps network gRPC between containers (`AUTH_SERVER_HOST`, `PAYMENT_SERVER_HOST`, etc.) until the profile is removed.

Removal (later): delete `split_control` profile and standalone `cmd/*` entrypoints; drop `make proto-grpc` requirement for default self-hosted; `SETTLEMENT_GRPC_ENABLED` defaults to off.

Done when: no docs or compose paths reference `split_control`; standalone control-plane binaries removed or thin wrappers only.
