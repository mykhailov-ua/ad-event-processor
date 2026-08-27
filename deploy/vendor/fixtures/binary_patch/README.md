# Binary patch lab fixture

Manual pentest **PT-D04**, **PT-D07** (`deploy/vendor/licensing_security_backlog.md` slug `binary_patch_lab_procedure`): verify that a single CFG patch or `.text` byte flip does not yield a full crack when guard and seed coupling are on.

Automated CI proxies (no live binary patch):

```bash
bash scripts/lab/binary_patch_lab.sh
```

## Build guarded release binary

```bash
export GARBLE_SEED="$(openssl rand -hex 16)"
LICENSE_GUARD=1 bash scripts/ci/release_garble.sh /tmp/patch-lab tracker
cp /tmp/patch-lab/tracker /tmp/patch-lab/tracker.orig
```

## On-disk `.text` patch (guard on)

1. Place a valid `var/license.jwt` on the lab host.
2. Patch one byte in the executable segment (disk copy):

```bash
# Find .text file offset via readelf -S, or use hexedit on a copy
cp /tmp/patch-lab/tracker.orig /tmp/patch-lab/tracker.patched
hexedit /tmp/patch-lab/tracker.patched
```

3. Start tracker with guard enabled:

```bash
export AD_EVENT_PROCESSOR_LICENSE_GUARD=1
export AD_EVENT_PROCESSOR_LICENSE_REQUIRED=1
export AD_EVENT_PROCESSOR_LICENSE_PATH=/path/to/var/license.jwt
/tmp/patch-lab/tracker.patched
```

| Guard | On-disk patch | In-memory patch (gdb write, if attach allowed) |
| :--- | :--- | :--- |
| Off | May run until recheck | May run |
| On | May run until first text hash probe (3-8 s) | Trip `text_tamper` -> `LicenseEpochInvalid` -> ingest EXPIRED |

**Pass:** ingest blocks or epoch invalid within one guard probe interval after in-memory patch. On-disk patch alone may not trip until the process maps the modified segment and the probe runs.

## In-memory patch (PT-D07)

When `LICENSE_GUARD=0` or ptrace attach is allowed (`ptrace_scope=0`), use gdb on a running guarded binary:

```bash
gdb -p "$(pgrep -n tracker)" -batch \
  -ex 'info proc mappings' \
  -ex 'set {char}0x<exec_addr>=0x90'
```

With guard on and ptrace blocked, prefer the on-disk copy workflow above.

## Patch `LicenseFilter` only (PT-D04)

Goal: bypass the obvious state check without valid MCK / feature seed.

1. Disassemble garbled tracker or use differential testing to locate filter return path.
2. Patch branch so `LicenseFilter` always allows.

**Pass (defense holds):** ingest may accept events at the filter layer, but over-cap RPS and OpenRTB remain blocked when seed coupling is on (`LicenseRPSFilter_seedCoupling`, `OpenRTBLicenseAllowed_seedCoupling`). Processor settlement and sealed assets also fail closed without valid seed.

```bash
go test ./internal/ingestion/ -run 'LicenseRPSFilter_seedCoupling|OpenRTBLicenseAllowed_seedCoupling' -count=1
```

## Guard trip decoupled from verify (PT-E08)

Patching near `ed25519.Verify` must not disable text hash trips. Unit harness:

```bash
go test -tags=license_guard ./internal/licensing/ \
  -run 'TestGuard_TripWithoutVerifyCall|TestGuard_TextTamper' -count=1
```

## Related gates

```bash
bash scripts/lab/binary_patch_lab.sh
bash scripts/test/license_red_team_extended.sh
make license-red-team
```
