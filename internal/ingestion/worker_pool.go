package ingestion

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

func finishOffloadCtx(ctx *connContext) {
	if ctx == nil {
		return
	}
	if ctx.offloadOnEnter != nil {
		ctx.offloadOnEnter()
	}
	if ctx.offloadBlock != nil {
		<-ctx.offloadBlock
	}
	if ctx.offloadWG != nil {
		ctx.offloadWG.Done()
	}
}

type Slot struct {
	ctx   *connContext
	ready atomic.Bool
}

type MPSCQueue struct {
	_     [8]uint64
	write uint64
	_     [8]uint64
	read  uint64
	_     [8]uint64
	mask  uint64
	ring  []Slot
}

func NewMPSCQueue(size uint64) *MPSCQueue {
	if size == 0 || (size&(size-1)) != 0 {
		size = 4096
	}
	return &MPSCQueue{
		mask: size - 1,
		ring: make([]Slot, size),
	}
}

func (q *MPSCQueue) PushCtx(ctx *connContext) bool {
	for {
		w := atomic.LoadUint64(&q.write)
		r := atomic.LoadUint64(&q.read)
		if w-r >= q.mask+1 {
			return false
		}
		if atomic.CompareAndSwapUint64(&q.write, w, w+1) {
			slot := &q.ring[w&q.mask]
			slot.ctx = ctx
			slot.ready.Store(true)
			return true
		}
	}
}

func (q *MPSCQueue) PopCtx() (*connContext, bool) {
	r := atomic.LoadUint64(&q.read)
	w := atomic.LoadUint64(&q.write)
	if r == w {
		return nil, false
	}
	slot := &q.ring[r&q.mask]
	if !slot.ready.Load() {
		return nil, false
	}
	ctx := slot.ctx
	slot.ctx = nil
	slot.ready.Store(false)
	atomic.StoreUint64(&q.read, r+1)
	return ctx, true
}

type Worker struct {
	pool  *PinnedWorkerPool
	id    int
	queue *MPSCQueue
	arena workerArena
}

func (w *Worker) start() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	run := func(ctx *connContext) {
		if h := w.pool.handler; h != nil {
			h.runOffloadedRequest(w.id, ctx)
		} else {
			finishOffloadCtx(ctx)
		}
		w.pool.wg.Done()
	}

	for {
		ctx, ok := w.queue.PopCtx()
		if ok {
			run(ctx)
			continue
		}

		if atomic.LoadInt32(&w.pool.closed) == 1 {
			if ctx, ok = w.queue.PopCtx(); !ok {
				break
			}
			run(ctx)
			continue
		}

		spin := 0
		for {
			if ctx, ok = w.queue.PopCtx(); ok {
				run(ctx)
				break
			}
			if atomic.LoadInt32(&w.pool.closed) == 1 {
				break
			}
			if spin < 10 {
				spin++
			} else if spin < 20 {
				spin++
				runtime.Gosched()
			} else {
				time.Sleep(time.Microsecond)
			}
		}
	}
}

type PinnedWorkerPool struct {
	workers []*Worker
	handler *AdsPacketHandler
	round   uint64
	wg      sync.WaitGroup
	closed  int32
}

func NewPinnedWorkerPool(size int, queueSize int) *PinnedWorkerPool {
	if size <= 0 {
		size = runtime.GOMAXPROCS(0)
	}
	qSize := uint64(queueSize)
	if qSize == 0 || (qSize&(qSize-1)) != 0 {
		qSize = 4096
	}

	p := &PinnedWorkerPool{
		workers: make([]*Worker, size),
	}
	for i := 0; i < size; i++ {
		w := &Worker{
			pool:  p,
			id:    i,
			queue: NewMPSCQueue(qSize),
		}
		p.workers[i] = w
		go w.start()
	}
	return p
}

func (p *PinnedWorkerPool) SubmitOffload(ctx *connContext, src []byte) bool {
	if atomic.LoadInt32(&p.closed) == 1 || ctx == nil {
		return false
	}
	p.wg.Add(1)

	idx := atomic.AddUint64(&p.round, 1) % uint64(len(p.workers))
	for i := 0; i < len(p.workers); i++ {
		widx := (idx + uint64(i)) % uint64(len(p.workers))
		w := p.workers[widx]
		if len(src) > 0 && ctx.offloadReqSlice == nil && ctx.offloadReqBuf == nil {
			if slot, buf, release, ok := w.arena.acquire(len(src)); ok {
				copy(buf, src)
				ctx.offloadReqSlice = buf
				ctx.offloadReqLen = len(src)
				ctx.offloadArenaWorker = int(widx)
				ctx.offloadArenaSlot = slot
				ctx.offloadRelease = release
			}
		}
		if w.queue.PushCtx(ctx) {
			return true
		}
		if ctx.offloadRelease != nil {
			ctx.offloadRelease()
			ctx.offloadRelease = nil
			ctx.offloadReqSlice = nil
			ctx.offloadArenaWorker = 0
			ctx.offloadArenaSlot = 0
		}
	}

	if len(src) > 0 && ctx.offloadReqSlice == nil && ctx.offloadReqBuf == nil {
		if len(src) > maxPoolObjectSize {
			heap := make([]byte, len(src))
			copy(heap, src)
			ctx.offloadReqSlice = heap
			ctx.offloadReqLen = len(src)
		} else {
			reqBufPtr := requestBufferPool.Get().(*[]byte)
			reqBytes := *reqBufPtr
			if cap(reqBytes) < len(src) {
				reqBytes = make([]byte, len(src))
				*reqBufPtr = reqBytes
			} else {
				reqBytes = reqBytes[:len(src)]
			}
			copy(reqBytes, src)
			ctx.offloadReqBuf = reqBufPtr
			ctx.offloadReqLen = len(src)
		}
		for i := 0; i < len(p.workers); i++ {
			widx := (idx + uint64(i)) % uint64(len(p.workers))
			if p.workers[widx].queue.PushCtx(ctx) {
				return true
			}
		}
		if ctx.offloadReqBuf != nil {
			putRequestBuffer(ctx.offloadReqBuf)
			ctx.offloadReqBuf = nil
		}
		ctx.offloadReqSlice = nil
		ctx.offloadReqLen = 0
	}

	p.wg.Done()
	return false
}

func (p *PinnedWorkerPool) Shutdown() {
	if atomic.CompareAndSwapInt32(&p.closed, 0, 1) {
		p.wg.Wait()
	}
}
