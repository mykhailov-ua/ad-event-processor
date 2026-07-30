package ingestion

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrInsufficientCustomerBalance = errors.New("insufficient customer balance for spend batch")

const (
	defaultLedgerBatchFlush = 10 * time.Second
	maxLedgerBatchSize      = 32
)

func MaxLedgerBatchSize() int {
	return maxLedgerBatchSize
}

var ErrCampaignSpendSkipped = errors.New("campaign spend row locked")

type SpendFlushItem struct {
	CampaignID          uuid.UUID
	AmountMicro         int64
	RtbCostMicro        int64
	TxID                string
	RedisRemainingMicro int64
	StrictFlush         bool
}

type SpendFlushOutcome struct {
	CampaignID uuid.UUID
	Err        error
}

type spendBatchFlusher interface {
	UpdateSpendBatch(ctx context.Context, items []SpendFlushItem) ([]SpendFlushOutcome, error)
}

type pendingRollup struct {
	amountMicro         int64
	txID                string
	idStr               string
	syncKey             string
	inFlightKey         string
	lockKey             string
	txKey               string
	dirtySet            string
	redisRemainingMicro int64
}

func ledgerBatchHash(txID string) string {
	return "spend_batch:" + txID
}
