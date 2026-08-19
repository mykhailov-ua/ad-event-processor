# Naming

**Public product name:** **ad-event-processor**.

**Engineering stack id:** **ad-event-processor** (Go module, compose services, internal docs).

Legacy tokens (`BidShard`, `espx`, `ESPX_`) are forbidden in new code and docs.

## Surfaces

| Surface | Name |
| :--- | :--- |
| `docs/INDEX.md`, `docs/START.md`, `docs/TRAFFIC.md`, `deploy/vendor/*` | `ad-event-processor` |
| `go.mod`, `internal/*`, `cmd/*`, `docs/DEVELOPMENT.md`, `docs/ARCHITECTURE.md` | `ad-event-processor` |
| Docker compose project / volumes | `ad_event_processor_*` (historical; not renamed in v1) |

Guard: `scripts/ci/check_no_legacy_naming.sh`. Admin UI product strings say **ad-event-processor**; API paths stay `/api/v1/*` on control `:8188`.

## Migration (upgrades)

After env renames (`REDIS_ADDRS`, `DB_DSN`, `AD_EVENT_PROCESSOR_LICENSE_*`): recreate compose stack and run `bash scripts/install/ad-event-processor-install.sh apply`. Details: [START.md](START.md).
