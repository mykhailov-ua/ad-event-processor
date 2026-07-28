// Package client sends spend sync batches to region-proxy ingress via the broker wire protocol.
package client

import (
	"fmt"
	"sync"
	"time"

	bclient "espx/pkg/broker/client"
	rserver "espx/pkg/regionproxy/server"
)

// Config tunes the region-proxy TCP client.
type Config struct {
	Addr     string
	RedisURL string
	Timeout  time.Duration
}

// Client appends spend sync payloads to the regional proxy WAL.
type Client struct {
	inner    *bclient.Client
	topicID  uint16
	register sync.Once
	regErr   error
}

// New builds a region-proxy ingress client.
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

// Close tears down the underlying TCP session.
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
		c.topicID, c.regErr = c.inner.RegisterTopic(rserver.DefaultIngressTopic)
	})
	return c.regErr
}

// ProduceSpendSyncPayload appends one spend sync JSON payload to the proxy WAL.
func (c *Client) ProduceSpendSyncPayload(payload []byte) (bclient.ProduceBatchResult, error) {
	if err := c.ensureTopic(); err != nil {
		return bclient.ProduceBatchResult{}, fmt.Errorf("region-proxy produce: %w", err)
	}
	result, err := c.inner.ProduceBatch(rserver.DefaultIngressTopic, c.topicID, [][]byte{payload})
	if err != nil {
		return bclient.ProduceBatchResult{}, fmt.Errorf("region-proxy produce: %w", err)
	}
	return result, nil
}
