# Index

Single-word index for `docs/`. Engineering stack id: **ad-event-processor** — [NAMING.md](NAMING.md).

## Start here

| Role | Read first |
| :--- | :--- |
| **Operator / on-prem** | [START.md](START.md) then [LICENSE.md](LICENSE.md) |
| **Developer** | [DEVELOPMENT.md](DEVELOPMENT.md) then [ARCHITECTURE.md](ARCHITECTURE.md), [TRADEOFFS.md](TRADEOFFS.md) |
| **Buyer / integrator** | [TRAFFIC.md](TRAFFIC.md) |
| **CI / merge gates** | [CI.md](CI.md) |

## All docs

| File | Topic |
| :--- | :--- |
| [ARCHITECTURE.md](ARCHITECTURE.md) | Topology, hot/cold path, Redis/PG/CH |
| [BILLING.md](BILLING.md) | USDT tiers, invoices, Cryptomus renewal |
| [CI.md](CI.md) | GitHub Actions, merge gates |
| [DEVELOPMENT.md](DEVELOPMENT.md) | Codegen, compose, tests, runbooks |
| [LICENSE.md](LICENSE.md) | Offline JWT license |
| [NAMING.md](NAMING.md) | Product vs stack identifiers |
| [PARSER.md](PARSER.md) | Ingress wire policy |
| [REGIONS.md](REGIONS.md) | Enterprise multi-region |
| [RTB.md](RTB.md) | OpenRTB shadow to live |
| [SHARDING.md](SHARDING.md) | Shard 0 failure matrix |
| [START.md](START.md) | Single-VPS installer |
| [TRAFFIC.md](TRAFFIC.md) | `/click`, `/track`, CAPI, macros |
| [TRADEOFFS.md](TRADEOFFS.md) | Architecture trade-offs, rejected alternatives |
| [TRIAL.md](TRIAL.md) | Pilot repeat-trial policy |
| [UI.md](UI.md) | Admin tokens, templates, anti-slop |
| [XDP.md](XDP.md) | Enterprise NIC-level XDP |

## Tooling

```bash
task scaffold -- my-service
task test-gen -- internal/my-service
bash scripts/ci/pr_fast.sh
```

SLAs: `.cursor/rules/core.mdc`, `make test-alloc-gate`.
