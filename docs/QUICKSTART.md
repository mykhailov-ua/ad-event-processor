# BidShard Quick Start (single VPS)

Self-hosted appliance on one Linux server. No Go required on the host.

## Requirements

- Ubuntu 22.04+ (or similar Linux), **4 vCPU / 8 GB RAM** minimum for gray-market Solo/Starter appliance (Hetzner CX31 / Contabo VPS S or similar)
- Kernel 5.15+ is fine for the appliance; edge XDP needs 6.1+ with BTF (optional)
- Root or sudo for first-time package install (Docker)

## Gray on-prem (pilot / USDT license)

Target path for self-hosted media-buying teams (offline JWT, no SaaS callback):

1. Provision a clean Ubuntu 22.04 VM (Hetzner / Contabo / equivalent), open `443` + admin port as needed.
2. Install with the one-liner or git path below; set `TRACKING_DOMAIN` / admin password in `deploy/installer/install.env`.
3. Paste the vendor-issued JWT (Settings → License, or `license-apply`) — see [PILOT_LICENSE.md](./PILOT_LICENSE.md). Entitlements reload without restart.
4. Keep product telemetry off (`AD_EVENT_PROCESSOR_TELEMETRY_OPT_IN=0`, `TELEMETRY_ENABLED=false` / `VENDOR_TELEMETRY_ENABLED=false`) so the appliance stays closed-contour.
5. Configure Telegram support channel for ops alerts via notifier settings (vendor channel ID is on the invoice / sales kit).
6. Bootstrap admin → create first campaign → enable edge **Expose click URL** → smoke `GET /click` (see [TRAFFIC_INTEGRATION.md](./TRAFFIC_INTEGRATION.md)).
7. Optional: **Settings → Domains** — verify tracking host health and run **Setup SSL** after DNS propagates.

Fresh VM goal: install → first campaign click through edge in **&lt; 45 min**. License renewal is JWT-only ([License tier upgrade](#license-tier-upgrade-no-reinstall)).

Vendor sales / USDT invoice: [SALES_KIT.md](../deploy/vendor/SALES_KIT.md), [USDT_INVOICE_TEMPLATE.md](../deploy/vendor/USDT_INVOICE_TEMPLATE.md), [sku.yaml](../deploy/vendor/sku.yaml).

## One-liner (release tarball)

```bash
curl -fsSL https://raw.githubusercontent.com/bidshard/bidshard/main/scripts/install/get.sh | bash
```

## Install from git

```bash
git clone <repo-url> bidshard && cd bidshard
cp deploy/installer/install.env.example deploy/installer/install.env
# edit admin email/password, TRACKING_DOMAIN, optional INGRESS_ENABLED=1
bash scripts/install/ad-event-processor-install.sh --yes   # runs host preflight first
```

After bootstrap via UI, open the checklist at `http://<host>:8188/install/done`.

## Automatic TLS (ingress)

Set in `deploy/installer/install.env`:

```bash
INGRESS_ENABLED=1
TRACKING_DOMAIN=trk.example.com
ADMIN_DOMAIN=admin.example.com
CADDY_ACME_EMAIL=ops@example.com
```

Re-run install or `bash scripts/dev/stack.sh single-vps`. Caddy obtains certificates and proxies:

- `https://trk.example.com` → tracker (`8181`)
- `https://admin.example.com` → control (`8188`)

Manual certs: `INGRESS_TLS_MODE=manual` and place `fullchain.pem` / `privkey.pem` in `deploy/ingress/certs/`.

## Release images (no local build)

```bash
AD_EVENT_PROCESSOR_USE_RELEASE_IMAGES=1
AD_EVENT_PROCESSOR_APP_IMAGE=ghcr.io/<owner>/ad-event-processor:v1.0.0
```

Legacy release-image environment variables from prior installs are still read for one release.

## Commands

```bash
bash scripts/install/ad-event-processor-install.sh status
bash scripts/install/ad-event-processor-install.sh apply
bash scripts/install/ad-event-processor-install.sh doctor
make release-installer VERSION=v1.0.0
```

Appliance Redis topology is **4 shards** (`REDIS_ADDRS` in `.env` / `install.compose.env`). After install or env changes, verify:

```bash
bash scripts/ops/verify_redis_topology.sh .env
# optional override: REDIS_SHARD_COUNT=4 bash scripts/ops/verify_redis_topology.sh .env
```

Six-shard dev layout (`redis-4`, `redis-5`) requires `bash scripts/dev/stack.sh infra` (compose profile `infra`), not `single-vps`.

## Advanced

```bash
make build-bin          # bin/tracker, bin/control, bin/ad-event-processor-install, …
bin/ad-event-processor-install preflight
```

**Pilot on-prem license (offline JWT, monthly renewal):** [docs/PILOT_LICENSE.md](./PILOT_LICENSE.md)

Commercial tiers and USDT sales flow (vendor): [deploy/vendor/SALES_KIT.md](../deploy/vendor/SALES_KIT.md).

## License tier upgrade (no reinstall)

Moving from **pilot → paid** or **Starter → Pro** (etc.) is a JWT swap only — same `deployment_id`, no stack rebuild.

1. Vendor issues invoice ([USDT template](../deploy/vendor/USDT_INVOICE_TEMPLATE.md)); after USDT confirm, you receive a new JWT for the target SKU (`solo` … `enterprise` in [sku.yaml](../deploy/vendor/sku.yaml)).
2. Apply the JWT (entitlements reload immediately; no container restart):

```bash
# Admin UI: Settings → License → paste JWT → Apply

bash scripts/install/ad-event-processor-install.sh license-apply '<JWT>'

curl -sS -X POST http://127.0.0.1:8188/api/v1/license/apply \
  -H "Content-Type: application/json" \
  -H "X-Admin-API-Key: $ADMIN_API_KEY" \
  -d '{"token":"<JWT>"}'
```

3. Open **Settings** — admin banner shows grace, renewal window (≤7 days), or campaign-cap warnings when approaching tier limits.

Higher tiers raise `max_rps`, `max_requests_per_day`, and `max_active_campaigns` enforced on the hot path and at campaign create.

## Upgrading an existing install

Runtime path and license env renames are documented in [NAMING.md §4](NAMING.md#4-migration-upgrades). After updating `.env` (`REDIS_ADDRS`, `DB_DSN`, `AD_EVENT_PROCESSOR_LICENSE_*`), recreate the compose stack and run:

```bash
bash scripts/install/ad-event-processor-install.sh apply
bash scripts/dev/stack.sh single-vps
```

See `deploy/installer/install.yaml.example` and `docs/DEVELOPMENT.md` for other compose profiles.

**Direct tracker access (`:8181`):** In development you can bypass nginx and post to gnet directly. `POST /track` still requires `Content-Length` and rejects chunked bodies — the same rules nginx enforces in production. See [PARSER_SECURITY.md](PARSER_SECURITY.md).
