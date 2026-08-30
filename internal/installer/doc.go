// Package installer implements cmd/installer CLI: appliance preflight, manifest render,
// compose bring-up, binary deploy, and on-host license helpers.
//
// Role:
//   - CLI (installer.go): preflight, provision, configure, up, bootstrap, apply, rollback,
//     doctor, license (install|activate|status|host-id).
//   - Render systemd units (render_systemd.go, render_edge_systemd.go), secrets.env (0600),
//     compose env for compose_dev, and install.yaml round-trip (apply_binary.go).
//   - BinaryDeploy copies tracker/processor binaries with health probes, versioned backup,
//     and rollback markers (apply_binary.go, rollback_cli.go).
//   - Doctor (doctor.go) runs local deps script plus GET management /api/v1/doctor when
//     ADMIN_API_KEY is set.
//   - License helpers (license.go) copy JWT to install root and print HostFingerprint/HWID.
//
// Topology:
//   - Invoked only from cmd/installer; no HTTP server or long-running worker in package.
//   - Paths resolve via config.InstallRootFromEnv and pkg/runtimepaths (secrets.env,
//     license.jwt); repo root via go.mod walk or AD_EVENT_PROCESSOR_REPO_ROOT.
//   - apply runs docker compose up -d for ProfileSingleVPS (--profile single_vps) or
//     ProfileComposeDev after writing manifests.
//
// Invariants:
//   - secrets.env and license.jwt never written world-readable (secrets mode 0600).
//   - apply --dry-run prints planned docker/render actions without mutating files or compose.
//   - Install profiles validated before render (profile.go); removed k8s profile rejected.
//   - Generated compose volume names keep ad_event_processor_* historical prefix.
//
// Defaults and limits:
//   - healthProbeTimeout 1 s for binary deploy health URL checks.
//   - Redis topology tests expect four shard addrs for single_vps appliance profile.
//
// Forbidden:
//   - Tracker/control runtime ingest or admin mutation (composition root only).
//   - Per-render Ed25519 JWT verify (license apply/status paths only).
//
// Verify:
//
//	go list -e ./internal/installer/...
//	go test ./internal/installer/ -short -run TestIdempotentApply -count=1
//	go test ./internal/installer/ -short -run TestGoldenRenderSystemd -count=1
//	go test ./internal/installer/ -short -run TestComposeRedisTopology_applianceFourShards -count=1
//	go test ./internal/installer/ -short -run TestBinaryDeployBadBinaryRollsBack -count=1
package installer
