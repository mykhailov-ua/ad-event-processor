// Package domainhealth probes tracking domain TLS, DNS, and HTTP health for operator dashboards.
//
// Role:
//   - Probe runs timed HEAD against tracking or admin health paths; classifies healthy/degraded/down and SSL state.
//   - ReputationChecker queries Google Safe Browsing and Facebook Graph scrape when API keys configured.
//   - NormalizeRole maps admin/custom/tracking roles for probe path selection.
//
// Topology:
//   - Called from opsadmin/platform readers; uses pkg/branding and pkg/platformconfig for URL defaults.
//   - Outbound HTTP from cold path only; must not run on tracker /track request path.
//
// Defaults and limits:
//   - probeTimeout 10 s; TLS dial timeout 5 s.
//   - degradedLatencyMs 2000 (HTTP 2xx/3xx still degraded when slower).
//   - sslExpiringHorizon 14 days (SSLExpiring forces HealthDegraded even when HTTP succeeds).
//   - reputationTimeout 8 s; reputationMaxJSONBytes 1 MiB per vendor response.
//
// Invariants:
//   - Probe respects caller context deadline layered on probeTimeout client timeout.
//   - Probe uses HEAD (not GET); redirects not followed (CheckRedirect returns ErrUseLastResponse).
//   - TLS verification enabled (InsecureSkipVerify false); probeTLS failure maps SSL to missing unless cert parsed.
//   - classifyHTTP: 2xx/3xx healthy unless latency >= degradedLatencyMs; 4xx/5xx down.
//   - SSL expiring or expired downgrades overall health from healthy to degraded with detail suffix.
//   - NewReputationChecker returns nil when both SafeBrowsing and Facebook tokens empty (Check is no-op).
//   - Reputation Check returns error on vendor HTTP/decode failure (fail closed); nil checker returns false, no error.
//   - Safe Browsing runs before Facebook when both configured; first flagged vendor wins.
//
// Tradeoffs:
//   - HEAD health probe vs GET (minimal bytes; misses body-only failure modes).
//   - SSL probe separate TCP/TLS dial before HTTP (extra RTT; cert expiry visible even when HTTP path fails).
//   - Fail-closed reputation API errors vs treat unknown as safe (operator sees probe error, not silent allow).
//   - Facebook graph blocked inferred from 4xx body keywords when JSON decode fails (vendor-specific heuristic).
//
// Forbidden:
//   - Import internal/* packages.
//
// Verify:
//
//	go test ./pkg/domainhealth/... -short -count=1
//	go test ./pkg/domainhealth/ -short -run TestClassifySSL|TestReputationChecker -count=1
package domainhealth
