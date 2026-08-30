package broker

import (
	"context"
	"time"

	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/broker/client"
)

// FraudBrokerSink publishes length-prefixed vtproto batches to a dedicated fraud broker topic
// when CH_INGEST_SOURCE=broker (replaces per-shard Redis fraud XADD on the cold path).
// Lane is separate from tracker BrokerProducer (main ad-events topic): own topic, partition, and
// consumer group in cmd/processor; wire layout matches brokerConcatLengthPrefixedMessages.
//
// Verify: go test ./internal/stream/broker/ -short -run TestBrokerProducer_EnqueueAndFlush -count=1
type FraudBrokerSink struct {
	client BrokerClient
	topic  string
}

func NewFraudBrokerSink(addr, redisURL, topic string, timeout time.Duration) (*FraudBrokerSink, error) {
	if addr == "" || topic == "" {
		return nil, ErrFraudBrokerSinkConfig
	}
	cli := client.NewClient(addr, timeout)
	if redisURL != "" {
		cli.SetRedisURL(redisURL)
	}
	if err := cli.Connect(); err != nil {
		return nil, err
	}
	return &FraudBrokerSink{client: cli, topic: topic}, nil
}

func NewFraudBrokerSinkWithClient(client BrokerClient, topic string) *FraudBrokerSink {
	return &FraudBrokerSink{client: client, topic: topic}
}

func (s *FraudBrokerSink) Topic() string {
	if s == nil {
		return ""
	}
	return s.topic
}

func (s *FraudBrokerSink) Produce(ctx context.Context, partition uint16, payloads [][]byte) error {
	if s == nil || s.client == nil || len(payloads) == 0 {
		return nil
	}
	// Wire: brokerConcatLengthPrefixedMessages joins N AdLogRecord/AdStreamEvent frames as
	// uvarint(len)||payload... for one mmap WAL Produce (same encoding as main events path).
	buf := brokerConcatLengthPrefixedMessages(payloads)
	if len(buf) == 0 {
		return nil
	}
	start := time.Now()
	_, err := s.client.Produce(ctx, s.topic, partition, buf)
	metrics.BrokerWriteDuration.WithLabelValues(s.topic).Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.BrokerProducedEventsTotal.WithLabelValues("fraud_error").Add(float64(len(payloads)))
		return err
	}
	metrics.BrokerProducedEventsTotal.WithLabelValues("fraud_ok").Add(float64(len(payloads)))
	return nil
}

func (s *FraudBrokerSink) Close() error {
	if s == nil || s.client == nil {
		return nil
	}
	return s.client.Close()
}
