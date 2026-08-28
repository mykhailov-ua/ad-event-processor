#!/usr/bin/env python3
"""Write package doc.go files from PACKAGE_DOCS. Run: python3 scripts/dev/sync_package_doc_go.py"""

from __future__ import annotations

import os
import textwrap

ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

SKIP_DIRS = {"db", "pb", "queries", "migrate", "dbtest"}


def doc(body: str) -> str:
    text = textwrap.dedent(body).strip("\n")
    lines: list[str] = []
    for line in text.splitlines():
        stripped = line.rstrip()
        if not stripped:
            lines.append("//")
        else:
            lines.append("// " + stripped)
    if lines:
        lines.append("//")
    return "\n".join(lines) + "\n"


PACKAGE_DOCS: dict[str, str] = {}


def add(rel: str, body: str) -> None:
    PACKAGE_DOCS[rel] = doc(body)


# --- internal: hot / ingest ---

add(
    "internal/ingestion/doc.go",
    """
    Package ingestion is the tracker hot path: gnet HTTP/1-2, /track and /click,
    unified filter, stream/broker producers, OpenRTB ingress, and RTB auction glue.

    Boundaries (hard):
      - Must NOT import internal/controlplane admin handlers.
      - Must NOT import internal/fraud ML scoring (batch sidecar only).

    SLA: ad_http_request_duration_seconds p95 < 50 ms (core.mdc).

    Verify (scoped):
      go test ./internal/ingestion/ -short -count=1
      go build -o /dev/null ./cmd/tracker/

    Alloc / parser gates (when touching handlers, filters, parsers):
      make test-alloc-gate
      bash scripts/ci/escape_heap_gate.sh

    Benchmarks (examples):
      go test ./internal/ingestion/ -run='^$' -bench='BenchmarkAdsPacketHandlerProto|BenchmarkParseOpenRTB26' -benchmem -count=1

    Integration / fault (operator machine):
      make test-integration
      make test-fault
    """,
)

add(
    "internal/ingestion/traceprobe/doc.go",
    """
    Package traceprobe holds optional ingest tracing hooks used from ingestion tests
    and diagnostics. Not on the production /track SLA path unless explicitly wired.
    """,
)

add(
    "internal/rtb/doc.go",
    """
    Package rtb implements the in-process OpenRTB auction engine (catalog scan,
    RunAuction, shadow/live modes). Called from ingestion on exchange routes.

    Verify:
      go test ./internal/rtb/ -short -count=1
      go test ./internal/rtb/ -run='^$' -bench=BenchmarkAuction -benchmem -count=1

  Budget: RunAuction p99 < 15 us; catalog scan p99 < 500 candidates (core.mdc).
    """,
)

add(
    "internal/openrtb/doc.go",
    """
    Package openrtb holds OpenRTB wire types, bid response writers, and exchange
    validation helpers shared by ingestion and tests. Cold-path sized payloads only
    on admin import paths, not per /track event.
    """,
)

add(
    "internal/domain/doc.go",
    """
    Package domain is shared hot-path vocabulary: Event, sharding, budget invariants,
    object pools. No json or db struct tags here.

    Financial invariant:
      current_spend <= budget_limit (AssertBudgetInvariant in tests).

    Verify:
      go test ./internal/domain/ -short -count=1
    """,
)

# --- internal: control / admin ---

add(
    "internal/controlplane/doc.go",
    """
    Package controlplane is the modular monolith composition root for cmd/control:
    Service, outbox workers, billing hooks, and *_bridge.go wiring to domain packages.

    Domain HTTP lives in internal/campaign, brand, flow, fraudadmin, etc. Bridges
    implement domain Host/Effects ports; they must not accumulate business rules
    (modular-monolith.mdc).

    Routes: /api/v1/* on :8188. Config mutations enqueue outbox_events in the same
    PG transaction as the domain change.

    Verify:
      go test ./internal/controlplane/ -short -count=1
      bash scripts/ci/cold_path_static_gate.sh
      bash scripts/ci/anti_slop_gate.sh

    Must NOT be imported by internal/ingestion.
    """,
)

add(
    "internal/controlplane/authz/doc.go",
    """
    Package authz provides request user context and RBAC role constants for admin
    handlers. Imported by domain packages (campaign) but not by ingestion.
    """,
)

add(
    "internal/controlplane/outboxpb/doc.go",
    """
    Package outboxpb holds generated protobuf types for control-plane outbox payloads.
    Hand-editing *.pb.go is forbidden; regenerate via make proto.
    """,
)

add(
    "internal/control/doc.go",
    """
    Package control is the cmd/control module runner: starts HTTP admin, payment
    webhooks, and in-process workers from a single process (modular monolith).
    """,
)

add(
    "internal/campaign/doc.go",
    """
    Package campaign owns campaign admin HTTP, Runtime, WizardStore, editor/migration
    routes, validators, and workers. Postgres side effects are invoked through
    controlplane Effects/DeliveryHost bridges.

    Import: may use controlplane/authz; must NOT import controlplane root.

    Verify:
      go test ./internal/campaign/ -short -count=1
    """,
)

add(
    "internal/brand/doc.go",
    """
    Package brand: /api/v1/brands and brand-creative admin. Store uses brand.Host
    implemented in controlplane brand_bridge.go.
    """,
)

add(
    "internal/flow/doc.go",
    """
    Package flow: /api/v1/flows, lander path validation, flow HTTP handlers.
    Flow bandit txs live in bandit_tx.go; PG mutations wired from controlplane.
    """,
)

add(
    "internal/supply/doc.go",
    """
    Package supply: sellers and ads.txt admin (/api/v1/supply/*), chain validation.
    """,
)

add(
    "internal/marginguard/doc.go",
    """
    Package marginguard: campaign margin policies, placement blocks, CH/PG reads for
    cost-over-revenue guardrails. Host port implemented in controlplane marginguard_bridge.go.

    Verify:
      go test ./internal/marginguard/ -short -count=1
    """,
)

add(
    "internal/platformadmin/doc.go",
    """
    Package platformadmin: platform-level admin settings and store; narrow Host port
    from controlplane platform_bridge.go.
    """,
)

add(
    "internal/fraudadmin/doc.go",
    """
    Package fraudadmin: /api/v1/fraud/* routes, integration health reader, policy presets.
    Must NOT import controlplane root (import cycle).
    """,
)

add(
    "internal/billingadmin/doc.go",
    """
    Package billingadmin: /api/v1/billing/*, workspace export, composite billing reads.
    """,
)

add(
    "internal/opsadmin/doc.go",
    """
    Package opsadmin: /api/v1/ops/*, ManagementOpsReader, stack health DTOs.
    Wired via ops_reader_bridge.go from controlplane Service deps.
    """,
)

add(
    "internal/settingsadmin/doc.go",
    """
    Package settingsadmin: deployment settings admin surface and store.
    """,
)

add(
    "internal/privacyadmin/doc.go",
    """
    Package privacyadmin: privacy and compliance-related admin routes and validators.
    """,
)

add(
    "internal/admin/doc.go",
    """
    Package admin: legacy /admin static and compatibility shims. Prefer /api/v1;
    do not extend HTMX surfaces (control-plane.mdc).
    """,
)

add(
    "internal/reports/doc.go",
    """
    Package reports: report catalog, ClickHouse query handlers, export hooks, fraud scrub.
    One-way: reportjob may import reports; reports must not import reportjob.
    """,
)

add(
    "internal/reportjob/doc.go",
    """
    Package reportjob: async report jobs, schedules, PG runner, validation-job HTTP.
    """,
)

add(
    "internal/outbox/doc.go",
    """
    Package outbox: shared outbox event helpers and types used across control-plane
    workers. Authoritative apply path is controlplane OutboxWorker + PG outbox_events.
    """,
)

add(
    "internal/governance/doc.go",
    """
    Package governance: governance policy helpers for admin and audit surfaces.
    """,
)

add(
    "internal/reconciliation/doc.go",
    """
    Package reconciliation: Postgres vs Redis recon helpers and worker support code.
    """,
)

add(
    "internal/smartalerts/doc.go",
    """
    Package smartalerts: alert routing and smart alert admin integration.
    """,
)

add(
    "internal/migrationsource/doc.go",
    """
    Package migrationsource: campaign migration source adapters for import/pull flows.
    """,
)

add(
    "internal/trialregistry/doc.go",
    """
    Package trialregistry: trial license registry storage and API helpers (licensing.mdc).
    """,
)

# --- internal: cold services ---

add(
    "internal/config/doc.go",
    """
    Package config loads environment configuration for all binaries. No business logic;
    cmd/*/main.go reads config and passes structs into internal packages.
    """,
)

add(
    "internal/database/doc.go",
    """
    Package database: Postgres pool, ClickHouse client facades, and shared DB helpers
    for cold path. Hot path uses Redis/Lua via ingestion, not this package, for filters.
    """,
)

add(
    "internal/payment/doc.go",
    """
    Package payment: Stripe/USDT checkout, webhooks, payment_outbox, settlement into
    balance_ledger. Async settlement only; no sync balance mutation in webhook handler.
    """,
)

add(
    "internal/ledger/doc.go",
    """
    Package ledger: balance_ledger lines, policy types, and ledger invariant helpers.
    Financial truth is Postgres; Redis budgets are operational limits reconciled async.
    """,
)

add(
    "internal/notify/doc.go",
    """
    Package notify: Slack/email dispatch helpers for ops and billing notifications.
    Prefer function-style senders over Provider interfaces (cold-path.mdc).
    """,
)

add(
    "internal/identity/doc.go",
    """
    Package identity: users, API keys, and auth persistence for admin and self-serve.
    """,
)

add(
    "internal/fraud/doc.go",
    """
    Package fraud: cold-path ML scoring, feature reads, training hooks. Production infer
    runs in cmd/fraud-scorer and cmd/ivt-detector sidecars.

    Hard rule: internal/ingestion must NOT import this package for scoring on /track.
    """,
)

add(
    "internal/automation/doc.go",
    """
    Package automation: scheduled automation rules and worker ticks for control plane.
    """,
)

add(
    "internal/postback/doc.go",
    """
    Package postback: advertiser postback dispatch and cmd/postback-sender support code.
    """,
)

add(
    "internal/costsync/doc.go",
    """
    Package costsync: third-party cost feed sync (e.g. PopAds) into reporting tables.
    """,
)

add(
    "internal/platformsync/doc.go",
    """
    Package platformsync: external platform catalog sync workers and adapters.
    """,
)

add(
    "internal/dedup/doc.go",
    """
    Package dedup: idempotency and dedup key helpers for control-plane writes.
    """,
)

add(
    "internal/edge/doc.go",
    """
    Package edge: XDP/BPF map types, blocklist store, edge sync helpers for cmd/edge-*.
    Compliance perimeter only; not tracker hot path (compliance.mdc, edge.mdc).
    """,
)

add(
    "internal/licensing/doc.go",
    """
    Package licensing: JWT verify, entitlements, trial tier gates. Tracker reads license
    snapshot from file; no per-request license HTTP on hot path.

    Verify:
      make license-verify
      bash scripts/ci/license_verify_tier.sh
    """,
)

add(
    "internal/licensing/embedkey/doc.go",
    """
    Package embedkey: build-time embedded public keys for license verification.
    """,
)

add(
    "internal/metrics/doc.go",
    """
    Package metrics: Prometheus metric registration shared across binaries.
    Hot path: avoid per-request dynamic label values in filter loops.
    """,
)

add(
    "internal/telemetry/doc.go",
    """
    Package telemetry: optional product telemetry hooks for control and tools binaries.
    """,
)

add(
    "internal/logpipeline/doc.go",
    """
    Package logpipeline: log shipper, compactor, evacuator shared pipeline types.
    """,
)

add(
    "internal/loadreport/doc.go",
    """
    Package loadreport: load test report parsing and aggregation for cmd/load-report.
    """,
)

add(
    "internal/installer/doc.go",
    """
    Package installer: release installer asset layout for cmd/installer.
    """,
)

add(
    "internal/integrationschema/doc.go",
    """
    Package integrationschema: JSON schema validation for integration configs.
    """,
)

add(
    "internal/openapi/doc.go",
    """
    Package openapi: OpenAPI bundle load and route metadata for admin and export tools.
    """,
)

add(
    "internal/openapivalidate/doc.go",
    """
    Package openapivalidate: validates OpenAPI request/response shapes in CI and tests.
    """,
)

add(
    "internal/traffictemplates/doc.go",
    """
    Package traffictemplates: traffic template codegen inputs for cmd/codegen-traffic-templates.
    """,
)

add(
    "internal/testutil/doc.go",
    """
    Package testutil: testcontainers and integration test helpers.

    Test-only: not imported from production code paths. Integration tests must use
    testing.Short() skip with integration: prefix (anti-slop.mdc).
    """,
)

# --- pkg ---

add(
    "pkg/coldpath/doc.go",
    """
    Package coldpath: shared cold HTTP helpers (ReadLimitedBody, DecodeRequestOrBadRequest,
    UUID parse). DefaultMaxBody = 64 KiB. No internal/* imports.

    Verify:
      go test ./pkg/coldpath/ -short -count=1
      bash scripts/ci/cold_path_json_gate.sh
    """,
)

add(
    "pkg/broker/doc.go",
    """
    Package broker: mmap WAL broker wire protocol root. Subpackages: client, consumer,
    protocol, log. Daemon server lives in internal/broker (cmd/broker).

    Verify:
      bash scripts/ci/pkg_boundary_gate.sh
      go build -o /dev/null ./cmd/broker/
    """,
)

add(
    "pkg/broker/client/doc.go",
    """
    Package client: broker producer client used from ingestion for live/shadow ingest.
    """,
)

add(
    "internal/broker/doc.go",
    """
    Package broker: gnet broker daemon (cmd/broker). Wire client/protocol in pkg/broker.
    """,
)

add(
    "pkg/broker/consumer/doc.go",
    """
    Package consumer: broker consumer groups draining to ClickHouse processor path.
    """,
)

add(
    "pkg/broker/protocol/doc.go",
    """
    Package protocol: broker frame codec and topic constants.
    """,
)

add(
    "pkg/broker/log/doc.go",
    """
    Package log: broker WAL segment format and replay helpers.
    """,
)

add(
    "pkg/faultproof/doc.go",
    """
    Package faultproof: structured fault_test telemetry logging (fault_proof gap=open/closed).
    """,
)

add(
    "pkg/branding/doc.go",
    "Package branding: white-label host and asset path helpers for admin and track.",
)

add(
    "pkg/clientip/doc.go",
    "Package clientip: trusted-proxy aware client IP extraction helpers.",
)

add(
    "pkg/dedupkey/doc.go",
    "Package dedupkey: canonical deduplication key hashing for admin idempotency.",
)

add(
    "internal/doctor/doc.go",
    """
    Package doctor: host and dependency health probes for cmd/operator and /api/v1/ops/doctor.
    """,
)

add(
    "pkg/domainhealth/doc.go",
    "Package domainhealth: DNS/TLS reachability probes for integration health.",
)

add(
    "pkg/gnetutil/doc.go",
    "Package gnetutil: gnet listener and buffer tuning shared by tracker.",
)

add(
    "pkg/httpresponse/doc.go",
    "Package httpresponse: pre-sized HTTP error bodies for cold admin handlers.",
)

add(
    "pkg/iogate/doc.go",
    "Package iogate: concurrency gates for Postgres/ClickHouse cold writers.",
)

add(
    "pkg/landerhost/doc.go",
    "Package landerhost: hosted lander URL host resolution for flow delivery.",
)

add(
    "pkg/legal/doc.go",
    "Package legal: legal snippet templates for admin and embed surfaces.",
)

add(
    "pkg/lifecycle/doc.go",
    "Package lifecycle: graceful shutdown coordination for long-running binaries.",
)

add(
    "pkg/logger/doc.go",
    "Package logger: slog setup and shard-aware logging for services.",
)

add(
    "pkg/moderatorintel/doc.go",
    "Package moderatorintel: residential/moderator intel API client types.",
)

add(
    "pkg/money/doc.go",
    "Package money: micro-unit money parse/format without float on hot paths.",
)

add(
    "pkg/naming/doc.go",
    "Package naming: legacy name guard helpers for CI scripts.",
)

add(
    "pkg/netaddr/doc.go",
    "Package netaddr: listen address normalization for cmd/*/main.",
)

add(
    "internal/pgfailover/doc.go",
    """
    Package pgfailover: Postgres primary/replica failover for tracker and controlplane shards.
    """,
)

add(
    "pkg/piihash/doc.go",
    "Package piihash: salted PII hashing for logs and CH columns.",
)

add(
    "pkg/platformconfig/doc.go",
    "Package platformconfig: platform feature flag file loader.",
)

add(
    "pkg/proxyupstream/doc.go",
    "Package proxyupstream: click proxy upstream URL builder for ingestion.",
)

add(
    "pkg/runtimepaths/doc.go",
    "Package runtimepaths: var/ and config path resolution for local dev.",
)

add(
    "pkg/supportbundle/doc.go",
    "Package supportbundle: operator support bundle tar layout.",
)

add(
    "pkg/regionproxy/client/doc.go",
    "Package client: region-proxy uplink client for multi-region enterprise.",
)

add(
    "internal/regionproxy/doc.go",
    """
    Package regionproxy: gnet region-proxy daemon (cmd/region-proxy). Client/wal/keygen in pkg/regionproxy.
    """,
)

add(
    "pkg/regionproxy/keygen/doc.go",
    "Package keygen: region-proxy signing key generation utilities.",
)

add(
    "pkg/regionproxy/opkey/doc.go",
    "Package opkey: operator key derivation for region-proxy auth.",
)

add(
    "pkg/regionproxy/quorum/doc.go",
    "Package quorum: quorum vote helpers for region-proxy failover.",
)

add(
    "pkg/regionproxy/uplink/doc.go",
    "Package uplink: region-proxy bidirectional sync stream.",
)

add(
    "pkg/regionproxy/wal/doc.go",
    "Package wal: region-proxy write-ahead log for cross-region replay.",
)

# --- cmd (package main) ---

CMD_DOCS = {
    "tracker": """
        Binary tracker: gnet ingest on /track, /click, OpenRTB, filters, stream/broker.

        Build:
          go build -o bin/tracker ./cmd/tracker/

        Verify (scoped):
          go test ./internal/ingestion/ -short -count=1
          make test-alloc-gate
    """,
    "processor": """
        Binary processor: consumes Redis streams or broker and writes ClickHouse / PG sinks.
    """,
    "control": """
        Binary control: modular monolith admin :8188, payment webhooks :8187, workers.
        Entry: internal/control module runner.
    """,
    "broker": "Binary broker: mmap WAL ingest broker (internal/broker).",
    "fraud-scorer": "Binary fraud-scorer: batch ML scoring sidecar (cold path only).",
    "ivt-detector": "Binary ivt-detector: IVT batch detector; pauses when outbox backlog high.",
    "campaign-shard": "Binary campaign-shard: Redis campaign config shard service for tracker.",
    "postback-sender": "Binary postback-sender: async advertiser postback worker.",
    "region-proxy": "Binary region-proxy: enterprise multi-region sync (regions.mdc).",
    "edge-xdp": "Binary edge-xdp: XDP flood drop program loader.",
    "edge-bpf-sync": "Binary edge-bpf-sync: sync Redis blocklists into BPF maps.",
    "log-shipper": "Binary log-shipper: ship structured logs off-host.",
    "log-compactor": "Binary log-compactor: compact log segments.",
    "log-evacuator": "Binary log-evacuator: evacuate logs under disk pressure.",
    "loadgen": "Binary loadgen: HTTP load generator for parser and ingest drills.",
    "load-report": "Binary load-report: summarize load test artifacts under var/.",
    "dlq": "Binary dlq: dead-letter queue inspector and replay tool.",
    "admin": "Binary admin: slim admin static server when split from control.",
    "operator": "Binary operator: CLI operator tasks against control API.",
    "installer": "Binary installer: package release artifacts for operators.",
    "license-issue": "Binary license-issue: sign deployment JWT licenses.",
    "trial-registry": "Binary trial-registry: trial tenant registry service.",
    "vendor-trial-bot": "Binary vendor-trial-bot: trial automation bot.",
    "license-asset-seal": "Binary license-asset-seal: seal release assets.",
    "migrate-cold-path": "Binary migrate-cold-path: one-shot cold schema migration helper.",
    "ml-validate": "Binary ml-validate: offline model validation.",
    "ml-replay": "Binary ml-replay: replay fraud features against a model file.",
    "openapi-export": "Binary openapi-export: export bundled OpenAPI for CI.",
    "codegen-traffic-templates": "Binary codegen-traffic-templates: regenerate traffic templates.",
    "patch-vtproto-hotpath": "Binary patch-vtproto-hotpath: post-process vtproto for hot path.",
    "perf-gate": "Binary perf-gate: local perf gate runner.",
    "bpf-collector": "Binary bpf-collector: BPF stats collector for load tests.",
    "alertmanager-telegram": "Binary alertmanager-telegram: Alertmanager to Telegram bridge.",
}

for cmd, body in CMD_DOCS.items():
    add(f"cmd/{cmd}/doc.go", body)


def package_name(path: str) -> str:
    if path.startswith("cmd/"):
        return "main"
    parts = path.split("/")
    return parts[-2]


def write_doc(rel_path: str, content: str) -> None:
    full = os.path.join(ROOT, rel_path)
    pkg = package_name(rel_path)
    if not content.strip().endswith(f"package {pkg}"):
        content = content.rstrip() + f"\npackage {pkg}\n"
    os.makedirs(os.path.dirname(full), exist_ok=True)
    with open(full, "w", encoding="utf-8") as f:
        f.write(content)


def main() -> None:
    written = 0
    for rel, content in sorted(PACKAGE_DOCS.items()):
        write_doc(rel, content)
        written += 1
        print(rel)
    print(f"wrote {written} doc.go files")


if __name__ == "__main__":
    main()
