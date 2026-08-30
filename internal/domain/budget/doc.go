// Package budget provides budget manager contracts, Postgres settlement, and spend invariant helpers.
//
// Role:
//   - BudgetManager.CheckAndSpend for RTB when RTB_BUDGET_AUTHORITY=rtb.
//   - SpendBatchFlusher and ledger batching for SyncWorker Postgres spend flush.
//   - PaymentSettlement credit/refund/chargeback legs; CTV gtax and margin economics splits.
//   - IngressCostConfig JSON parse; recon snapshot Lua script; campaign registry broker publish.
//
// Topology:
//   - Imported by internal/rtb, processor settlement, domain.SyncWorker, reconciliation workers.
//   - ReadBudgetInvariant joins Postgres campaigns row with shard.BudgetCampaignKey and CampaignSyncKey.
//   - Not registered on admin HTTP handlers; controlplane bridges call settlement stores directly.
//
// Invariants:
//   - VerifyBudgetInvariant: redis spend matches PG current_spend + sync_delta within tolerance.
//   - AssertBudgetInvariant is the test helper wrapper; fault tests call after spend paths.
//   - CheckAndSpend idempotency is enforced by caller click_id or dedup key, not inside BudgetManager.
//   - Spend batch flush skips locked rows with ErrCampaignSpendSkipped without aborting the batch.
//
// Forbidden:
//   - json or db struct tags on hot-path budget interfaces in this package.
//   - Direct admin HTTP handler registration from budget package.
//
// Defaults and limits:
//   - budgetInvariantToleranceMicro: 1 micro-unit for AssertBudgetInvariant / VerifyBudgetInvariant.
//   - maxLedgerBatchSize: 32; defaultLedgerBatchFlush: 10s (MaxLedgerBatchSize, DefaultLedgerBatchFlush).
//   - DefaultCampaignUpdateBrokerTopic: campaigns:update; RegistryFullSyncPayload: *.
//   - PublishCampaignUpdateBroker default timeout: 3s when caller passes zero.
//
// Verify:
//
//	go test ./internal/domain/budget/... -short -count=1
//	go test ./internal/domain/budget/... -short -run TestReadBudgetInvariants_emptyIDs -count=1
//	go test ./internal/domain/budget/... -short -run TestParseIngressCostConfigJSON -count=1
package budget
