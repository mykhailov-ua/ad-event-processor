package ingestion

import (
	"context"
	"time"

	"github.com/bidshard/ad-event-processor/internal/metrics"
	"github.com/bidshard/ad-event-processor/pkg/broker/client"
)

// FraudBrokerSink produces length-prefixed vtproto fraud events to a dedicated broker topic.
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
