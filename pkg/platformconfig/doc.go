// Package platformconfig defines install.yaml platform settings schema, patch merge, and Redis render helpers.
//
// Role:
//   - Config struct maps tracking domain, ingress schema, stripe block, and edge expose flags.
//   - validate.go rejects invalid patches; render.go writes install.yaml; redis_addrs.go maps shard URLs.
//
// Topology:
//   - Imported by internal/platformadmin and pkg/domainhealth; persisted under settings key platform_config.
//
// Defaults and limits:
//   - ProfileSingleVPS default appliance profile.
//   - IngressAdEventProcessorNative and IngressOpenRTB3 canonical ingress_schema values.
//
// Invariants:
//   - Patch merge deep-merges known fields; unknown keys rejected in validate.
//   - Secrets fields omitted from JSON audit diffs at caller.
//
// Forbidden:
//   - Import internal/* packages.
//
// Verify:
//
//	go test ./pkg/platformconfig/... -short -count=1
package platformconfig
