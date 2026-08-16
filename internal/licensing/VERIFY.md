# Licensing verification catalog

Property IDs referenced by tests in this package. V2 phases add rows in the same file before implementation merges.

## Authorization (P-C2)

Predicate: `Authz(token, host, t) ≜ V ∧ B ∧ R ∧ S`

| ID | Property | Test |
| --- | --- | --- |
| P-C2-01 | Invalid signature or payload ⇒ reject | `TestVerifyJWT/Tampered_Signature`, `TestVerifyJWT/Tampered_Payload` |
| P-C2-02 | Hard bind mismatch ⇒ reject | `TestVerifyDeploymentBind_hardMismatch`, `TestVerifyDeploymentBind_hwidMismatch`, `TestIntegration_LicenseProtection_hwidMismatchBlocksIngest` |
| P-C2-03 | Revoked ⇒ not ingestible | `TestClaimsRevoked_inJWT`, `TestDetermineEffectiveState_revokedAndJwtExpired` |
| P-C2-04 | Past grace ⇒ expired | `TestVerifyLicenseFile_validAndExpired`, `TestProperty_P_C3_02_JwtStateMonotonic` |
| P-C2-05 | PG ACTIVE without valid file JWT ⇒ ingest blocked | `TestIntegration_LicenseProtection_emptyPGRowBlocksIngest`, `TestIntegration_LicenseProtection_fakePGRowWithoutJWTBlocked` |

HWID v2 bind uses `hwid_hash` when present; legacy `bind.fingerprint` when `hwid_hash` is absent (`TestVerifyDeploymentBind_legacyFingerprintMatch`, `TestVerifyDeploymentBind_hwidTakesPrecedenceOverFingerprint`).

## JWT state machine (P-C3)

| ID | Property | Test |
| --- | --- | --- |
| P-C3-01 | `DetermineState` is pure in `(claims, t, revoked)` | `TestProperty_P_C3_01_StatePure` |
| P-C3-02 | Non-revoked JWT state is monotonic non-increasing in time | `TestProperty_P_C3_02_JwtStateMonotonic` |
| P-C3-03 | `IngestAllowed` false iff state ∈ {EXPIRED, REVOKED} | `TestIngestAllowed`, `TestProperty_P_C3_03_IngestAllowedStates` |

## Entitlement lattice (P-C4)

| ID | Property | Test |
| --- | --- | --- |
| P-C4-01 | `Effective(e, e)` stable under re-merge | `TestProperty_P_C4_01_EffectiveIdempotent` |
| P-C4-02 | `Effective` associative merge | `TestProperty_P_C4_02_EffectiveAssociative` |
| P-C4-03 | Customer limits never exceed deployment ceiling | `TestProperty_P_C4_03_DeploymentCeiling` |

## HWID v2 (V2-A)

| ID | Property | Test |
| --- | --- | --- |
| P-HWID-01 | Argon2id hash deterministic for fixed telemetry | `TestHWID_Deterministic`, `TestHWID_GoldenVectors` |
| P-HWID-02 | Missing DMI does not panic; stable hash | `TestHWID_VMFallback` |

## Embedded verify key (V2-D.3)

| ID | Property | Test |
| --- | --- | --- |
| P-KEY-01 | XOR-masked embedded key decodes to production Ed25519 pubkey | `TestPublicKey_EmbeddedObfuscation` |
| P-KEY-02 | Plaintext production pubkey hex not in `public_key.go` | `TestPublicKey_sourceNotPlainEmbeddedHex` |

## Release asset seal salt (V2-D.4)

| ID | Property | Test / gate |
| --- | --- | --- |
| P-D4-01 | `sha256(vault:tag)` derivation matches CI | `TestDeriveReleaseAssetSealSalt_matchesCIFormula` |
| P-D4-02 | Wrong release salt ⇒ AEAD open fails | `TestSealAsset_wrongReleaseSaltRejected` |
| P-D4-03 | CI smoke with `ASSET_SEAL_SALT` set | `bash scripts/ci/asset_seal_salt_smoke.sh` |

## HWID path obfuscation (V2-D.2)

| ID | Property | Test / gate |
| --- | --- | --- |
| P-D2-01 | XOR path tables decode to expected sysfs targets | `TestDecodeHWIDPath_knownIDs` (linux) |
| P-D2-02 | amd64 uses raw `openat`/`read` syscall shim | `hwid_read_amd64.s` |
| P-D2-03 | Release tracker has no plaintext HWID paths | `bash scripts/ci/hwid_strings_gate.sh` |

## MCK and asset sealing (V2-B)

| ID | Property | Test |
| --- | --- | --- |
| P-C6-01 | MCK deterministic for fixed JWT + HWID + deployment_id | `TestDeriveMCK_GoldenVector` |
| P-C6-02 | Different HWID ⇒ different MCK | `TestDeriveMCK_Sensitivity` |
| P-C6-03 | Seal/open round-trip; wrong MCK ⇒ open fails | `TestSealAsset_roundTrip`, `TestSealAsset_wrongMCK`, `TestSealAsset_bitFlipRejected` |
| P-C6-04 | Feature seed coupling gates RPS/OpenRTB when enabled | `TestFeatureSeed_couplingGates`, `TestLicenseRPSFilter_seedCouplingBlocksWithoutValidSeed`, `TestOpenRTBLicenseAllowed_seedCouplingBlocksActiveLicense` |
| P-C6-05 | Per-release `ASSET_SEAL_SALT` mixed into AEAD HKDF | `TestDeriveReleaseAssetSealSalt_matchesCIFormula`, `TestSealAsset_releaseSaltRoundTrip`, `TestSealAsset_wrongReleaseSaltRejected` |
| P-C6-08 | Valid license + sealed blob → `ebpf.NewCollection` | `TestSealedBPF_ValidMCKLoadsCollection` (linux + bpf2go) |
| P-C6-09 | Sealed vs unsealed XDP attach on lab host (V2-B.D2) | `bash scripts/test/sealed_bpf_xdp_smoke.sh` (`harness=kernel_xdp_attach_lo_generic`) |
| P-C6-10 | Sealed `unified-filter.lua` decrypt at tracker startup (V2-B.4) | `TestResolveUnifiedFilterLua_*`; `license_lua_seal_fail_total` on failure |

Dev mode (`AD_EVENT_PROCESSOR_LICENSE_MODE=dev`) skips seed coupling and uses unsealed assets (`TestLicenseAssetsUnsealed_devMode`).

## Clock skew watch (V2-C.5)

| ID | Property | Test |
| --- | --- | --- |
| P-C3-04 | Wall clock rewind vs monotonic ⇒ license expired | `TestSkewWatch_ClockRewind`, `TestSkewWatch_StepBackwardBetweenSamples` |
| P-C3-05 | NTP-sized drift within threshold ⇒ not expired | `TestSkewWatch_AllowsNTPSkewWithinThreshold` |
| P-C3-06 | Global skew watch forces expired state | `TestEvaluateClockSkew_globalWatch` |
| P-C3-07 | Registry recheck on skew ⇒ ingest blocked (red-team step 10) | `TestRegistry_licenseRecheck_clockSkewBlocksIngest`; `make license-red-team-extended` |

Disabled in `dev` mode (`TestLicenseSkewWatch_devModeDisabled`).

## License guard (V2-C.1–C.4)

| ID | Property | Test / gate |
| --- | --- | --- |
| P-C8-01 | TracerPid > 0 trips guard + epoch invalidate | `TestGuard_TracerPid` (`-tags=license_guard`, linux) |
| P-C8-02 | Suspicious maps trip guard | `TestGuard_SuspiciousMap` |
| P-C8-03 | `.text` hash mismatch trips guard | `TestGuard_TextTamper` |
| P-C8-04 | Env kill switch disables guard | `TestLicenseGuardEnv_killSwitch` |
| P-C8-05 | Ptrace watchdog child: busy tracer slot trips guard | `TestGuard_PtraceWatchdogHandshakeBusy` |
| P-C8-06 | Ptrace watchdog re-exec child claims parent tracer | `guard_watchdog_linux.go`; `AD_EVENT_PROCESSOR_LICENSE_GUARD_PTRACE=0` to disable |
| P-C8-07 | Guard off: engineering tooling documented | `bash scripts/test/license_guard_off_smoke.sh` (V2-C.D2) |
| P-C8-08 | Fault suite does not require `license_guard` tag | `bash scripts/ci/license_guard_fault_gate.sh` (V2-C.D3) |
| P-C8-09 | Release strings gate (symbols + pubkey + license plaintext) | `bash scripts/ci/release_strings_gate.sh`; `public_key_strings_gate.sh` (V2-C.D8) |
| P-C8-10 | gdb attach denied with guard + ptrace watchdog | `bash scripts/test/license_gdb_guard_smoke.sh` (`harness=license_guard_release`) |

Dev/CI builds omit `license_guard` tag; release-garble enables it (`scripts/ci/release_garble.sh`). Ops overrides: `AD_EVENT_PROCESSOR_LICENSE_GUARD=0` (all probes), `AD_EVENT_PROCESSOR_LICENSE_GUARD_PTRACE=0` (ptrace child only).

**Lab (V2-C.D1):** Full garbled `tracker` gdb drill on release host — same harness label; automated smoke uses test binary subprocess.

## Symbol scatter (V2-C.6)

| ID | Property | Test / gate |
| --- | --- | --- |
| P-C6-06 | Hot path uses inline state deny (`EXPIRED`/`REVOKED`), not `IngestAllowed` call | `filter_license.go` review; `TestProperty_P_C3_03_IngestAllowedStates` |
| P-C6-07 | Cold paths scatter inline bitmask checks | `deployment_gate.go`, `tier_policy.go`, controlplane |

`IngestAllowed` remains for P-C3-03 property tests only (`ingest_gate.go`).

## Garble literals policy (V2-D.1)

| ID | Property | Test / gate |
| --- | --- | --- |
| P-D1-01 | Default: tracker `literals=0`, control/processor `literals=1` | `bash scripts/ci/garble_literals_policy_gate.sh` |
| P-D1-02 | `GARBLE_LITERALS` overrides all binaries | `garble_literals_policy_gate.sh` |
| P-D1-03 | `internal/ingestion` package `//garble:ignore` for tracker literals eval | `internal/ingestion/garble_policy.go` |
| P-D1-04 | Tracker size delta with `-literals` (manual) | `make garble-literals-eval` |

Tracker p99 (+10% budget) requires `make load-test-bpf` on literals tracker — not automated in eval script.

## Garbled release license math (V2-D.5)

| ID | Property | Test / gate |
| --- | --- | --- |
| P-D5-01 | Garbled tracker/processor/control build | `RELEASE_GARBLE=1 bash scripts/ci/release_garble.sh` |
| P-D5-02 | Garbled binary strings gate | `bash scripts/ci/release_strings_gate.sh bin/garbled-release/tracker` |
| P-D5-03 | License math after garble build | `make license-red-team-garbled` |

Non-goals: unbreakable protection on root-owned hosts; hot-path per-request HWID or pubkey decode (startup/recheck only).

## Tier verification (M0–M3)

Consolidated gate: `make license-verify` (`scripts/ci/license_verify_tier.sh`).

| Tier | Scope | Automated in gate |
| --- | --- | --- |
| M0 | Baseline P-C2/C3/C4 + red-team | M0.2–M0.6 |
| M1 | Crypto rigor (HWID, rapid) | M1.2–M1.4; M1.1 HKDF vectors; M1.5 fuzz when `LICENSE_VERIFY_FUZZ=1` |
| M2 | MCK, seal, BPF, Lua, seed | M2.1–M2.5; M2.6 OpenSSL HKDF when `openssl kdf` available |
| M3 | Release / formal | M3.3 when `LICENSE_VERIFY_GARBLED=1`; TLC/Alloy manual |

Hot-path alloc subset (V2-C.D6): `make license-alloc-gate`. Full ingest alloc gate: `make test-alloc-gate`.
