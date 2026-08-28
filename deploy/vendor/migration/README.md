# External tracker migration maps

Static maps for `internal/migrationsource` when importing campaigns from Keitaro or Binom.

## Files

| File | Role |
| :--- | :--- |
| `keitaro_macros.yaml` | Keitaro URL token -> click query key |
| `keitaro_sources.yaml` | Keitaro traffic source label -> bundled `traffic_*` slug |
| `binom_macros.yaml` | Binom tokens |
| `binom_sources.yaml` | Binom source labels |

## Source kinds (honest contract)

File upload remains the default. Live HTTP pull is available for `keitaro_admin_api` and `binom_report_api` via `POST /api/v1/campaigns/migrate/pull/preview` and `POST /api/v1/campaigns/migrate/pull/import` (POST body only; token never logged or echoed).

| `source_kind` | Wire format | Required fields per campaign |
| :--- | :--- | :--- |
| `keitaro_json` | Normalized interchange (operator ETL or hand-built) | `name`, `tracking_url` |
| `keitaro_admin_api` | `GET /admin_api/v1/campaigns` array or `{"campaigns":[]}` | `name`, `alias`, `domain` or `tracker_domain` (or `tracking_url` override) |
| `binom_json` | Normalized interchange | `name`, `tracking_url` |
| `binom_report_api` | Binom campaign report rows array | `name`, `url`, `ts_name` recommended |
| `native_v1` | ad-event-processor export bundle | use `/api/v1/campaigns/import` |

### Interchange vs wire

- **Interchange** (`keitaro_json`, `binom_json`): flat campaign rows with `tracking_url`, `lander_url`, `postback_url`, `budget`. Use when an ETL script already flattened tracker data.
- **Wire** (`keitaro_admin_api`, `binom_report_api`): field names from vendor APIs. Admin API JSON passed to `keitaro_json` **must fail** (holdout: missing `tracking_url`).

### Fields not imported as budget

| Source field | Why ignored |
| :--- | :--- |
| Keitaro `cost_value` | CPC/CPA bid metadata, not campaign cap |
| Binom report `cost` | Spend for report window, not budget |

Only explicit `budget` on interchange or enrichment rows maps to `budget_limit_micro`.

### Keitaro Admin API enrichment

Core API returns `traffic_source_id`, not the label. Operators merge `traffic_source` from `GET /admin_api/v1/traffic_sources` in an export script, or preview shows `unknown_traffic_source`.

`parameters` is appended to `https://{domain}/{alias}`.

### Binom report gaps

Report rows have `url` but no `lander_url` or `postback_url`. Preview emits `lander_external_only`; operators set lander and re-enter postback secrets manually.

### Keitaro interchange streams

`keitaro_json` campaigns may include `streams[]` with `paths[]` (rotation weights, lander/offer refs). Import maps the first stream into the campaign flow snapshot; additional streams emit `multiple_streams_truncated`. Unsupported filter nodes in `unmapped_nodes` emit `stream_node_unmapped` warnings.

### Live pull

| `source_kind` | Default path | Auth |
| :--- | :--- | :--- |
| `keitaro_admin_api` | `GET /admin_api/v1/campaigns` | `Api-Key` header |
| `binom_report_api` | `GET /public/api/v1/campaign/list` | `api_key` query param |

Pull uses `ReadLimitedBody` (1 MiB cap) and a 30 s HTTP timeout. Pull failure returns an error before any import TX runs.

## v1 scope

- Landers remain external URLs; hosted ZIP must be re-uploaded manually after import.
- Postback secrets are never imported; operators re-enter tokens in CAPI and Postbacks.
- Streams import path weights and external lander/offer URLs; hosted lander ZIP and complex filter nodes are not imported.

Loader: `internal/migrationsource/maps.go` (`MapsRootDir` mirrors `integrationschema.SchemaRootDir`).

CI: `bash scripts/ci/migration_maps_gate.sh`.

Tests: `internal/migrationsource/wire_holdout_test.go` (wrong `source_kind` for wire JSON must fail).
