# BidShard documentation

Single-word index for `docs/`. Engineering stack id: **ad-event-processor** — [NAMING.md](NAMING.md).

## Start here

| Role | Read first |
| :--- | :--- |
| **Operator / on-prem** | [QUICKSTART.md](QUICKSTART.md) then [LICENSE.md](LICENSE.md) |
| **Developer** | [DEVELOPMENT.md](DEVELOPMENT.md) then [ARCHITECTURE.md](ARCHITECTURE.md) |
| **Buyer / integrator** | [TRAFFIC.md](TRAFFIC.md) |
| **CI / merge gates** | [DEVELOPMENT.md](DEVELOPMENT.md) section 6 |

## All docs

| File | Topic |
| :--- | :--- |
| [ARCHITECTURE.md](ARCHITECTURE.md) | Topology, hot/cold path, Redis/PG/CH, enterprise policy |
| [DEVELOPMENT.md](DEVELOPMENT.md) | Codegen, compose, tests, CI/BPF gates, runbooks |
| [QUICKSTART.md](QUICKSTART.md) | Single-VPS installer |
| [NAMING.md](NAMING.md) | BidShard vs ad-event-processor |
| [TRAFFIC.md](TRAFFIC.md) | `/click`, `/track`, CAPI, macros, DMR |
| [LICENSE.md](LICENSE.md) | Offline JWT license |
| [PARSER.md](PARSER.md) | Ingress wire policy, chaos drills |
| [SHARDING.md](SHARDING.md) | Shard 0 failure matrix |
| [RTB.md](RTB.md) | OpenRTB shadow to live |
| [XDP.md](XDP.md) | Enterprise NIC-level XDP |
| [REGIONS.md](REGIONS.md) | Enterprise multi-region |

## Tooling

```bash
task scaffold -- my-service
task test-gen -- internal/my-service
bash scripts/ci/pr_fast.sh
```

SLAs: `.cursor/rules/global/core.mdc`, `make test-alloc-gate`.
