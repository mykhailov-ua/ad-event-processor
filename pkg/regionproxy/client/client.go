package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	bclient "ad-event-processor/pkg/broker/client"
	rserver "ad-event-processor/pkg/regionproxy/server"
)

type Config struct {
	Addr     string
	RedisURL string
	Timeout  time.Duration
}

type Client struct {
	inner    *bclient.Client
	topicID  uint16
	register sync.Once
	regErr   error
}

func New(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	inner := bclient.NewClient(cfg.Addr, timeout)
	if cfg.RedisURL != "" {
		inner.SetRedisURL(cfg.RedisURL)
	}
	return &Client{inner: inner}
}

func (c *Client) Close() error {
	if c == nil || c.inner == nil {
		return nil
	}
	return c.inner.Close()
}

func (c *Client) ensureTopic() error {
	if c == nil || c.inner == nil {
		return fmt.Errorf("region-proxy client: unavailable")
	}
	c.register.Do(func() {
		c.topicID, c.regErr = c.inner.RegisterTopic(context.Background(), rserver.DefaultIngressTopic)
	})
	return c.regErr
}

func (c *Client) ProduceSpendSyncPayload(payload []byte) (bclient.ProduceBatchResult, error) {
	if err := c.ensureTopic(); err != nil {
		return bclient.ProduceBatchResult{}, fmt.Errorf("region-proxy produce: %w", err)
	}
	result, err := c.inner.ProduceBatch(context.Background(), rserver.DefaultIngressTopic, c.topicID, [][]byte{payload})
	if err != nil {
		return bclient.ProduceBatchResult{}, fmt.Errorf("region-proxy produce: %w", err)
	}
	return result, nil
}
