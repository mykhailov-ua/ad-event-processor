# Real-Time Bidding (RTB) Engine

BidShard features a high-performance, in-process RTB engine designed for the 2026 programmatic landscape. It provides a private, self-hosted alternative to expensive SaaS ad servers, giving you complete data ownership and sub-millisecond auction latencies.

## 1. Executive Summary for Media Buyers

For media buying teams and arbitrageurs, BidShard’s RTB engine is built to maximize ROI by eliminating the "SaaS tax" and providing unparalleled speed.

*   **Zero Volume-Based Fees**: Unlike SaaS trackers that charge more as you scale, BidShard is self-hosted. You process billions of bid requests for the flat cost of your hosting.
*   **Total Data Privacy**: Your winning angles, PMP deals, and bidding strategies never leave your infrastructure. No SaaS provider can "spy" on your profitable setups.
*   **Instant Budget Protection**: Our engine uses atomic Redis locks to stop spending the millisecond a budget limit is reached, eliminating the "afterburn" common in other platforms.
*   **High-Fidelity Fraud Defense**: Integrated ML-scoring and pre-bid IVT filtering ensure you only bid on high-quality human traffic, saving your budget for real conversions.
*   **PMP & Direct Deals**: Secure premium inventory through Private Marketplace (PMP) deals with full support for Deal IDs and floor price enforcement.

## 2. Technical Feature List (Engineering Deep-Dive)

For engineers, BidShard provides a zero-alloc, low-latency auction environment that scales horizontally across bare-metal or cloud instances.

### 2.1 Performance & Scale
*   **Sub-15µs Auction Latency**: The core `RunAuction` logic executes in under 15 microseconds (p99) on standard hardware, leaving maximum headroom for network I/O.
*   **Zero-Allocation Path**: The hot path for parsing OpenRTB 2.6 and running the auction produces 0 heap allocations, minimizing GC pressure and Stop-The-World pauses.
*   **DFA-Based Parsing**: High-speed, security-hardened DFA parsers for OpenRTB JSON and Protobuf ingress, with strict scan budgets (`ORTB_SCAN_MAX_BYTES`) to prevent CPU exhaustion attacks.
*   **Structure of Arrays (SoA) Catalog**: Campaign and creative metadata are materialized into parallel slices (SoA) for cache-friendly linear scans and early-exit optimization.

### 2.2 OpenRTB & Integration
*   **OpenRTB 2.6 Exchange**: A dedicated `POST /openrtb/bid` endpoint for SSP/Exchange partners, supporting Display and Video (VAST/VPAID) formats.
*   **Dual-Path RTB**:
    *   **Direct Ingest**: Add RTB auctions to your standard `/track` endpoints via `RTB_MODE=live`.
    *   **Exchange Path**: Use the tracker as a standalone OpenRTB exchange for external demand partners.
*   **Flexible Budget Authority**:
    *   `redis`: High-speed Lua-based budgeting (default).
    *   `rtb`: In-process CAS (Compare-and-Swap) for ultra-low latency budget debits, bypassing Redis round-trips.
*   **Standard Macro Support**: Full support for OpenRTB macros (`{AUCTION_ID}`, `{CLICK_URL}`, `{PRICE}`, etc.) in creative and tracking URLs.

### 2.3 Smart Targeting & Filtering
*   **Multi-Dimensional Targeting**: In-process targeting for Geo (MaxMind), Device, Category (BCAT), and custom segments.
*   **ML-Driven Boosts**: Injects fraud scores and quality signals from offline ML models (LightGBM/ONNX) directly into the auction ranking logic.
*   **Pre-Bid IVT Rejection**: Automatically rejects datacenter, proxy, and known bot IPs before the auction even begins, saving CPU cycles and partner QPS.
*   **Frequency Capping & Pacing**: Real-time frequency cap enforcement and daily pacing snapshots to smooth out delivery over 24 hours.

### 2.4 Operational Excellence
*   **Tie-Breaking & Clearing**: Sophisticated tie-breaking (Weight > Bid) and configurable clearing modes (`first-price`, `second-price`, or `reserve-only`).
*   **Snapshot Recovery**: Auction state and catalogs can be snapshotted to disk (v4 wire format) for rapid recovery and shard consistency.
*   **Shadow Mode (`shadow`)**: Run the auction in production, log the winners, and analyze performance without spending live budget or returning bids.
*   **Shadow-Diff Analysis**: Compare shadow auction results against live data to tune floors and targeting before going live.
*   **Async ClickHouse Logging**: All auction outcomes, bid requests, and exchange errors are logged asynchronously to ClickHouse for massive-scale analytics.
*   **Reconcile API**: Export transaction-level logs for partner reconciliation and discrepancy resolution.
*   **Doctor API Integration**: Real-time health checks for the RTB catalog, budget store, and exchange QPS throttlers.

## 3. Getting Started

*   **Deployment**: See the [Quick Start Guide](QUICKSTART.md).
*   **Operations**: Refer to the [RTB Production Runbook](RTB_PRODUCTION_RUNBOOK.md) for shadow→live promotion and reconciliation.
*   **Security**: Review [Parser Security](PARSER_SECURITY.md) for ingress hardening details.

## 4. Admin UI (onboarding)

| Route | Purpose |
| :--- | :--- |
| `/rtb/integration` | Integration profile, `POST /openrtb/bid` URL + copy, edge expose hint, read-only `RTB_MODE`, validate-bid smoke fixture, shadow-diff summary |
| `/rtb/deals` | PMP deal list (API wired; full CRUD UI backlog — see [`.cursor/MILESTONE.md`](../.cursor/MILESTONE.md) §1.3) |
| Campaign → **Integration** | `/track` RTB_MODE readout vs exchange endpoint; distinct from OpenRTB partner path |

Enable optional edge path: **Platform settings → Expose OpenRTB bid endpoint on edge** — [TRAFFIC_INTEGRATION.md §5](TRAFFIC_INTEGRATION.md#5-enabling-optional-edge-paths).

**APIs without dedicated UI yet:** `POST /api/v1/rtb/floors/apply`, `GET /api/v1/rtb/reconcile/export` — use CLI/runbook until UI ships.
