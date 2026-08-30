// Package coldpath provides shared cold HTTP, JSON, UUID, and Postgres migration helpers.
//
// Role:
//   - ReadLimitedBody, DecodeRequestOrBadRequest, DecodeBody for admin/payment/webhook handlers.
//   - WritePaginatedJSON sets X-Total-Count for list endpoints; Paginate/PaginatedList for cursor/offset lists.
//   - ApplyTrackedSchemaMigrations: goose-style SQL dirs with public.tracked_migrations ledger.
//   - MarshalOutbox/UnmarshalOutbox with optional proto magic prefix; PG error helpers for idempotency branches.
//   - ParseDryRun reads X-Dry-Run header or dry_run query for admin mutation previews.
//
// Topology:
//   - Imported by internal/controlplane, internal/payment, cmd/migrate-cold-path.
//   - Uses pkg/httpresponse for JSON error envelopes; stdlib encoding/json only.
//
// Defaults and limits:
//   - DefaultMaxBody 65536 (64 KiB).
//   - SelfServePaymentIntentMaxBody 16 KiB.
//   - PaymentWebhookMaxBody 64 KiB.
//   - AlertmanagerWebhookMaxBody 1 MiB.
//   - RegionIngestMaxBody 4 MiB (multi-region batch ingress).
//
// Invariants:
//   - ReadLimitedBody wraps r.Body with http.MaxBytesReader; oversize read fails before buffering full body.
//   - DecodeRequestOrBadRequest maps read/decode failures to 400 BAD_REQUEST without leaking parse details.
//   - ApplyTrackedSchemaMigrations applies .sql files in lexicographic order; records filename in tracked_migrations after success.
//   - Goose Down sections and StatementBegin/End markers stripped; only Up SQL executes.
//   - migrationAlreadyApplied PG codes (42P06, 42P07, 42701, 42710, 42723) record filename without re-running DDL.
//   - OutboxProtoMagic 0x1f prefixes registered codec payloads; unregistered types marshal as plain JSON.
//   - IsUniqueViolation matches PG SQLSTATE 23505 only.
//
// Tradeoffs:
//   - encoding/json on cold path vs hot-path DFA/vtproto stack (admin latency acceptable; shared handler helpers).
//   - Embedded tracked_migrations runner vs external goose CLI (sidecar schemas self-apply on pool open).
//   - Idempotent migration skip on already-exists errors vs hard fail (safe reruns on partially applied appliances).
//   - Proto magic-byte outbox lane vs JSON-only (compact registered payloads; legacy rows stay JSON-decodable).
//
// Forbidden:
//   - pkg/* must not import internal/*.
//   - Not for hot-path /track parse (use ingest DFA/vtproto stack).
//
// Verify:
//
//	go test ./pkg/coldpath/... -short -count=1
//	go test ./pkg/coldpath/ -short -run TestReadLimitedBody -count=1
//	bash scripts/ci/static/cold_path_json.sh
package coldpath
