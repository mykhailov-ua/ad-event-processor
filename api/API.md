# api

Wire contracts and codegen inputs. **Admin REST is OpenAPI-first** — do not land handler-only routes without spec update.

Cross-ref: [docs/DEVELOPMENT.md](DEVELOPMENT.md), [internal/INTERNAL.md](../internal/INTERNAL.md).

---

## Layout

```
api/
  openapi/           # Admin REST — paths/, components/schemas/, bundle
  events.proto       # Event protobuf (ingest/processor)
  outbox.proto       # Outbox event protobuf
  vast.proto         # VAST messages
  buf.yaml           # Buf module config
  buf.gen*.yaml      # Codegen plugins (vtproto, etc.)
  gen/               # Generated output (after make proto)
```

---

## OpenAPI (`api/openapi/`)

| File | Role |
| :--- | :--- |
| `openapi.yaml` | Root document — `$ref` paths and schemas |
| `paths/*.yaml` | Per-domain operations |
| `components/schemas/*.yaml` | DTO shapes |
| `paths/_generated_routes.yaml` | Stubs from `routeCatalog` until documented |
| `openapi.bundle.yaml` | Single-file bundle for Spectral/oasdiff |

### Workflow for new `/api/v1` routes

1. Add `paths/<domain>.yaml` + `components/schemas/<domain>.yaml`.
2. Register in `internal/openapi/documented_routes.go`; `$ref` from `openapi.yaml`.
3. Run `make openapi-export` and `make openapi-types` (when `web/` exists).
4. Implement Go handler DTO to match schema — field names must align.
5. Add `internal/controlplane/openapi_<domain>_test.go` parity test.
6. Gate: `bash scripts/ci/admin/openapi.sh`.

**Breaking changes:** `bash scripts/ci/admin/openapi_breaking.sh` (oasdiff on bundle). Document in PR and release notes.

**Optional runtime validation:** `OPENAPI_REQUEST_VALIDATION=1` on control — kin-openapi middleware for selected routes.

### Permissions

Document on operation: `x-permissions: [campaigns:write]`. Handler must enforce same strings.

### Unstable routes

Mark `x-unstable: true` and add ignore line to `api/openapi/breaking_err_ignore.txt` until stable.

---

## Protobuf (`*.proto`)

| Proto | Consumers |
| :--- | :--- |
| `events.proto` | Tracker ingest, processor batching |
| `outbox.proto` | Outbox worker payloads |
| `vast.proto` | VAST-related surfaces |

```bash
make proto    # buf generate + patch-vtproto-hotpath
```

**Hot path rule:** vtproto patched for `appendReuseBytes` — zero-alloc parse on `/track`. Do not hand-edit `*.pb.go`.

**Test:** `make test-alloc-gate`, parser fuzz nightly on ingest FSM targets.

---

## Codegen commands

| Command | Output |
| :--- | :--- |
| `make openapi-export` | Bundle + `_generated_routes.yaml` |
| `make openapi-types` | `web/src/types/generated/openapi.d.ts` |
| `make proto` | `api/gen/`, internal `pb/` |
| `make gen` | sqlc (not under `api/` but often paired) |

---

## Contract ↔ code mapping

| Layer | Location |
| :--- | :--- |
| OpenAPI schema | `api/openapi/components/schemas/` |
| Go JSON DTO | `internal/<domain>/handlers.go` or `controlplane/*_handlers.go` |
| Documented route list | `internal/openapi/documented_routes.go` |
| Parity tests | `internal/controlplane/openapi_*_test.go` |
| TS types (when UI returns) | `web/src/types/generated/` |

**Rule:** PATCH body fields must exist on Go struct before UI field ships.

---

## Ingress wire (not OpenAPI)

Tracker hot path wire formats are **not** OpenAPI:

| Endpoint | Format |
| :--- | :--- |
| `/track` | JSON (DFA parser), protobuf, OpenRTB 3 FSM |
| `/click` | Query string macros |
| `/openrtb/bid` | OpenRTB 2.x JSON |

Document parser limits in `.cursor/rules/parser.mdc`. Chaos test: `TestChaos_CrossHop_NginxGnet`.

---

## Limits

| Limit | Detail |
| :--- | :--- |
| Admin body size | 64 KiB via `pkg/coldpath` |
| OpenAPI stub routes | Generated until domain documented — do not treat stubs as shipped API |
| `web/` absent | TS typegen skipped; Go/OpenAPI gate still required |

---

## Verification

```bash
bash scripts/ci/admin/openapi.sh
go test ./internal/openapi/ -count=1
go test ./internal/controlplane/ -run TestOpenAPI_ -count=1
```

---

## Common mistakes

1. **Handler-only route** — fails review; spec first.
2. **json tag drift** — parity test must catch snake_case mismatches.
3. **Editing bundle by hand** — regenerate via `make openapi-export`.
4. **Using `encoding/json` on hot ingest** — use DFA/vtproto paths in `internal/ingest/`.
