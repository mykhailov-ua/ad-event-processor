package payment

import (
	"encoding/json"
	"time"

	"github.com/bidshard/ad-event-processor/internal/payment/db"

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
	Detail             json.RawMessage
}

type reconDetailStatus struct {
	Status string `json:"status"`
}

type reconDetailOrphanTopup struct {
	OrphanTopupMicro int64 `json:"orphan_topup_micro"`
}

type reconDetailDeadOutbox struct {
	OutboxID  int64  `json:"outbox_id"`
	EventType string `json:"event_type"`
	LastError string `json:"last_error"`
	Attempts  int32  `json:"attempts"`
}

func marshalReconDetail(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return b
}
