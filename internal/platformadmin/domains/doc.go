// Package domains holds custom domain health, DNS parking, Cloudflare integration, and TLS gate HTTP for platformadmin.
//
// Role:
//   - domain_health.go syncs platformconfig domain targets into domain_health_status and runs periodic probes.
//   - domain_health_handlers.go registers /api/v1/domains CRUD, park, probe, ssl/setup, and ops tls-allowed routes.
//   - domain_park.go parks hostnames into domain pools with Cloudflare DNS records.
//   - cloudflare_client.go wraps Cloudflare zone, DNS, and SSL status API calls with bounded response bodies.
//   - StartDomainHealthWorker runs probe ticks; parent platformadmin re-exports types via domains_export.go aliases.
//
// Topology:
//   - Subpackage of platformadmin; DomainHealthHost supplies PG pool, config, and optional Cloudflare client.
//   - TLS allow queries serve edge Caddy ask endpoints without admin session (token-gated when configured).
//
// Invariants:
//   - Custom domain delete limited to role=custom rows; built-in platform targets resync from platformconfig.
//   - Reputation-unsafe pool domains move to banned status (TestDomainHealth_reputationUnsafeBansPool).
//   - Cloudflare client nil when API token unset; park and ssl routes fail closed with service unavailable.
//   - IsTLSAllowed checks domain_health_status and domain pool membership before edge cert issuance.
//
// Forbidden:
//   - Hot-path tracker or ingest imports.
//   - Direct platformconfig mutation from domains package (patch flows stay in platformadmin store).
//
// Verify:
//   go list -e ./internal/platformadmin/domains/
//   go test ./internal/platformadmin/domains/ -short -run 'TestCloudflare|TestApplyReputation|TestDomainHealthTLSAllowed_tokenRequired|TestCloudflareRecordType' -count=1
//   go test ./internal/platformadmin/domains/ -short -run TestCloudflareClient_ListZones -count=1
//   go test ./internal/platformadmin/domains/ -short -run TestApplyReputationToProbe -count=1
//   go test ./internal/platformadmin/domains/ -run TestDomainHealth_markPoolDomainBanned -count=1
package domains
