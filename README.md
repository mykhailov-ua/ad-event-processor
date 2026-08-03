# BidShard (eSPX)

The ultimate self-hosted ad tracking, event ingestion, and real-time bidding (RTB) platform designed for Ad Networks, Media Buying Teams (Arbitrageurs), and Publishers in the 2026 programmatic landscape.

BidShard is a private, high-performance alternative to expensive SaaS trackers and ad servers. By hosting the platform on your own infrastructure, you gain complete data ownership, eliminate volume-based monthly fees, and achieve sub-80ms tracking latencies.

---

## Why BidShard in 2026?

### For Media Buying (Arbitrage) Teams
- **Server-Side Click Redirects**: Use `GET /click` on your tracking domain for one-hop `302` redirects to landers. Filters, budget, and fraud run before the redirect; `gclid`, `ttclid`, and custom sub-IDs pass through to the offer URL.
- **Zero-Latency Tracking**: Slow redirects kill conversions. BidShard processes tracking requests in under 80 milliseconds (p99), ensuring your users reach landers instantly without traffic loss.
- **Real-Time Budget Protection**: Traditional trackers suffer from budget "afterburn" - continuing to spend money minutes after a campaign is paused. BidShard uses atomic Redis locks to stop campaign spending instantly the millisecond a budget limit is reached.
- **Privacy Sandbox & Cookie-less Readiness**: Native adaptors for Google Privacy Sandbox (Topics API, Protected Audience API) and first-party signal enrichment. Track conversions accurately without third-party cookies.
- **Advanced Bot & Fraud Filtering**: Filter out non-human traffic, residential proxies, and search crawlers before they hit your landing pages. Save your budget for real users.
- **100% Spy-Proof Data Privacy**: SaaS trackers can spy on your profitable campaign angles, landers, and offers. With a self-hosted instance, your campaign data is entirely private and secure.

### For Ad Networks & Publishers
- **OpenRTB 2.6 exchange**: `POST /openrtb/bid` on tracker for SSP partners (display + video, PMP deals, shadow→live). See [RTB production runbook](docs/RTB_PRODUCTION_RUNBOOK.md).
- **ML-Driven Traffic Scoring**: Run offline and near-real-time batch analysis using LightGBM and ONNX Isolation Forest models to score traffic quality, update blocklists, or execute silent "ghost" invalid traffic (IVT) drop actions.
- **Supply Path Optimization (SPO)**: Integrated `sellers.json` and SupplyChain object support to provide full transparency to buyers and attract premium demand.
- **CTV & Live Event Handling**: Optimized for high-concurrency environments, supporting Concurrent Streams API for large-scale live events.
- **Integrated Payment Gateways**: Accept deposits automatically via Stripe (credit cards) or directly via cryptocurrency (USDT ERC-20/TRC-20) with automated ledger updates and fraud holds.
- **Unmatched Infrastructure Savings**: Process billions of ad events monthly on standard bare-metal servers. Eliminate five-figure SaaS bills and scale your business profitably.

---

## BidShard vs. SaaS Trackers

| Feature | BidShard (Self-Hosted) | Typical SaaS Tracker |
| :--- | :---: | :---: |
| **Monthly Cost** | **Flat hosting fee** (independent of volume) | **Volume-based** (scales exponentially with traffic) |
| **Data Privacy** | **100% Private** (hosted on your own servers) | **Shared** (SaaS providers can view your setups) |
| **Budget Protection** | **Instant (Atomic)** (zero overspend) | **Delayed** (leads to budget overruns) |
| **2026 Privacy Compliance** | **Full (Sandbox/DCR)** | **Limited / Third-party dependent** |
| **RTB Support** | **Built-in (OpenRTB 2.6 exchange + in-process auction)** | **None / Basic** |
| **Click Redirect (`GET /click`)** | **Built-in (302, macro + passthrough)** | **Volume-priced add-on** |

---

## Core Features

- **High-Volume Ingestion**: Built on a custom epoll-based network engine (`gnet`) to handle hundreds of thousands of requests per second without breaking a sweat.
- **Local Quanta & Full-Skip Ingestion**: Trackers reserve budget quotas in local memory to perform local atomic debits. Under Full-Skip mode, the hot path bypasses synchronous Redis network round-trips (RTT) entirely, executing local CAS debits and in-memory idempotency checks before writing asynchronously to streams.
- **Multi-Lane Sharding & Fallbacks**: Shards campaigns based on CRC32 Castagnoli hash, supports high-volume campaign multi-lane sub-sharding (splitting budgets into 4 parallel lanes to avoid Redis single-thread bottlenecks), and provides triple-shard redundancy (Primary A, Primary B, and Reserve) with automatic circuit-breaker fallback.
- **Telegram Mini App Integration**: Built-in edge-proxy and anti-fraud layer for Telegram Mini App and bot traffic. Validates `initData` HMAC/Ed25519 signatures, maps users, runs tracking redirects via `GET /tg/click`, and provides specialized performance reports (KPIs, funnels, premium breakdowns).
- **Click Redirect (`GET /click`)**: Server-side `302` redirects for arbitrage and affiliate traffic. Runs the same `FilterEngine` as `POST /track`, resolves brand creative landing URLs with macros (`{click_id}`, `{sub1}`-`{sub5}`, `{user_id}`), and forwards attribution query parameters (`gclid`, `ttclid`, UTM) to the destination.
- **Atomic Budgeting**: Real-time budget tracking, frequency capping, and pacing executed directly inside Redis memory.
- **eBPF/XDP Network Protection**: Block malicious bots and DDoS attacks directly at the network card level, saving CPU resources for clean traffic.
- **Transactional Ledger**: A double-entry accounting system stores all advertiser balances in micro-units, preventing rounding errors and financial discrepancies.
- **Columnar Analytics**: Powered by ClickHouse for lightning-fast reporting over billions of raw events.

---

## Documentation for Engineers

If you are a developer, system administrator, or DevOps engineer looking to deploy, configure, or modify BidShard, please refer to our technical documentation:

- **[Quick Start (single VPS install)](docs/QUICKSTART.md)**: Interactive installer script, platform config bootstrap, and Doctor API.
- **[RTB production runbook](docs/RTB_PRODUCTION_RUNBOOK.md)**: OpenRTB 2.6 shadow→live, reconcile export, CH retention.
- **[System Architecture & Data Flow](docs/ARCHITECTURE.md)**: Deep dive into the network topology, Redis sharding, PostgreSQL ledger, ClickHouse spooling, and the request lifecycle.
- **[Development & Deployment Guide](docs/DEVELOPMENT.md)**: Step-by-step instructions for local environment setup, code generation, Docker Compose profiles, testing, and multi-region deployment.
