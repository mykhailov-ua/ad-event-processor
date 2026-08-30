// Package gnetutil provides shared gnet connection idle and max-lifetime policies.
//
// Role:
//   - ConnPolicy configures read idle and max lifetime defaults for tracker and broker gnet servers.
//   - ApplyConnPolicy sets gnet socket options on accept.
//
// Defaults and limits:
//   - DefaultConnReadIdle 30s.
//   - DefaultConnMaxLifetime 120s.
//
// Topology:
//   - Imported by internal/broker and internal/ingest/gnet; panjf2000/gnet v2 only besides stdlib.
//
// Forbidden:
//   - Import internal/* packages.
//
// Verify:
//
//	go test ./pkg/gnetutil/... -short -count=1
package gnetutil
