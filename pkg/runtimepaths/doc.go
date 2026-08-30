// Package runtimepaths defines canonical /etc and /run paths for appliance installs and compose volumes.
//
// Role:
//   - Constants and helpers for EtcDir, RunDir, license/secrets paths, and unix socket locations.
//   - Legacy path aliases (LegacyEtcDir, LegacyRunDir, LegacyComposeVolume) preserved one release for migrated installs.
//
// Layout constants:
//   - EtcDir /etc/ad-event-processor; RunDir /run/ad-event-processor.
//   - ComposeRunVolume ad_event_processor_run docker volume name.
//   - ContainerLicense equals LicensePath (EtcDir/license.jwt).
//   - Sockets: PostgresSocketDir, RedisSocket(shard), BrokerGnetSocket, BrokerHealthSocket, RegionProxyGnetSocket, RegionProxyHealthSocket, ClickHouseNativeSocket, ControlHTTPSocket, TrackerSocket(instance).
//
// Topology:
//   - Imported by cmd installers, cmd/region-proxy flags, internal/config env defaults, internal/installer, pkg/platformconfig.
//   - Stdlib only; no runtime file creation in this package.
//
// Invariants:
//   - All returned paths are absolute under EtcDir or RunDir; RedisSocket and TrackerSocket format shard/instance suffixes deterministically.
//   - Legacy* constants map to prior stack slug paths for symlink/migration docs only; new installs use EtcDir/RunDir.
//
// Tradeoffs:
//   - Central path table avoids drift between systemd units, compose mounts, and Go dial defaults; operators override via flags/env at wire sites, not here.
//
// Forbidden:
//   - Import internal/* packages.
//
// Verify:
// go test ./pkg/runtimepaths/... -short -run TestRuntimePaths -count=1
package runtimepaths
