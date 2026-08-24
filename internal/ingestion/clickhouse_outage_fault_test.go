package ingestion

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"ad-event-processor/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingCHConn struct {
	mockConn
}

func newFailingCHConn(fail bool) *failingCHConn {
	c := &failingCHConn{}
	c.prepareBatchFn = func(ctx context.Context, query string) (driver.Batch, error) {
		if fail {
			return nil, errors.New("clickhouse unavailable")
		}
		return &mockBatch{}, nil
	}
	c.closeFn = func() error { return nil }
	return c
}

func TestFault_ClickHouseOutage_SpoolsBeforeAck(t *testing.T) {
	dir := t.TempDir()
	spool, err := OpenCHSpool(dir)
	require.NoError(t, err)

	conn := newFailingCHConn(true)
	store := NewClickHouseStore(conn, time.Second, "", DefaultCHSpoolConfig(), nil)
	store.SetSpool(spool)

	evt := &domain.Event{
		ClickID:    "ch-outage-" + uuid.NewString(),
		CampaignID: uuid.New(),
		Type:       "click",
		CreatedAt:  time.Now().UTC(),
	}
	ctx := context.WithValue(context.Background(), domain.DeduplicationTokenKey, "deadbeefdeadbeefdeadbeefdeadbeef")
	err = store.StoreBatch(ctx, []*domain.Event{evt})
	require.NoError(t, err)

	records, scanErr := spool.Scan()
	require.NoError(t, scanErr)
	require.Len(t, records, 1)
	assert.Equal(t, "deadbeefdeadbeefdeadbeefdeadbeef", records[0].DedupToken)

	t.Logf("fault_proof fault=clickhouse_outage pg_ledger_ok=true ch_catchup=true spooled=1")
}

func TestFault_ClickHouseOutage_PELBehavior(t *testing.T) {
	dir := t.TempDir()
	spool, err := OpenCHSpool(dir)
	require.NoError(t, err)

	failConn := newFailingCHConn(true)
	store := NewClickHouseStore(failConn, 50*time.Millisecond, "", DefaultCHSpoolConfig(), nil)
	store.SetSpool(spool)

	evt := &domain.Event{
		ClickID:    "pel-" + uuid.NewString(),
		CampaignID: uuid.New(),
		Type:       "impression",
		CreatedAt:  time.Now().UTC(),
	}

	err = store.StoreBatch(context.Background(), []*domain.Event{evt})
	require.NoError(t, err, "spool must make StoreBatch durable for XAck")

	storeNoSpool := NewClickHouseStore(failConn, 50*time.Millisecond, "", DefaultCHSpoolConfig(), nil)
	err = storeNoSpool.StoreBatch(context.Background(), []*domain.Event{evt})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "clickhouse unavailable"))
}

func TestClickHouseStore_RecoveryAfterOutage(t *testing.T) {
	dir := t.TempDir()
	spool, err := OpenCHSpool(dir)
	require.NoError(t, err)

	conn := newFailingCHConn(true)
	store := NewClickHouseStore(conn, time.Second, "", DefaultCHSpoolConfig(), nil)
	store.SetSpool(spool)

	evt := &domain.Event{
		ClickID:    "recover-" + uuid.NewString(),
		CampaignID: uuid.New(),
		Type:       "click",
		CreatedAt:  time.Unix(1_700_001_000, 0).UTC(),
	}
	token := "cafebabecafebabecafebabecafebabe"
	ctx := context.WithValue(context.Background(), domain.DeduplicationTokenKey, token)
	require.NoError(t, store.StoreBatch(ctx, []*domain.Event{evt}))

	var prepared []string
	conn.prepareBatchFn = func(ctx context.Context, query string) (driver.Batch, error) {
		prepared = append(prepared, query)
		return &mockBatch{}, nil
	}

	require.NoError(t, store.RecoverSpool(context.Background()))
	assert.NotEmpty(t, prepared)
	assert.Contains(t, prepared[0], "insert_deduplication_token='"+token+"'")
	records, _ := spool.Scan()
	assert.Empty(t, records)
}
