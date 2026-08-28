package controlplane

import (
	"ad-event-processor/internal/governance"
	"ad-event-processor/internal/reconciliation"
)

type BrokerPendingDeltaReader = reconciliation.BrokerPendingDeltaReader

type GlobalSpendReconciler = reconciliation.GlobalSpendReconciler

type GlobalSpendReconcilerConfig = reconciliation.GlobalSpendReconcilerConfig

var NewGlobalSpendReconciler = reconciliation.NewGlobalSpendReconciler

type QuotaRepairPayload = governance.QuotaRepairPayload

type ReconciliationAdjustPayload = reconciliation.ReconciliationAdjustPayload

type ReconService = reconciliation.ReconService

var NewReconService = reconciliation.NewReconService
