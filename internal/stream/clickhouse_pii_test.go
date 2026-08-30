package stream

import (
	"context"
	"strings"
	"testing"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/pkg/piihash"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Holdout: default-table append must use hashed ip/ua columns (indices 3,4,5), never raw strings.
func TestClickHouseStore_StoreBatch_hashesPII(t *testing.T) {
	hasher := piihash.TestHasher()
	var captured [][]any

	connMock := &mockConn{
		prepareBatchFn: func(ctx context.Context, query string) (driver.Batch, error) {
			return &mockBatch{
				appendFn: func(v ...any) error {
					row := append([]any(nil), v...)
					captured = append(captured, row)
					return nil
				},
			}, nil
		},
	}

	store := NewClickHouseStore(connMock, 100*time.Millisecond, "", DefaultClickHouseSpoolConfig(), nil)
	store.SetPIIHasher(hasher)

	rawIP := "203.0.113.10"
	rawUA := "Mozilla/5.0 test"
	evt := &domain.Event{
		ClickID:    "click-pii",
		CampaignID: uuid.New(),
		Type:       "impression",
		IP:         rawIP,
		UA:         rawUA,
		CreatedAt:  time.Now(),
	}

	require.NoError(t, store.StoreBatch(context.Background(), []*domain.Event{evt}))
	require.Len(t, captured, 1)

	ipHash := hasher.HashIP(rawIP)
	uaHash := hasher.HashUA(rawUA)
	assert.Equal(t, piihash.FixedString16(ipHash), captured[0][3])
	assert.Equal(t, piihash.FixedString16(uaHash), captured[0][4])
	assert.Equal(t, uint8(1), captured[0][5])

	for _, row := range captured {
		for _, v := range row {
			if s, ok := v.(string); ok {
				assert.NotEqual(t, rawIP, s)
				assert.NotEqual(t, rawUA, s)
			}
		}
	}
}

// Holdout: fraud_events user_id_hash is column 2; silent_reject maps via fraudSilentRejectFlag in insertTable.
func TestClickHouseStore_StoreBatch_fraudEventUserIDHash(t *testing.T) {
	hasher := piihash.TestHasher()
	var captured [][]any

	connMock := &mockConn{
		prepareBatchFn: func(ctx context.Context, query string) (driver.Batch, error) {
			if !strings.Contains(query, "fraud_events") {
				return &mockBatch{}, nil
			}
			return &mockBatch{
				appendFn: func(v ...any) error {
					captured = append(captured, append([]any(nil), v...))
					return nil
				},
			}, nil
		},
	}

	store := NewClickHouseStore(connMock, 100*time.Millisecond, "", DefaultClickHouseSpoolConfig(), nil)
	store.SetPIIHasher(hasher)

	evt := &domain.Event{
		ClickID:     "fraud-1",
		CampaignID:  uuid.New(),
		UserID:      "user-secret",
		Type:        "click",
		IP:          "10.0.0.1",
		UA:          "bot",
		FraudReason: "test",
		FraudScore:  90,
		CreatedAt:   time.Now(),
	}

	require.NoError(t, store.StoreBatch(context.Background(), []*domain.Event{evt}))
	require.Len(t, captured, 1)
	assert.Equal(t, piihash.FixedString16(hasher.HashUserID("user-secret")), captured[0][2])
}
