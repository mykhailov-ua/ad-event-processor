package fraud

import (
	"context"
	"testing"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/stream/broker"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockFraudBrokerProduceClient struct {
	calls int
}

func (m *mockFraudBrokerProduceClient) Produce(_ context.Context, topic string, partition uint16, payload []byte) (uint64, error) {
	m.calls++
	return 1, nil
}

func (m *mockFraudBrokerProduceClient) Close() error { return nil }

func TestFraudStreamWriter_brokerFlushSkipsRedis(t *testing.T) {
	redisClient := &countingRedisXAdd{}
	q := NewFraudStreamWriter([]redis.UniversalClient{redisClient}, "fraud-stream", 1000)
	require.NotNil(t, q)
	defer q.Stop()

	mock := &mockFraudBrokerProduceClient{}
	q.SetBrokerSink(broker.NewFraudBrokerSinkWithClient(mock, "ad-fraud-events"))

	evt := &domain.Event{
		ClickID:     "click-1",
		CampaignID:  uuid.New(),
		Type:        "click",
		FraudReason: "geo",
	}
	require.True(t, q.Enqueue(1, evt))
	q.Stop()
	assert.Equal(t, 0, int(redisClient.xadds.Load()))
	assert.Equal(t, 1, mock.calls)
}
