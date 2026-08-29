package broker

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"path/filepath"
	"testing"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/ingest/pb"
	sbroker "ad-event-processor/internal/stream/broker"
	blog "ad-event-processor/pkg/broker/log"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockReplayStore struct {
	replayed []*domain.Event
}

func (m *mockReplayStore) StoreBatch(ctx context.Context, events []*domain.Event) error {
	m.replayed = append(m.replayed, events...)
	return nil
}

func (m *mockReplayStore) Close() error {
	return nil
}

func TestBrokerReplay_Integrity(t *testing.T) {
	dataDir := t.TempDir()
	topic := "ad-events"
	partDir := filepath.Join(dataDir, topic, "partition_0")

	partLog, err := blog.NewPartitionLog(context.Background(), partDir, 1024*1024*1024, 4096)
	require.NoError(t, err)

	expectedHasher := sha256.New()
	baseTime := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)

	for i := range 100 {
		clickID := uuid.New().String()
		evtType := "click"
		campID := uuid.New()

		expectedHasher.Write([]byte(clickID))
		expectedHasher.Write([]byte(evtType))

		pbEvt := &pb.AdStreamEvent{
			ClickId:       []byte(clickID),
			EventType:     []byte(evtType),
			CampaignId:    campID[:],
			CreatedAtUnix: baseTime.Add(time.Duration(i) * time.Minute).Unix(),
		}

		size := pbEvt.SizeVT()
		var lenBuf [binary.MaxVarintLen64]byte
		nLen := binary.PutUvarint(lenBuf[:], uint64(size))
		buf := append([]byte{}, lenBuf[:nLen]...)
		startIdx := len(buf)
		buf = append(buf, make([]byte, size)...)
		_, err := pbEvt.MarshalToSizedBufferVT(buf[startIdx:])
		require.NoError(t, err)

		_, err = partLog.Append(buf)
		require.NoError(t, err)
	}

	require.NoError(t, partLog.Close())
	expectedHash := hex.EncodeToString(expectedHasher.Sum(nil))

	mockStore := &mockReplayStore{}
	replayer := NewReplayer(ReplayConfig{
		DataDir:   dataDir,
		Topic:     topic,
		BatchSize: 20,
	}, mockStore, sbroker.ParseBrokerPayloadStream)

	res, err := replayer.Replay(context.Background())
	require.NoError(t, err)

	assert.Equal(t, int64(100), res.EventsRead)
	assert.Equal(t, int64(100), res.EventsReplayed)
	assert.Equal(t, expectedHash, res.PayloadHash, "integrity hash must match replayed payloads")
	assert.Equal(t, 100, len(mockStore.replayed))
}

func TestBrokerReplay_TimestampFiltering(t *testing.T) {
	dataDir := t.TempDir()
	topic := "ad-events-filter"
	partDir := filepath.Join(dataDir, topic, "partition_0")

	partLog, err := blog.NewPartitionLog(context.Background(), partDir, 1024*1024*1024, 4096)
	require.NoError(t, err)

	baseTime := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)

	for i := range 10 {
		pbEvt := &pb.AdStreamEvent{
			ClickId:       []byte(uuid.New().String()),
			EventType:     []byte("impression"),
			CreatedAtUnix: baseTime.Add(time.Duration(i) * time.Minute).Unix(),
		}
		size := pbEvt.SizeVT()
		var lenBuf [binary.MaxVarintLen64]byte
		nLen := binary.PutUvarint(lenBuf[:], uint64(size))
		buf := append([]byte{}, lenBuf[:nLen]...)
		startIdx := len(buf)
		buf = append(buf, make([]byte, size)...)
		_, err := pbEvt.MarshalToSizedBufferVT(buf[startIdx:])
		require.NoError(t, err)

		_, err = partLog.Append(buf)
		require.NoError(t, err)
	}

	require.NoError(t, partLog.Close())

	fromTime := baseTime.Add(3 * time.Minute)
	toTime := baseTime.Add(6 * time.Minute)

	mockStore := &mockReplayStore{}
	replayer := NewReplayer(ReplayConfig{
		DataDir:   dataDir,
		Topic:     topic,
		From:      fromTime,
		To:        toTime,
		BatchSize: 10,
	}, mockStore, sbroker.ParseBrokerPayloadStream)

	res, err := replayer.Replay(context.Background())
	require.NoError(t, err)

	assert.Equal(t, int64(10), res.EventsRead)
	assert.Equal(t, int64(4), res.EventsReplayed)
	assert.Equal(t, 4, len(mockStore.replayed))
}
