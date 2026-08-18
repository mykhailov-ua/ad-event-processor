# Pilot: offline license (on-prem)

Closed-network installs with **no license server**, **no outbound license pings**. You receive a signed JWT monthly after payment and paste it into the install or admin UI.

## Vendor (you) — first-time setup

```bash
# Once: generate keypair (private key stays local, never in customer tarball)
bash scripts/vendor/gen_license_keypair.sh
git add deploy/vendor/license_public.key
# deploy/vendor/license_private.key is gitignored — back up offline
```

Issue a monthly license for a customer:

```bash
export AD_EVENT_PROCESSOR_LICENSE_PRIVATE_KEY_FILE=deploy/vendor/license_private.key

go run ./cmd/license-issue \
  --sku pilot \
  --customer "Acme Media" \
  --deployment-id "<uuid-from-customer-settings-or-email>" \
  --fingerprint "<host-fingerprint-from-install>" \
  --days 35 \
  --out /tmp/acme-license.jwt

# Email / deliver the JWT file contents to the customer (single line)
```

Record `deployment_id` per customer — reuse the same ID on renewal JWTs.

### Pilot trial registry (repeat-trial gate)

Pilot issue checks a **vendor-local** file registry (`deploy/vendor/trial_registry.json` by default). See [TRIAL_ABUSE.md](TRIAL_ABUSE.md).

```bash
export BIDSHARD_VENDOR_TRIAL_REGISTRY=deploy/vendor/trial_registry.json

go run ./cmd/license-issue \
  --sku pilot \
  --customer "Acme Media" \
  --telegram-id "<buyer-telegram-user-id>" \
  --deployment-id "<uuid>" \
  --out /tmp/acme-pilot.jwt

# After bundle: record hwid anchor
go run ./cmd/license-issue \
  --record-hwid \
  --deployment-id "<uuid>" \
  --hwid-v2 "<hwid_v2-from-support-bundle>"

# Pilot ended without conversion
go run ./cmd/license-issue \
  --trial-mark-expired \
  --deployment-id "<uuid>"

# Bulk expire stale pilots (cron on vendor host)
go run ./cmd/trial-registry expire-stale

# Optional: Telegram capture bot (vendor workstation; no signing key on bot host)
export BIDSHARD_VENDOR_TRIAL_BOT_TOKEN="<from @BotFather>"
go run ./cmd/vendor-trial-bot run

# Review queue and issue pilot JWT
go run ./cmd/trial-registry list-pending
go run ./cmd/license-issue \
  --approve-pending "<pending-id>" \
  --out /tmp/acme-pilot.jwt

# Reject spam signups
go run ./cmd/trial-registry reject-pending --id "<pending-id>" --reason "spam"

# Paid conversion: same deployment_id, then mark converted
go run ./cmd/license-issue \
  --sku starter \
  --customer "Acme Media" \
  --deployment-id "<uuid>" \
  --hwid-v2 "<hwid_v2>" \
  --mark-converted \
  --out /tmp/acme-starter.jwt
```

Force re-issue (audit): `BIDSHARD_VENDOR_TRIAL_FORCE=1` plus `--force --force-reason "..."`.

## Customer — first install

In `deploy/installer/install.env`:

```bash
AD_EVENT_PROCESSOR_LICENSE_MODE=file
AD_EVENT_PROCESSOR_LICENSE_KEY=<paste full JWT line from vendor>
TELEMETRY_ENABLED=false
```

Then:

```bash
bash scripts/install/ad-event-processor-install.sh --yes
```

Install writes `license.jwt`, sets `AD_EVENT_PROCESSOR_LICENSE_MODE=file`, clears license server URL, disables product telemetry ping.

## Customer — monthly renewal (after payment)

**Option A — Admin UI:** Settings → License renewal → paste new JWT → Apply.

**Option B — CLI on the server:**

```bash
bash scripts/install/ad-event-processor-install.sh license-apply '<JWT>'
```

**Option C — API:**

```bash
curl -sS -X POST http://127.0.0.1:8188/api/v1/license/apply \
  -H "Content-Type: application/json" \
  -H "X-Admin-API-Key: $ADMIN_API_KEY" \
  -d '{"token":"<JWT>"}'
```

No restart required — entitlements reload immediately.

## License states

| State | Tracking |
|-------|----------|
| ACTIVE | Normal |
| GRACE | JWT expired but within `grace_days` (7) — renew soon |
| EXPIRED | Tracking blocked |

Pilot SKU (`deploy/vendor/sku.yaml`): 10-day validity, 5k RPS cap, OpenRTB off, 7-day grace, **hard bind** (host fingerprint).

## Host fingerprint (hard bind)

On first install the installer prints `deployment fingerprint=...`. Include this value when requesting a renewal JWT:

```bash
go run ./cmd/license-issue \
  --sku pilot \
  --customer "Acme Media" \
  --deployment-id "<uuid>" \
  --fingerprint "<host-fingerprint-from-install>" \
  --out /tmp/acme-license.jwt
```

Mismatching fingerprint on another VPS blocks ingest even with a copied `license.jwt`.

### HWID v2 (Argon2id, recommended for new renewals)

Renewal JWTs may include an `hwid_hash` claim (Argon2id over DMI UUID, root disk id, NIC MAC, and CPU model/cores). The support bundle and doctor output export `hwid_v2` alongside the legacy `host_fingerprint`.

Issue with HWID v2 (preferred when the customer bundle includes `hwid_v2`):

```bash
go run ./cmd/license-issue \
  --sku pilot \
  --customer "Acme Media" \
  --deployment-id "<uuid>" \
  --hwid-v2 "<hwid_v2-from-support-bundle>" \
  --out /tmp/acme-license.jwt
```

When `hwid_hash` is present in the JWT, verification uses HWID v2 only; legacy `--fingerprint` remains valid for older tokens without `hwid_hash`.

Regenerate golden HWID vectors (vendor CI only): `bash scripts/security/license_vector_gen.sh`

## Vendor renewal checklist (support)

Before issuing a renewal JWT:

1. Customer ticket includes **same** `deployment_id` as prior invoice (Settings → License or support bundle `version.json`).
2. Compare `host_fingerprint` or `hwid_v2` from support bundle with values used at first install (`AD_EVENT_PROCESSOR_DEPLOYMENT_FINGERPRINT` on server or install log).
3. Reject renewal when fingerprint, HWID v2, or deployment_id changed without a signed migration note.
4. Issue JWT with matching `--deployment-id` and `--hwid-v2` (or legacy `--fingerprint` when `hwid_hash` is absent).

## Revocation (chargeback)

Issue a revocation JWT — ingest stops after customer applies it or tracker rechecks the file:

```bash
go run ./cmd/license-issue \
  --sku pilot \
  --customer "Acme Media" \
  --deployment-id "<uuid>" \
  --fingerprint "<host-fingerprint>" \
  --revoke \
  --out /tmp/acme-revoke.jwt
```

## License red-team (vendor / CI)

```bash
make license-red-team
```

Runs integration tests for empty PG row, fake ACTIVE row, and expired JWT bypass scenarios.

## Closed contour checklist

| Check | Setting |
|-------|---------|
| No license server | `AD_EVENT_PROCESSOR_LICENSE_MODE=file`, empty `AD_EVENT_PROCESSOR_LICENSE_SERVER` |
| No product telemetry | `AD_EVENT_PROCESSOR_TELEMETRY_OPT_IN=0` |
| Verify key in image/env | `deploy/vendor/license_public.key` or `AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY` |
| Status | `GET /api/v1/license/status` or Overview license banner |

## Support bundle

Ask customer for `deployment_id` from Settings or `GET /api/v1/meta` before issuing renewal JWT.

## Release images (SKU surface)

| Tag suffix | Binaries | Use case |
|------------|----------|----------|
| `v1.0.0-pilot` (default tag) | tracker, processor, control | Full single-VPS pilot |
| `v1.0.0-pilot-ingest` | tracker, processor | Ingest-only workers (no admin binary in image) |

Set `AD_EVENT_PROCESSOR_APP_IMAGE=ghcr.io/<org>/ad-event-processor:<tag>-ingest` for ingest-only deployments.

## EULA

Install requires `--accept-eula` (non-interactive) or interactive acceptance during `ad-event-processor-install.sh up`.

First admin login shows a click-through if EULA was not recorded at bootstrap. Text: `pkg/legal/EULA.txt`, version `2026-01`.

## Garble literals (Release hardening)

Release garble builds apply `-literals` per binary:

| Binary | Default `-literals` |
| --- | --- |
| `tracker` | off (hot path; enable only after eval) |
| `processor` | on |
| `control` | on |

Override all: `GARBLE_LITERALS=0|1`. Per binary: `GARBLE_LITERALS_TRACKER=1`, etc.

Evaluate tracker size impact before enabling literals on ingest:

```bash
make garble-literals-eval
make garble-literals-policy-gate
```

p99 acceptance (+10% vs baseline): `make garble-literals-p99-smoke` (load-test stack + Prometheus). Size-only eval: `make garble-literals-eval`.

## Sealed BPF + XDP attach lab (Sealed assets)

Valid JWT on bound host must load sealed edge BPF and attach XDP like the unsealed baseline:

```bash
# Prereq: clang + BTF
sudo apt install -y clang llvm libbpf-dev linux-libc-dev
make bpf-edge-prereq-gate          # go generate + sealed unit tests; skip if no clang

sudo SEALED_BPF_XDP_SMOKE=1 make sealed-bpf-xdp-smoke
```

Harness: `kernel_xdp_attach_lo_generic` on `lo`; drop proof via `prog.Test` on the same maps (`drop_assertion=prog_test_same_maps`), not raw loopback RX.

### Sealed unified-filter.lua (V2-B.4)

Vendor seal (enterprise images):

```bash
go run ./cmd/license-asset-seal \
  --label unified-filter \
  --in internal/ingestion/unified-filter.lua \
  --out internal/ingestion/unified_filter_sealed.bin \
  --license license.jwt
```

Runtime: tracker calls `InitUnifiedFilterLua()` at cold startup. When `AD_EVENT_PROCESSOR_LICENSE_MODE=enterprise` and `unified_filter_sealed.bin` exists, decrypts with MCK from `license.jwt`. Missing blob falls back to embedded `//go:embed` script (pilot transition). Override path: `AD_EVENT_PROCESSOR_UNIFIED_FILTER_SEALED_BLOB`.

## License guard (V2-C)

Release garble builds (`license_guard` tag) enable cold-path anti-debug probes on tracker, processor, and control.

| Env | Effect |
| --- | --- |
| `AD_EVENT_PROCESSOR_LICENSE_GUARD=0` | Disable all guard probes (gdb/strace friendly) |
| `AD_EVENT_PROCESSOR_LICENSE_GUARD_PTRACE=0` | Disable ptrace watchdog child only; keep TracerPid/maps/text checks |

Ptrace watchdog (V2-C.2): on startup a re-exec child attaches to the parent with `PTRACE_ATTACH`, occupying the tracer slot so external debuggers get `EBUSY`. If another tracer is already present, the license epoch is invalidated (`ptrace_busy`). When `kernel.yama.ptrace_scope` blocks attach, the watchdog skips with a warning (no trip).

```bash
go test -tags=license_guard ./internal/licensing/ -run Guard -count=1
make license-guard-off-smoke
make license-guard-fault-gate
make license-gdb-guard-smoke          # automated gdb attach denial (harness=license_guard_release)
make license-red-team-extended        # red-team steps 7–10 (automated subset)
```

### Lab gdb drill (V2-C.D1)

Automated smoke (test binary subprocess):

```bash
make license-gdb-guard-smoke
# or: LICENSE_GDB_SMOKE=1 go test -tags=license_guard ./internal/licensing/ -run GDBAttachDenied -count=1
```

Full release tracker on a lab host:

```bash
RELEASE_GARBLE=1 LICENSE_GUARD=1 make release-garble
# run tracker with valid license.jwt, then:
gdb -p "$(pgrep -n tracker)"
```

Harness: `license_guard_release`. Expect `ptrace: Operation not permitted` / attach failure, or license epoch invalid within the background probe window. Use `AD_EVENT_PROCESSOR_LICENSE_GUARD=0` to confirm gdb works when guard is intentionally disabled (C.D2).

