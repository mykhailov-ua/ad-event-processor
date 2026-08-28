# Licensing security reference

Pentest catalog and operator runbook for offline Ed25519 JWT licensing, Argon2id HWID bind, `garble` release builds, runtime guard, and MCK-sealed assets.

**Status:** Hardening slugs shipped (2026-08-26). New work adds pentest rows or operator controls only; do not reopen shipped slugs without a regression.

**Canonical implementation:** `internal/licensing/`, `internal/ingestion/registry_license.go`, `scripts/security/license_red_team.sh`, `.cursor/rules/licensing.mdc`.

**Out of scope:** online license servers, HSM on buyer VPS, marketing claims about unbreakable protection.

---

## Threat model (buyer VPS)

| Attacker capability | Realistic goal | Primary controls today |
| :--- | :--- | :--- |
| Root on appliance | Run without paying; clone license | HWID bind, trial registry at issue time, host activation limits |
| Root on appliance | Patch binary or hook memory | `license_guard` (Linux release), text hash, ptrace watchdog, maps scan |
| Root on appliance | Replace pubkey / JWT without patch | Embedded pubkey only in production profile (`pubkey_production_fail_closed`) |
| Binary-only reverser | Find branch after `ed25519.Verify` | garble, release strings gates, seed coupling, MCK, decoy surfaces |
| Operator mistake | Leave guard off in prod | Env kill switches documented; production profile should fail closed |

Offline licensing cannot prevent a determined root owner forever. Goal: raise cost above SKU price and block **mass redistributable** cracks across releases.

---

## Current baseline (code truth)

| Layer | Implementation | Release default |
| :--- | :--- | :--- |
| JWT | Ed25519 (`crypto/ed25519`), 3-part JWS (`verify.go`) | On |
| Pubkey | XOR-masked embed + file/env override (`public_key.go`) | Embed + optional file |
| HWID v2 | Argon2id t=3, m=65536 KiB, p=4 (`hwid.go`) | On Linux |
| Hot ingest gate | `LicenseFilter` + `LicenseRPSFilter` on atomic snapshot | `LICENSE_REQUIRED` in production |
| Seed coupling | MCK -> feature seed gates RPS/OpenRTB (`runtime_seed.go`) | `LICENSE_MODE=file\|enterprise` |
| Sealed assets | ChaCha20-Poly1305 via HKDF(MCK) (`asset_cipher.go`) | When `ASSET_SEAL` on |
| Runtime guard | `license_guard` tag: text SHA256, TracerPid, maps needles, ptrace child | `LICENSE_GUARD=1` in `release_garble.sh` |
| garble | `mvdan.cc/garble@v0.15.0`; literals default on control/processor only | `RELEASE_GARBLE=1` |
| License recheck | `StartFileLicenseRecheck` in tracker, processor, control | On when license required |
| MCK consumers | unified-filter sealed lua, edge BPF, processor CH policy, control runtime policy | On |

---

## Reference: tool and spec practices

### Argon2id (RFC 9106)

HWID here is **binding fingerprinting**, not password storage.

| Parameter | This repo (`hwid.go`) | Notes |
| :--- | :--- | :--- |
| Type | `argon2.IDKey` (Argon2id) | Correct choice |
| Time (`t`) | 3 | Boot-time cache only |
| Memory (`m`) | 65536 KiB (64 MiB) | Do not raise on hot path |
| Parallelism (`p`) | 4 | Match cores |

**Do not** raise `t`/`m` on hot path: `HostHWID()` is `sync.Once` at startup only.

### Ed25519 JWT (RFC 8032, RFC 7519)

| Practice | Status |
| :--- | :--- |
| Algorithm Ed25519 only (`alg` pinned to EdDSA) | Shipped |
| `kid` rotation | `key_resolver.go` cohort paths under `deploy/vendor/keys/<kid>/` |
| Max token size 16 KiB | Shipped |
| Private key never in buyer image | Shipped |

### garble (mvdan.cc/garble)

| Practice | This repo |
| :--- | :--- |
| Pin version | `GARBLE_VERSION=v0.15.0` in `release_garble.sh` |
| Pin seed | Required in release CI (`release_garble_seed_ok`) |
| `-literals` | Default on control/processor; off on tracker |
| Build tags | `license_guard` separate from garble; release uses both |

### Runtime guard (Linux)

| Practice | Limitation |
| :--- | :--- |
| Ptrace watchdog | Bypass via `LICENSE_GUARD=0` or VM |
| `.text` SHA256 via `/proc/self/mem` | In-memory patch detected |
| Maps substring scan | Trivial to rename gadget `.so` |
| Trip behavior | `InvalidateLicenseEpoch` -> EXPIRED ingest |

---

## Pentest catalog

Entrypoint: `bash scripts/security/license_pentest.sh` (tiers A-C automated; tier D manual). Also `make license-pentest`, `make license-red-team`, `make license-verify`.

Run from repo root on Linux amd64 unless noted.

### Tier A - Automated CI parity (no garble)

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

### Tier B - Garbled release binary

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

### Tier C - Runtime guard lab (Linux + gdb)

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

### Tier D - Manual root attacker (documented, not CI)

| ID | Procedure | Pass criteria (defense holds) |
| :--- | :--- | :--- |
| PT-D01 | Config pubkey injection | Production profile ignores env/file pubkey override |
| PT-D02 | Guard kill switch | `AD_EVENT_PROCESSOR_LICENSE_GUARD=0` on release build; lab-only |
| PT-D03 | HWID sysfs spoof | `deploy/vendor/fixtures/hwid_spoof/README.md` |
| PT-D04 | Patch `LicenseFilter` only | `bash scripts/lab/binary_patch_lab.sh` |
| PT-D05 | Pubkey XOR recovery | Attacker still needs private key or verify patch |
| PT-D06 | Frida gadget maps | Guard trips within probe interval |
| PT-D07 | `.text` in-memory patch | `deploy/vendor/fixtures/binary_patch/README.md` |
| PT-D08 | Replace `license_public.key` on disk | Same residual risk as PT-D01 |

### Tier E - Misdirection

| ID | Test | Pass criteria |
| :--- | :--- | :--- |
| PT-E01 | Decoy verify not in prod call graph | `go test ./internal/licensing/ -run DecoyUnreachable` PASS |
| PT-E02 | Patch decoy branch only | Ingest still blocked with valid JWT on disk |
| PT-E03 | Patch `LicenseFilter` only | Seed coupling blocks over-cap RPS (`PT-A04`) |
| PT-E04 | Patch pubkey to decoy XOR block | JWT verify or MCK path fails |
| PT-E05 | Open decoy sealed blob | Production `OpenAsset` still requires valid MCK |
| PT-E06 | Hot path disasm/strings | No `VerifyJWT` anchor on `/track` build |
| PT-E07 | Status API vs filter | `GET /api/v1/license/status` matches `LicenseFilter` on clean profile |
| PT-E08 | Guard trip vs verify offset | `text_tamper` invalidates epoch without patching verify |

```bash
go test ./internal/licensing/ -run 'Decoy|Misdirect' -count=1
go test ./internal/ingestion/ -run 'LicenseRPSFilter_seedCoupling|LicenseFilter' -count=1
bash scripts/ci/release_strings_gate.sh bin/garbled-release/tracker
```

### Tier F - Cost amplification

| ID | Test | Pass criteria |
| :--- | :--- | :--- |
| PT-F01 | MCK stretch on recheck | Argon2id (~64 MiB) on recheck path only |
| PT-F02 | Tracker-only crack insufficient | Processor/control still blocked |
| PT-F03 | Feature JWT patch without MCK byte | OpenRTB denied without matching `mck` bit |
| PT-F04 | JWT file swap without sidecar HMAC | Recheck fails before ingest |
| PT-F05 | `kid` change invalidates MCK | Golden vector fails `DeriveMCK` |
| PT-F06 | Release `mck_info_label` bump | `testdata/mck_derivation.json` updated |
| PT-F07 | Recheck jitter | Two hosts: recheck not aligned to wall 5m boundary |

```bash
go test ./internal/licensing/ -run 'DeriveMCK|MCK_|ArgonRecheck' -count=1
go test ./internal/ingestion/ -run 'seedCoupling|OpenRTBLicenseAllowed' -count=1
go test ./internal/controlplane/ -run 'License|FeatureSeed' -count=1
```

---

## Misdirection design (reverser cost)

**Goal:** No single branch where a patch yields full ingest. Legitimate customers see unchanged behavior on a clean production profile.

**Not in scope:** honeypot env vars, fake `ACTIVE` in admin API, panic on tamper.

| Rule | Rationale |
| :--- | :--- |
| No monolithic `IsLicensed()` | One bool is one patch target |
| Verify only on cold/background paths | Hot path reads `fileLicenseSnapshot` only |
| State is derived | MCK -> `feature_seed` -> `SeedGateRPS` / sealed `OpenAsset` |
| Decoy surfaces unreachable in prod | Never called from `recheckLicenseFile` |
| Guard trip decoupled from Verify | Patch near `ed25519.Verify` does not disable text hash |

### Real vs decoy catalog (maintain when adding symbols)

| Symbol / area | Role |
| :--- | :--- |
| `recheckLicenseFile` | **Real** snapshot writer |
| `LicenseFilter.Check` | **Real** ingest gate |
| `DeriveMCK` / `OpenAsset` | **Real** sealed assets + seed |
| `StartLicenseGuard` | **Real** tamper response |
| `decoyVerify*` / `decoyPubKey*` | **Decoy** only |
| `DeploymentCredentialRefresh` | **Decoy** cold |

---

## Cost amplification and SPOF distribution

```text
license.jwt (disk)
  -> Ed25519 verify + HWID bind (recheck)
  -> MCK = HKDF(ikm=sig|payload|hwid|kid, salt=deployment_id, info=license-mck-v2)
  -> feature_seed = f(MCK) -> SeedGateRPS / SeedGateOpenRTB
  -> OpenAsset(MCK, label) -> unified-filter.lua, edge BPF
  -> guard epoch / skew watch -> seed_valid=false
```

| Surface | Allowed | Forbidden |
| :--- | :--- | :--- |
| `HostHWID()` boot | Argon2id m=64MiB t=3 p=4 | - |
| `recheckLicenseFile` | MCK stretch (`StretchMCKForRecheck`) | - |
| `LicenseFilter.Check`, `/track` | atomic load only | Argon2, Ed25519, HKDF |

### Banned cost patterns

| Pattern | Why |
| :--- | :--- |
| Argon2 on hot path | SLA / alloc gates |
| Ed25519 per `/track` event | Cheap for attacker |
| Degrade legit customer on clean profile | Support cost |

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

---

## CI mapping

| Gate | Covers |
| :--- | :--- |
| `make license-red-team` | PT-A*, partial PT-C |
| `make license-verify` | Tier A + optional garble |
| `license_red_team_extended.sh` | PT-A04-A06, PT-C01, PT-D04/D07/E08 proxies |
| `license_red_team_garbled.sh` | PT-B* |
| `license_pentest.sh` | Tiers A-C orchestration |
| `scripts/lab/binary_patch_lab.sh` | PT-D04, PT-D07, PT-E08 proxies |
| `license_hot_path_anchor_gate.sh` | PT-E06 |
| `license_verify_tier.sh` entangle-* | MCK, seal, HKDF differential |

---

## Related

- `deploy/vendor/KEYS.md` - key rotation
- `deploy/vendor/sku.yaml` - RPS/host limits enforced after crypto passes
- `.cursor/rules/licensing.mdc` - kill switches and trial registry
- `scripts/security/license_red_team.sh` - baseline automated red team
