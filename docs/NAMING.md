# Naming: BidShard vs ad-event-processor

**Public product:** **BidShard** (README, sales kit, buyer-facing guides).

**Engineering stack id:** **ad-event-processor** (Go module path, compose service names, internal docs). Do not use legacy **espx** in new code or customer docs.

## Where each name appears

| Surface | Name |
| :--- | :--- |
| `README.md`, `docs/QUICKSTART.md`, `docs/TRAFFIC.md`, `deploy/vendor/*` | BidShard |
| `go.mod`, `internal/*`, `cmd/*`, `docs/DEVELOPMENT.md`, `docs/ARCHITECTURE.md` | ad-event-processor |
| Docker compose project / volumes | `ad_event_processor_*` (historical prefix; not renamed in v1 appliance) |

## Prefix allowlist (migration)

Paths still containing `espx` in env var names or scripts are listed in `scripts/ci/check_no_espx.sh`. New symbols must not add `espx` or `ESPX_` except documented allowlist entries.

## Admin UI

Embedded SPA: product strings say **BidShard**; API paths remain `/api/v1/*` on control `:8188`.
