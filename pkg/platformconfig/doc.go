// Package platformconfig defines install.yaml platform settings schema, patch merge, and compose render helpers.
//
// Role:
//   - Config and Patch model tracking domain, ingress schema, Stripe block, edge expose flags, and appliance profile.
//   - Parse, Marshal, Default, MergeDefaults load and persist settings key platform_config (JSON in PG settings KV).
//   - Patch.Apply pointer-merge updates; Validate normalizes tracking_domain and enforces profile/ingress/stripe rules.
//   - RestartRequiredFields lists config deltas that need process restart (ingress, telemetry, edge, stripe, profile).
//   - RenderComposeEnv and RenderInstallYAML emit install.compose.env and install.yaml fragments for installer.
//   - RedisAddrsForProfile and RedisAddrsUDS build comma-separated UDS paths via pkg/runtimepaths.
//   - Public, RedactConfig, MaskSecret strip secrets from admin API responses; BootstrapRequest wraps first-boot payload.
//   - NormalizeIngressSchema maps legacy ingress slugs (via pkg/naming) to IngressAdEventProcessorNative.
//
// Topology:
//   - Cold: internal/platformadmin HTTP, internal/installer apply/up, internal/controlplane platform_config reader.
//   - pkg/domainhealth ResolveHost reuse for DNS probes; pkg/branding header on generated compose env.
//   - Imports pkg/naming and pkg/runtimepaths only from pkg/*; no internal/* imports.
//
// Defaults and limits:
//   - ProfileSingleVPS default appliance profile; compose_dev allowed in Validate only (not Default()).
//   - IngressAdEventProcessorNative and IngressOpenRTB3 canonical ingress_schema values.
//   - Default currency USD, timezone UTC, telemetry_enabled true, edge_expose_click true, network_interface eth0.
//   - RedisShardCountAppliance 4 UDS shards; RedisShardCountInfra 6 reserved for future multi-host profiles.
//
// Invariants:
//   - Parse empty string returns Default(); non-empty JSON runs MergeDefaults, NormalizeIngressSchema, then Validate.
//   - Patch.Apply runs MergeDefaults and Validate on result; omitted Stripe secret fields preserve base secrets.
//   - Validate rejects edge_xdp when profile is compose_dev.
//   - Stripe.Enabled requires non-empty stripe.secret_key.
//   - default_currency must be empty or exactly three letters after trim.
//   - tracking_domain strips scheme, path, and trailing slash; stored host-only lowercase.
//   - EdgeExposeRedisSettings maps to edge_expose_click and edge_expose_openrtb string true/false for Redis sync.
//   - RenderComposeEnv always sets REDIS_ADDRS from RedisAddrsForProfile (single_vps uses four UDS shards).
//
// Tradeoffs:
//   - Pointer patch merge vs full document replace: partial admin PATCH keeps Stripe secrets without re-posting keys.
//   - NormalizeIngressSchema warns and rewrites legacy slugs vs hard error: installer YAML from old releases still boots.
//   - compose_dev blocks edge_xdp: XDP attach is unsafe in default dev compose network namespace.
//   - Fixed four-shard UDS list for single_vps vs env-driven REDIS_ADDRS: appliance layout matches runtimepaths socket layout.
//   - JSON unknown fields ignored by encoding/json: forward-compatible reads; schema evolution adds struct fields explicitly.
//
// Forbidden:
//   - Import internal/* packages.
//   - Storing stripe secrets in PublicView.Config (use RedactConfig and MaskedSecrets).
//
// Verify:
//
//	go test ./pkg/platformconfig/... -short -count=1
//	go test ./pkg/platformconfig/ -short -run TestPatchPreservesSecrets -count=1
//	go test ./pkg/platformconfig/ -short -run TestNormalizeIngressSchema -count=1
//	go test ./pkg/platformconfig/ -short -run TestRedisAddrsForProfile_singleVPSNeverSixShards -count=1
package platformconfig
