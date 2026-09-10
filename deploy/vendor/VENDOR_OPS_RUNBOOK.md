# Vendor ops runbook (Controlled Pilot)

Internal. Manual sales year 1: Telegram qualification, offer acceptance, pilot JWT. No calls for standard pilot/paid ([OUTREACH.md](./OUTREACH.md)).

## Prerequisites

| Item | Path / command |
| :--- | :--- |
| Offer version | `deploy/vendor/offer_meta.json` + `VENDOR_OFFER_VERSION` on vendor VPS |
| Offer contact overrides | Optional `deploy/vendor/offer_vendor.env` (email, Telegram, website) |
| Private signing key | `deploy/vendor/license_private.key` on vendor VPS only |
| Trial registry | `deploy/vendor/trial_registry.json` |
| License vendor API | `make deploy-license-vendor-api` |
| Trial bot | `make deploy-vendor-trial-bot` |

## Pilot flow

1. Buyer reads https://bidshard.com/offer.html and accepts via site gate (UX) or Telegram `/accept` (authoritative).
2. Buyer messages @bidshardsupportbot; sales qualifies (RPS, VPS, use case).
3. Telegram `/accept` records acceptance in `trial_registry.json` (`offer_acceptances[]`).
4. Buyer sends `/trial` or sales enqueues via API.
5. Buyer sends HWID (`go run ./cmd/installer license host-id` on VPS).
6. Operator issues pilot JWT:

```bash
go run ./cmd/license-issue --sku pilot \
  --customer "Buyer Name" \
  --telegram-id "<id>" \
  --hwid-v2 "<hwid>" \
  --deployment-id "<uuid>" \
  --approve-pending "<pending-id>"
```

Or vendor API:

```bash
curl -sf -H "Authorization: Bearer $LICENSE_VENDOR_API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"hwid_v2":"<hwid>"}' \
  "http://127.0.0.1:8199/api/v1/vendor/trial-requests/<id>/issue"
```

7. Deliver JWT securely (Telegram DM). Buyer: `install.sh license-apply '<jwt>'` or `/activate`.

**Gate:** pilot JWT is rejected without recorded offer acceptance for the telegram id (`ErrOfferNotAccepted`).


## Network and Enterprise (paid)

Full qualification, hardware audit, topology review, onboarding phases, and customer SLA text:

- [NETWORK_ENTERPRISE_RUNBOOK.md](./NETWORK_ENTERPRISE_RUNBOOK.md) (Appendix A = Network SLA, Appendix B = Enterprise SLA)
- Technical deploy: [ENTERPRISE_DEPLOY.md](./ENTERPRISE_DEPLOY.md)

Summary:

1. Offer acceptance + discovery intake (load, hardware audit, topology) **before** invoice.
2. USDT payment per [INVOICE.md](./INVOICE.md).
3. Issue `--sku network` or `--sku enterprise`; attach SLA appendix in Telegram.
4. Network: multi-region cutover within 14 days target; Enterprise: add XDP + optional platform API phases.

## Paid flow (manual)

1. Same offer acceptance on file.
2. Send `deploy/vendor/INVOICE.md` filled (SKU, USDT, `deployment_id`).
3. Confirm USDT tx hash.
4. Issue paid JWT with same `deployment_id`:

```bash
go run ./cmd/license-issue --sku pro \
  --customer "Buyer" \
  --deployment-id "<uuid>" \
  --hwid-v2 "<hwid>" \
  --mark-converted
```

## Renewal

Re-issue JWT with same `deployment_id` before `exp` + grace. No reinstall.

## Revoke

```bash
go run ./cmd/license-issue --revoke --deployment-id "<uuid>" ...
```

## CLI helpers

```bash
go run ./cmd/trial-registry list-pending
go run ./cmd/trial-registry accept-offer --telegram-id 123456789
go run ./cmd/trial-registry reject-pending --id <uuid> --reason "spam"
```

## Offer version bump

1. Edit `deploy/vendor/offer_meta.json` and `PUBLIC_OFFER.md` if needed.
2. Set `VENDOR_OFFER_VERSION` on vendor VPS (bot + license-vendor-api).
3. `bash scripts/ops/deploy_marketing.sh` (sources `offer_vendor.env`).
4. `bash scripts/ci/static/offer_version_gate.sh`

## Evidence per lead

| Field | Source |
| :--- | :--- |
| offer_version | registry `offer_acceptances` |
| accepted_at | registry |
| source | `telegram` / `vendor_api` / `web` (web = UX only until bot deeplink) |
| deployment_id | pending or paid JWT |
| hwid | issue request |
| jwt exp | issue output |

See also: `OFFER_IMPLEMENTATION_GUIDE.md`, `KEYS.md`, `INVOICE.md`, `SALES.md`.
