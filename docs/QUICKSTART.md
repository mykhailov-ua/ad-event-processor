# BidShard Quick Start (single VPS)

For engineers and operators deploying a self-hosted appliance profile.

## Requirements

- Ubuntu 22.04+ (or similar Linux)
- Docker Engine + Compose plugin
- 8 GB RAM recommended
- Open ports: tracker, control (`MANAGEMENT_PORT`, default 8188), Redis/Postgres internal

## Install (interactive)

From the repository root:

```bash
bash scripts/install/bidshard-install.sh
```

The script will:

1. Prompt for tracking domain, currency, timezone, Stripe (optional), admin credentials
2. Start the `single_vps` Docker stack
3. Bootstrap platform config via `POST /api/v1/settings/platform/bootstrap`
4. Print the control URL and click redirect template

## After install

- **Control UI / API:** `http://<host>:8188`
- **Platform settings (same fields as install):** `GET/PATCH /api/v1/settings/platform`
- **System health (Doctor):** `GET /api/v1/ops/doctor`
- **Click redirect:** `https://<tracking_domain>/click?campaign_id=...&sub1=...`

## Apply config changes

When settings require a stack restart (ingress schema, Stripe keys, telemetry):

```bash
bash scripts/install/bidshard-install.sh apply
```

Or from the UI: `POST /api/v1/settings/platform/apply` (writes `install.compose.env`), then restart containers.

## Alternative: Go installer

```bash
go run ./cmd/installer configure --interactive
go run ./cmd/installer up
go run ./cmd/installer bootstrap
go run ./cmd/installer apply
```

See `deploy/installer/install.yaml.example` and `docs/DEVELOPMENT.md` for advanced profiles.
