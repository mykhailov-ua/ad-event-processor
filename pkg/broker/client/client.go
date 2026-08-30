package client

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"ad-event-processor/pkg/broker/protocol"
	"ad-event-processor/pkg/netaddr"

	"github.com/redis/go-redis/v9"
)

// MessageIterator walks fetch message blob: per record length u32 (includes offset u64) + offset u64 + payload.
type MessageIterator struct {
	data          []byte
	idx           int
	count         uint32
	curr          uint32
	Offset        uint64
	Payload       []byte
	HighWatermark uint64
}

func (it *MessageIterator) Next() bool {
	if it.curr >= it.count || it.idx+12 > len(it.data) {
		return false
	}
	length := binary.BigEndian.Uint32(it.data[it.idx : it.idx+4]) // total bytes after length field (offset+payload).
	it.Offset = binary.BigEndian.Uint64(it.data[it.idx+4 : it.idx+12])
	payloadLen := int(length) - 8 // length counts 8-byte offset prefix.
	if it.idx+12+payloadLen > len(it.data) {
		return false
	}
	it.Payload = it.data[it.idx+12 : it.idx+12+payloadLen]
	it.idx += 12 + payloadLen
	it.curr++
	return true
}

type Client struct {
	addr        string
	conn        net.Conn
	mu          sync.Mutex // serializes wire I/O; one in-flight request per Client
	nextSeq     uint64
	readBuf     []byte // reused by ReadFrameConn; Fetch iterator aliases into this buffer
	writeBuf    []byte // encode scratch for Produce/Fetch/offset frames
	lenBuf      []byte
	timeout     time.Duration
	redisURL    string
	redisClient redis.Cmdable
	fetchIter   MessageIterator // single iterator per Client; not safe under concurrent Fetch
}

func NewClient(addr string, timeout time.Duration) *Client {
	return &Client{
		addr:     addr,
		timeout:  timeout,
		readBuf:  make([]byte, 1024*1024),
		writeBuf: make([]byte, 1024*1024),
		lenBuf:   make([]byte, 4),
	}
}

func (c *Client) SetRedisURL(url string) {
	c.redisURL = url
}

func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connectLocked()
}

func (c *Client) connectLocked() error {
	if c.conn != nil {
		return nil
	}

	// Lazy dial: Connect or first Produce/Fetch; netaddr handles tcp vs unix socket.
	conn, err := netaddr.DialTimeout(c.addr, c.timeout)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var err error
	if c.conn != nil {
		err = c.conn.Close()
		c.conn = nil
	}
	if c.redisClient != nil {
		if cl, ok := c.redisClient.(interface{ Close() error }); ok {
			_ = cl.Close()
		}
		c.redisClient = nil
	}
	return err
}

func (c *Client) getConn() (net.Conn, error) {
	if c.conn == nil {
		if err := c.connectLocked(); err != nil {
			return nil, err
		}
	}
	return c.conn, nil
}

func (c *Client) Produce(ctx context.Context, topic string, partition uint16, payload []byte) (uint64, error) {
	var lastErr error
	// Up to 5 attempts: conn reset + Redis leader re-resolve on retryable broker status 4-7.
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
				if newAddr, rErr := c.resolveLeaderAddr(ctx, topic, partition); rErr == nil && newAddr != c.addr {
					c.addr = newAddr
				}
			}
			continue
		}

		seq := atomic.AddUint64(&c.nextSeq, 1)
		req := protocol.EncodeProduceRequest(c.writeBuf, seq, topic, partition, payload)

		if c.timeout > 0 {
			// Per-request deadline on broker gnet conn; zero timeout = blocking read/write.
			_ = conn.SetDeadline(time.Now().Add(c.timeout))
		}

		if _, err := conn.Write(req); err != nil {
			_ = c.closeRawConn() // drop conn on syscall error; next attempt redials
			c.mu.Unlock()
			lastErr = err
			if c.redisURL != "" {
				if newAddr, rErr := c.resolveLeaderAddr(ctx, topic, partition); rErr == nil && newAddr != c.addr {
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
				if newAddr, rErr := c.resolveLeaderAddr(ctx, topic, partition); rErr == nil && newAddr != c.addr {
					c.addr = newAddr
				}
			}
			continue
		}

		if cmd != protocol.CmdProduceResp {
			c.mu.Unlock()
			return 0, fmt.Errorf("unexpected command response: %d", cmd)
		}

		if respSeq != seq {
			c.mu.Unlock()
			return 0, fmt.Errorf("sequence mismatch: expected %d, got %d", seq, respSeq)
		}

		if len(respPayload) < 9 {
			c.mu.Unlock()
			return 0, errors.New("malformed produce response payload")
		}

		status := respPayload[0]
		// Retryable broker statuses 4-7: not leader, stale epoch, catching up, overloaded.
		if status == 4 || status == 5 || status == 6 || status == 7 {
			_ = c.closeRawConn()
			c.mu.Unlock()
			switch status {
			case 5:
				lastErr = errors.New("stale fencing epoch")
			case 6:
				lastErr = errors.New("leader catching up")
			case 7:
				lastErr = errors.New("broker overloaded")
			default:
				lastErr = errors.New("not leader") // status 4
			}
			if c.redisURL != "" {
				if newAddr, rErr := c.resolveLeaderAddr(ctx, topic, partition); rErr == nil && newAddr != c.addr {
					c.addr = newAddr
				}
			}
			continue
		}

		if status != 0 {
			c.mu.Unlock()
			return 0, fmt.Errorf("broker error status: %d", status)
		}

		offsetVal := binary.BigEndian.Uint64(respPayload[1:9])
		c.mu.Unlock()
		return offsetVal, nil
	}

	return 0, fmt.Errorf("failed after 5 attempts, last error: %w", lastErr)
}

func (c *Client) Fetch(ctx context.Context, topic string, partition uint16, startOffset uint64, maxBytes uint32) (*MessageIterator, error) {
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
				if newAddr, rErr := c.resolveLeaderAddr(ctx, topic, partition); rErr == nil && newAddr != c.addr {
					c.addr = newAddr
				}
			}
			continue
		}

		seq := atomic.AddUint64(&c.nextSeq, 1)
		req := protocol.EncodeFetchRequest(c.writeBuf, seq, topic, partition, startOffset, maxBytes)

		if c.timeout > 0 {
			// Per-request deadline on broker gnet conn; zero timeout = blocking read/write.
			_ = conn.SetDeadline(time.Now().Add(c.timeout))
		}

		if _, err := conn.Write(req); err != nil {
			_ = c.closeRawConn() // drop conn on syscall error; next attempt redials
			c.mu.Unlock()
			lastErr = err
			if c.redisURL != "" {
				if newAddr, rErr := c.resolveLeaderAddr(ctx, topic, partition); rErr == nil && newAddr != c.addr {
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
				if newAddr, rErr := c.resolveLeaderAddr(ctx, topic, partition); rErr == nil && newAddr != c.addr {
					c.addr = newAddr
				}
			}
			continue
		}

		if cmd != protocol.CmdFetchResp {
			c.mu.Unlock()
			return nil, fmt.Errorf("unexpected command response: %d", cmd)
		}

		if respSeq != seq {
			c.mu.Unlock()
			return nil, fmt.Errorf("sequence mismatch: expected %d, got %d", seq, respSeq)
		}

		if len(respPayload) < protocol.FetchRespMetaLen {
			c.mu.Unlock()
			return nil, errors.New("malformed fetch response payload")
		}

		status := respPayload[0]
		// Retryable broker statuses 4-7: not leader, stale epoch, catching up, overloaded.
		if status == 4 || status == 5 || status == 6 || status == 7 {
			_ = c.closeRawConn()
			c.mu.Unlock()
			switch status {
			case 5:
				lastErr = errors.New("stale fencing epoch")
			case 6:
				lastErr = errors.New("leader catching up")
			case 7:
				lastErr = errors.New("broker overloaded")
			default:
				lastErr = errors.New("not leader") // status 4
			}
			if c.redisURL != "" {
				if newAddr, rErr := c.resolveLeaderAddr(ctx, topic, partition); rErr == nil && newAddr != c.addr {
					c.addr = newAddr
				}
			}
			continue
		}

		if status != 0 {
			c.mu.Unlock()
			return nil, fmt.Errorf("broker error status: %d", status)
		}

		msgCount := binary.BigEndian.Uint32(respPayload[1:5])
		highWatermark := binary.BigEndian.Uint64(respPayload[5:13])
		messagesData := respPayload[protocol.FetchRespMetaLen:]

		// Iterator views messagesData inside readBuf; invalid after next Fetch or Close.
		c.fetchIter.data = messagesData
		c.fetchIter.idx = 0
		c.fetchIter.count = msgCount
		c.fetchIter.curr = 0
		c.fetchIter.Offset = 0
		c.fetchIter.Payload = nil
		c.fetchIter.HighWatermark = highWatermark

		c.mu.Unlock()
		return &c.fetchIter, nil
	}

	return nil, fmt.Errorf("failed after 5 attempts, last error: %w", lastErr)
}

func (c *Client) CommitOffset(topic string, partition uint16, group string, offset uint64) (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, err := c.getConn()
	if err != nil {
		return 0, err
	}

	seq := atomic.AddUint64(&c.nextSeq, 1)
	req := protocol.EncodeCommitOffsetRequest(c.writeBuf, seq, topic, partition, group, offset)

	if c.timeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(c.timeout))
	}
	if _, err := conn.Write(req); err != nil {
		_ = c.closeRawConn()
		return 0, err
	}

	cmd, respSeq, respPayload, err := protocol.ReadFrameConn(conn, c.readBuf, c.lenBuf)
	if err != nil {
		_ = c.closeRawConn()
		return 0, err
	}
	if cmd != protocol.CmdCommitOffsetResp {
		return 0, fmt.Errorf("unexpected command response: %d", cmd)
	}
	if respSeq != seq {
		return 0, fmt.Errorf("sequence mismatch: expected %d, got %d", seq, respSeq)
	}

	status, stored, err := protocol.DecodeCommitOffsetResponse(respPayload)
	if err != nil {
		return 0, err
	}
	if status != 0 {
		return 0, fmt.Errorf("broker commit offset status: %d", status)
	}
	return stored, nil
}

func (c *Client) CommittedOffset(topic string, partition uint16, group string) (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, err := c.getConn()
	if err != nil {
		return 0, err
	}

	seq := atomic.AddUint64(&c.nextSeq, 1)
	req := protocol.EncodeCommittedOffsetRequest(c.writeBuf, seq, topic, partition, group)

	if c.timeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(c.timeout))
	}
	if _, err := conn.Write(req); err != nil {
		_ = c.closeRawConn()
		return 0, err
	}

	cmd, respSeq, respPayload, err := protocol.ReadFrameConn(conn, c.readBuf, c.lenBuf)
	if err != nil {
		_ = c.closeRawConn()
		return 0, err
	}
	if cmd != protocol.CmdCommittedOffsetResp {
		return 0, fmt.Errorf("unexpected command response: %d", cmd)
	}
	if respSeq != seq {
		return 0, fmt.Errorf("sequence mismatch: expected %d, got %d", seq, respSeq)
	}

	status, offset, err := protocol.DecodeCommittedOffsetResponse(respPayload)
	if err != nil {
		return 0, err
	}
	// OffsetStatusStoreUnavailable (2): fail-open read path returns offset 0 without error.
	if status == protocol.OffsetStatusStoreUnavailable {
		return 0, nil
	}
	if status != protocol.OffsetStatusOK {
		return 0, fmt.Errorf("broker committed offset status: %d", status)
	}
	return offset, nil
}

func (c *Client) closeRawConn() error {
	// Called under mu after wire failure; nils conn so getConn redials on next RPC.
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

func (c *Client) resolveLeaderAddr(ctx context.Context, topic string, partition uint16) (string, error) {
	if c.redisURL == "" {
		return "", errors.New("redis URL not set")
	}
	if c.redisClient == nil {
		redisClient, err := netaddr.ParseRedisURL(c.redisURL, "")
		if err != nil {
			return "", err
		}
		c.redisClient = redisClient
	}
	tpKey := protocol.TopicPartitionID(topic, partition)
	lookupCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	// Redis leader election keys: topics:{tp}:leader -> broker id; brokers:{id} -> dial address.
	leaderID, err := c.redisClient.Get(lookupCtx, "ad_event_processor:topics:"+tpKey+":leader").Result()
	if err != nil {
		return "", err
	}
	return c.redisClient.Get(lookupCtx, "ad_event_processor:brokers:"+leaderID).Result()
}
