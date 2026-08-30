// Package runtimepaths defines canonical /etc and /run paths for appliance installs and compose volumes.
//
// Role:
//   - EtcDir, RunDir, LicensePath, PostgresSocketDir, RedisSocket(shard) for operator docs and wire defaults.
//   - Legacy path aliases preserved one release for migrated installs.
//
// Layout constants:
//   - EtcDir /etc/ad-event-processor.
//   - RunDir /run/ad-event-processor.
//   - ComposeRunVolume ad_event_processor_run docker volume name.
//
// Topology:
//   - Imported by cmd installers, config loaders, and compose scripts; stdlib only.
//
// Forbidden:
//   - Import internal/* packages.
//
// Verify:
//
//	go test ./pkg/runtimepaths/... -short -count=1
package runtimepaths
