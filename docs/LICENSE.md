# Pilot: offline license (on-prem)

Closed-network installs: no license server, no outbound license pings. Vendor delivers signed JWT monthly; customer pastes into install or admin UI.

## Vendor setup

```bash
bash scripts/vendor/gen_license_keypair.sh
git add deploy/vendor/license_public.key
# deploy/vendor/license_private.key gitignored — back up offline
```

Issue monthly JWT:

```bash
export AD_EVENT_PROCESSOR_LICENSE_PRIVATE_KEY_FILE=deploy/vendor/license_private.key

go run ./cmd/license-issue \
  --sku pilot \
  --customer "Acme Media" \
  --deployment-id "<uuid>" \
  --fingerprint "<host-fingerprint>" \
  --days 35 \
  --out /tmp/acme-license.jwt
```

Reuse same `deployment_id` on renewals. USDT billing: [BILLING.md](BILLING.md).

### Trial registry

Pilot checks vendor-local registry (`deploy/vendor/trial_registry.json`). See [TRIAL.md](TRIAL.md).

```bash
export BIDSHARD_VENDOR_TRIAL_REGISTRY=deploy/vendor/trial_registry.json

go run ./cmd/license-issue --sku pilot --customer "Acme" --telegram-id "<id>" --deployment-id "<uuid>" --out /tmp/acme-pilot.jwt
go run ./cmd/license-issue --record-hwid --deployment-id "<uuid>" --hwid-v2 "<hwid_v2>"
go run ./cmd/license-issue --trial-mark-expired --deployment-id "<uuid>"
go run ./cmd/trial-registry expire-stale
go run ./cmd/trial-registry list-pending
go run ./cmd/license-issue --approve-pending "<pending-id>" --out /tmp/acme-pilot.jwt
go run ./cmd/license-issue --sku starter --deployment-id "<uuid>" --hwid-v2 "<hwid_v2>" --mark-converted --out /tmp/acme-starter.jwt
```

Force re-issue: `BIDSHARD_VENDOR_TRIAL_FORCE=1` + `--force --force-reason "..."`.

## Customer install

`deploy/installer/install.env`:

```bash
AD_EVENT_PROCESSOR_LICENSE_MODE=file
AD_EVENT_PROCESSOR_LICENSE_KEY=<JWT line>
TELEMETRY_ENABLED=false
```

```bash
bash scripts/install/ad-event-processor-install.sh --yes
```

## Renewal

- **Admin:** Settings → License renewal → paste JWT → Apply.
- **CLI:** `bash scripts/install/ad-event-processor-install.sh license-apply '<JWT>'`
- **API:**

```bash
curl -sS -X POST http://127.0.0.1:8188/api/v1/license/apply \
  -H "Content-Type: application/json" \
  -H "X-Admin-API-Key: $ADMIN_API_KEY" \
  -d '{"token":"<JWT>"}'
```

No restart — entitlements reload immediately.

## License states

| State | Tracking |
|-------|----------|
| ACTIVE | Normal |
| GRACE | JWT expired, within `grace_days` (7) |
| EXPIRED | Blocked |

Pilot SKU (`deploy/vendor/sku.yaml`): 10-day validity, 5k RPS cap, OpenRTB off, 7-day grace, hard bind.

## Host fingerprint / HWID v2

Installer prints `deployment fingerprint=...` — include in renewal request. Mismatch blocks ingest on another VPS.

HWID v2 (Argon2id, preferred): bundle exports `hwid_v2`.

```bash
go run ./cmd/license-issue \
  --sku pilot --customer "Acme" --deployment-id "<uuid>" \
  --hwid-v2 "<hwid_v2-from-support-bundle>" \
  --out /tmp/acme-license.jwt
```

When JWT has `hwid_hash`, verification uses HWID v2 only; legacy `--fingerprint` for older tokens without `hwid_hash`.

Vectors: `bash scripts/security/license_vector_gen.sh`

## Verification catalog

Authz: `Authz(token, host, t) ≜ V ∧ B ∧ R ∧ S`. Gate: `make license-verify`.

| ID | Property |
| --- | --- |
| P-C2-01 | Invalid signature or payload ⇒ reject |
| P-C2-02 | Hard bind mismatch ⇒ reject |
| P-C2-03 | Revoked ⇒ not ingestible |
| P-C2-04 | Past grace ⇒ expired |
| P-C2-05 | PG ACTIVE without valid file JWT ⇒ ingest blocked |
| P-C3-01 | `DetermineState` pure in `(claims, t, revoked)` |
| P-C3-02 | Non-revoked JWT state monotonic non-increasing |
| P-C3-03 | `IngestAllowed` false iff state ∈ {EXPIRED, REVOKED} |
| P-C3-04 | Wall clock rewind vs monotonic ⇒ expired |
| P-C3-05 | NTP-sized drift within threshold ⇒ not expired |
| P-C3-06 | Global skew watch forces expired state |
| P-C3-07 | Registry recheck on skew ⇒ ingest blocked |
| P-C4-01 | `Effective(e, e)` stable under re-merge |
| P-C4-02 | `Effective` associative merge |
| P-C4-03 | Customer limits never exceed deployment ceiling |
| P-HWID-01 | Argon2id hash deterministic for fixed telemetry |
| P-HWID-02 | Missing DMI does not panic; stable hash |
| P-KEY-01 | XOR-masked embedded key decodes to production Ed25519 pubkey |
| P-KEY-02 | Plaintext production pubkey hex not in `public_key.go` |
| P-D2-01 | XOR path tables decode to expected sysfs targets |
| P-D2-02 | amd64 uses raw `openat`/`read` syscall shim |
| P-D2-03 | Release tracker has no plaintext HWID paths |
| P-D4-01 | `sha256(vault:tag)` derivation matches CI |
| P-D4-02 | Wrong release salt ⇒ AEAD open fails |
| P-D4-03 | CI smoke with `ASSET_SEAL_SALT` set |
| P-C6-01 | MCK deterministic for fixed JWT + HWID + deployment_id |
| P-C6-02 | Different HWID ⇒ different MCK |
| P-C6-03 | Seal/open round-trip; wrong MCK ⇒ open fails |
| P-C6-04 | Feature seed coupling gates RPS/OpenRTB when enabled |
| P-C6-05 | Per-release `ASSET_SEAL_SALT` mixed into AEAD HKDF |
| P-C6-06 | Hot path uses inline state deny, not `IngestAllowed` call |
| P-C6-07 | Cold paths scatter inline bitmask checks |
| P-C6-08 | Valid license + sealed blob ⇒ `ebpf.NewCollection` |
| P-C6-09 | Sealed vs unsealed XDP attach on lab host |
| P-C6-10 | Sealed `unified-filter.lua` decrypt at tracker startup |
| P-C8-01 | TracerPid > 0 trips guard + epoch invalidate |
| P-C8-02 | Suspicious maps trip guard |
| P-C8-03 | `.text` hash mismatch trips guard |
| P-C8-04 | Env kill switch disables guard |
| P-C8-05 | Ptrace watchdog child: busy tracer slot trips guard |
| P-C8-06 | Ptrace watchdog re-exec child claims parent tracer |
| P-C8-07 | Guard off: engineering tooling documented |
| P-C8-08 | Fault suite does not require `license_guard` tag |
| P-C8-09 | Release strings gate (symbols + pubkey + license plaintext) |
| P-C8-10 | gdb attach denied with guard + ptrace watchdog |
| P-D1-01 | Default: tracker `literals=0`, control/processor `literals=1` |
| P-D1-02 | `GARBLE_LITERALS` overrides all binaries |
| P-D1-03 | `internal/ingestion` package `//garble:ignore` for tracker literals eval |
| P-D1-04 | Tracker p99 with `-literals` within +10% baseline (lab) |
| P-D5-01 | Garbled tracker/processor/control build |
| P-D5-02 | Garbled binary strings gate |
| P-D5-03 | License math after garble build |
| P-QA-01 | Nightly fuzz workflow documents JWT/HWID targets |
| P-QA-02 | Short fuzz smoke + extended red-team |
| P-QA-03 | Garbled tracker build + license alloc subset |
| P-QA-04 | Full garbled release (tracker/processor/control) |

## Renewal checklist

1. Same `deployment_id` as prior invoice.
2. Compare `host_fingerprint` or `hwid_v2` from support bundle with install values.
3. Reject fingerprint/HWID/deployment_id change without signed migration note.
4. Issue with matching `--deployment-id` and `--hwid-v2` (or legacy `--fingerprint`).

## Revocation

```bash
go run ./cmd/license-issue \
  --sku pilot --deployment-id "<uuid>" --fingerprint "<fp>" \
  --revoke --out /tmp/acme-revoke.jwt
```

## Red-team / closed contour

```bash
make license-red-team
```

| Check | Setting |
|-------|---------|
| No license server | `AD_EVENT_PROCESSOR_LICENSE_MODE=file`, empty `AD_EVENT_PROCESSOR_LICENSE_SERVER` |
| No telemetry | `AD_EVENT_PROCESSOR_TELEMETRY_OPT_IN=0` |
| Verify key | `deploy/vendor/license_public.key` or `AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY` |
| Status | `GET /api/v1/license/status` |

Support bundle: ask for `deployment_id` from Settings or `GET /api/v1/meta`.

## Release images

| Tag suffix | Binaries | Use |
|------------|----------|-----|
| `v1.0.0-pilot` | tracker, processor, control | Full single-VPS |
| `v1.0.0-pilot-ingest` | tracker, processor | Ingest-only |

`AD_EVENT_PROCESSOR_APP_IMAGE=ghcr.io/<org>/ad-event-processor:<tag>-ingest`

## EULA

Install: `--accept-eula` or interactive during `ad-event-processor-install.sh up`. Text: `pkg/legal/EULA.txt`, version `2026-01`.

## Garble literals (release)

| Binary | Default `-literals` |
| --- | --- |
| tracker | off |
| processor | on |
| control | on |

Override: `GARBLE_LITERALS=0|1`, `GARBLE_LITERALS_TRACKER=1`. Eval: `make garble-literals-eval`, `make garble-literals-policy-gate`, `make garble-literals-p99-smoke`.

## Sealed assets

Valid JWT on bound host loads sealed BPF + XDP:

```bash
sudo apt install -y clang llvm libbpf-dev linux-libc-dev
make bpf-edge-prereq-gate
sudo SEALED_BPF_XDP_SMOKE=1 make sealed-bpf-xdp-smoke
```

Seal unified-filter.lua:

```bash
go run ./cmd/license-asset-seal \
  --label unified-filter \
  --in internal/ingestion/unified-filter.lua \
  --out internal/ingestion/unified_filter_sealed.bin \
  --license var/license.jwt
```

Runtime: `InitUnifiedFilterLua()` at cold startup. Enterprise mode + blob → decrypt via MCK from JWT. Fallback: embedded script. Override: `AD_EVENT_PROCESSOR_UNIFIED_FILTER_SEALED_BLOB`.

## License guard (V2-C)

Release garble (`license_guard` tag): cold-path anti-debug on tracker, processor, control.

| Env | Effect |
| --- | --- |
| `AD_EVENT_PROCESSOR_LICENSE_GUARD=0` | Disable all probes |
| `AD_EVENT_PROCESSOR_LICENSE_GUARD_PTRACE=0` | Disable ptrace watchdog only |

```bash
go test -tags=license_guard ./internal/licensing/ -run Guard -count=1
make license-guard-off-smoke license-guard-fault-gate license-gdb-guard-smoke license-red-team-extended
```

Lab gdb drill: `RELEASE_GARBLE=1 LICENSE_GUARD=1 make release-garble` then `gdb -p "$(pgrep -n tracker)"`. Expect attach failure or epoch invalid. Disable guard: `AD_EVENT_PROCESSOR_LICENSE_GUARD=0`.
