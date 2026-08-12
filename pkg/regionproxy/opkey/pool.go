package opkey

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bidshard/ad-event-processor/pkg/regionproxy/wal"
)

type Config struct {
	NodeID       string
	QueueSize    uint64
	Watermark    int64
	PollInterval time.Duration
	BatchSize    int
}

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

func (p *Pool) Start() {
	p.wg.Add(1)
	go p.loop()
}

func (p *Pool) Stop() {
	close(p.closeCh)
	p.wg.Wait()
}

func (p *Pool) Depth() int64 {
	return p.queue.Depth()
}

func (p *Pool) Watermark() int64 {
	return p.watermark
}

func (p *Pool) ShouldShed() bool {
	return p.Depth() > p.watermark
}

func (p *Pool) Enqueued() uint64 {
	return p.enqueued.Load()
}

func (p *Pool) ShedTotal() uint64 {
	return p.shedTotal.Load()
}

func (p *Pool) Release(slot *Slot) {
	if slot == nil {
		return
	}
	atomic.StoreUint32(&slot.flags, 0)
	slot.Seq = 0
	p.slotPool.Put(slot)
}

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
