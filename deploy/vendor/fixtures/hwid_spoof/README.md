# HWID spoof lab fixture

Manual pentest: verify that a license bound to donor hardware does not activate when telemetry is spoofed on another host.

## Telemetry inputs (HWID v2)

Argon2id hash covers these fields (`internal/licensing/hwid.go`):

| Field | Typical source |
| :--- | :--- |
| DMI UUID | `/sys/class/dmi/id/product_uuid` |
| Root disk id | mountinfo / block serial under `/sys/block/` |
| MAC | first non-loopback, non-docker NIC under `/sys/class/net/` |
| CPU model | `/proc/cpuinfo` `model name` |
| CPU cores | `runtime.NumCPU()` |

When `AD_EVENT_PROCESSOR_LICENSE_HWID_V3=1`, Argon2 input also includes systemd `machine-id` (`/etc/machine-id` or dbus path). Re-issue donor JWT after enabling v3.

`GET /api/v1/license/status` field `hwid_inputs` lists the same telemetry (machine id redacted when v3 is off).

Legacy `HostFingerprint()` (separate from v2) also uses `/etc/machine-id` and install paths.

## QEMU / libvirt recipe

1. Issue a pilot JWT on **donor** with hard bind:

```bash
# On donor: read hwid_v2 from admin (control must be running with valid session)
# curl -sS http://127.0.0.1:8188/api/v1/license/status -H "Authorization: Bearer $TOKEN" | jq -r .hwid_v2

export AD_EVENT_PROCESSOR_LICENSE_PRIVATE_KEY_FILE=deploy/vendor/license_private.key
go run ./cmd/license-issue \
  --sku pilot --customer "Donor" \
  --deployment-id "<uuid>" \
  --hwid-v2 "<donor_hwid_hex_from_status_api>" \
  --out /tmp/donor.jwt
```

2. On **target** VM, bind-mount fake sysfs before starting tracker:

```bash
mkdir -p /tmp/hwid_spoof/dmi /tmp/hwid_spoof/net/eth0
echo "donor-dmi-uuid" > /tmp/hwid_spoof/dmi/product_uuid
echo "de:ad:be:ef:00:01" > /tmp/hwid_spoof/net/eth0/address

# Example docker run (adjust image and paths):
docker run --rm -it \
  -v /tmp/hwid_spoof/dmi/product_uuid:/sys/class/dmi/id/product_uuid:ro \
  -v /tmp/hwid_spoof/net/eth0/address:/sys/class/net/eth0/address:ro \
  -v /tmp/donor.jwt:/app/var/license.jwt:ro \
  your-tracker-image
```

3. **Pass (defense holds):** ingest blocked or license recheck sets `EXPIRED` when any telemetry field differs from donor.

4. **Fail (residual risk):** donor JWT works on target with only partial spoof (identify missing field from `hwid_inputs` on status API and extend bind-mount coverage).

## Collect hash for issue CLI

```bash
bash scripts/lab/hwid_collect.sh
```

Or from admin: `GET /api/v1/license/status` field `hwid_v2` (`licensing.mdc`).

Unit test determinism only (fixed telemetry, not live host):

```bash
go test ./internal/licensing/ -run TestHWID_Deterministic -count=1
```

## Related gates

```bash
bash scripts/ci/license/hwid_strings.sh
go test ./tests/integration/ -run LicenseProtection_hwid -count=1
```
