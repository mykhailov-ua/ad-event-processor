package opkey

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"espx/pkg/regionproxy/wal"
)

// Config tunes the OpKeyPool ring and WAL drain loop.
type Config struct {
	NodeID       string
	QueueSize    uint64
	Watermark    int64
	PollInterval time.Duration
	BatchSize    int
}

// Pool derives operation keys from WAL dedup-ready records on a pinned thread.
type Pool struct {
	wal        *wal.WAL
	cfg        Config
	queue      *MPSCQueue
	ids        idGen
	watermark  int64
	closeCh    chan struct{}
	wg         sync.WaitGroup
	slotPool   sync.Pool
	lastWalSeq atomic.Uint64
	enqueued   atomic.Uint64
	shedTotal  atomic.Uint64
}

// New builds an OpKeyPool for w.
func New(w *wal.WAL, cfg Config) *Pool {
	if cfg.QueueSize == 0 || (cfg.QueueSize&(cfg.QueueSize-1)) != 0 {
		cfg.QueueSize = 4096
	}
	if cfg.Watermark <= 0 {
		cfg.Watermark = 1000
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = time.Millisecond
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 64
	}
	p := &Pool{
		wal:       w,
		cfg:       cfg,
		queue:     NewMPSCQueue(cfg.QueueSize),
		ids:       newIDGen(cfg.NodeID),
		watermark: cfg.Watermark,
		closeCh:   make(chan struct{}),
	}
	p.slotPool.New = func() any { return &Slot{} }
	return p
}

// Start launches the pinned WAL drain goroutine.
func (p *Pool) Start() {
	p.wg.Add(1)
	go p.loop()
}

// Stop waits for the drain goroutine to exit.
func (p *Pool) Stop() {
	close(p.closeCh)
	p.wg.Wait()
}

// Depth returns the current MPSC queue depth.
func (p *Pool) Depth() int64 {
	return p.queue.Depth()
}

// Watermark returns the shed threshold.
func (p *Pool) Watermark() int64 {
	return p.watermark
}

// ShouldShed reports whether ingress should reject new work due to pool pressure.
func (p *Pool) ShouldShed() bool {
	return p.Depth() > p.watermark
}

// Enqueued returns slots successfully pushed to the ring.
func (p *Pool) Enqueued() uint64 {
	return p.enqueued.Load()
}

// ShedTotal returns ingress shed events due to pool pressure.
func (p *Pool) ShedTotal() uint64 {
	return p.shedTotal.Load()
}

// Release returns a dequeued slot to the pool.
func (p *Pool) Release(slot *Slot) {
	if slot == nil {
		return
	}
	atomic.StoreUint32(&slot.flags, 0)
	slot.Seq = 0
	p.slotPool.Put(slot)
}

// TryEnqueue assigns op_id, sets OpKeyFlagDerived, and pushes slot when depth <= watermark.
func (p *Pool) TryEnqueue(seq uint64, factorU [32]byte) ([16]byte, bool) {
	if p.Depth() > p.watermark {
		p.recordShed()
		return [16]byte{}, false
	}
	slot := p.slotPool.Get().(*Slot)
	slot.Seq = seq
	slot.FactorU = factorU
	p.ids.next(&slot.OpID)
	var opID [16]byte
	copy(opID[:], slot.OpID[:])
	slot.setDerived()
	if !p.queue.Push(slot) {
		p.slotPool.Put(slot)
		return [16]byte{}, false
	}
	p.enqueued.Add(1)
	setDepth(float64(p.queue.Depth()))
	return opID, true
}

// Dequeue pops one derived slot for booking/uplink workers.
func (p *Pool) Dequeue() (*Slot, bool) {
	slot, ok := p.queue.Pop()
	if ok {
		setDepth(float64(p.queue.Depth()))
	}
	return slot, ok
}

func (p *Pool) recordShed() {
	p.shedTotal.Add(1)
	incIngressShed()
}

func (p *Pool) loop() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer p.wg.Done()

	ticker := time.NewTicker(p.cfg.PollInterval)
	defer ticker.Stop()

	for {
		p.drainWAL()

		select {
		case <-p.closeCh:
			return
		case <-ticker.C:
		}
	}
}

func (p *Pool) drainWAL() {
	from := p.lastWalSeq.Load()
	processed := p.wal.ScanDedupReady(from, p.cfg.BatchSize, func(seq uint64, factorU [32]byte) bool {
		if _, ok := p.TryEnqueue(seq, factorU); !ok {
			return false
		}
		p.lastWalSeq.Store(seq + 1)
		return true
	})
	if processed > 0 {
		setDepth(float64(p.queue.Depth()))
	}
}
