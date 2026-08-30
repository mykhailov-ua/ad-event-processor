// Package netintel provides GeoIP, CIDR LPM, proxy/VPN, and TLS impersonation feeds for local filters.
//
// Role:
//   - GeoProvider (MaxMind) for country and anonymous-IP lookup on filter path.
//   - CIDR block tables with RCU snapshot reload; DC ASN, residential proxy ring, moderator intel feeds.
//   - ProxyVPNConnType classification used by Geo and segment filters before UnifiedFilter.
//
// Topology:
//   - Feed loaders run on background ticks or startup; hot path reads atomic.Pointer snapshots only.
//   - Fail-closed on refresh error after first successful load; fail-open on first boot when feed missing.
//
// Invariants:
//   - RCU publish: readers see full gen A or gen B snapshot; no torn prefix tables during swap.
//   - Corrupt or missing refresh after a good snapshot retains the previous snapshot (not wiped).
//
// Forbidden:
//   - Per-request HTTP download or Postgres fetch inside FilterEngine.Check.
//
// Verify:
// go test ./internal/filter/netintel/... -short -count=1
// go test ./internal/filter/netintel/ -short -run TestCIDR_RCUSwap_ConcurrentReaders -count=1
// go test ./internal/filter/netintel/ -short -run TestCIDR_FeedRefreshFailClosed -count=1
//
// Tradeoffs:
//   - RCU atomic.Pointer snapshot vs RWMutex on filter path: readers on Tier B take one atomic load with
//     no lock; background loader publishes full gen B then swaps pointer; rejected per-lookup mutex (tail
//     latency under refresh) and rejected in-place table mutation (torn prefix reads).
//   - Fail-closed on refresh error after first successful load vs fail-open on first boot when feed missing:
//     retain last good snapshot on corrupt/partial refresh (TestCIDR_FeedRefreshFailClosed); empty boot
//     allows pass with L1 fail-open log until first good load; rejected wiping table on failed refresh
//     (false negatives on DC/proxy blocks) and rejected hard 503 on missing initial feed (dev bootstrap).
//   - Embedded mmap GeoIP (MaxMind on disk) vs external geo RPC per request: local lookup ~microseconds,
//     no network RTT on Check; rejected HTTP GeoIP service on hot path. CDN-terminated traffic: edge
//     headers supplement but tracker still owns country/anonymous-IP for filter chain after edge pass.
//   - CIDR LPM in Go vs pushing all DC/proxy prefixes to XDP: L7 filters need connection-type and segment
//     signals nginx/XDP do not have; edge drops known L3/L4 floods and synced host keys; netintel feeds
//     classify residual traffic before UnifiedFilter. Rejected duplicating full prefix set only at edge
//     (rotating proxies evade host maps per edge.mdc).
//   - Background tick reload vs pub/sub per feed row: periodic snapshot rebuild keeps hot path simple;
//     rejected LISTEN/NOTIFY into tracker for intel feeds (cold-path pattern only).
//   - Proxy/VPN/residential ring and moderator intel as separate RCU tables: independent refresh failure
//     domains; one bad feed does not clear others when prior snapshot exists.
package netintel
