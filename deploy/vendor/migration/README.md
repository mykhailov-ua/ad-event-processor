# External tracker migration maps

Static maps for `internal/migrationsource` when importing campaigns from Keitaro or Binom.

## Files

| File | Role |
| :--- | :--- |
| `keitaro_macros.yaml` | Keitaro URL token -> click query key |
| `keitaro_sources.yaml` | Keitaro traffic source label -> bundled `traffic_*` slug |
| `binom_macros.yaml` | Binom tokens (v2 adapter) |
| `binom_sources.yaml` | Binom source labels (v2 adapter) |

## v1 scope

- File upload JSON only (no live Keitaro/Binom API pull).
- Landers remain external URLs; hosted ZIP must be re-uploaded manually after import.
- Postback secrets are never imported; operators re-enter tokens in CAPI & Postbacks.

Loader: `internal/migrationsource/maps.go` (`MapsRootDir` mirrors `integrationschema.SchemaRootDir`).

CI: `bash scripts/ci/migration_maps_gate.sh`.
