// Package traffictemplates merges integration schemas and curated sidecar metadata into the traffic-source template catalog.
//
// Role:
//   - codegen.go Generate reads deploy/schemas inbound_tokens YAML plus deploy/vendor/traffic_source_ui.yaml sidecar.
//   - cmd/codegen-traffic-templates writes internal/traffictemplates/generated_templates.json and optional admin TS export.
//   - CountBundledTrafficSchemas tracks traffic_source entries in integrationschema.BundledIntegrationTemplateCatalog.
//   - Campaign wizard and import paths validate traffic_template_id via internal/campaign click preset helpers.
//
// Topology:
//   - Build-time / CI catalog only; not loaded per /track request.
//   - Consumers: cmd/codegen-traffic-templates, campaign TemplateCatalog import, wizard traffic-source step.
//
// Invariants:
//   - Curated sidecar templates win over auto-generated rows for the same bundled_slug.
//   - direct-custom template is always appended last for manual sub_id entry.
//   - codegen -check fails when bundled slug coverage drops below catalog count or JSON is stale.
//
// Forbidden:
//   - Hot-path dynamic template fetch or Postgres reads on tracker ingest.
//   - Hand-editing generated_templates.json without rerunning codegen.
//
// Verify:
//   go test ./internal/traffictemplates/ -short -run TestGenerate_coversCatalogAndCuratedMeta -count=1
//   go run ./cmd/codegen-traffic-templates -check
package traffictemplates
