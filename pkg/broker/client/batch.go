package client

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"ad-event-processor/pkg/broker/protocol"
)

func (c *Client) RegisterTopic(ctx context.Context, topic string) (uint16, error) {
	var lastErr error
	for attempt := range 5 {
		if attempt > 0 {
			time.Sleep(500 * time.Millisecond)
		}

		c.mu.Lock()
		conn, err := c.getConn()
		if err != nil {
			c.mu.Unlock()
			lastErr = err
			if c.redisURL != "" {
				if newAddr, rErr := c.resolveLeaderAddr(ctx, topic, 0); rErr == nil && newAddr != c.addr {
					c.addr = newAddr
				}
			}
			continue
		}

		seq := atomic.AddUint64(&c.nextSeq, 1)
		req := protocol.EncodeRegisterTopicRequest(c.writeBuf, seq, topic)
		if c.timeout > 0 {
			_ = conn.SetDeadline(time.Now().Add(c.timeout))
		}
		if _, err := conn.Write(req); err != nil {
			_ = c.closeRawConn()
			c.mu.Unlock()
			lastErr = err
			if c.redisURL != "" {
				if newAddr, rErr := c.resolveLeaderAddr(ctx, topic, 0); rErr == nil && newAddr != c.addr {
					c.addr = newAddr
				}
			}
			continue
		}

		cmd, respSeq, respPayload, err := protocol.ReadFrameConn(conn, c.readBuf, c.lenBuf)
		if err != nil {
			_ = c.closeRawConn()
			c.mu.Unlock()
			lastErr = err
			if c.redisURL != "" {
				if newAddr, rErr := c.resolveLeaderAddr(ctx, topic, 0); rErr == nil && newAddr != c.addr {
					c.addr = newAddr
				}
			}
			continue
		}
		if cmd != protocol.CmdRegisterTopicResp {
			c.mu.Unlock()
			return 0, fmt.Errorf("unexpected command response: %d", cmd)
		}
		if respSeq != seq {
			c.mu.Unlock()
			return 0, fmt.Errorf("sequence mismatch: expected %d, got %d", seq, respSeq)
		}
		status, topicID, err := protocol.DecodeRegisterTopicResponse(respPayload)
		if err != nil {
			c.mu.Unlock()
			return 0, err
		}
		if status == 4 || status == 5 || status == 6 || status == 7 {
			_ = c.closeRawConn()
			c.mu.Unlock()
			lastErr = brokerStatusError(status)
			if c.redisURL != "" {
				if newAddr, rErr := c.resolveLeaderAddr(ctx, topic, 0); rErr == nil && newAddr != c.addr {
					c.addr = newAddr
				}
			}
			continue
		}
		if status != 0 {
			c.mu.Unlock()
			return 0, fmt.Errorf("broker register topic status: %d", status)
		}
		c.mu.Unlock()
		return topicID, nil
	}
	return 0, fmt.Errorf("register topic after 5 attempts: %w", lastErr)
}

type ProduceBatchResult struct {
	Offset    uint64
	Committed uint32
}

func (c *Client) ProduceBatch(ctx context.Context, topic string, topicID uint16, payloads [][]byte) (ProduceBatchResult, error) {
	var zero ProduceBatchResult
	if len(payloads) == 0 {
		return zero, errors.New("broker client: empty produce batch")
	}

	var batch []byte
	for _, payload := range payloads {
		batch = protocol.AppendBatchMessage(batch, topicID, payload)
	}

	var lastErr error
	for attempt := range 5 {
		if attempt > 0 {
			time.Sleep(500 * time.Millisecond)
		}

		c.mu.Lock()
		conn, err := c.getConn()
		if err != nil {
			c.mu.Unlock()
			lastErr = err
			if c.redisURL != "" {
				if newAddr, rErr := c.resolveLeaderAddr(ctx, topic, 0); rErr == nil && newAddr != c.addr {
					c.addr = newAddr
				}
			}
			continue
		}

		seq := atomic.AddUint64(&c.nextSeq, 1)
		req := protocol.EncodeProduceBatchRequest(c.writeBuf, seq, batch)
		if c.timeout > 0 {
			_ = conn.SetDeadline(time.Now().Add(c.timeout))
		}
		if _, err := conn.Write(req); err != nil {
			_ = c.closeRawConn()
			c.mu.Unlock()
			lastErr = err
			if c.redisURL != "" {
				if newAddr, rErr := c.resolveLeaderAddr(ctx, topic, 0); rErr == nil && newAddr != c.addr {
					c.addr = newAddr
				}
			}
			continue
		}

		cmd, respSeq, respPayload, err := protocol.ReadFrameConn(conn, c.readBuf, c.lenBuf)
		if err != nil {
			_ = c.closeRawConn()
			c.mu.Unlock()
			lastErr = err
			if c.redisURL != "" {
				if newAddr, rErr := c.resolveLeaderAddr(ctx, topic, 0); rErr == nil && newAddr != c.addr {
					c.addr = newAddr
				}
			}
			continue
		}
		if cmd != protocol.CmdProduceBatchResp {
			c.mu.Unlock()
			return zero, fmt.Errorf("unexpected command response: %d", cmd)
		}
		if respSeq != seq {
			c.mu.Unlock()
			return zero, fmt.Errorf("sequence mismatch: expected %d, got %d", seq, respSeq)
		}
		status, offset, committed, err := protocol.DecodeProduceBatchResponse(respPayload)
		if err != nil {
			c.mu.Unlock()
			return zero, err
		}
		if status == 4 || status == 5 || status == 6 || status == 7 {
			_ = c.closeRawConn()
			c.mu.Unlock()
			lastErr = brokerStatusError(status)
			if c.redisURL != "" {
				if newAddr, rErr := c.resolveLeaderAddr(ctx, topic, 0); rErr == nil && newAddr != c.addr {
					c.addr = newAddr
				}
			}
			continue
		}
		if status != 0 {
			c.mu.Unlock()
			return zero, fmt.Errorf("broker produce batch status: %d", status)
		}
		c.mu.Unlock()
		return ProduceBatchResult{Offset: offset, Committed: committed}, nil
	}
	return zero, fmt.Errorf("produce batch after 5 attempts: %w", lastErr)
}

func brokerStatusError(status byte) error {
	switch status {
	case 5:
		return errors.New("stale fencing epoch")
	case 6:
		return errors.New("leader catching up")
	case 7:
		return errors.New("broker overloaded")
	default:
		return errors.New("not leader")
	}
}
