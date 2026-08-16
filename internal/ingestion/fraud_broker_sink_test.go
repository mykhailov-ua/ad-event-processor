package ingestion

import (
	"context"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockFraudBrokerClient struct {
	calls       int
	lastTopic   string
	lastPart    uint16
	lastPayload []byte
	err         error
}

func (m *mockFraudBrokerClient) Produce(_ context.Context, topic string, partition uint16, payload []byte) (uint64, error) {
	m.calls++
	m.lastTopic = topic
	m.lastPart = partition
	m.lastPayload = append([]byte(nil), payload...)
	if m.err != nil {
		return 0, m.err
	}
	return 1, nil
}

func (m *mockFraudBrokerClient) Close() error { return nil }

func TestBrokerConcatLengthPrefixedMessages(t *testing.T) {
	out := brokerConcatLengthPrefixedMessages([][]byte{[]byte("ab"), []byte("c")})
	require.NotEmpty(t, out)
}

func TestFraudBrokerSink_Produce(t *testing.T) {
	mock := &mockFraudBrokerClient{}
	sink := NewFraudBrokerSinkWithClient(mock, "ad-fraud-events")
	err := sink.Produce(context.Background(), 2, [][]byte{[]byte("payload")})
	require.NoError(t, err)
	assert.Equal(t, 1, mock.calls)
	assert.Equal(t, "ad-fraud-events", mock.lastTopic)
	assert.Equal(t, uint16(2), mock.lastPart)
	assert.NotEmpty(t, mock.lastPayload)
}

func TestFraudStreamWriter_brokerFlushSkipsRedis(t *testing.T) {
	rdb := &countingRedisXAdd{}
	q := NewFraudStreamWriter([]redis.UniversalClient{rdb}, "fraud-stream", 1000)
	require.NotNil(t, q)

	mock := &mockFraudBrokerClient{}
	q.SetBrokerSink(NewFraudBrokerSinkWithClient(mock, "ad-fraud-events"))

	evt := &domain.Event{
		ClickID:     "click-1",
		CampaignID:  uuid.New(),
		Type:        "click",
		FraudReason: "geo",
	}
	require.True(t, q.Enqueue(1, evt))
	q.Stop()
	assert.Equal(t, 0, int(rdb.xadds.Load()))
	assert.Equal(t, 1, mock.calls)
}
