# docs

Human-readable architecture and operator guides. **Engineering constraints live in `.cursor/rules/`** — do not duplicate agent rule prose here.

---

## Documents

| File | Audience | Content |
| :--- | :--- | :--- |
| [ARCHITECTURE.md](ARCHITECTURE.md) | Engineers | Hot/cold boundary, topology, ports, Redis sharding, wire policy |
| [DEVELOPMENT.md](DEVELOPMENT.md) | Engineers | Clone bootstrap, codegen, compose profiles, test tiers, OpenAPI workflow, licensing ops |
| [INTEGRATIONS.md](INTEGRATIONS.md) | Operators | Cost Sync, CAPI, traffic templates, platform sync, migration import |
| [AUTO_OPTIMIZATION.md](AUTO_OPTIMIZATION.md) | Engineers | Traffic weight rules (CR/EPC/ROI/revenue), test tiers, hot-path SLA |
| [COMMAND_PALETTE.md](COMMAND_PALETTE.md) | Engineers | Admin Ctrl+K global search, API + UI rollout, cold-path SLA |

---

## Related maps (by directory)

| Path | File |
| :--- | :--- |
| Runtime configs | [deploy/DEPLOY.md](../deploy/DEPLOY.md) |
| Binaries | [cmd/CMD.md](../cmd/CMD.md) |
| Go packages | [internal/INTERNAL.md](../internal/INTERNAL.md) |
| Shared libs | [pkg/PKG.md](../pkg/PKG.md) |
| Contracts | [api/API.md](../api/API.md) |
| Automation | [scripts/SCRIPTS.md](../scripts/SCRIPTS.md) |
| Test tiers | [tests/TESTS.md](../tests/TESTS.md) |
| ML training | [model/MODEL.md](../model/MODEL.md) |
| CI | [WORKFLOWS.md](../.github/workflows/WORKFLOWS.md) |

---

## Vendor / commercial

| Path | File |
| :--- | :--- |
| Vendor index | [deploy/vendor/VENDOR.md](../deploy/vendor/VENDOR.md) |
| Buyer features | [deploy/vendor/MARKETING.md](../deploy/vendor/MARKETING.md) |
| Internal sales | [deploy/vendor/SALES.md](../deploy/vendor/SALES.md) |
| Fraud ops | [deploy/vendor/ANTIFRAUD.md](../deploy/vendor/ANTIFRAUD.md) |
| SKU limits | [deploy/vendor/sku.yaml](../deploy/vendor/sku.yaml) |

---

## Reading order for new engineers

1. [ARCHITECTURE.md](ARCHITECTURE.md) — hot/cold split and topology.
2. [internal/INTERNAL.md](../internal/INTERNAL.md) — package map and hot path rules.
3. [DEVELOPMENT.md](DEVELOPMENT.md) — bootstrap and verify commands.
4. [deploy/DEPLOY.md](../deploy/DEPLOY.md) — run the stack locally.
5. `.cursor/rules/hot-path.mdc` — detailed ingest constraints when editing tracker.

---

## What belongs here vs rules

| Put in `docs/` | Put in `.cursor/rules/` |
| :--- | :--- |
| Topology diagrams, port tables | SLA ceilings, CI gate names |
| Operator integration steps | Agent verification policy |
| OpenAPI workflow how-to | Import matrix, banned patterns |
| Compose profile descriptions | Anti-slop lie modes |

---

## Maintenance

- Update `ARCHITECTURE.md` when ports, topology, or store roles change.
- Update `INTEGRATIONS.md` when network count, API paths, or explicit non-goals change.
- Update `DEVELOPMENT.md` when bootstrap or Makefile targets change.
- Keep [deploy/clickhouse/init.sql](../deploy/clickhouse/init.sql) and architecture DDL section in sync.

---

## Verification

Docs are not gated directly — code gates enforce claims:

```bash
bash scripts/ci/naming/antifraud_doc.sh   # ANTIFRAUD.md vs code
bash scripts/ci/pr_fast.sh
```

Doc-only changes do not require compile; doc+code claims must match (`anti-slop.mdc`).
