# BidShard Quick Start (single VPS)

Self-hosted appliance on one Linux server. No Go required on the host.

## Requirements

- Ubuntu 22.04+ (or similar Linux), 8 GB RAM recommended
- Kernel 5.15+ is fine for the appliance; edge XDP needs 6.1+ with BTF (optional)
- Root or sudo for first-time package install (Docker)

## One-liner (release tarball)

```bash
curl -fsSL https://raw.githubusercontent.com/bidshard/bidshard/main/scripts/install/get.sh | bash
```

## Install from git

```bash
git clone <repo-url> bidshard && cd bidshard
cp deploy/installer/install.env.example deploy/installer/install.env
# edit admin email/password, TRACKING_DOMAIN, optional INGRESS_ENABLED=1
bash scripts/install/bidshard-install.sh --yes
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
ESPX_USE_RELEASE_IMAGES=1
ESPX_APP_IMAGE=ghcr.io/<owner>/bidshard:v1.0.0
```

## Commands

```bash
bash scripts/install/bidshard-install.sh status
bash scripts/install/bidshard-install.sh apply
bash scripts/install/bidshard-install.sh doctor
make release-installer VERSION=v1.0.0
```

## Advanced

```bash
make build-bin          # bin/tracker, bin/control, bin/espx-install, …
bin/espx-install preflight
```

**Pilot on-prem license (offline JWT, monthly renewal):** [docs/PILOT_LICENSE.md](./PILOT_LICENSE.md)

See `deploy/installer/install.yaml.example` and `docs/DEVELOPMENT.md` for other compose profiles.
