package shard

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"ad-event-processor/pkg/broker/client"
)

func PublishSlotMapReload(ctx context.Context, brokerURL, brokerRedisURL, topic string, timeout time.Duration, version int32, routingEpoch int64) error {
	if brokerURL == "" {
		return nil
	}
	if topic == "" {
		topic = DefaultSlotMapReloadTopic
	}
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	payload, err := EncodeSlotMapReloadMessage(version, routingEpoch)
	if err != nil {
		return err
	}
	cli := client.NewClient(brokerURL, timeout)
	if brokerRedisURL != "" {
		cli.SetRedisURL(brokerRedisURL)
	}
	if err := cli.Connect(); err != nil {
		return fmt.Errorf("slot map reload publish connect: %w", err)
	}
	defer func() { _ = cli.Close() }()
	publishCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if _, err := cli.Produce(publishCtx, topic, 0, payload); err != nil {
		return fmt.Errorf("slot map reload publish: %w", err)
	}
	slog.Info("published slot map reload", "topic", topic, "version", version)
	return nil
}
