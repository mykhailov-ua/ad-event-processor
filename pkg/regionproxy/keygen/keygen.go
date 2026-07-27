// Package keygen runs a pinned-thread worker that derives factor_u for WAL records.
package keygen

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"espx/pkg/dedupkey"
	"espx/pkg/regionproxy/wal"
)

// Config tunes the KeyGen polling loop.
type Config struct {
	RegionCode   uint8
	NodeID       string
	PollInterval time.Duration
	BatchSize    int
}

// KeyGen derives factor_u for appended WAL records and sets WalFlagDedupReady.
type KeyGen struct {
	wal     *wal.WAL
	cfg     Config
	closeCh chan struct{}
	wg      sync.WaitGroup

	processed atomic.Uint64
}

// New builds a KeyGen worker for w.
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

// Start launches the pinned KeyGen goroutine.
func (k *KeyGen) Start() {
	k.wg.Add(1)
	go k.loop()
}

// Stop waits for the KeyGen goroutine to exit.
func (k *KeyGen) Stop() {
	close(k.closeCh)
	k.wg.Wait()
}

// Processed returns the number of records marked WalFlagDedupReady.
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
