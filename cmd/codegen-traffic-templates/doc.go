// Package main regenerates bundled traffic-source UI templates JSON.
//
// Role:
//   - Merge integrationschema catalog + deploy/vendor/traffic_source_ui.yaml sidecar.
//   - Write internal/traffictemplates/generated_templates.json.
//   - -check exits 1 when on-disk output differs (CI stale guard).
//   - Fail when bundled slug coverage is below traffictemplates.CountBundledTrafficSchemas().
//
// Topology:
//   - Build-time / make gen helper; no long-running server.
//   - Reads integrationschema.SchemaRootDir() and repo-root paths via -root (default ".").
//
// Invariants:
//   - Exit 1 on generate, coverage, marshal, or write errors.
//   - -check compares exact rendered bytes including trailing newline.
//
// Forbidden:
//   - Not a loadgen or simulation binary; does not emit HTTP traffic.
//
// Verify:
// go run ./cmd/codegen-traffic-templates
// go run ./cmd/codegen-traffic-templates -check
package main
