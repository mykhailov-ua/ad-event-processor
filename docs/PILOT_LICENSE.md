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
export ESPX_LICENSE_PRIVATE_KEY_FILE=deploy/vendor/license_private.key

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

## Customer — first install

In `deploy/installer/install.env`:

```bash
ESPX_LICENSE_MODE=file
ESPX_LICENSE_KEY=<paste full JWT line from vendor>
TELEMETRY_ENABLED=false
```

Then:

```bash
bash scripts/install/bidshard-install.sh --yes
```

Install writes `license.jwt`, sets `ESPX_LICENSE_MODE=file`, clears license server URL, disables product telemetry ping.

## Customer — monthly renewal (after payment)

**Option A — Admin UI:** Settings → License renewal → paste new JWT → Apply.

**Option B — CLI on the server:**

```bash
bash scripts/install/bidshard-install.sh license-apply '<JWT>'
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

Pilot SKU (`deploy/vendor/sku.yaml`): 35-day validity, 7-day grace, **hard bind** (host fingerprint).

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

## Vendor renewal checklist (support)

Before issuing a renewal JWT:

1. Customer ticket includes **same** `deployment_id` as prior invoice (Settings → License or support bundle `version.json`).
2. Compare `host_fingerprint` from support bundle with the fingerprint used at first install (`ESPX_DEPLOYMENT_FINGERPRINT` on server or install log).
3. Reject renewal when fingerprint or deployment_id changed without a signed migration note.
4. Issue JWT with matching `--deployment-id` and `--fingerprint`.

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
| No license server | `ESPX_LICENSE_MODE=file`, empty `ESPX_LICENSE_SERVER` |
| No product telemetry | `ESPX_TELEMETRY_OPT_IN=0` |
| Verify key in image/env | `deploy/vendor/license_public.key` or `ESPX_LICENSE_PUBLIC_KEY` |
| Status | `GET /api/v1/license/status` or Overview license banner |

## Support bundle

Ask customer for `deployment_id` from Settings or `GET /api/v1/meta` before issuing renewal JWT.

## Release images (SKU surface)

| Tag suffix | Binaries | Use case |
|------------|----------|----------|
| `v1.0.0-pilot` (default tag) | tracker, processor, control | Full single-VPS pilot |
| `v1.0.0-pilot-ingest` | tracker, processor | Ingest-only workers (no admin binary in image) |

Set `ESPX_APP_IMAGE=ghcr.io/<org>/bidshard:<tag>-ingest` for ingest-only deployments.

## EULA

Install requires `--accept-eula` (non-interactive) or interactive acceptance during `bidshard-install.sh up`.

First admin login shows a click-through if EULA was not recorded at bootstrap. Text: `pkg/legal/EULA.txt`, version `2026-01`.

## Garble literals (optional)

Evaluate before enabling `GARBLE_LITERALS=1` in release builds:

```bash
make garble-literals-eval
```

