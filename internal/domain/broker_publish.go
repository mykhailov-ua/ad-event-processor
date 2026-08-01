package domain

import (
	"fmt"
	"log/slog"
	"time"

	"espx/pkg/broker/client"
)

const (
	DefaultCampaignUpdateBrokerTopic = "campaigns:update"
	RegistryFullSyncPayload          = "*"
)

func IsRegistryFullSyncPayload(payload string) bool {
	return payload == RegistryFullSyncPayload
}

func PublishCampaignUpdateBroker(brokerURL, brokerRedisURL, topic string, timeout time.Duration, campaignID string) error {
	if brokerURL == "" || campaignID == "" {
		return nil
	}
	if topic == "" {
		topic = DefaultCampaignUpdateBrokerTopic
	}
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	cli := client.NewClient(brokerURL, timeout)
	if brokerRedisURL != "" {
		cli.SetRedisURL(brokerRedisURL)
	}
	if err := cli.Connect(); err != nil {
		return fmt.Errorf("campaign update broker connect: %w", err)
	}
	defer cli.Close()
	_, err := cli.Produce(topic, 0, []byte(campaignID))
	return err
}

func PublishSlotMapReload(brokerURL, brokerRedisURL, topic string, timeout time.Duration, version int32, routingEpoch int64) error {
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
	defer cli.Close()
	if _, err := cli.Produce(topic, 0, payload); err != nil {
		return fmt.Errorf("slot map reload publish: %w", err)
	}
	slog.Info("published slot map reload", "topic", topic, "version", version)
	return nil
}
