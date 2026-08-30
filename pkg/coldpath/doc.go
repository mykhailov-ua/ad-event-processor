// Package coldpath provides shared cold HTTP, JSON, UUID, and Postgres migration helpers.
//
// Role:
//   - ReadLimitedBody, DecodeRequestOrBadRequest, DecodeBody for admin/payment/webhook handlers.
//   - WritePaginatedJSON sets X-Total-Count for list endpoints.
//   - ApplyTrackedSchemaMigrations: goose-style SQL dirs with public.tracked_migrations ledger.
//   - PG error helpers (IsUniqueViolation, etc.) for handler idempotency branches.
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
// Forbidden:
//   - pkg/* must not import internal/*.
//   - Not for hot-path /track parse (use ingest DFA/vtproto stack).
//
// Verify:
// go test ./pkg/coldpath/... -short -count=1
// bash scripts/ci/static/cold_path_json.sh
package coldpath
