// Package uplink forwards region-proxy WAL batches to global D3 ingest.
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

	"espx/pkg/regionproxy/opkey"
	"espx/pkg/regionproxy/wal"

	"github.com/google/uuid"
)

// Config tunes the uplink worker loop.
type Config struct {
	RegionCode     uint8
	NodeID         string
	SourceEpoch    uint32
	GlobalURL      string
	APIKey         string
	PollInterval   time.Duration
	BatchSize      int
	HTTPTimeout    time.Duration
	BatchCommitter *opkey.BatchCommitter
}

// Worker dequeues OpKey slots and posts them to global management ingest.
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

// New builds an uplink worker.
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

// Start launches the pinned uplink goroutine.
func (u *Worker) Start() {
	u.wg.Add(1)
	go u.loop()
}

// Stop waits for the uplink goroutine to exit.
func (u *Worker) Stop() {
	close(u.closeCh)
	u.wg.Wait()
}

// Forwarded returns batches claimed for uplink.
func (u *Worker) Forwarded() uint64 {
	return u.forwarded.Load()
}

// Acked returns batches acknowledged by global ingest.
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
	claimed, err := u.wal.TryClaimForward(slot.Seq)
	if err != nil || !claimed {
		return
	}
	u.forwarded.Add(1)

	payload, err := u.wal.ReadRecordPayload(slot.Seq)
	if err != nil {
		return
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
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), u.cfg.HTTPTimeout)
	defer cancel()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u.cfg.GlobalURL, bytes.NewReader(body))
	if err != nil {
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if u.cfg.APIKey != "" {
		httpReq.Header.Set("X-Admin-API-Key", u.cfg.APIKey)
	}

	resp, err := u.client.Do(httpReq)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return
	}
	if err := u.wal.MarkRemoteAcked(slot.Seq); err == nil {
		u.acked.Add(1)
	}
}

// ForwardOnce posts one slot synchronously (tests).
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
