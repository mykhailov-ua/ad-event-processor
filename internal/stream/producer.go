package stream

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/ingest/pb"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/stream/codec"
	"ad-event-processor/internal/telemetry"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

var ErrQueueFull = errors.New("producer queue full")

type PregeneratedID struct {
	UUID   uuid.UUID
	String string
}

type IDRingBuffer struct {
	buffer []PregeneratedID
	size   uint32
	mask   uint32
	_      [56]byte
	head   uint32
	_      [56]byte
	tail   uint32
}

func NewIDRingBuffer(size uint32) *IDRingBuffer {
	if size == 0 {
		size = 4096
	}
	p := uint32(1)
	for p < size {
		p <<= 1
	}
	size = p

	rb := &IDRingBuffer{
		buffer: make([]PregeneratedID, size),
		size:   size,
		mask:   size - 1,
	}

	for i := uint32(0); i < size; i++ {
		id := filter.NewFastUUID()
		rb.buffer[i] = PregeneratedID{
			UUID:   id,
			String: id.String(),
		}
	}
	rb.tail = size

	go rb.refillWorker()

	return rb
}

func (rb *IDRingBuffer) Next() PregeneratedID {
	for {
		h := atomic.LoadUint32(&rb.head)
		t := atomic.LoadUint32(&rb.tail)
		if h == t {
			id := filter.NewFastUUID()
			return PregeneratedID{
				UUID:   id,
				String: id.String(),
			}
		}

		item := rb.buffer[h&rb.mask]

		if atomic.CompareAndSwapUint32(&rb.head, h, h+1) {
			return item
		}
	}
}

func (rb *IDRingBuffer) refillWorker() {
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		h := atomic.LoadUint32(&rb.head)
		t := atomic.LoadUint32(&rb.tail)

		available := t - h
		if available < rb.size/2 {
			toFill := rb.size - available
			for i := uint32(0); i < toFill; i++ {
				id := filter.NewFastUUID()
				idx := (t + i) & rb.mask
				rb.buffer[idx] = PregeneratedID{
					UUID:   id,
					String: id.String(),
				}
			}
			atomic.StoreUint32(&rb.tail, t+toFill)
		}
	}
}

var globalIDRingBuffer = NewIDRingBuffer(16384)

const defaultStreamProducerQueueCap = 50000

const defaultStreamProducerMaxBatch = 500

const defaultStreamProducerFlushWait = 20 * time.Millisecond

type StreamProducerConfig struct {
	QueueCap     int
	MaxBatchSize int
	FlushWait    time.Duration
}

type StreamProducer struct {
	redisClient   redis.UniversalClient
	streamName    string
	maxStreamLen  int64
	writeTimeout  time.Duration
	queue         chan *[]byte
	queueCap      uint32
	queueDepth    atomic.Uint32
	queueReserved atomic.Uint32
	maxBatchSize  int
	flushWait     time.Duration
	flushChan     chan chan struct{}
	closeCh       chan struct{}
	wg            sync.WaitGroup
}

func NewStreamProducer(
	redisClient redis.UniversalClient,
	streamName string,
	maxStreamLen int,
	writeTimeout time.Duration,
	tuning ...StreamProducerConfig,
) *StreamProducer {
	queueCap := defaultStreamProducerQueueCap
	maxBatchSize := defaultStreamProducerMaxBatch
	flushWait := defaultStreamProducerFlushWait
	if len(tuning) > 0 {
		if tuning[0].QueueCap > 0 {
			queueCap = tuning[0].QueueCap
		}
		if tuning[0].MaxBatchSize > 0 {
			maxBatchSize = tuning[0].MaxBatchSize
		}
		if tuning[0].FlushWait > 0 {
			flushWait = tuning[0].FlushWait
		}
	}
	p := &StreamProducer{
		redisClient:  redisClient,
		streamName:   streamName,
		maxStreamLen: int64(maxStreamLen),
		writeTimeout: writeTimeout,
		queue:        make(chan *[]byte, queueCap),
		queueCap:     uint32(queueCap),
		maxBatchSize: maxBatchSize,
		flushWait:    flushWait,
		flushChan:    make(chan chan struct{}),
		closeCh:      make(chan struct{}),
	}
	p.wg.Add(1)
	go p.worker()
	return p
}

func (p *StreamProducer) admissionLimit(admissionPct int) uint32 {
	if admissionPct <= 0 {
		return p.queueCap
	}
	if admissionPct > 100 {
		admissionPct = 100
	}
	limit := uint64(p.queueCap) * uint64(admissionPct) / 100
	if limit == 0 {
		return 1
	}
	return uint32(limit)
}

func (p *StreamProducer) occupied() uint32 {
	return p.queueDepth.Load() + p.queueReserved.Load()
}

func (p *StreamProducer) TryReserve(admissionPct int) bool {
	limit := p.admissionLimit(admissionPct)
	for {
		occupied := p.occupied()
		if occupied >= limit {
			return false
		}
		r := p.queueReserved.Load()
		if !p.queueReserved.CompareAndSwap(r, r+1) {
			continue
		}
		if p.occupied() > limit {
			p.ReleaseReserve()
			return false
		}
		return true
	}
}

func (p *StreamProducer) ReleaseReserve() {
	p.queueReserved.Add(^uint32(0))
}

func (p *StreamProducer) Process(evt *domain.Event) error {
	return p.process(evt, false)
}

func (p *StreamProducer) ProcessReserved(evt *domain.Event) error {
	return p.process(evt, true)
}

func (p *StreamProducer) process(evt *domain.Event, reserved bool) error {
	if reserved {
		defer p.ReleaseReserve()
	}
	if evt.ClickID == "" {
		evt.ClickID = globalIDRingBuffer.Next().String
	}

	pbEvt := codec.StreamEventPool.Get().(*pb.AdStreamEvent)
	pbEvt.ClickId = UnsafeBytes(evt.ClickID)
	pbEvt.CampaignId = evt.CampaignID[:]
	pbEvt.EventType = UnsafeBytes(evt.Type)
	pbEvt.Payload = evt.Payload
	pbEvt.Ip = UnsafeBytes(evt.IP)
	pbEvt.Ua = UnsafeBytes(evt.UA)
	pbEvt.UserId = UnsafeBytes(evt.UserID)
	pbEvt.CreatedAtUnix = evt.CreatedAt.Unix()
	pbEvt.FraudScore = evt.FraudScore
	pbEvt.FraudReason = UnsafeBytes(evt.FraudReason)
	pbEvt.SilentRejectEvent = evt.SilentRejectEvent
	pbEvt.ReviewRoutedEvent = evt.ReviewRoutedEvent

	size := pbEvt.SizeVT()
	bufPtr := codec.ByteBufPool.Get().(*[]byte)
	buf := *bufPtr
	if cap(buf) < size {
		buf = make([]byte, size)
	} else {
		buf = buf[:size]
	}

	n, err := pbEvt.MarshalToSizedBufferVT(buf)
	if err != nil {
		ClearAdStreamEvent(pbEvt)
		codec.StreamEventPool.Put(pbEvt)
		*bufPtr = buf
		codec.ByteBufPool.Put(bufPtr)
		metrics.EventsDropped.Inc()
		telemetry.RecordRejected()
		return err
	}
	data := buf[:n]
	*bufPtr = data

	ClearAdStreamEvent(pbEvt)
	codec.StreamEventPool.Put(pbEvt)

	select {
	case p.queue <- bufPtr:
		p.queueDepth.Add(1)
		return nil
	default:
		*bufPtr = buf[:cap(buf)]
		codec.ByteBufPool.Put(bufPtr)
		metrics.EventsDropped.Inc()
		telemetry.RecordRejected()
		return ErrQueueFull
	}
}

func (p *StreamProducer) QueueDepth() int {
	return int(p.queueDepth.Load())
}

func (p *StreamProducer) QueueCapacity() int {
	return int(p.queueCap)
}

func (p *StreamProducer) QueuePressurePct() int {
	queueCapLimit := int(p.queueCap)
	if queueCapLimit == 0 {
		return 0
	}
	depth := int(p.occupied())
	return depth * 100 / queueCapLimit
}

func NewStreamProducerQueueForTest(queueCap, fill int) *StreamProducer {
	if queueCap <= 0 {
		queueCap = 1
	}
	if fill < 0 {
		fill = 0
	}
	if fill > queueCap {
		fill = queueCap
	}
	p := &StreamProducer{
		queue:    make(chan *[]byte, queueCap),
		queueCap: uint32(queueCap),
	}
	for range fill {
		buf := make([]byte, 1, 4)
		item := &buf
		p.queue <- item
		p.queueDepth.Add(1)
	}
	return p
}

func (p *StreamProducer) dequeueOne() {
	p.queueDepth.Add(^uint32(0))
}

func (p *StreamProducer) Close() {
	select {
	case <-p.closeCh:
		return
	default:
		close(p.closeCh)
		p.wg.Wait()
	}
}

func (p *StreamProducer) Flush() {
	done := make(chan struct{})
	select {
	case p.flushChan <- done:
		<-done
	case <-p.closeCh:
	}
}

func (p *StreamProducer) worker() {
	defer p.wg.Done()

	maxBatchSize := p.maxBatchSize
	if maxBatchSize <= 0 {
		maxBatchSize = defaultStreamProducerMaxBatch
	}
	maxFlushWait := p.flushWait
	if maxFlushWait <= 0 {
		maxFlushWait = defaultStreamProducerFlushWait
	}

	batch := make([]*[]byte, 0, maxBatchSize)
	ticker := time.NewTicker(maxFlushWait)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		p.flushBatch(batch)
		batch = batch[:0]
	}

	for {
		select {
		case <-p.closeCh:
			for {
				select {
				case item := <-p.queue:
					p.dequeueOne()
					batch = append(batch, item)
					if len(batch) >= maxBatchSize {
						flush()
					}
				default:
					flush()
					return
				}
			}

		case item := <-p.queue:
			p.dequeueOne()
			batch = append(batch, item)
			if len(batch) >= maxBatchSize {
				flush()
				ticker.Reset(maxFlushWait)
			}

		case <-ticker.C:
			flush()

		case done := <-p.flushChan:
			for {
				select {
				case item := <-p.queue:
					p.dequeueOne()
					batch = append(batch, item)
					if len(batch) >= maxBatchSize {
						flush()
					}
				default:
					goto doneDraining
				}
			}
		doneDraining:
			flush()
			close(done)
		}
	}
}

func (p *StreamProducer) flushBatch(batch []*[]byte) {
	ctx, cancel := context.WithTimeout(context.Background(), p.writeTimeout)
	defer cancel()

	pipe := p.redisClient.Pipeline()

	n := len(batch)
	if n > p.maxBatchSize {
		n = p.maxBatchSize
	}
	if n <= 0 {
		return
	}

	wrapSlots := make([]*ByteSliceValue, n)
	valueSlots := make([]*[]any, n)
	for i := range n {
		bufPtr := batch[i]
		wrap := codec.ByteSliceValuePool.Get().(*ByteSliceValue)
		wrap.B = *bufPtr
		wrapSlots[i] = wrap

		valuesPtr := codec.ProducerValuesPool.Get().(*[]any)
		values := *valuesPtr
		values[1] = wrap
		valueSlots[i] = valuesPtr

		pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: p.streamName,
			MaxLen: p.maxStreamLen,
			Approx: true,
			Values: values,
		})
	}

	_, err := pipe.Exec(ctx)

	for i := range n {
		bufPtr := batch[i]
		*bufPtr = (*bufPtr)[:cap(*bufPtr)]
		codec.ByteBufPool.Put(bufPtr)
		codec.ByteSliceValuePool.Put(wrapSlots[i])
		codec.ProducerValuesPool.Put(valueSlots[i])
	}

	if err != nil {
		metrics.EventsDropped.Add(float64(n))
		for range n {
			telemetry.RecordRejected()
		}
		return
	}

	metrics.EventsProcessed.Add(float64(n))
	for range n {
		telemetry.RecordAccepted()
	}
}
