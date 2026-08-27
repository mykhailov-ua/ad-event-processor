# Licensing security backlog

Hardening and pentest catalog for offline Ed25519 JWT licensing, Argon2id HWID bind, `garble` release builds, runtime guard, and MCK-sealed assets.

**Canonical implementation:** `internal/licensing/`, `internal/ingestion/registry_license.go`, `scripts/security/license_red_team.sh`, `.cursor/rules/licensing.mdc`.

**Out of scope:** online license servers, hardware security modules (HSM) on buyer VPS, marketing claims about "unbreakable" protection.

Cross-reference slugs in PR descriptions. Do not close a slug until every applicable gate below is checked.

---

## Threat model (buyer VPS)

| Attacker capability | Realistic goal | Primary controls today |
| :--- | :--- | :--- |
| Root on appliance | Run without paying; clone license | HWID bind, trial registry at issue time, host activation limits |
| Root on appliance | Patch binary or hook memory | `license_guard` (Linux release), text hash, ptrace watchdog, maps scan |
| Root on appliance | Replace pubkey / JWT without patch | Embedded pubkey only in production profile (`pubkey_production_fail_closed`) |
| Binary-only reverser | Find branch after `ed25519.Verify` | garble, release strings gates, seed coupling, MCK |
| Operator mistake | Leave guard off in prod | Env kill switches documented; production profile should fail closed |

Offline licensing cannot prevent a determined root owner forever. Goal: raise cost above SKU price and block **mass redistributable** cracks across releases.

---

## Priority legend

| Label | Meaning |
| :--- | :--- |
| `sec_p0` | Closes a practical bypass (config injection, release guard off, weak JWT parsing) |
| `sec_p1` | Raises reverser cost (garble policy, Argon tuning, pentest automation) |
| `sec_p2` | Depth hardening (extra telemetry, anti-Frida needles, hot-path probes) |
| `misdirect_p1` | Structural misdirection: no single identifiable crack point; legit path unchanged |
| `cost_p1` | Raises reverser CPU/RAM or multi-binary work; cold/background only |
| `cost_p2` | Release churn, jitter, cross-process sync; lower implementation risk |

---

## Global close checklist (every slug)

Rule: `anti-slop.mdc` agent checklist

- [ ] Every new symbol resolves (`go build` / `go test -c` on touched packages)
- [ ] Hot path does not import cold fraud scoring; license checks stay snapshot-based (`hot-path.mdc`)
- [ ] Verification commands pasted in PR with package path (`quality.mdc`)
- [ ] No doc claim that microbench or unit test equals production crack resistance
- [ ] `bash scripts/ci/pr_fast.sh` scoped to touched packages when Go changes land
- [ ] `make license-verify` or `make license-red-team` when licensing paths change (`licensing.mdc`)

Rule: licensing security

- [ ] Pentest step added or updated in **Pentest catalog** section when behavior changes
- [ ] Kill switches (`LICENSE_GUARD=0`, env pubkey) documented in operator runbook, not removed without replacement
- [ ] `bash scripts/ci/check_no_legacy_naming.sh` clean on touched `deploy/vendor/` prose

Rule: misdirection (decoy surfaces)

- [x] Decoy functions are **not** reachable from production call graph (holdout test or `go test` call-graph audit)
- [ ] No fake `ACTIVE` in admin API or metrics when ingest is blocked (`anti-slop.mdc` lie modes)
- [ ] No honeypot env vars documented in operator install paths
- [ ] Hot path still has zero `ed25519.Verify` and no new heap allocs (`hot-path.mdc`)
- [ ] `make license-red-team` and integration license tests pass unchanged on clean profile

Rule: cost amplification (memory-hard / multi-binary)

- [ ] Argon2id and other memory-hard KDF run only on boot, recheck goroutine, or tamper probe - never in `LicenseFilter.Check` or `/track`
- [x] New MCK consumers fail closed when `FeatureSeedValid()` false; no partial appliance crack (tracker-only) without explicit dev profile
- [x] `mck_derivation.json` / HKDF golden vectors updated when `mckInfoLabel` or IKM layout changes
- [ ] Benchmark recheck stretch: document ms and RSS on reference SKU in PR when touching Argon2 on recheck

---

## Current baseline (code truth)

| Layer | Implementation | Release default |
| :--- | :--- | :--- |
| JWT | Ed25519 (`crypto/ed25519`), 3-part JWS (`verify.go`) | On |
| Pubkey | XOR-masked embed + file/env override (`public_key.go`) | Embed + optional file |
| HWID v2 | Argon2id t=3, m=65536 KiB, p=4 (`hwid.go`) | On Linux |
| HWID inputs | DMI UUID, root disk id/serial, first NIC MAC, CPU model, cores | Linux sysfs |
| Legacy bind | SHA256(`/etc/machine-id` + install paths) (`fingerprint.go`) | When `bind.fingerprint` set |
| Hot ingest gate | `LicenseFilter` + `LicenseRPSFilter` on atomic snapshot | `LICENSE_REQUIRED` in production |
| Seed coupling | MCK -> feature seed gates RPS/OpenRTB (`runtime_seed.go`) | `LICENSE_MODE=file\|enterprise` |
| Sealed assets | ChaCha20-Poly1305 via HKDF(MCK) (`asset_cipher.go`) | When `ASSET_SEAL` on |
| Runtime guard | `license_guard` tag: text SHA256, TracerPid, maps needles, ptrace child | `LICENSE_GUARD=1` in `release_garble.sh` |
| garble | `mvdan.cc/garble@v0.15.0`; literals default on control/processor only | `RELEASE_GARBLE=1` |
| CI red team | `make license-red-team`, `license_verify_tier.sh`, extended steps 7-10 | Linux gates |
| License recheck | `StartFileLicenseRecheck` in tracker (registry), processor, control | guard only |
| MCK consumers | unified-filter sealed lua, edge BPF sealed, processor CH policy, control runtime policy | processor/control sealed assets: **shipped** |

---

## Summary

| Slug | Priority | Gap | Est. effort |
| :--- | :--- | :--- | :--- |
| `licensing_pentest_playbook` | sec_p0 | No single operator pentest runbook | S | **Shipped** |
| `ed25519_alg_pin_and_claims` | sec_p0 | Header `alg` not pinned to EdDSA | S | **Shipped** |
| `pubkey_production_fail_closed` | sec_p0 | Env/file pubkey override on prod profile | M | **Shipped** |
| `license_guard_release_matrix` | sec_p0 | Garbled red-team build sets `LICENSE_GUARD=0` | S | **Shipped** |
| `argon2_hwid_params_rfc9106` | sec_p1 | Params not documented vs RFC 9106 / OWASP | S | **Shipped** |
| `argon2_hwid_telemetry_expand` | sec_p2 | No disk serial fallback chain doc; no machine-id in v2 | M | **Shipped** |
| `garble_seed_and_literals_policy` | sec_p1 | Params not documented vs RFC 9106 / OWASP | S | **Shipped** |
| `garble_strings_surface_audit` | sec_p1 | XOR masks recoverable; expand forbidden patterns | S | **Shipped** |
| `license_guard_ptrace_fail_closed` | sec_p1 | Yama `skip` continues with warn only | M | **Shipped** |
| `license_guard_frida_needles` | sec_p2 | Maps scan list is short | S | **Shipped** |
| `mck_seed_coupling_mandatory_release` | sec_p1 | Seed off in dev/unsealed mode only by design | S | **Shipped** |
| `asset_seal_salt_release_gate` | sec_p1 | `ASSET_SEAL_SALT` optional in CI | S | **Shipped** |
| `hwid_spoof_lab_fixture` | sec_p1 | No documented KVM/sysfs spoof drill | M | **Shipped** |
| `binary_patch_lab_procedure` | sec_p2 | Manual CFG patch not in CI catalog | S (doc only) | **Shipped** |
| `license_decoy_verify_unreachable` | misdirect_p1 | Single obvious Verify branch is the real gate | M | **Shipped** |
| `license_decoy_pubkey_candidate` | misdirect_p1 | One recoverable XOR pubkey block in binary | S | **Shipped** |
| `license_state_derivatives_only` | misdirect_p1 | `state==ACTIVE` patchable without MCK/seed path | M | **Shipped** |
| `license_sealed_decoy_embed` | misdirect_p1 | Obvious plaintext Lua/BPF looks like the asset gate | M | **Shipped** |
| `license_hot_path_no_crypto_anchor` | misdirect_p1 | Reverser can anchor on Verify near ingest | S (audit gate) | **Shipped** |
| `license_guard_decoupled_trip` | misdirect_p2 | Guard trip correlates visibly with patched branch | S | **Shipped** |
| `license_cold_near_gates_catalog` | misdirect_p2 | Too few decoy cold functions that look like enforcement | S | **Shipped** |
| `mck_argon2_recheck_stretch` | cost_p1 | MCK derivation cheap; brute-force JWT/HWID combos low RAM cost | M | **Shipped** |
| `license_recheck_processor_control` | cost_p1 | Crack tracker-only sufficient for partial appliance | M | **Shipped** |
| `entitlements_mck_bitwise_coupling` | cost_p1 | JWT feature flags patchable without MCK byte agreement | M | **Shipped** |
| `license_file_hmac_sidecar` | cost_p1 | Replace `license.jwt` on disk without second check | S | **Shipped** |
| `mck_hkdf_kid_in_ikm` | cost_p1 | `kid` rotation does not invalidate derived MCK | S | **Shipped** |
| `sealed_assets_processor_control` | cost_p1 | Only tracker consumes sealed blobs today | M | **Shipped** |
| `processor_settlement_seed_gate` | cost_p1 | Processor cold path ignores `FeatureSeedValid` | M | **Shipped** |
| `mck_info_label_release_version` | cost_p1 | No per-release break of old cracks via HKDF info bump | S | **Shipped** |
| `guard_tamper_argon2_before_trip` | cost_p2 | Automated patch fuzz cheap vs guard | S | **Shipped** |
| `license_recheck_jitter` | cost_p2 | Fixed 5m recheck window easy to race in lab | S | **Shipped** |
| `license_epoch_pubsub_sync` | cost_p2 | Tracker/control/processor epoch drift after partial crack | M | **Shipped** |

---

## Reference: tool and spec practices

### Argon2id (RFC 9106)

HWID here is **binding fingerprinting**, not password storage. Argon2id still adds CPU/RAM cost to brute-force synthetic HWIDs and slows mass cloning scripts.

| Parameter | This repo (`hwid.go`) | RFC 9106 guidance | Notes |
| :--- | :--- | :--- | :--- |
| Type | `argon2.IDKey` (Argon2id) | Prefer Argon2id when unsure | Correct choice |
| Time (`t`) | 3 | Tune to target latency | ~tens of ms on modern CPU; OK for boot-time cache |
| Memory (`m`) | 65536 KiB (64 MiB) | Raise for stronger binding | OWASP often cites 19-46 MiB minimum for passwords; 64 MiB is reasonable |
| Parallelism (`p`) | 4 | Match cores | Do not exceed physical cores on appliance SKU |
| Salt | Fixed app salt (XOR blob) | Unique per deployment for passwords | **Acceptable for HWID** if salt is not attacker-chosen; document as product salt |
| Output | 32 bytes hex | - | Used as `hwid_hash` claim at issue time |

**Do not** raise `t`/`m` on hot path: `HostHWID()` is `sync.Once` at startup only.

### Ed25519 JWT (RFC 8032, RFC 7519)

| Practice | Status | Action |
| :--- | :--- | :--- |
| Algorithm Ed25519 only | Partial | Reject `alg` != `EdDSA` in header before verify |
| `kid` rotation | Shipped | `key_resolver.go` cohort paths under `deploy/vendor/keys/<kid>/` |
| Max token size | Shipped | 16 KiB (`maxLicenseTokenBytes`) |
| Claims: `exp`, `nbf`, `revoked` | Shipped | `DetermineState`, `file_verify.go` |
| Private key never in buyer image | Shipped | `license_private.key` gitignored; issue on vendor plane only |
| Constant-time compare on HWID | Shipped | String equality on hex digest |

### garble (mvdan.cc/garble)

| Practice | This repo | Action |
| :--- | :--- | :--- |
| Pin version | `GARBLE_VERSION=v0.15.0` in `release_garble.sh` | Keep pinned in CI |
| Pin seed | `GARBLE_SEED` warned if unset | **Require** in release CI for reproducible builds and diffable pentest |
| `-literals` | Default on control/processor; off on tracker | Run `garble_literals_p99_smoke.sh` before enabling on tracker |
| `-tiny` | Not used | Stay off unless alloc gate proves safe |
| `//garble:ignore` | `internal/ingestion/garble_policy.go` only for ingestion package marker | Audit any new ignores in `internal/licensing` |
| Build tags | `license_guard` separate from garble | Release: both on (`release_garble.sh`) |

### Runtime guard (Linux)

| Practice | This repo | Limitation |
| :--- | :--- | :--- |
| `PR_SET_DUMPABLE(0)` | `guard_linux.go` | Reduces core dumps; not full anti-debug |
| Ptrace watchdog | Child `PtraceAttach` on parent | Blocks second debugger; bypass via `LICENSE_GUARD=0` or VM |
| `.text` SHA256 via `/proc/self/mem` | Baseline at start; probe every 3-8 s | In-memory patch detected; disk patch at rest not detected until load |
| Maps substring scan | frida/gdb-related needles XOR-encoded | Trivial to rename gadget `.so` |
| Trip behavior | `InvalidateLicenseEpoch` -> EXPIRED ingest | Silent degrade, not panic (by design) |

---

## `licensing_pentest_playbook`

**Priority:** sec_p0

**Gap:** Red-team scripts are split across `license_red_team.sh`, extended, garbled, and gdb smoke. Operators need one ordered pentest with expected outcomes.

**Target:** Single entrypoint script + this section as canonical manual supplement.

### Implementation

1. Add `bash scripts/security/license_pentest.sh` (automated tiers A-D).
2. Link from `docs/DEVELOPMENT.md` under license verify section.
3. Nightly optional: `LICENSE_PENTEST_GARBLED=1` tier D.

### Pentest catalog

Run from repo root on Linux amd64 unless noted.

#### Tier A - Automated CI parity (no garble)

```bash
make license-red-team
make license-verify
bash scripts/test/license_red_team_extended.sh
```

| ID | Test | Pass criteria |
| :--- | :--- | :--- |
| PT-A01 | Invalid JWT signature | `go test ./internal/licensing/ -run Revoked` PASS |
| PT-A02 | HWID mismatch blocks ingest | `TestIntegration_LicenseProtection_hwidMismatchBlocksIngest` PASS |
| PT-A03 | PG ACTIVE without JWT | `fakePGRowWithoutJWTBlocked` PASS |
| PT-A04 | Seed coupling without MCK | `LicenseRPSFilter_seedCoupling` PASS |
| PT-A05 | Sealed Lua invalid MCK | `ResolveUnifiedFilterLua` PASS |
| PT-A06 | Clock skew | `SkewWatch` + registry skew tests PASS |
| PT-A07 | HWID path strings absent | `bash scripts/ci/hwid_strings_gate.sh` PASS |
| PT-A08 | Pubkey hex absent | `bash scripts/ci/public_key_strings_gate.sh` PASS |

#### Tier B - Garbled release binary

```bash
export LICENSE_VERIFY_GARBLED=1
bash scripts/ci/license_red_team_garbled.sh
bash scripts/ci/release_strings_gate.sh bin/garbled-release/tracker
```

| ID | Test | Pass criteria |
| :--- | :--- | :--- |
| PT-B01 | Forbidden strings | `release_strings_gate.sh` exit 0 |
| PT-B02 | garble literals policy | `garble_literals_policy_gate.sh` exit 0 |
| PT-B03 | Red team on garbled tree | `license_red_team.sh` exit 0 |

**Known gap:** none for Tier B guard (`license_guard_release_matrix` shipped 2026-08-26).

#### Tier C - Runtime guard lab (Linux + gdb)

```bash
export LICENSE_GDB_SMOKE=1
bash scripts/test/license_gdb_guard_smoke.sh
go test -tags=license_guard ./internal/licensing/ -run Guard -count=1
```

| ID | Test | Pass criteria |
| :--- | :--- | :--- |
| PT-C01 | GDB attach denied | gdb output contains `Operation not permitted`, `EBUSY`, or `ptrace:` |
| PT-C02 | Simulated TracerPid | `TestGuard_TracerPid` PASS |
| PT-C03 | Simulated text tamper | `TestGuard_TextTamper` PASS |
| PT-C04 | Guard off smoke documents kill switch | `license_guard_off_smoke.sh` PASS |

#### Tier D - Manual root attacker (documented, not CI)

| ID | Procedure | Pass criteria (defense holds) |
| :--- | :--- | :--- |
| PT-D01 | Config pubkey injection | Set `AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY` to attacker key, place attacker-signed `var/license.jwt` | Ingest EXPIRED: production profile ignores env/file pubkey (`pubkey_production_fail_closed`) |
| PT-D02 | Guard kill switch | `AD_EVENT_PROCESSOR_LICENSE_GUARD=0` on release build | Process starts; gdb attach succeeds; document as lab-only |
| PT-D03 | HWID sysfs spoof | KVM guest with bind-mounted fake `product_uuid`, disk serial, MAC | Valid license on donor HWID works only when telemetry matches; mismatch -> EXPIRED |
| PT-D04 | Patch `LicenseFilter` only | Binary patch or runtime return 0 from filter | Ingest may pass state check but RPS/OpenRTB blocked when seed coupling on (`PT-A04`); `bash scripts/lab/binary_patch_lab.sh` |
| PT-D05 | Pubkey XOR recovery | Recover embed from `masked^mask` in binary | Attacker can derive vendor pubkey offline; must still forge JWT private key or patch verify |
| PT-D06 | Frida gadget maps | Inject `frida-agent.so`, check `/proc/self/maps` | Guard trips within one probe interval (3-8 s) -> ingest EXPIRED |
| PT-D07 | `.text` in-memory patch | `gdb` write byte in executable segment (if attach allowed) | `text_tamper` trip when guard on; manual steps in `deploy/vendor/fixtures/binary_patch/README.md` |
| PT-D08 | Replace `license_public.key` on disk | Drop attacker pubkey at `deploy/vendor/license_public.key` | Same residual risk as PT-D01 |

#### Tier E - Misdirection (automated + manual)

| ID | Test | Pass criteria |
| :--- | :--- | :--- |
| PT-E01 | Decoy verify not in prod call graph | Holdout: `go test ./internal/licensing/ -run DecoyUnreachable` PASS |
| PT-E02 | Patch decoy branch only | Ingest still blocked with valid JWT on disk (integration or lab script) |
| PT-E03 | Patch `LicenseFilter` only | Seed coupling blocks over-cap RPS (`PT-A04`) |
| PT-E04 | Patch pubkey to decoy XOR block | JWT verify fails or MCK path fails; ingest blocked |
| PT-E05 | Open decoy sealed blob | `OpenAsset` with production label still requires valid MCK |
| PT-E06 | Hot path disasm/strings | No `VerifyJWT` / `ed25519` symbol anchor on `/track` build (`release_strings_gate`) |
| PT-E07 | Status API vs filter | `GET /api/v1/license/status` state matches `LicenseFilter` on clean profile |
| PT-E08 | Guard trip vs verify offset | `text_tamper` invalidates epoch without patching verify function (unit test) |

```bash
# Tier E subset (after misdirection slugs land)
go test ./internal/licensing/ -run 'Decoy|Misdirect' -count=1
go test ./internal/ingestion/ -run 'LicenseRPSFilter_seedCoupling|LicenseFilter' -count=1
bash scripts/ci/release_strings_gate.sh bin/garbled-release/tracker
```

#### Tier F - Cost amplification (after cost slugs land)

| ID | Test | Pass criteria |
| :--- | :--- | :--- |
| PT-F01 | MCK stretch on recheck | Invalid JWT brute script: each attempt triggers Argon2id (~64 MiB) on recheck path only |
| PT-F02 | Tracker-only crack insufficient | Patch tracker ingest; processor settlement or control enterprise route still blocked |
| PT-F03 | Feature JWT patch without MCK byte | `open_rtb` claim on without matching `mck[16]` bit -> OpenRTB denied |
| PT-F04 | JWT file swap without sidecar HMAC | Replace `var/license.jwt`; recheck fails before ingest |
| PT-F05 | `kid` change invalidates MCK | Re-issue with new `kid`; old MCK golden vector fails `DeriveMCK` |
| PT-F06 | Release `mck_info_label` bump | `testdata/mck_derivation.json` updated; old label yields wrong seed |
| PT-F07 | Recheck jitter | Two hosts same license: recheck not aligned to wall 5m boundary |

```bash
go test ./internal/licensing/ -run 'DeriveMCK|MCK_|ArgonRecheck' -count=1
go test ./internal/ingestion/ -run 'seedCoupling|OpenRTBLicenseAllowed' -count=1
# After processor slug:
go test ./internal/controlplane/ -run 'License|FeatureSeed' -count=1
```

### Close checklist

- [x] `scripts/security/license_pentest.sh` runs tiers A-C
- [x] `make license-pentest` entrypoint
- [x] `docs/DEVELOPMENT.md` links to this slug
- [x] Tier D steps PASS/FAIL table updated after hardening slugs land

---

## `ed25519_alg_pin_and_claims`

**Priority:** sec_p0

**Status:** Shipped.

**Gap (was):** `VerifyJWT` verified signature with provided key but did not require JWS header `alg:"EdDSA"`.

### Implementation

1. In `VerifyJWT` / `VerifyJWTResolved`, decode header JSON; require `alg` in allowlist (`EdDSA` only).
2. Reject `none` and empty alg before crypto.
3. Holdout: `TestVerifyJWT_rejectsWrongAlg` with `HS256` header + Ed25519 body.

### Pentest

```bash
go test ./internal/licensing/ -run 'WrongAlg|AlgPin' -count=1
```

Craft token with header `{"alg":"HS256","kid":"2026-01"}` and Ed25519 sig; expect `ErrInvalidAlgorithm`.

### Done gates

- [x] `validateJWTHeaderAlg` rejects `none`, empty, and non-EdDSA before verify
- [x] `go test ./internal/licensing/ -run 'WrongAlg|none alg|empty alg' -count=1`

---

## `pubkey_production_fail_closed`

**Priority:** sec_p0

**Status:** Shipped.

**Gap (was):** `ResolvePublicKey()` honored `AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY` and filesystem paths before embed on production profile.

### Implementation

1. When `ProfileFromEnv() == "production"` and `LicenseRequiredFromEnv()`, ignore env/file overrides unless `AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY_OVERRIDE=1` (lab only).
2. Log one warning at startup if override used.
3. Document in install script: production compose must not set pubkey env.
4. Holdout integration: PT-D01 becomes PASS (ingest blocked).

### Pentest

```bash
# After implementation
AD_EVENT_PROCESSOR_PROFILE=production AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY="$(cat attacker.pub)" \
  go test ./tests/integration/ -run LicenseProtection_productionProfileIgnoresAttackerPubKeyEnv -count=1
```

### Done gates

- [x] Production profile uses embedded pubkey unless `AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY_OVERRIDE=1`
- [x] Override logs one startup warning
- [x] Unit + integration holdout tests

---

## `license_guard_release_matrix`

**Priority:** sec_p0

**Status:** Shipped.

**Gap (was):** `scripts/ci/license_red_team_garbled.sh` exported `LICENSE_GUARD=0`, so garbled release artifact was not guard-tested in CI.

### Implementation

1. Split garbled red-team into `LICENSE_GUARD=1` (default) and optional `LICENSE_GUARD=0` smoke for dev docs.
2. `release_garble.sh` already adds `license_guard` tag when `LICENSE_GUARD=1`.
3. Run `LICENSE_GDB_SMOKE=1` against `bin/garbled-release/tracker` built with guard on.

### Pentest

```bash
LICENSE_GUARD=1 bash scripts/ci/release_garble.sh /tmp/guarded tracker
LICENSE_GDB_SMOKE=1 go test -tags=license_guard ./internal/licensing/ -run GDBAttachDenied -count=1
bash scripts/ci/release_strings_gate.sh /tmp/guarded/tracker
```

### Done gates

- [x] `license_red_team_garbled.sh` defaults `LICENSE_GUARD=1`
- [x] Guard unit tests run after garbled build
- [x] `LICENSE_GUARD=0` remains available for lab smoke (`license_guard_off_smoke.sh`)

---

## `argon2_hwid_params_rfc9106`

**Priority:** sec_p1

**Status:** **Shipped** (2026-08-26)

**Gap:** Parameters work but are not justified in operator docs; risk of arbitrary tuning without measurement.

### Implementation

1. Table in `deploy/vendor/KEYS.md` citing RFC 9106 Argon2id m=64 MiB, t=3, p=4.
2. `TestHWID_ArgonParamsDocumented` + RFC comment in `hwid.go`.
3. `BenchmarkHostHWID` in `hwid_bench_test.go`.

### Pentest

| Step | Command | Pass |
| :--- | :--- | :--- |
| Determinism | `go test ./internal/licensing/ -run 'HWID|HashHWID' -count=1` | Same telemetry -> same hash (P-HWID-01) |
| Sensitivity | `TestDeriveMCK_Sensitivity` | Different HWID -> different MCK |

---

## `argon2_hwid_telemetry_expand`

**Priority:** sec_p2

**Status:** **Shipped** (2026-08-26)

**Gap:** `machine-id` is in legacy fingerprint only, not Argon2 v2; no PCIe/NIC ring claims (do not add fake signals). Optional: stable `systemd-id128` when DMI absent on cloud.

### Implementation

1. Document v1 telemetry fields in `GET /api/v1/license/status` (`hwid_v2` derivation inputs).
2. Optional v2: add `MachineID` field to `HWIDTelemetry` behind feature flag `hwid_v3` with migration note for re-issue.
3. Do **not** add ethtool/NIC ring buffers without measurable bind stability data.

### Pentest (HWID spoof lab)

```bash
# Manual: spawn VM, bind-mount fake sysfs paths from deploy/vendor/fixtures/hwid_spoof/ (create fixture)
go run ./cmd/license-issue --hwid-v2 "$(./scripts/lab/hwid_hash.sh)" ...
```

Pass: mismatch vs donor license -> `ErrFingerprintMismatch` on recheck.

---

## `garble_seed_and_literals_policy`

**Priority:** sec_p1

**Status:** **Shipped** (2026-08-26)

**Gap:** `GARBLE_SEED` optional locally; tracker `-literals` impact measured only when `GARBLE_LITERALS_P99_SMOKE=1`.

### Implementation

1. CI release: fail if `GARBLE_SEED` unset (`release_garble.sh` via `release_garble_seed_ok`; `RELEASE_GARBLE_SKIP_SEED=1` local only).
2. Record decision: tracker stays `GARBLE_LITERALS=0` unless `garble_literals_p99_smoke.sh` passes with <12% p99 regression.
3. Document `GARBLE_LITERALS_TRACKER=1` as experimental in `docs/DEVELOPMENT.md`.
4. `garble_literals_policy_gate.sh` asserts seed policy; `garble_literals_eval.sh` exports default eval seed.

### Pentest

```bash
bash scripts/ci/garble_literals_policy_gate.sh
GARBLE_LITERALS_P99_SMOKE=1 bash scripts/test/garble_literals_p99_smoke.sh
```

---

## `garble_strings_surface_audit`

**Priority:** sec_p1

**Status:** **Shipped** (2026-08-26)

**Gap:** XOR pubkey masks are recoverable; gates block obvious hex but not reconstructed key.

### Implementation

1. `scripts/lib/release_strings_patterns.sh`: core symbol anchors on all garbled binaries; extended patterns (`license_public`, `embeddedPubKey`, `EdDSA`) and raw embedkey byte needles on processor/control only (`-literals=1`; tracker literals off keeps path strings).
2. `release_strings_gate.sh` scans garbled tracker/processor/control in `license_red_team_garbled.sh` and tier B pentest.
3. Split embedkey material into `internal/licensing/embedkey/` (`material.go`, `xor_mask.go`).

### Pentest

```bash
bash scripts/ci/release_garble.sh /tmp/garble-audit tracker
python3 - <<'PY'
# Recover XOR embed from binary - document that reverser can; defense is legal + MCK + updates
import re, sys
# Manual step: compare recovered key to vendor pubkey out-of-band
PY
strings /tmp/garble-audit/tracker | rg -i 'license|ed25519|jwt' || true
```

Pass: no `VerifyJWT`, `internal/licensing`, or raw vendor pubkey hex.

---

## `license_guard_ptrace_fail_closed`

**Priority:** sec_p1

**Status:** **Shipped** (2026-08-26)

**Gap:** When ptrace watchdog returns `skip` (Yama `ptrace_scope`), process continues with warn only.

### Implementation

1. `AD_EVENT_PROCESSOR_LICENSE_GUARD_PTRACE_REQUIRED=1` trips guard on `skip`; default on production profile when license required.
2. Dev laptops: set `GUARD_PTRACE_REQUIRED=0` or use `LICENSE_MODE=dev`.
3. `docs/DEVELOPMENT.md` documents `ptrace_scope`.

### Pentest

```bash
# ptrace_scope=1 (restrict attach)
cat /proc/sys/kernel/yama/ptrace_scope
bash scripts/test/license_gdb_guard_smoke.sh
# After hardening with PTRACE_REQUIRED=1: expect startup failure or trip
```

---

## `license_guard_frida_needles`

**Priority:** sec_p2

**Status:** **Shipped** (2026-08-26)

**Gap:** `guardSuspiciousMapNeedles` is a fixed XOR list (frida, gdb, vmm).

### Implementation

1. Add needles for `linjector`, `gum-js`, common Frida agent path fragments (ASCII only per `naming.mdc`).
2. Property test: needles must not appear verbatim in binary (`go test -tags=license_guard`).
3. Rotate needles when public Frida default paths change.

### Pentest

PT-D06: load Frida with renamed `.so` -> expect bypass (document residual risk). Default agent name -> guard trip within 8 s.

---

## `mck_seed_coupling_mandatory_release`

**Priority:** sec_p1

**Status:** **Shipped** (2026-08-26)

**Gap:** `LicenseSeedCouplingEnabled()` false in dev/unsealed mode by design; release must always enable for sealed SKUs.

### Implementation

1. `scripts/ci/mck_seed_coupling_release_gate.sh` asserts install templates default `LICENSE_MODE=file` (not dev).
2. Wired in `license_red_team.sh`.

### Pentest

```bash
go test ./internal/ingestion/ -run 'seedCoupling|OpenRTBLicenseAllowed_seed' -count=1
```

Patch-only license bypass without valid MCK: RPS filter returns `ErrRateLimitExceeded` when `max_rps > 0`.

---

## `asset_seal_salt_release_gate`

**Priority:** sec_p1

**Status:** **Shipped** (2026-08-26)

**Gap:** `ASSET_SEAL_SALT` injected via `-ldflags` only when set; release without salt weakens HKDF salt for asset AEAD.

### Implementation

1. `release_garble.sh` requires `ASSET_SEAL_SALT` (same local skip as `GARBLE_SEED`).
2. `asset_seal_salt_smoke.sh` fails when `RELEASE_GARBLE=1` without salt.
3. `license_red_team_garbled.sh` exports default eval salt; GitHub `release-images.yaml` already computes per-tag salt.

### Pentest

```bash
ASSET_SEAL_SALT="$(openssl rand -hex 32)" bash scripts/ci/asset_seal_salt_smoke.sh
go test ./internal/edge/ -run Sealed -count=1
go test ./internal/ingestion/ -run ResolveUnifiedFilterLua -count=1
```

Tamper sealed blob one byte -> `ErrSealTampered`.

---

## `hwid_spoof_lab_fixture`

**Priority:** sec_p1

**Status:** **Shipped** (2026-08-26)

**Gap:** No reproducible HWID spoof environment for QA.

### Implementation

1. `deploy/vendor/fixtures/hwid_spoof/README.md` with bind-mount recipes for QEMU/libvirt.
2. `scripts/lab/hwid_collect.sh` + `TestHWID_LabCollectPrint` / `LabCollectHWID()`.

### Pentest

Manual PT-D03 with fixture; record hash match/mismatch in test report template.

---

## `binary_patch_lab_procedure`

**Priority:** sec_p2

**Status:** **Shipped** (2026-08-26)

**Gap:** CFG patching narrative untested; document expected behavior vs guard.

### Implementation

1. Manual procedure: `deploy/vendor/fixtures/binary_patch/README.md`.
2. CI catalog: `bash scripts/lab/binary_patch_lab.sh` (PT-D04/D07/E08 automated proxies).
3. Wired in `license_red_team_extended.sh` step 11 and `license_pentest.sh` tier D.

### Procedure (lab only)

1. Build guarded release: `LICENSE_GUARD=1 bash scripts/ci/release_garble.sh /tmp/patch-lab tracker`.
2. Copy binary; use `hexedit` or `gdb` to patch one byte in `.text` (if attach blocked, patch on disk and run).
3. Start tracker with valid license.

| Guard | On-disk patch | In-memory patch (if attach works) |
| :--- | :--- | :--- |
| Off | May run until recheck | May run |
| On | May run until first text hash probe | Trip `text_tamper` -> EXPIRED |

4. Patch branch in `LicenseFilter` equivalent (find via differential testing): ingest may work; verify seed coupling still blocks over-cap RPS.

Pass for defense: combined layers block realistic crack path per PT-D04 + PT-A04.

---

## Misdirection design (reverser cost)

**Goal:** After static and dynamic analysis, there is no single branch where a patch yields full ingest. Legitimate customers see unchanged behavior on a clean production profile.

**Not in scope:** honeypot env vars, fake `ACTIVE` in admin API, delayed failures for drama, or any UX aimed at the attacker. Misdirection is **structural noise** only.

### Architecture rules

| Rule | Rationale |
| :--- | :--- |
| No monolithic `IsLicensed()` | One bool is one patch target |
| Verify only on cold/background paths | Hot path reads `fileLicenseSnapshot` only (`registry_license.go`) |
| State is derived | `MCK` -> `feature_seed` -> `SeedGateRPS` / sealed `OpenAsset`; independent of a single JWT bool |
| Decoy surfaces are unreachable in prod | Dead code that **looks** live in disasm; never called from `recheckLicenseFile` |
| Guard trip is decoupled from Verify | `InvalidateLicenseEpoch` in guard loop; patch near `ed25519.Verify` does not disable text hash |
| Recheck interval (~5m) | Runtime state can diverge from static CFG expectations |

### Layer map (what actually blocks ingest)

| Layer | Blocks ingest when invalid | Obvious in disasm? |
| :--- | :--- | :--- |
| `LicenseFilter` | `EXPIRED` / `REVOKED` snapshot | Often (misdirect: decoy filter elsewhere) |
| `LicenseRPSFilter` | Over cap + seed coupling | Medium |
| `SeedGateRPS` / `SeedGateOpenRTB` | Invalid `feature_seed` | Low (bitwise mix) |
| MCK + `OpenAsset` | Sealed Lua / edge BPF | Low (HKDF label) |
| `LicenseEpochInvalid` / guard | Epoch invalid after tamper | Low (background goroutine) |
| HWID bind | Recheck fails file verify | Medium (cold path) |

### Banned misdirection patterns

| Pattern | Why banned |
| :--- | :--- |
| API reports ACTIVE while filter rejects | Operator and integrator trust break; `anti-slop.mdc` |
| Metrics lie on clean profile | Same |
| Honeypot `*_BYPASS` env | Teaches attackers; no structural benefit |
| Panic or crash on tamper | Obvious signal and ops blast radius |
| Crypto or guard logic on `/track` request path | SLA and alloc gates |

---

## `license_decoy_verify_unreachable`

**Priority:** misdirect_p1

**Status:** **Shipped** (2026-08-26)

**Gap:** Reverser searching for `ed25519.Verify` or a post-verify branch finds one CFG that appears to gate runtime. Today the real gate is snapshot + derivatives; there is no deliberate decoy chain.

### Implementation

1. `internal/licensing/decoy_verify.go` + `decoy_public_key.go`: decoy verify chain with separate XOR pubkey.
2. `decoyLicensed` not wired to `ingestion.Registry`.
3. `TestDecoyVerify_UnreachableFromRecheck` scans production wiring files.

### Pentest

PT-E01, PT-E02. Patch decoy branch to always succeed; with valid `var/license.jwt` on disk, `LicenseFilter` still enforces snapshot.

```bash
go test ./internal/licensing/ -run DecoyUnreachable -count=1
```

---

## `license_decoy_pubkey_candidate`

**Priority:** misdirect_p1

**Status:** **Shipped** (2026-08-26)

**Gap:** XOR recovery yields exactly one 32-byte candidate tied to real verify.

### Implementation

1. `decoy_public_key.go`: `decoyPubKeyMasked` / `decoyPubKeyMask` (not vendor key).
2. `release_strings_gate` still forbids vendor pubkey hex; decoy masks allowed.
3. `TestDecoyEmbeddedPublicKey_distinctFromProduction`.

### Pentest

PT-E04. Patch embed to decoy XOR result; valid vendor-signed JWT must fail verify on decoy key.

---

## `license_state_derivatives_only`

**Priority:** misdirect_p1

**Status:** **Shipped** (2026-08-26)

**Gap:** Patching snapshot `state` to `ACTIVE` may suffice when seed coupling off (dev). Release must require derivatives.

### Implementation

1. Ingest uses filter + `SeedGateRPS` / `SeedGateOpenRTB`; `IngestAllowed` test/catalog only.
2. `TestLicenseStatePatchInsufficient`: ACTIVE snapshot + invalid seed -> RPS filter rejects.

### Pentest

PT-E03, existing `LicenseRPSFilter_seedCoupling`, `OpenRTBLicenseAllowed_seedCoupling`.

```bash
go test ./internal/ingestion/ -run 'seedCoupling|OpenRTBLicenseAllowed_seed' -count=1
```

---

## `license_sealed_decoy_embed`

**Priority:** misdirect_p1

**Status:** **Shipped** (2026-08-26)

**Gap:** Sealed assets are the hidden gate; only one blob label exists in tree. Reverser finding any plaintext Lua assumes it is the Redis script.

### Implementation

1. `internal/ingestion/decoy_unified_filter.lua` + `decoy_assets.go` (not used by `resolveUnifiedFilterLuaSource`).
2. `internal/edge/decoy_assets.go` ELF-shaped stub (not used by `loadEdgeObjectsFromSealed`).
3. Holdout tests in `decoy_assets_test.go` (ingestion + edge).

### Pentest

PT-E05. Replace decoy embed bytes in binary; unified filter still resolves only via MCK.

```bash
go test ./internal/ingestion/ -run 'ResolveUnifiedFilterLua|Decoy' -count=1
go test ./internal/edge/ -run 'Sealed|Decoy' -count=1
```

---

## `license_hot_path_no_crypto_anchor`

**Priority:** misdirect_p1

**Status:** **Shipped** (2026-08-26)

**Gap:** No CI assertion that tracker hot packages do not import `crypto/ed25519` for license.

### Implementation

1. `scripts/ci/license_hot_path_anchor_gate.sh`: forbids `ed25519.Verify` / `VerifyJWT` in `handler*.go`, `filter_license*.go`; registry recheck may call `VerifyLicenseFile` only.
2. Wired in `license_red_team.sh` Linux block.

### Pentest

PT-E06.

```bash
bash scripts/ci/license_hot_path_anchor_gate.sh
bash scripts/ci/release_strings_gate.sh bin/garbled-release/tracker
```

---

## `license_guard_decoupled_trip`

**Priority:** misdirect_p2

**Status:** **Shipped** (2026-08-26)

**Gap:** Guard and verify live in same package; reverser may assume one patch disables both.

### Implementation

1. Guard trips via `runGuardProbe` without calling verify (`TestGuard_TripWithoutVerifyCall`, `TestGuard_TextTamper`).
2. Decoy verify in separate files from `guard_linux.go`.

### Pentest

PT-E08.

```bash
go test -tags=license_guard ./internal/licensing/ -run 'Guard_TextTamper|Guard_TripWithout' -count=1
```

---

## `license_cold_near_gates_catalog`

**Priority:** misdirect_p2

**Status:** **Shipped** (2026-08-26)

**Gap:** Few cold functions **look** like enforcement (`CheckHostActivation`, skew watch, `IngestAllowed` in tests). Reverser short list is small.

### Implementation

1. `decoy_cold.go`: `DeploymentCredentialRefresh`, `RuntimeEntitlementSnapshot` (no snapshot effect).
2. `cmd/control/main.go` calls decoy cold paths once at startup when `LICENSE_MODE=dev` (debug log only).

### Real vs decoy catalog (maintain when adding slugs)

| Symbol / area | Role |
| :--- | :--- |
| `recheckLicenseFile` | **Real** snapshot writer |
| `VerifyLicenseFile` | **Real** JWT + bind |
| `LicenseFilter.Check` | **Real** ingest gate |
| `DeriveMCK` / `OpenAsset` | **Real** sealed assets + seed |
| `StartLicenseGuard` / `runGuardProbe` | **Real** tamper response |
| `CheckHostActivation` | **Real** multi-host bind (cold) |
| `IngestAllowed` | Catalog/tests; not hot path |
| `decoyVerify*` | **Decoy** only (`decoy_verify.go`) |
| `decoyPubKey*` | **Decoy** only (`decoy_public_key.go`) |
| `DeploymentCredentialRefresh` / `RuntimeEntitlementSnapshot` | **Decoy** cold (`decoy_cold.go`) |

### Pentest

Manual: enumerate exported licensing symbols in garbled binary; confirm multiple verify-like clusters; patch each cluster in lab - only one affects ingest (PT-E02).

---

## Cost amplification and SPOF distribution

**Goal:** No single crack point yields full appliance behavior. Each bypass attempt pays CPU/RAM (cold path) or repeats work across binaries/releases. Legitimate customer: unchanged SLA on `/track`.

**Offline model limit:** Root owner can always win eventually. Target economics: stable mass crack costs more than SKU + per-release maintenance.

### Independence chain (today)

```text
license.jwt (disk)
  -> Ed25519 verify + HWID bind (recheck, tracker)
  -> MCK = HKDF(ikm=sig|payload|hwid|kid, salt=deployment_id, info=license-mck-v2)
  -> feature_seed = f(MCK) -> SeedGateRPS / SeedGateOpenRTB
  -> OpenAsset(MCK, label) -> unified-filter.lua, edge BPF
  -> guard epoch / skew watch -> seed_valid=false
```

**Remaining SPOF:** `fileLicenseSnapshot` on tracker; processor/control lack full recheck/MCK chain.

### Memory-hard placement (RFC 9106)

| Surface | Allowed | Forbidden |
| :--- | :--- | :--- |
| `HostHWID()` boot (`sync.Once`) | Argon2id m=64MiB t=3 p=4 | - |
| `recheckLicenseFile` goroutine | Argon2id stretch of MCK (`StretchMCKForRecheck`) | - |
| `runGuardProbe` on tamper signal | optional Argon2 before trip (proposed) | - |
| `LicenseFilter.Check`, `/track` | atomic load only | Argon2, Ed25519, HKDF |

Ed25519 verify (~10 us) is **not** a cost amplifier - do not call per event.

### Reverser effort model (indicative)

| Work item | Hours (skilled) |
| :--- | ---: |
| garble layout + strings gates | 15-25 |
| locate real verify vs decoy | 10-20 |
| reverse MCK IKM + HKDF info | 15-30 |
| sealed lua + BPF labels | 10-15 |
| seed bitwise gates | 5-10 |
| guard decoupled from verify | 10-20 |
| per-release garble seed diff | 8-16 |

Full appliance crack (tracker + processor + control + sealed assets): **~80-150 h** plus **8-16 h/release** maintenance.

### Banned cost patterns

| Pattern | Why |
| :--- | :--- |
| Argon2 on hot path | SLA / alloc gates |
| Ed25519 per `/track` event | cheap for attacker, expensive for you |
| Online-only enforcement | contradicts offline JWT model |
| Degrade legit customer on clean profile | support cost |

---

## `mck_argon2_recheck_stretch`

**Priority:** cost_p1

**Status:** **Shipped** (2026-08-26)

**Gap:** `DeriveMCK` is HKDF-only (fast). Attacker scripts can try many JWT/HWID combinations at negligible RAM cost between rechecks.

**Target:** After successful HKDF MCK, derive `MCK_work = Argon2id(MCK, salt=deployment_id, m=64MiB, t=3, p=4)` on **recheck only**. `FeatureSeedFromMCK` reads `MCK_work`.

### Implementation

1. `StretchMCKForRecheck` in `mck_stretch.go` (reuses `hwidArgon*` constants).
2. `registry_license.recheckLicenseFile` calls `DeriveMCKWorkForRecheckFromLicenseFile` for seed coupling.
3. `testdata/mck_derivation.json` `mck_stretch_v1` fixtures; `BenchmarkStretchMCKForRecheck`.
4. `DeriveMCK` / `OpenAsset` paths unchanged (unstretched HKDF MCK).

### Pentest

PT-F01. Script 1000 invalid tokens: observe ~64 MiB alloc per recheck attempt, not per track event.

```bash
go test ./internal/licensing/ -run 'DeriveMCK|ArgonRecheck|FeatureSeed' -count=1
```

---

## `license_recheck_processor_control`

**Priority:** cost_p1

**Status:** **Shipped** (2026-08-26)

**Gap:** `StartLicenseRecheck` only in `cmd/tracker/main.go`. Partial crack: patch tracker snapshot, leave control API and processor settlement.

### Implementation

1. `internal/licensing/file_watcher.go`: shared `RecheckLicenseFile` + `StartFileLicenseRecheck` (MCK stretch + `PublishFeatureSeed`).
2. `registry_license.go` delegates recheck to licensing package.
3. `cmd/processor/main.go` and `cmd/control/main.go` start file recheck when license required.
4. `licenseIngestReady` requires `FeatureSeedValid()` when seed coupling on; `LicenseWatcher` publishes seed on verify.

### Pentest

PT-F02.

```bash
go test ./tests/integration/ -run LicenseProtection -count=1
# Lab: patch tracker LicenseFilter only; processor/control must fail seed or recheck gate
```

---

## `entitlements_mck_bitwise_coupling`

**Priority:** cost_p1

**Status:** Shipped.

**Gap:** `ent.Features` from JWT JSON alone gates OpenRTB after `SeedGateOpenRTB`. Patching claims in memory may suffice without MCK.

**Target:** Critical features require **both** JWT claim and MCK byte predicate, e.g. `open_rtb_allowed = ent.Features.OpenRTBEnabled() && (mck[16]&0x01 != 0)`.

### Implementation

1. Store `mckFeatureBits` byte in snapshot beside `feature_seed` (set on recheck from stretched MCK).
2. Update `SeedGateOpenRTB` and any RTB live flags in `license_gate.go` / `OpenRTBAllowed`.
3. Document bit map in `jwt_claims.go` comment (one line per bit).
4. Holdout: JWT with `rtb_live` true, `mckFeatureBits&1==0` -> OpenRTB off.

### Pentest

PT-F03.

```bash
go test ./internal/ingestion/ -run 'OpenRTBLicenseAllowed|seedCoupling' -count=1
```

---

## `license_file_hmac_sidecar`

**Priority:** cost_p1

**Status:** Shipped.

**Gap:** Root can replace `var/license.jwt` with another file; recheck re-verifies signature but file swap is a single obvious target.

**Target:** Sidecar `var/license.jwt.mac` = HMAC-SHA256(`MCK_work`, file_bytes). Recheck rejects mismatch before Ed25519 (or after verify, before snapshot publish).

### Implementation

1. `internal/licensing/file_mac.go`: `WriteLicenseMAC`, `VerifyLicenseMAC` using MCK from recheck path.
2. Regenerate sidecar on successful `POST /api/v1/license/apply` and on first valid recheck.
3. File permissions: same owner as JWT, `0600`.
4. Holdout: valid JWT content with wrong MAC -> snapshot stays EXPIRED.

### Pentest

PT-F04.

```bash
go test ./internal/licensing/ -run 'LicenseMAC|FileMAC' -count=1
```

---

## `mck_hkdf_kid_in_ikm`

**Priority:** cost_p1

**Status:** Shipped.

**Gap:** `DeriveMCK` IKM is `sig|payload|hwid` only. `kid` rotation changes verify key but not MCK for same JWT body if attacker re-signs.

**Target:** Append `kid` bytes to IKM (or separate HKDF info segment). `license-issue` and golden vectors must include `kid` from header.

### Implementation

1. Extend `DeriveMCK` to decode header `kid`; append to IKM.
2. Bump label to `license-mck-v2` in same slug or pair with `mck_info_label_release_version`.
3. Update `mck_derivation.json`, `license_verify_tier.sh` entangle-1.

### Pentest

PT-F05.

```bash
go test ./internal/licensing/ -run 'DeriveMCK|JWTKeyID' -count=1
```

---

## `sealed_assets_processor_control`

**Priority:** cost_p1

**Status:** Shipped.

**Gap:** Sealed consumers: tracker unified-filter, edge BPF only. Reverser can ignore processor/control binaries for ingest-only crack.

**Target:** At least one sealed blob per control and processor (policy JSON, processor CH ingest config, or ML batch schema) opened via `OpenAsset` with unique `AssetLabel*`.

### Implementation

1. Add labels in `asset_cipher.go` constants.
2. Build-time seal in release pipeline (`ASSET_SEAL_SALT` required).
3. `cmd/processor/main.go`: fail startup or degrade cold path when open fails (not tracker hot path).
4. Extend `license_verify_tier.sh` entangle-4 with processor/control tests.

### Pentest

PT-F02 extension: processor without valid MCK does not start sealed worker.

```bash
go test ./internal/ingestion/ -run ResolveUnifiedFilterLua -count=1
go test ./internal/edge/ -run Sealed -count=1
```

---

## `processor_settlement_seed_gate`

**Priority:** cost_p1

**Status:** Shipped.

**Gap:** Processor runs conversion settlement without `FeatureSeedValid` check. Tracker crack may leave settlement/cold analytics working.

**Target:** Processor settlement hook: skip or mark conversions when `!licensing.FeatureSeedValid()` under production license profile (align with `license_recheck_processor_control`).

### Implementation

1. Wire `licensing` watcher into processor main (shared with slug above).
2. Check in conversion batch path before payout/postback enqueue (cold path - OK per `architecture.mdc`).
3. Holdout: `processor_settlement_seed_gate_test.go` with invalid seed.

### Pentest

PT-F02.

```bash
go test ./cmd/processor/... -run SeedGate -count=1
# or package under internal/ingestion processor hooks
```

---

## `mck_info_label_release_version`

**Priority:** cost_p1

**Status:** Shipped.

**Gap:** `mckInfoLabel = "license-mck-v1"` static. Old cracks survive across releases if binary patch is stable.

**Target:** Per major release bump `license-mck-v2` (or embed minor in `ASSET_SEAL_SALT` rotation policy). Vendor re-issues JWT not required; buyer replaces binary + license file recheck refreshes MCK from same JWT **only if** IKM layout unchanged - when label bumps, old patched binaries compute wrong seed.

### Implementation

1. Const or ldflags `-X licensing.mckInfoLabel=...` set in `release_garble.sh`.
2. Document in `KEYS.md`: release notes must mention MCK label bump.
3. CI: `mck_derivation.json` matches release label.

### Pentest

PT-F06.

```bash
go test ./internal/licensing/ -run 'MCK_Golden|mck_derivation' -count=1
```

---

## `guard_tamper_argon2_before_trip`

**Priority:** cost_p2

**Status:** Shipped.

**Gap:** Guard trip is cheap; automated patch fuzz can hammer probes.

**Target:** On positive `suspicious_map` or `text_tamper` **before** `tripGuard`, run one Argon2id iteration (m=64MiB, t=1) keyed from `GuardTextFingerprint(hash)` to burn RAM/CPU on scripted attacks.

### Implementation

1. `guard_linux.go`: `runTamperStretch(reason string)` only when probe positive, not on clean loop.
2. Timeout cap: max 500 ms; skip stretch if `LICENSE_GUARD_STRETCH=0` (lab).
3. Holdout: tamper probe still trips after stretch.

### Pentest

Lab only: trigger fake maps scanner; measure RSS spike once per trip.

```bash
go test -tags=license_guard ./internal/licensing/ -run 'Guard_.*Stretch|Guard_TextTamper' -count=1
```

---

## `license_recheck_jitter`

**Priority:** cost_p2

**Status:** Shipped.

**Gap:** Fixed `5m` recheck (`LicenseFileRecheckInterval`) lets lab scripts race immediately after patch.

**Target:** Interval = base + jitter in `[0, deployment_hash % 120s]` per process start.

### Implementation

1. `registry_license.go`: compute jitter from `claims.DeploymentID` or file inode hash.
2. Document mean interval still ~5m for ops.
3. Holdout: two processes same license, recheck times differ (PT-F07).

### Pentest

PT-F07 (manual or test with injected clock).

---

## `license_epoch_pubsub_sync`

**Priority:** cost_p2

**Status:** Shipped.

**Gap:** Tracker, control, processor each hold local epoch/seed atomics. Partial hook of one process leaves others inconsistent in complex cracks.

**Target:** On recheck or guard trip, publish `license:epoch` on Redis (existing infra); subscribers invalidate local seed. Increases work to crack **all** processes or subscribe fake messages.

### Implementation

1. Publisher in `licensing` watcher after snapshot update.
2. Subscribers in control/processor/tracker `StartLicenseRecheck` companion.
3. Fail closed if Redis unavailable: keep local recheck only (no regression).
4. Do not add Redis to `/track` path.

### Pentest

PT-F02: trip guard on tracker; processor receives epoch invalid within one pubsub RTT + local poll.

```bash
go test ./internal/licensing/ -run 'EpochPubsub|LicenseEpoch' -count=1
```

---

## Operator hardening checklist (ship with install)

| Control | Production setting |
| :--- | :--- |
| License mode | `AD_EVENT_PROCESSOR_LICENSE_MODE=file` |
| License required | `AD_EVENT_PROCESSOR_LICENSE_REQUIRED=1` (production profile) |
| Guard | `AD_EVENT_PROCESSOR_LICENSE_GUARD=1` |
| Ptrace watchdog | `AD_EVENT_PROCESSOR_LICENSE_GUARD_PTRACE=1` |
| Pubkey override | **Unset** `AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY` |
| Private key on appliance | **Absent** |
| JWT path | `var/license.jwt` root-owned `0600` |
| Kernel | Document `yama/ptrace_scope`; prefer `1` or `2` with understanding of watchdog behavior |
| Filesystem | Read-only `/usr` or immutable tracker binary where ops model allows |

---

## CI mapping

| Gate | Slugs covered |
| :--- | :--- |
| `make license-red-team` | PT-A*, partial PT-C |
| `make license-verify` | Tier A + optional garble |
| `license_red_team_extended.sh` | PT-A04-A06, PT-C01, PT-D04/D07/E08 proxies |
| `license_red_team_garbled.sh` | PT-B* (fix guard gap) |
| `hwid_strings_gate.sh` | PT-A07 |
| `public_key_strings_gate.sh` | PT-A08 |
| `license_pentest.sh` | Tiers A-C orchestration (`make license-pentest`); tier D proxies |
| `scripts/lab/binary_patch_lab.sh` | PT-D04, PT-D07, PT-E08 (automated proxies) |
| `license_hot_path_anchor_gate.sh` (planned) | PT-E06 misdirection |
| `license_verify_tier.sh` entangle-* | MCK, seal, HKDF differential |

### Recommended implementation order (all tracks)

```text
# sec_p0
pubkey_production_fail_closed, license_guard_release_matrix, ed25519_alg_pin_and_claims

# misdirect (no hot-path crypto)
license_hot_path_no_crypto_anchor
license_state_derivatives_only
license_decoy_pubkey_candidate
license_decoy_verify_unreachable
license_sealed_decoy_embed

# cost (cold path only)
mck_argon2_recheck_stretch
license_file_hmac_sidecar
mck_hkdf_kid_in_ikm
license_recheck_processor_control
sealed_assets_processor_control
processor_settlement_seed_gate
entitlements_mck_bitwise_coupling
license_epoch_pubsub_sync

# release churn
mck_info_label_release_version
asset_seal_salt_release_gate
garble_seed_and_literals_policy

# depth
guard_tamper_argon2_before_trip
license_recheck_jitter
license_guard_decoupled_trip
license_cold_near_gates_catalog
```

### Economic check (per slug PR)

| Question | Pass |
| :--- | :--- |
| Does this run on `/track`? | Must be no for memory-hard |
| Does tracker-only patch still leave settlement/admin broken? | Must be yes after processor/control slugs |
| Does crack survive next release without rework? | Should be no when label/salt/garble seed bump |
| Legit pilot on clean profile | `make license-red-team` pass |

---


## Related

- `deploy/vendor/KEYS.md` - key rotation
- `deploy/vendor/sku.yaml` - RPS/host limits enforced after crypto passes
- `.cursor/rules/licensing.mdc` - kill switches and trial registry
- `scripts/security/license_red_team.sh` - baseline automated red team
