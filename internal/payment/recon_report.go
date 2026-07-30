package payment

import (
	"time"

	"espx/internal/payment/db"

	"github.com/google/uuid"
)

type FinancialReconSummary struct {
	RunID            int64
	PeriodStart      time.Time
	PeriodEnd        time.Time
	IntentsChecked   int
	FindingsCount    int
	FindingsByKind   map[string]int
	TopupAligned     int
	TopupMissing     int
	DeadOutboxRows   int
	SettlementFailed int
}

type FinancialReconFinding struct {
	Kind               db.PaymentFinancialFindingKind
	PaymentIntentID    uuid.UUID
	CustomerID         uuid.UUID
	PaymentAmountMicro int64
	LedgerAmountMicro  int64
	DeltaMicro         int64
	Detail             map[string]any
}
