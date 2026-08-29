# Ed25519 license keys

Per-cohort public keys for offline JWT verification on customer installs.

- Default pilot key: [license_public.key](./license_public.key) (`kid`: `2026-01`).
- Layout: `keys/<kid>/license_public.key` (private keys local only, gitignored).

Issue: `go run ./cmd/license-issue`. Verify catalog: `make license-verify`.

## HWID v2 (Argon2id)

Host bind hash (`hwid_v2` in license status API) uses Argon2id per [RFC 9106](https://www.rfc-editor.org/rfc/rfc9106) (`argon2.IDKey`):

| Parameter | Value | Notes |
| :--- | :--- | :--- |
| Type | Argon2id | `argon2.IDKey` |
| Memory (`m`) | 65536 KiB | 64 MiB |
| Iterations (`t`) | 3 | |
| Parallelism (`p`) | 4 | |
| Output length | 32 bytes | hex-encoded as 64-char `hwid_v2` |

Telemetry fields hashed (v2 default): DMI UUID, root disk id, MAC, CPU model, core count (`internal/licensing/hwid.go`).

Optional v3 (`AD_EVENT_PROCESSOR_LICENSE_HWID_V3=1`): appends `/etc/machine-id` (or dbus path) as a sixth field. Existing licenses must be re-issued after enabling v3 on a host.

`GET /api/v1/license/status` exposes live telemetry in `hwid_inputs` (machine id omitted unless v3 is enabled).

Lab collection (same code path as production):

```bash
bash scripts/lab/hwid_collect.sh
```

Reference bench on a 4-vCPU VPS: `go test ./internal/licensing/ -bench=BenchmarkHostHWID -benchmem` ~120–180 ms/op (one-time at process start).

License recheck seed coupling (`StretchMCKForRecheck`): same Argon2id params, ~120-180 ms per recheck (not per `/track` event). Bench: `go test ./internal/licensing/ -bench=BenchmarkStretchMCKForRecheck -benchmem`.

## MCK HKDF info label

`DeriveMCK` uses HKDF-SHA256 with `info = MCKInfoLabel()` (default `license-mck-v2`). IKM is `sig|payload|hwid|kid`; salt is `deployment_id`.

| Build | Label source |
| :--- | :--- |
| Dev / `go test` | `DefaultMCKInfoLabel` in `internal/licensing/mck_info_label.go` |
| Garbled release | `-X ad-event-processor/internal/licensing.buildMCKInfoLabel=...` from `scripts/ci/release_garble.sh` (`MCK_INFO_LABEL`, default `license-mck-v2`) |

Bump `MCK_INFO_LABEL` on major licensing releases so patched old binaries derive the wrong `feature_seed` and sealed assets fail to open. Golden vectors: `internal/licensing/testdata/mck_derivation.json` field `mck_info_label` must match `MCKInfoLabel()`. Regenerate: `WRITE_MCK_VECTORS=1 go test ./internal/licensing/ -run TestGenMCKVectorArtifacts -count=1`.

Gate: `bash scripts/ci/static/mck_info_label.sh`.
