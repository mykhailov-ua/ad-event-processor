# Start

Self-hosted appliance on one Linux VPS. No Go on the host.

## Requirements

- Ubuntu 22.04+ (or similar), **4 vCPU / 8 GB RAM** minimum (Hetzner CX31 / Contabo VPS S)
- Kernel 5.15+ for appliance; edge XDP needs 6.1+ with BTF (optional)
- Root or sudo for first-time Docker install

## Gray on-prem (pilot / USDT license)

1. Provision clean Ubuntu 22.04 VM; open `443` + admin port as needed.
2. Install (one-liner or git path); set `TRACKING_DOMAIN` / admin password in `deploy/installer/install.env`.
3. Paste vendor JWT (Settings → License, or `license-apply`) — [LICENSE.md](./LICENSE.md). Entitlements reload without restart.
4. Keep telemetry off (`AD_EVENT_PROCESSOR_TELEMETRY_OPT_IN=0`, `TELEMETRY_ENABLED=false`).
5. Bootstrap admin → first campaign → edge **Expose click URL** → smoke `GET /click` — [TRAFFIC.md](./TRAFFIC.md).
6. Optional: **Settings → Domains** → verify host → **Setup SSL** after DNS.

Goal: install → first campaign click in **< 45 min**. Renewal is JWT-only (see [License tier upgrade](#license-tier-upgrade)).

Sales / billing: [BILLING.md](./BILLING.md).

## Install

**One-liner (release tarball):**

```bash
curl -fsSL https://raw.githubusercontent.com/ad-event-processor/ad-event-processor/main/scripts/install/get.sh | bash
```

**From git:**

```bash
git clone <repo-url> ad-event-processor && cd ad-event-processor
cp deploy/installer/install.env.example deploy/installer/install.env
# edit admin email/password, TRACKING_DOMAIN, optional INGRESS_ENABLED=1
bash scripts/install/ad-event-processor-install.sh --yes
```

Post-bootstrap checklist: `http://<host>:8188/install/done`.

## Automatic TLS

In `deploy/installer/install.env`:

```bash
INGRESS_ENABLED=1
TRACKING_DOMAIN=trk.example.com
ADMIN_DOMAIN=admin.example.com
CADDY_ACME_EMAIL=ops@example.com
```

Re-run install or `bash scripts/dev/stack.sh single-vps`. Caddy proxies `trk` → `:8181`, `admin` → `:8188`. Manual certs: `INGRESS_TLS_MODE=manual` + PEMs in `deploy/ingress/certs/`.

## Release images

```bash
AD_EVENT_PROCESSOR_USE_RELEASE_IMAGES=1
AD_EVENT_PROCESSOR_APP_IMAGE=ghcr.io/<owner>/ad-event-processor:v1.0.0
```

## Commands

```bash
bash scripts/install/ad-event-processor-install.sh status|apply|doctor
make release-installer VERSION=v1.0.0
bash scripts/ops/verify_redis_topology.sh .env   # 4-shard appliance default
```

Six-shard dev layout: `bash scripts/dev/stack.sh infra` (not `single-vps`).

## License tier upgrade

Pilot → paid or tier bump = JWT swap only — same `deployment_id`, no stack rebuild.

```bash
# Admin UI: Settings → License → paste JWT → Apply
bash scripts/install/ad-event-processor-install.sh license-apply '<JWT>'
```

After USDT confirm, vendor issues JWT for target SKU (`starter` … `enterprise` in `deploy/vendor/sku.yaml`). Higher tiers raise `max_rps` and `max_activations`.

## Upgrade existing install

Env renames: [NAMING.md §4](NAMING.md#4-migration-upgrades). After updating `.env`:

```bash
bash scripts/install/ad-event-processor-install.sh apply
bash scripts/dev/stack.sh single-vps
```

See `deploy/installer/install.yaml.example` and [DEVELOPMENT.md](./DEVELOPMENT.md).
