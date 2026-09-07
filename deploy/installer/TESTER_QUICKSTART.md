# Tester install (bare VPS)

Linux amd64, Docker, sudo. Ports: 8188 (admin), 8181 (track).

## 1. Install

```bash
tar -xzf ad-event-processor-installer-*.tar.gz
cd ad-event-processor
sudo bash install.sh systemd up
```

No `install.env` edits required for a first run on a clean VPS.

## 2. Activate in browser

Open the URL printed at the end (usually `http://<server-ip>:8188/activate`).

Paste license JWT, email, password, team name. Done.

## Optional

Domains and TLS: set `ADMIN_DOMAIN`, `TRACKING_DOMAIN`, `INGRESS_ENABLED=1`, `CADDY_ACME_EMAIL` in `deploy/installer/install.env`, then re-run `sudo bash install.sh systemd up`.

Broken or partial Postgres from an older attempt:

```bash
sudo bash install.sh bootstrap-schema
sudo systemctl restart ad-event-processor-control ad-event-processor-tracker ad-event-processor-processor
```

Status: `bash install.sh status` / `bash install.sh doctor`
