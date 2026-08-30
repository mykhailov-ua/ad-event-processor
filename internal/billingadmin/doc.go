// Package billingadmin serves operator billing HTTP and billing maintenance workers.
//
// Role:
//   - HTTP: /api/v1/billing/*, /api/v1/customers/{id}/balance|ledger|wallet, disputes, cost-sync admin.
//   - Crypto billing webhook: POST /api/v1/billing/crypto/webhook on management listener.
//   - Workers: VolumeMeterWorker, LedgerInvariantWorker, CreditScoringWorker, UsageDailyFlushWorker.
//   - Composite reads (statement, forecast, invariant) join PG ledger with optional CH metrics.
//
// Topology:
//   - Handlers registered from controlplane/register.go; Host port satisfied by controlplane billing_bridge.
//   - Self-serve billing reads alias campaign balance DTOs; writes stay on operator permissions.
//   - Export jobs for workspace billing land in billingadmin/export handlers + reportjob when async.
//
// Invariants:
//   - Operator money fields come from PG; display formatting in handler DTOs (micros or *_display).
//   - Ledger invariant worker interval: LEDGER_INVARIANT_INTERVAL_HOURS default 24.
//   - Volume meter default interval 1 h unless VOLUME_METER_INTERVAL set; source from VOLUME_METER_SOURCE.
//
// Forbidden:
//   - KEYS/FLUSHALL/FLUSHDB Redis commands (cold_path_static_gate.sh).
//   - N+1 customer loops on changed handlers without batch sqlc queries.
//
// Verify:
//
//	go test ./internal/billingadmin/ -short -count=1
//	bash scripts/ci/static/cold_path_static.sh
package billingadmin
