package controlplane

import (
	"encoding/json"
	"time"

	db "github.com/bidshard/ad-event-processor/internal/domain/db"
)

type auditReasonChange struct {
	Reason string `json:"reason"`
}

type auditIdChange struct {
	ID int64 `json:"id"`
}

type auditIdempotencyMeta struct {
	IdempotencyKey string `json:"idempotency_key"`
}

type auditOutboxEventMeta struct {
	OutboxEventID int64 `json:"outbox_event_id"`
}

type auditTxSourceMeta struct {
	TxSource string `json:"tx_source"`
}

type auditQuotaRepairMeta struct {
	OutboxEventID int64  `json:"outbox_event_id"`
	Reason        string `json:"reason"`
	RepairMicro   int64  `json:"repair_micro"`
}

type auditQuotaDeadShardRelease struct {
	ShardID      int   `json:"shard_id"`
	RowsReleased int64 `json:"rows_released"`
}

type auditCreateCustomerChange struct {
	Name    string `json:"name"`
	Balance int64  `json:"balance"`
}

type auditAmountChange struct {
	Amount int64 `json:"amount"`
}

type auditPaymentSettlementChange struct {
	Amount          int64  `json:"amount"`
	PaymentIntentID string `json:"payment_intent_id"`
	Provider        string `json:"provider"`
	ProviderRef     string `json:"provider_ref"`
}

type auditPaymentRefundChange struct {
	Amount           int64  `json:"amount"`
	PaymentIntentID  string `json:"payment_intent_id"`
	Provider         string `json:"provider"`
	ProviderRefundID string `json:"provider_refund_id"`
}

type auditPaymentDisputeChange struct {
	Amount            int64  `json:"amount"`
	PaymentIntentID   string `json:"payment_intent_id"`
	Provider          string `json:"provider"`
	ProviderDisputeID string `json:"provider_dispute_id"`
}

type auditOverdraftChange struct {
	OldOverdraft string `json:"old_overdraft"`
	NewOverdraft string `json:"new_overdraft"`
}

type auditPacingChange struct {
	OldPacingMode string `json:"old_pacing_mode"`
	NewPacingMode string `json:"new_pacing_mode"`
}

type auditCampaignAdminChange struct {
	Name            string   `json:"name"`
	DailyBudget     int64    `json:"daily_budget_micro"`
	Timezone        string   `json:"timezone"`
	TargetCountries []string `json:"target_countries,omitempty"`
}

type auditBrandFcapChange struct {
	OldFreqLimit  int32 `json:"old_freq_limit"`
	OldFreqWindow int32 `json:"old_freq_window"`
	NewFreqLimit  int32 `json:"new_freq_limit"`
	NewFreqWindow int32 `json:"new_freq_window"`
}

type auditCampaignScheduleChange struct {
	StartAt      *time.Time `json:"start_at,omitempty"`
	EndAt        *time.Time `json:"end_at,omitempty"`
	DaypartHours []int16    `json:"daypart_hours,omitempty"`
}

type auditPacingLoopAdjustment struct {
	OldPacing string `json:"old_pacing"`
	NewPacing string `json:"new_pacing"`
	Spend     string `json:"spend"`
	Expected  string `json:"expected"`
	Curve     string `json:"curve"`
}

type auditCampaignFraudChange struct {
	FraudThresholdPass    uint8 `json:"fraud_threshold_pass"`
	FraudThresholdSuspect uint8 `json:"fraud_threshold_suspect"`
	FraudThresholdIVT     uint8 `json:"fraud_threshold_ivt"`
	FraudThresholdBlock   uint8 `json:"fraud_threshold_block"`
	GhostIVTEnabled       bool  `json:"ghost_ivt_enabled"`
	BehaviorFlags         int32 `json:"behavior_flags"`
}

type auditCreateCampaignChange struct {
	Name         string                `json:"name"`
	BudgetLimit  int64                 `json:"budget_limit"`
	Status       db.CampaignStatusType `json:"status"`
	StartAt      *time.Time            `json:"start_at,omitempty"`
	EndAt        *time.Time            `json:"end_at,omitempty"`
	DaypartHours []int16               `json:"daypart_hours,omitempty"`
}

type auditCohortSnapshotChange struct {
	Name     string `json:"name"`
	Active   bool   `json:"active"`
	Variants int    `json:"variants"`
}

type auditSellerCreateChange struct {
	SellerID string `json:"seller_id"`
	Domain   string `json:"domain"`
}

type auditSellerUpdateChange struct {
	ID       int64  `json:"id"`
	SellerID string `json:"seller_id"`
}

type auditAdsTxtDomainChange struct {
	Domain string `json:"domain"`
}

type auditSupplyChainChange struct {
	OldNodes json.RawMessage `json:"old_nodes"`
	NewNodes json.RawMessage `json:"new_nodes"`
}

type auditAutoscaleBudgetTransfer struct {
	OldBudget string  `json:"old_budget"`
	NewBudget string  `json:"new_budget"`
	CTR       float64 `json:"ctr"`
	Target    string  `json:"target,omitempty"`
	Source    string  `json:"source,omitempty"`
}

type auditEmergencyBreakerChange struct {
	Active bool   `json:"active"`
	Reason string `json:"reason"`
}

type auditRtbDealCreateChange struct {
	DealID string `json:"deal_id"`
}

type auditRtbDealUpdateChange struct {
	ID     int64  `json:"id"`
	DealID string `json:"deal_id"`
}

type auditRtbDealDeleteChange struct {
	ID     int64  `json:"id"`
	DealID string `json:"deal_id"`
}

type auditSlotMapVersionCreated struct {
	BaseVersion int32           `json:"base_version"`
	NewVersion  int32           `json:"new_version"`
	Overrides   json.RawMessage `json:"overrides"`
}

type auditSlotMapMarkMigrating struct {
	Version     int32           `json:"version"`
	Slots       json.RawMessage `json:"slots"`
	TargetShard int16           `json:"target_shard"`
}

type auditSlotMapActivated struct {
	Version          int32 `json:"version"`
	MigratedSlots    int   `json:"migrated_slots"`
	MigrationCutover bool  `json:"migration_cutover"`
}

type auditSlotMapRollback struct {
	FromVersion int32 `json:"from_version"`
	ToVersion   int32 `json:"to_version"`
}

type auditLicenseApplyChange struct {
	DeploymentID string `json:"deployment_id"`
	ValidUntil   string `json:"valid_until"`
	CustomerName string `json:"customer_name,omitempty"`
	Plan         string `json:"plan,omitempty"`
	Revoked      bool   `json:"revoked,omitempty"`
}
