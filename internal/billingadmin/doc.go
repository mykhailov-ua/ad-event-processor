// Package billingadmin serves operator billing HTTP, crypto billing webhooks, cost-sync
// admin, workspace usage export, and billing maintenance workers for cmd/control.
//
// Role:
//   - HTTPHandlers (handlers.go): /api/v1/billing/invoices*, /api/v1/billing/invariant,
//     /api/v1/billing/summary, /api/v1/billing/forecast, tax profile, void invoice;
//     /api/v1/customers/{id}/balance|ledger|wallet|billing/statement|billing/forecast;
//     /api/v1/disputes; self-serve GET /api/v1/selfserve/billing/statement.
//   - ExportHTTPHandlers: POST/GET /api/v1/billing/exports and download (async jobs).
//   - CostSyncHTTPHandlers: /api/v1/cost-sync/* credentials, networks, run, history.
//   - CryptoWebhookHandlers: POST /api/v1/billing/crypto/webhook (management listener).
//   - Workspace usage CSV: GET /api/v1/billing/usage/export (workspace_handlers.go).
//   - CompositeReadService joins PG ledger with optional CH metrics (forecast timeout 1.5 s).
//   - Workers: VolumeMeterWorker, LedgerInvariantWorker, CreditScoringWorker,
//     UsageDailyFlushWorker (optional).
//
// Topology:
//   - Routes registered from controlplane adminapi_wire.go RouteRegistry; billing_bridge.go
//     implements CreditHost and customer balance reads on Service.
//   - Volume meter and ledger invariant workers started in controlplane serve_workers.go;
//     credit scoring in controlplane workers.go (24 h ticker).
//   - Usage daily flush only when USAGE_DAILY_FLUSH_ENABLED=1 and ClickHouse is configured.
//   - Self-serve billing reads alias campaign balance DTOs; writes require operator permissions.
//
// Invariants:
//   - Operator money fields sourced from Postgres balance_ledger; micros or *_display in DTOs.
//   - Ledger invariant scan tolerance +/-1 micro-unit (ledgerInvariantToleranceMicro).
//   - Fleet invariant endpoint scans up to fleetInvariantScanLimit (500) customers per request.
//   - Export chunk default 10 MiB (DefaultExportChunkMaxBytes); usage export batches 500 rows.
//   - Crypto webhook verifies provider signature before crediting ledger.
//   - cost-sync sync_interval_minutes must be 15, 30, 60, or 1440 when set.
//
// Defaults and limits:
//   - LEDGER_INVARIANT_INTERVAL_HOURS default 24 (config.LedgerInvariantIntervalHours).
//   - VOLUME_METER_INTERVAL default 1 h (time.ParseDuration); disable with VOLUME_METER_ENABLED=0.
//   - VOLUME_METER_SOURCE default pg; clickhouseQuery reads accepted_events from CH when set
//     and ClickHouse client is wired (VolumeMeterSourceCH).
//   - USAGE_DAILY_FLUSH_INTERVAL default 24 h when USAGE_DAILY_FLUSH_ENABLED=1.
//   - Credit scoring interval fixed 24 h in controlplane; knobs CREDIT_SCORING_MIN_AGE_DAYS,
//     CREDIT_SCORING_MATURE_AGE_DAYS, CREDIT_SCORING_MID_TIER_PERCENT,
//     CREDIT_SCORING_MATURE_PERCENT, CREDIT_SCORING_MAX_CAP,
//     CREDIT_SCORING_RECON_LAG_THRESHOLD_MICRO, CREDIT_SCORING_RECON_LAG_PENALTY_PCT.
//   - Handler bodies use pkg/coldpath.DefaultMaxBody unless export-specific limits apply.
//
// Forbidden:
//   - KEYS, FLUSHALL, FLUSHDB Redis commands (cold_path_static_gate.sh).
//   - N+1 customer loops on changed handlers without batch sqlc queries.
//   - Tracker hot path imports.
//
// Verify:
//
//	go list -e ./internal/billingadmin/...
//	go test ./internal/billingadmin/ -short -run TestBilling_parseMoneyMicro -count=1
//	go test ./internal/billingadmin/ -short -run TestBillingCryptoWebhook -count=1
//	bash scripts/ci/static/cold_path_static.sh
package billingadmin
