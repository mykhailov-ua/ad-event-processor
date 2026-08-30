// Package provider holds per-network HTTP clients for costsync ad spend report ingestion.
//
// Role:
//   - FetchNetworkCosts in fetch.go dispatches by network string to vendor fetch*Costs implementations.
//   - oauth_meta.go, oauth_google.go, oauth_networks.go refresh OAuth tokens (Meta, Google, TikTok, Snapchat, LinkedIn, Pinterest, TrafficStars, Revcontent, Mondiad, Microsoft Ads).
//   - stat_rows.go and http_response.go parse vendor JSON/CSV with ReadLimitedBody caps (pkg/coldpath.DefaultMaxBody).
//   - token_mapping.go parses placement-field and attribution-mode settings on credentials.
//   - types.go defines CostLine, Credential, LineType (spend/revenue), and TokenMapping shared with parent costsync via type aliases.
//
// Supported networks (FetchNetworkCosts switch):
//
//	facebook, taboola, outbrain, google, tonic_rsoc, system1_rsoc, tiktok, propellerads, mgid, adsterra,
//	exoclick, hilltopads, clickadu, popads, revcontent, microsoft_ads, snapchat, linkedin, pinterest,
//	trafficstars, richads, galaksion, mondiad, juicyads, evadav.
//
// Topology:
//   - Subpackage of costsync; no HTTP routes. Called only from costsync.Worker syncCredentialRun.
//   - RSOC providers (tonic_rsoc, system1_rsoc) read revenue/EPC fixtures in parent costsync tests.
//   - OAuth app credentials passed from costsync.OAuthConfig at worker construction (internal/control/run.go).
//
// Invariants:
//   - API credentials supplied by caller after PG decrypt; never logged in fetch paths.
//   - Unsupported network returns error from FetchNetworkCosts (no silent empty import).
//   - Vendor HTTP responses size-limited before JSON decode.
//   - Pagination and vendor rate limits handled inside each provider_* file; worker applies overall sync timeout.
//
// Forbidden:
//   - Hot-path ingest imports.
//   - Direct PG/CH writes (parent costsync persists rows).
//
// Verify:
//
//	go list -e ./internal/costsync/provider/
//	go test ./internal/costsync/provider/ -short -count=1
//	go test ./internal/costsync/provider/ -short -run TestFetchFacebookCosts_Httptest -count=1
//	go test ./internal/costsync/ -short -run TestTikTokCostSync_integration -count=1
package provider
