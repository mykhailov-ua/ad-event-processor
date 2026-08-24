package uplink

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"ad-event-processor/pkg/regionproxy/opkey"
	"ad-event-processor/pkg/regionproxy/wal"

	"github.com/google/uuid"
)

type Config struct {
	RegionCode          uint8
	NodeID              string
	SourceEpoch         uint32
	GlobalURL           string
	APIKey              string
	PollInterval        time.Duration
	BatchSize           int
	HTTPTimeout         time.Duration
	ForwardMaxAttempts  int
	ForwardRetryBackoff time.Duration
	BatchCommitter      *opkey.BatchCommitter
}

type Worker struct {
	wal    *wal.WAL
	pool   *opkey.Pool
	client *http.Client
	cfg    Config

	closeCh    chan struct{}
	wg         sync.WaitGroup
	forwarded  atomic.Uint64
	acked      atomic.Uint64
	quorumHeld atomic.Uint64
}

func New(w *wal.WAL, pool *opkey.Pool, cfg Config) *Worker {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = time.Millisecond
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 32
	}
	if cfg.HTTPTimeout <= 0 {
		cfg.HTTPTimeout = 5 * time.Second
	}
	if cfg.ForwardMaxAttempts <= 0 {
		cfg.ForwardMaxAttempts = 3
	}
	if cfg.ForwardRetryBackoff <= 0 {
		cfg.ForwardRetryBackoff = 50 * time.Millisecond
	}
	return &Worker{
		wal:  w,
		pool: pool,
		cfg:  cfg,
		client: &http.Client{
			Timeout: cfg.HTTPTimeout,
		},
		closeCh: make(chan struct{}),
	}
}

func (u *Worker) Start() {
	u.wg.Add(1)
	go u.loop()
}

func (u *Worker) Stop() {
	close(u.closeCh)
	u.wg.Wait()
}

func (u *Worker) Forwarded() uint64 {
	return u.forwarded.Load()
}

func (u *Worker) Acked() uint64 {
	return u.acked.Load()
}

type ingestRequest struct {
	RegionCode  uint8  `json:"region_code"`
	NodeID      string `json:"node_id"`
	SourceEpoch uint32 `json:"source_epoch"`
	Seq         uint64 `json:"seq"`
	FactorU     string `json:"factor_u"`
	Payload     []byte `json:"payload"`
	OpID        string `json:"op_id,omitempty"`
}

func (u *Worker) loop() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer u.wg.Done()

	ticker := time.NewTicker(u.cfg.PollInterval)
	defer ticker.Stop()

	for {
		u.drainBatch()

		select {
		case <-u.closeCh:
			return
		case <-ticker.C:
		}
	}
}

func (u *Worker) drainBatch() {
	if u.pool == nil || u.cfg.GlobalURL == "" {
		return
	}
	for i := 0; i < u.cfg.BatchSize; i++ {
		slot, ok := u.pool.Dequeue()
		if !ok {
			return
		}
		if u.cfg.BatchCommitter != nil {
			ready, err := u.cfg.BatchCommitter.PrepareForward(context.Background(), slot)
			if err != nil || !ready {
				u.quorumHeld.Add(1)
				u.pool.Release(slot)
				continue
			}
		}
		u.forwardSlot(slot)
		if u.cfg.BatchCommitter != nil {
			u.cfg.BatchCommitter.Complete(context.Background(), slot)
		}
		u.pool.Release(slot)
	}
}

func (u *Worker) forwardSlot(slot *opkey.Slot) {
	if slot == nil {
		return
	}
	maxAttempts := u.cfg.ForwardMaxAttempts
	backoff := u.cfg.ForwardRetryBackoff

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			time.Sleep(backoff * time.Duration(attempt-1))
		}
		claimed, err := u.wal.TryClaimForward(slot.Seq)
		if err != nil {
			continue
		}
		if !claimed {
			return
		}
		u.forwarded.Add(1)

		if err := u.forwardAttempt(slot); err != nil {
			_ = u.wal.UnclaimForward(slot.Seq)
			if attempt == maxAttempts {
				return
			}
			continue
		}
		if err := u.wal.MarkRemoteAcked(slot.Seq); err == nil {
			u.acked.Add(1)
		}
		return
	}
}

func (u *Worker) forwardAttempt(slot *opkey.Slot) error {
	payload, err := u.wal.ReadRecordPayload(slot.Seq)
	if err != nil {
		return err
	}
	var factor uuid.UUID
	copy(factor[:], slot.FactorU[:])
	var opID uuid.UUID
	copy(opID[:], slot.OpID[:])

	reqBody := ingestRequest{
		RegionCode:  u.cfg.RegionCode,
		NodeID:      u.cfg.NodeID,
		SourceEpoch: u.cfg.SourceEpoch,
		Seq:         slot.Seq,
		FactorU:     factor.String(),
		Payload:     payload,
		OpID:        opID.String(),
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), u.cfg.HTTPTimeout)
	defer cancel()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u.cfg.GlobalURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if u.cfg.APIKey != "" {
		httpReq.Header.Set("X-Admin-API-Key", u.cfg.APIKey)
	}

	resp, err := u.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("region proxy uplink seq=%d: http %d", slot.Seq, resp.StatusCode)
	}
	return nil
}

func (u *Worker) ForwardOnce(slot *opkey.Slot) error {
	if slot == nil {
		return fmt.Errorf("region proxy uplink: nil slot")
	}
	before := u.acked.Load()
	u.forwardSlot(slot)
	if u.acked.Load() == before {
		return fmt.Errorf("region proxy uplink seq=%d: global ingest failed", slot.Seq)
	}
	return nil
}
