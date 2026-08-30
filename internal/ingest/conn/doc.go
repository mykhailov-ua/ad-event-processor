// Package conn hosts TLS fingerprint tables and residential-intel feed loaders for ingest.
//
// Role:
//   - TLSFingerprintTable RCU snapshots for JA3/JA4 block and allow lists (sorted CRC32-IEEE hashes).
//   - JA4 browser corpus (embedded + optional dir overlay) for UA-vs-JA4 family mismatch checks.
//   - ResidentialIntelFeedLoader merges file prefixes and optional Redis intel:residential:* keys
//     into filter.ResidentialIntelTable for edge-parity conn-type signals.
//   - TLSFingerprintFeedLoader and ResidentialIntelFeedLoader run from cmd/tracker background ticks.
//
// Topology:
//   - TLSFingerprintTable.active and ja4BrowserCorpusActive are atomic.Pointer snapshots; Tier B
//     readers load once per Match*/ShouldBlock*/JA4BrowserCorpusMismatch call with no locks.
//   - Feed loaders use time.Ticker loops (TLSFingerprintFeedRefresh / ResidentialIntelFeedRefresh);
//     RefreshOnce and ReloadOnce exist for startup wiring.
//   - Blocklist files: ja3_blocklist.txt (ja3:/ja4: lines or bare JA3); allowlists ja3_allowlist.txt
//     and ja4_allowlist.txt merged for parseTLSFingerprintAllowFeed.
//   - Residential file external_residential.txt uses filter.ParseProxyVPNFeedLine; Redis keys
//     intel:residential:<ip> JSON {residential_proxy,vpn,proxy} qualify when isResidentialFarm.
//
// Invariants:
//   - Fingerprint payload length 1..tlsFingerprintMaxLen (512); empty or over-max never matches.
//   - Allowlist wins over blocklist: ShouldBlockJA3/JA4 returns false when Match*Allowed is true.
//   - BuildTLSFingerprintSnapshot sorts all four hash slices before publish; lookup is binary search.
//   - JA4BrowserCorpusMismatch is false when corpus unset, prefix unknown, or UA family unclassified.
//   - ResidentialIntelTable.PublishPrefixes only after at least one prefix; empty reload skips publish.
//
// Contracts:
//   - TLS feed line: optional ja3:/ja4: prefix; bare line treated as JA3; # comments and blank skipped.
//   - JA4 corpus line: prefix=family[,family] (chrome,firefox,safari,okhttp,go); embedded ships at init.
//   - JA4BrowserCorpusMismatchFn registered on filter package from ja4_corpus init.
//   - TLSFingerprintImpersonating and UAClaimsChromeNotChromium delegate to internal/filter.
//
// Tradeoffs:
//   - Rejected per-request file or Redis fetch on Tier B /track (background ticker reload only).
//   - Rejected string-key map for TLS lists: sorted uint32 CRC32 slices give zero-alloc binary search.
//   - Rejected RW mutex on hot readers: atomic.Pointer RCU swap after loader builds new snapshot.
//   - TLS feed refresh fail-open when table.Ready(): keep last snapshot, increment refresh error metric.
//   - TLS feed fail-closed only before first successful publish (TLSFingerprintUninitialized=1).
//   - Residential reload fail-open when table.Ready() and merged prefix list empty: retain prior LPM.
//   - Redis SCAN for residential intel runs in loader goroutine only; encoding/json not on request path.
//   - JA4 corpus: embed baseline always loaded; dir overlay merges into new snapshot on TLS feed refresh.
//
// Forbidden:
//   - Synchronous network or disk fetch inside FilterEngine.Check or Tier A gnet thread.
//   - Mutating published FingerprintSnapshot or ja4BrowserCorpusSnapshot in place after Publish.
//
// Verify:
//
//	go test ./internal/ingest/ -short -count=1
package conn
