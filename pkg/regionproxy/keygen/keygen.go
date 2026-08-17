// Package keygen implements regionproxy keygen helpers.
package keygen

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bidshard/ad-event-processor/pkg/dedupkey"
	"github.com/bidshard/ad-event-processor/pkg/regionproxy/wal"
)

type Config struct {
	RegionCode   uint8
	NodeID       string
	PollInterval time.Duration
	BatchSize    int
}

type KeyGen struct {
	wal     *wal.WAL
	cfg     Config
	closeCh chan struct{}
	wg      sync.WaitGroup

	processed atomic.Uint64
}

func New(w *wal.WAL, cfg Config) *KeyGen {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = time.Millisecond
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 64
	}
	return &KeyGen{
		wal:     w,
		cfg:     cfg,
		closeCh: make(chan struct{}),
	}
}

func (k *KeyGen) Start() {
	k.wg.Add(1)
	go k.loop()
}

func (k *KeyGen) Stop() {
	close(k.closeCh)
	k.wg.Wait()
}

func (k *KeyGen) Processed() uint64 {
	return k.processed.Load()
}

func (k *KeyGen) loop() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer k.wg.Done()

	ticker := time.NewTicker(k.cfg.PollInterval)
	defer ticker.Stop()

	var canonicalBuf [wal.MaxPayloadSize + 64]byte

	for {
		k.drainBatch(&canonicalBuf)

		select {
		case <-k.closeCh:
			return
		case <-ticker.C:
		}
	}
}

func (k *KeyGen) drainBatch(canonicalBuf *[wal.MaxPayloadSize + 64]byte) {
	depth := k.wal.KeyGenQueueDepth()
	setQueueDepth(float64(depth))

	n, err := k.wal.ProcessPendingKeyGen(k.cfg.BatchSize, func(seq uint64, payload []byte) ([32]byte, error) {
		start := time.Now()
		canon := dedupkey.WriteCanonicalProxyBatchPayload(canonicalBuf[:0], seq, payload)
		id := dedupkey.FactorU(canon)
		observeLag(time.Since(start).Seconds())
		var out [32]byte
		copy(out[:], id[:])
		return out, nil
	})
	if err != nil {
		_ = err
	}
	if n > 0 {
		k.processed.Add(uint64(n))
		incRate(float64(n))
	}
}
