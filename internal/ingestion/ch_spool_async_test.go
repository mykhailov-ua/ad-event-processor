package ingestion

import (
	"strconv"
	"testing"
	"time"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCHSpool_AsyncFlusher(t *testing.T) {
	dir := t.TempDir()
	spool, err := OpenCHSpool(dir)
	require.NoError(t, err)

	spool.StartAsyncFlusher(10 * time.Millisecond)

	campID := uuid.New()
	evt := &domain.Event{
		CampaignID: campID,
		UserID:     "user_async",
		Type:       "click",
	}

	err = spool.AppendDurably("token-async-1", []*domain.Event{evt})
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	records, err := spool.Scan()
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "token-async-1", records[0].DedupToken)

	require.NoError(t, spool.Close())
}

func BenchmarkCHSpool_AsyncAppend(b *testing.B) {
	dir := b.TempDir()
	spool, err := OpenCHSpool(dir)
	require.NoError(b, err)
	defer func() { _ = spool.Close() }()

	spool.StartAsyncFlusher(50 * time.Millisecond)

	campID := uuid.New()
	evt := &domain.Event{
		CampaignID: campID,
		UserID:     "user_bench",
		Type:       "click",
	}

	b.ReportAllocs()
	benchN := 0
	for b.Loop() {
		token := "token-" + strconv.Itoa(benchN)
		if err := spool.AppendDurably(token, []*domain.Event{evt}); err != nil {
			b.Fatal(err)
		}
		benchN++
	}
}
