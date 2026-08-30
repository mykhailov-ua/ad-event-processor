// Package integrationschema validates bundled traffic-source and affiliate integration
// JSON/YAML schemas and builds tracking URLs from parsed documents.
//
// Role:
//   - ParseDocument and kind-specific validators (schema.go) for inbound tokens, outbound
//     postback, affiliate receive postback, and status_mapping feeds.
//   - BundledIntegrationTemplateCatalog and LoadBundledTemplate (catalog.go) read deploy/schemas
//     templates; SchemaRootDir resolves install root or repo deploy/schemas.
//   - URL builders: BuildInboundTrackingURL, BuildAffiliateReceivePanelURL, OutboundURLTemplateFromBody.
//   - Consumers: internal/campaign TemplateCatalog import paths; cmd/codegen-traffic-templates.
//
// Topology:
//   - Cold-path library; no HTTP routes or Postgres in package root.
//   - YAML bundled templates parsed to JSON-shaped structs; strict JSON decode for API bodies.
//
// Invariants:
//   - decodeStrict uses json.Decoder.DisallowUnknownFields on strict API decode paths.
//   - Single JSON object per document; trailing JSON rejected after first object.
//   - Version field required on versioned schema kinds (InboundTokensSchema version 1, etc.).
//   - MaxBodyBytes 64 KiB; MaxURLTemplate follows postback.MaxRenderedURLLen; token/name length caps.
//
// Schema kinds (Kind):
//   - inbound_tokens, outbound_postback, affiliate_receive_postback, status_mapping.
//
// Forbidden:
//   - Tracker hot path (internal/ingest) imports.
//
// Verify:
//
//	go list -e ./internal/integrationschema/...
//	go test ./internal/integrationschema/ -short -run TestIntegrationSchema_InvalidUnknownField -count=1
//	go test ./internal/integrationschema/ -short -run TestBundledIntegrationTemplateCatalog_parse -count=1
//	go test ./internal/integrationschema/ -short -run TestIntegrationSchema_ParseBundledYAML -count=1
package integrationschema
