// Package doctor runs install health probes and exposes ops doctor HTTP and CLI bundle export.
//
// Role:
//   - DoctorHTTPHandlers serves GET /api/v1/ops/doctor (shards:read) with platform probe report.
//   - Run and RunPlatform execute Probe implementations; WriteBundle archives probe JSON for support.
//   - MVSSChecklist and DeployProfileChecklist validate env/deploy profile without live probes.
//   - remediation.go maps probe ids to operator hints (CheckHint); no auto-fix.
//
// Topology:
//   - Wired from internal/controlplane/adminapi_wire.go; cmd/operator doctor subcommand calls WriteBundle.
//   - ProbeDeps injects config, Postgres pool, Redis, ClickHouse, license diagnostics, XDP stats reader.
//   - RunPlatform appends rtb_config, edge_xdp, and dns (when tracking domain set) to DefaultProbes.
//
// Invariants:
//   - Doctor HTTP GET is read-only; probes must not mutate production state.
//   - Overall status is worst-of: fail beats warn beats pass (OverallStatus, Report.ExitCode).
//   - Report.ExitCode: 0 pass, 1 warn, 2 fail.
//   - Default Run timeout is 60s; HTTP handler passes Options.Timeout 30s.
//
// Forbidden:
//   - Auto-apply license, nginx reload, or sysctl changes from doctor handlers or probes.
//   - Import internal/ingest hot-path handlers or filter packages.
//
// Defaults and limits:
//   - DefaultProbes: kernel, sysctl, listen, redis, slotmap, clickhouse, disk, tls; license when
//     config.LicenseProbeEnabled; RunPlatform adds rtb_config, edge_xdp, optional dns.
//   - Deploy profiles: ingest_only, network_operator, analytics_ml (DeployProfileChecklist).
//
// Verify:
//
//	go test ./internal/doctor/ -short -count=1
//	go test ./internal/doctor/ -short -run TestMVSSChecklistTelemetryDefault -count=1
//	go test ./internal/doctor/ -short -run TestSlotMapProbe_passWhenPostgresMatchesHTTP -count=1
package doctor
