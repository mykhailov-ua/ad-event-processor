# Tester install (bare VPS)

Linux amd64, Docker, sudo. Ports: 8188 (admin), 8181 (track).

## 0. Get the installer

### Option A: bidshard.com releases (pilot channel)

```bash
curl -fsSL -o installer.tar.gz https://bidshard.com/releases/ad-event-processor-installer.tar.gz
tar -xzf installer.tar.gz
cd ad-event-processor
```

Pilot tarballs include `bin/control`, `bin/tracker`, `bin/processor`, and `bin/broker` for `systemd up`.

### Option B: Marketing bootstrap

```bash
curl -fsSL https://bidshard.com/get.sh | bash
```

`get.sh` downloads from `https://bidshard.com/releases/` first, then GitHub Releases, then git clone.

### Option C: GitHub Release (after first `v*` tag)

```bash
VERSION=v0.1.0
curl -fsSL -o installer.tar.gz \
  "https://github.com/mykhailov-ua/ad-event-processor/releases/download/${VERSION}/ad-event-processor-installer.tar.gz"
tar -xzf installer.tar.gz
cd ad-event-processor
```

Tagged releases ship garbled binaries extracted from the pilot GHCR image in CI.

### Option D: Docker compose + GHCR (no host binaries)

For docker install mode, set in `deploy/installer/install.env` before `install.sh docker up`:

```bash
AD_EVENT_PROCESSOR_USE_RELEASE_IMAGES=1
AD_EVENT_PROCESSOR_APP_IMAGE=ghcr.io/mykhailov-ua/ad-event-processor:<tag>
```

Image tags match GitHub Release tags (for example `v0.1.0`). Pilot image contains `control`, `tracker`, `processor`.

## 1. Install

```bash
sudo bash install.sh systemd up
```

No `install.env` edits required for a first run on a clean VPS.

`systemd up` requires `bin/control`, `bin/tracker`, `bin/processor`, and `bin/broker`. Use a GitHub Release tarball (Option A), not a bare git clone.

## 2. Activate in browser

Open the URL printed at the end (usually `http://<server-ip>:8188/activate`).

Paste license JWT, email, password, team name. Done.

Or apply JWT from shell (control must be up):

```bash
sudo bash install.sh license-apply '<paste-jwt-here>'
```

This calls `POST /api/v1/license/apply`, writes `license.jwt`, and bootstraps sidecar files (`.mac`, `.clock` when enterprise/file mode).

## 3. License security checklist (vendor, before first customer)

| Step | Command / action |
| :--- | :--- |
| Private signing key | Keep `deploy/vendor/license_private.key` off customer VPS and out of git |
| Host HWID | On customer VPS: `./bin/ad-event-processor-install license host-id` (or admin activate page) |
| Offer acceptance | Pilot buyer sends `/accept` then `/trial` in Telegram; or vendor records `trial-registry accept-offer` |
| Issue pilot JWT | `go run ./cmd/license-issue --sku pilot --customer "..." --telegram-id "<id>" --deployment-id "<uuid>" --hwid-v2 "<hwid>" --out license.jwt` |
| Trial registry | Set `VENDOR_TRIAL_REGISTRY=deploy/vendor/trial_registry.json` when issuing pilots |
| Production profile | Customer `.env`: `AD_EVENT_PROCESSOR_PROFILE=production`, `AD_EVENT_PROCESSOR_LICENSE_MODE=file`, `AD_EVENT_PROCESSOR_LICENSE_REQUIRED=1` |

Pilot JWT issuance fails without offer acceptance for the current offer version (`2026-09-09` unless bumped in `deploy/vendor/offer_meta.json`).

Do not ship production with `AD_EVENT_PROCESSOR_LICENSE_GUARD=0` or `AD_EVENT_PROCESSOR_LICENSE_SKEW_WATCH=0` — in production profile these kill switches are ignored anyway.

## 4. License security checklist (on VPS after apply)

| Check | Expected |
| :--- | :--- |
| `GET /api/v1/license/status` | `state` is `ACTIVE` or `GRACE` |
| License files | `/etc/ad-event-processor/license.jwt` (+ `.mac` and `.clock` in file/enterprise mode) |
| Clock tamper | Do not roll system time backward; metric `license_clock_anchor_total` should stay 0 |
| Renew | New JWT with same `deployment_id`; no reinstall required |

Verify:

```bash
curl -sf -H "X-Admin-API-Key: $ADMIN_API_KEY" http://127.0.0.1:8188/api/v1/license/status | jq .
ls -la /etc/ad-event-processor/license.jwt*
```

## Optional

Domains and TLS: set `ADMIN_DOMAIN`, `TRACKING_DOMAIN`, `INGRESS_ENABLED=1`, `CADDY_ACME_EMAIL` in `deploy/installer/install.env`, then re-run `sudo bash install.sh systemd up`.

Broken or partial Postgres from an older attempt:

```bash
sudo bash install.sh bootstrap-schema
sudo systemctl restart ad-event-processor-control ad-event-processor-tracker ad-event-processor-processor
```

Status: `bash install.sh status` / `bash install.sh doctor`

Red-team (vendor machine): `make license-red-team` / `make license-verify`
