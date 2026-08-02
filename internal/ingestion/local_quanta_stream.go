package ingestion

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"espx/internal/domain"
	"espx/internal/ingestion/pb"
	"espx/internal/metrics"

	redis "github.com/redis/go-redis/v9"
)

const (
	localQuantaStreamCapacity = 8192
	localQuantaStreamMask     = localQuantaStreamCapacity - 1
	localQuantaStreamUsable   = localQuantaStreamCapacity - 1
	localQuantaStreamBatch    = 128
	localQuantaStreamFlush    = 2 * time.Millisecond

	localQuantaSlotClickMax   = 128
	localQuantaSlotUserMax    = 128
	localQuantaSlotTypeMax    = 32
	localQuantaSlotIPMax      = 64
	localQuantaSlotUAMax      = 512
	localQuantaSlotPayloadMax = 2048
)

type localQuantaStreamSlot struct {
	ready      atomic.Uint32
	shard      uint8
	_          [3]byte
	campaignID [16]byte
	createdAt  int64

	clickLen   uint16
	userLen    uint16
	typeLen    uint16
	ipLen      uint16
	uaLen      uint16
	payloadLen uint16

	clickID [localQuantaSlotClickMax]byte
	userID  [localQuantaSlotUserMax]byte
	evtType [localQuantaSlotTypeMax]byte
	ip      [localQuantaSlotIPMax]byte
	ua      [localQuantaSlotUAMax]byte
	payload [localQuantaSlotPayloadMax]byte
}

type LocalQuantaStreamPublisher struct {
	_           [64]byte
	writeCursor uint64
	_           [64]byte
	allocCursor uint64
	_           [64]byte
	readCursor  uint64
	_           [64]byte
	slots       [localQuantaStreamCapacity]localQuantaStreamSlot

	stream         string
	maxLen         int64
	rdbs           []redis.UniversalClient
	idemTTL        time.Duration
	idem           *LocalClickIdemCache
	writeTimeout   time.Duration
	idemKeyScratch [localQuantaSlotClickMax + 20]byte

	stopCh chan struct{}
	wg     sync.WaitGroup
}

type LocalQuantaStreamPublisherConfig struct {
	Rdbs           []redis.UniversalClient
	StreamName     string
	MaxLen         int
	IdempotencyTTL time.Duration
	IdemCache      *LocalClickIdemCache
	WriteTimeout   time.Duration
}

func NewLocalQuantaStreamPublisher(cfg LocalQuantaStreamPublisherConfig) *LocalQuantaStreamPublisher {
	if len(cfg.Rdbs) == 0 || cfg.StreamName == "" {
		return nil
	}
	if cfg.IdempotencyTTL <= 0 {
		cfg.IdempotencyTTL = time.Hour
	}
	if cfg.WriteTimeout <= 0 {
		cfg.WriteTimeout = 50 * time.Millisecond
	}
	p := &LocalQuantaStreamPublisher{
		stream:       cfg.StreamName,
		maxLen:       int64(cfg.MaxLen),
		rdbs:         cfg.Rdbs,
		idemTTL:      cfg.IdempotencyTTL,
		idem:         cfg.IdemCache,
		writeTimeout: cfg.WriteTimeout,
		stopCh:       make(chan struct{}),
	}
	p.wg.Add(1)
	go p.worker()
	return p
}

func (p *LocalQuantaStreamPublisher) IdemCache() *LocalClickIdemCache {
	return p.idem
}

func copyLocalQuantaField(dst []byte, s string) int {
	n := len(s)
	if n > len(dst) {
		n = len(dst)
	}
	if n > 0 {
		copy(dst[:n], s[:n])
	}
	return n
}

func fillLocalQuantaStreamSlot(slot *localQuantaStreamSlot, shard int, evt *domain.Event) {
	slot.ready.Store(0)
	slot.shard = uint8(shard)
	copy(slot.campaignID[:], evt.CampaignID[:])
	if evt.CreatedAt.IsZero() {
		slot.createdAt = time.Now().Unix()
	} else {
		slot.createdAt = evt.CreatedAt.Unix()
	}
	slot.clickLen = uint16(copyLocalQuantaField(slot.clickID[:], evt.ClickID))
	slot.userLen = uint16(copyLocalQuantaField(slot.userID[:], evt.UserID))
	slot.typeLen = uint16(copyLocalQuantaField(slot.evtType[:], evt.Type))
	slot.ipLen = uint16(copyLocalQuantaField(slot.ip[:], evt.IP))
	slot.uaLen = uint16(copyLocalQuantaField(slot.ua[:], evt.UA))
	slot.payloadLen = uint16(copyLocalQuantaField(slot.payload[:], unsafeString(evt.Payload)))
	slot.ready.Store(1)
}

func (p *LocalQuantaStreamPublisher) Enqueue(shard int, evt *domain.Event) bool {
	if p == nil || evt == nil {
		return false
	}
	if shard < 0 || shard >= len(p.rdbs) {
		shard = 0
	}
	for {
		alloc := atomic.LoadUint64(&p.allocCursor)
		read := atomic.LoadUint64(&p.readCursor)
		if alloc-read >= localQuantaStreamUsable {
			metrics.LocalQuotaStreamDropTotal.Inc()
			return false
		}
		if !atomic.CompareAndSwapUint64(&p.allocCursor, alloc, alloc+1) {
			continue
		}
		idx := alloc & localQuantaStreamMask
		slot := &p.slots[idx]
		if slot.ready.Load() != 0 {
			metrics.LocalQuotaStreamDropTotal.Inc()
			return false
		}
		fillLocalQuantaStreamSlot(slot, shard, evt)
		atomic.StoreUint64(&p.writeCursor, alloc+1)
		return true
	}
}

func (p *LocalQuantaStreamPublisher) Pending() uint64 {
	write := atomic.LoadUint64(&p.writeCursor)
	read := atomic.LoadUint64(&p.readCursor)
	if write < read {
		return 0
	}
	return write - read
}

func (p *LocalQuantaStreamPublisher) Close() {
	if p == nil {
		return
	}
	close(p.stopCh)
	p.wg.Wait()
}

func (p *LocalQuantaStreamPublisher) worker() {
	defer p.wg.Done()
	ticker := time.NewTicker(localQuantaStreamFlush)
	defer ticker.Stop()
	for {
		select {
		case <-p.stopCh:
			p.drain(true)
			return
		case <-ticker.C:
			p.drain(false)
		}
	}
}

func (p *LocalQuantaStreamPublisher) drain(final bool) {
	batch := make([]*localQuantaStreamSlot, 0, localQuantaStreamBatch)
	for {
		read := atomic.LoadUint64(&p.readCursor)
		write := atomic.LoadUint64(&p.writeCursor)
		if read >= write {
			break
		}
		idx := read & localQuantaStreamMask
		slot := &p.slots[idx]
		if slot.ready.Load() != 1 {
			break
		}
		batch = append(batch, slot)
		atomic.StoreUint64(&p.readCursor, read+1)
		if len(batch) >= localQuantaStreamBatch {
			break
		}
	}
	if len(batch) > 0 {
		p.flushBatch(batch)
	}
	if final {
		for atomic.LoadUint64(&p.writeCursor) != atomic.LoadUint64(&p.readCursor) {
			p.drain(false)
		}
	}
}

func (p *LocalQuantaStreamPublisher) appendIdemKey(scratch []byte, clickLen int, slot *localQuantaStreamSlot) string {
	const prefix = "idempotency:click:"
	n := copy(scratch, prefix)
	n += copy(scratch[n:], slot.clickID[:clickLen])
	return unsafeString(scratch[:n])
}

func (p *LocalQuantaStreamPublisher) flushBatch(batch []*localQuantaStreamSlot) {
	ctx, cancel := context.WithTimeout(context.Background(), p.writeTimeout)
	defer cancel()

	flushed := 0
	for _, slot := range batch {
		if p.flushOne(ctx, slot) {
			flushed++
		}
		slot.ready.Store(0)
	}
	if flushed > 0 {
		metrics.LocalQuotaStreamFlushTotal.Add(float64(flushed))
	}
}

func (p *LocalQuantaStreamPublisher) flushOne(ctx context.Context, slot *localQuantaStreamSlot) bool {
	data, wrap, bufPtr := marshalLocalQuantaStreamSlot(slot)
	if data == nil {
		metrics.LocalQuotaStreamWriteErrorTotal.Inc()
		return false
	}
	defer func() {
		byteSliceValuePool.Put(wrap)
		byteBufPool.Put(bufPtr)
	}()

	shard := int(slot.shard)
	if shard < 0 || shard >= len(p.rdbs) {
		shard = 0
	}
	rdb := p.rdbs[shard]

	if slot.clickLen > 0 {
		idemKey := p.appendIdemKey(p.idemKeyScratch[:], int(slot.clickLen), slot)
		ok, err := rdb.SetNX(ctx, idemKey, "1", p.idemTTL).Result()
		if err != nil {
			metrics.LocalQuotaStreamWriteErrorTotal.Inc()
			return false
		}
		if !ok {
			return false
		}
	}

	_, err := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: p.stream,
		MaxLen: p.maxLen,
		Approx: true,
		Values: []any{"d", wrap},
	}).Result()
	if err != nil {
		metrics.LocalQuotaStreamWriteErrorTotal.Inc()
		return false
	}
	return true
}

func marshalLocalQuantaStreamSlot(slot *localQuantaStreamSlot) ([]byte, *ByteSliceValue, *[]byte) {
	pbEvt := streamEventPool.Get().(*pb.AdStreamEvent)
	DeepResetAdStreamEvent(pbEvt)
	pbEvt.ClickId = slot.clickID[:slot.clickLen]
	pbEvt.CampaignId = slot.campaignID[:]
	pbEvt.EventType = slot.evtType[:slot.typeLen]
	pbEvt.Payload = slot.payload[:slot.payloadLen]
	pbEvt.Ip = slot.ip[:slot.ipLen]
	pbEvt.Ua = slot.ua[:slot.uaLen]
	if slot.userLen > 0 {
		pbEvt.UserId = slot.userID[:slot.userLen]
	}
	if slot.createdAt > 0 {
		pbEvt.CreatedAtUnix = slot.createdAt
	}

	size := pbEvt.SizeVT()
	bufPtr := byteBufPool.Get().(*[]byte)
	buf := *bufPtr
	if cap(buf) < size {
		buf = make([]byte, size)
	} else {
		buf = buf[:size]
	}
	n, err := pbEvt.MarshalToSizedBufferVT(buf)
	ClearAdStreamEvent(pbEvt)
	streamEventPool.Put(pbEvt)
	if err != nil || n <= 0 {
		*bufPtr = buf
		byteBufPool.Put(bufPtr)
		return nil, nil, nil
	}
	data := buf[:n]
	wrap := byteSliceValuePool.Get().(*ByteSliceValue)
	wrap.b = data
	*bufPtr = buf
	return data, wrap, bufPtr
}
