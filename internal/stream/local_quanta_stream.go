package stream

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/ingest/pb"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/stream/codec"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

const (
	localQuantaStreamCapacity = 8192
	localQuantaStreamMask     = localQuantaStreamCapacity - 1
	localQuantaStreamUsable   = localQuantaStreamCapacity - 1 // one slot reserved to distinguish full vs empty
	localQuantaStreamBatch    = 128
	localQuantaStreamFlush    = 2 * time.Millisecond

	localQuantaSlotClickMax = 128
	localQuantaSlotUserMax  = 128
)

const LocalQuantaStreamUsable = localQuantaStreamUsable

// localQuantaStreamSlot: fixed cell in the per-shard MPSC ring. ready=1 publishes slot to laneWorker;
// inline arrays avoid heap on Tier B Enqueue after local-quanta full-skip debit.
type localQuantaStreamSlot struct {
	ready       atomic.Uint32
	shard       uint8
	campaignID  [16]byte
	customerID  [16]byte
	amountMicro int64

	fcapPrefix [64]byte
	fcapLen    uint16
	freqLimit  uint32
	freqWindow int32

	userID  [localQuantaSlotUserMax]byte
	userLen uint16

	clickID  [localQuantaSlotClickMax]byte
	clickLen uint16

	data   []byte
	wrap   *ByteSliceValue
	bufPtr *[]byte
}

// localQuantaStreamLane: power-of-two MPSC ring per Redis shard. allocCursor claims slots on Tier B;
// writeCursor publishes; readCursor drained by laneWorker (separate cache lines on alloc/read/write).
type localQuantaStreamLane struct {
	_           [64]byte
	writeCursor uint64
	_           [64]byte
	allocCursor uint64
	_           [64]byte
	readCursor  uint64
	_           [64]byte
	slots       []localQuantaStreamSlot
}

// LocalQuantaStreamPublisher: async stream lane for LOCAL_QUOTA_MODE=live full-skip. Tier B Enqueue
// after local debit; laneWorker batches XADD + budget:sync INCRBY + fcap INCR off the request path.
// When StreamProducer is wired, stream name is fcap:ignored so only one Go writer issues XADD.
type LocalQuantaStreamPublisher struct {
	stream       string
	maxLen       int64
	redisShards  []redis.UniversalClient
	idemTTL      time.Duration
	idem         *LocalClickIdemCache
	writeTimeout time.Duration

	stopCh chan struct{}
	wg     sync.WaitGroup

	initOnce sync.Once
	lanes    []localQuantaStreamLane
}

type LocalQuantaStreamPublisherConfig struct {
	RedisShards    []redis.UniversalClient
	StreamName     string
	MaxLen         int
	IdempotencyTTL time.Duration
	IdemCache      *LocalClickIdemCache
	WriteTimeout   time.Duration
}

func (p *LocalQuantaStreamPublisher) ensureLanes() {
	if p == nil {
		return
	}
	p.initOnce.Do(func() {
		if len(p.lanes) > 0 {
			return
		}
		if len(p.redisShards) == 0 {
			p.redisShards = []redis.UniversalClient{nil}
		}
		p.lanes = make([]localQuantaStreamLane, len(p.redisShards))
		for i := range p.lanes {
			p.lanes[i].slots = make([]localQuantaStreamSlot, localQuantaStreamCapacity)
		}
	})
}

func NewLocalQuantaStreamPublisher(cfg LocalQuantaStreamPublisherConfig) *LocalQuantaStreamPublisher {
	if len(cfg.RedisShards) == 0 || cfg.StreamName == "" {
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
		redisShards:  cfg.RedisShards,
		idemTTL:      cfg.IdempotencyTTL,
		idem:         cfg.IdemCache,
		writeTimeout: cfg.WriteTimeout,
		stopCh:       make(chan struct{}),
	}
	p.ensureLanes()
	p.wg.Add(len(p.lanes))
	for i := range p.lanes {
		go p.laneWorker(i)
	}
	return p
}

func NewLocalQuantaStreamPublisherForTest(
	shards []redis.UniversalClient,
	streamName string,
	maxLen int,
	idem *LocalClickIdemCache,
	writeTimeout time.Duration,
) *LocalQuantaStreamPublisher {
	return NewLocalQuantaStreamPublisher(LocalQuantaStreamPublisherConfig{
		StreamName:     streamName,
		MaxLen:         maxLen,
		RedisShards:    shards,
		IdempotencyTTL: time.Hour,
		IdemCache:      idem,
		WriteTimeout:   writeTimeout,
	})
}

func (p *LocalQuantaStreamPublisher) IdemCache() *LocalClickIdemCache {
	return p.idem
}

func (p *LocalQuantaStreamPublisher) SetStreamName(name string) {
	if p == nil {
		return
	}
	p.stream = name
}

func (p *LocalQuantaStreamPublisher) StreamName() string {
	if p == nil {
		return ""
	}
	return p.stream
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

func marshalEventToProto(evt *domain.Event) ([]byte, *ByteSliceValue, *[]byte) {
	pbEvt := codec.StreamEventPool.Get().(*pb.AdStreamEvent)
	DeepResetAdStreamEvent(pbEvt)
	pbEvt.ClickId = UnsafeBytes(evt.ClickID)
	pbEvt.CampaignId = evt.CampaignID[:]
	pbEvt.EventType = UnsafeBytes(evt.Type)
	pbEvt.Payload = evt.Payload
	pbEvt.Ip = UnsafeBytes(evt.IP)
	pbEvt.Ua = UnsafeBytes(evt.UA)
	if len(evt.UserID) > 0 {
		pbEvt.UserId = UnsafeBytes(evt.UserID)
	}
	if !evt.CreatedAt.IsZero() {
		pbEvt.CreatedAtUnix = evt.CreatedAt.Unix()
	} else {
		pbEvt.CreatedAtUnix = time.Now().Unix()
	}

	size := pbEvt.SizeVT()
	bufPtr := codec.ByteBufPool.Get().(*[]byte)
	buf := *bufPtr
	if cap(buf) < size {
		buf = make([]byte, size)
	} else {
		buf = buf[:size]
	}
	n, err := pbEvt.MarshalToSizedBufferVT(buf)
	ClearAdStreamEvent(pbEvt)
	codec.StreamEventPool.Put(pbEvt)
	if err != nil || n <= 0 {
		*bufPtr = buf
		codec.ByteBufPool.Put(bufPtr)
		return nil, nil, nil
	}
	data := buf[:n]
	wrap := codec.ByteSliceValuePool.Get().(*ByteSliceValue)
	wrap.B = data
	*bufPtr = buf
	return data, wrap, bufPtr
}

func fillLocalQuantaStreamSlot(slot *localQuantaStreamSlot, shard int, evt *domain.Event, camp *domain.Campaign, amountMicro int64, data []byte, wrap *ByteSliceValue, bufPtr *[]byte) {
	slot.ready.Store(0)
	slot.shard = uint8(shard)
	copy(slot.campaignID[:], evt.CampaignID[:])
	if camp != nil {
		copy(slot.customerID[:], camp.CustomerID[:])
	}
	slot.amountMicro = amountMicro

	slot.clickLen = uint16(copyLocalQuantaField(slot.clickID[:], evt.ClickID))
	slot.userLen = uint16(copyLocalQuantaField(slot.userID[:], evt.UserID))

	slot.freqLimit = 0
	slot.freqWindow = 0
	slot.fcapLen = 0
	if camp != nil && camp.FreqLimit > 0 {
		slot.freqLimit = uint32(camp.FreqLimit)
		slot.freqWindow = camp.FreqWindow
		slot.fcapLen = uint16(copyLocalQuantaField(slot.fcapPrefix[:], fcapKeyPrefixForDebit(camp, evt.UserID, evt.ClickID)))
	}

	slot.data = data
	slot.wrap = wrap
	slot.bufPtr = bufPtr

	slot.ready.Store(1)
}

// Enqueue claims the next lane slot on Tier B after local TrySpendDebit. False on ring full (drop
// metric); no TryReserve here — ingest TryReserve covers StreamProducer/BrokerProducer admission.
func (p *LocalQuantaStreamPublisher) Enqueue(shard int, evt *domain.Event, camp *domain.Campaign, amountMicro int64) bool {
	if p == nil || evt == nil {
		return false
	}
	p.ensureLanes()
	if shard < 0 || shard >= len(p.lanes) {
		shard = 0
	}

	data, wrap, bufPtr := marshalEventToProto(evt)
	if data == nil {
		metrics.LocalQuotaStreamWriteErrorTotal.Inc()
		return false
	}

	lane := &p.lanes[shard]
	for {
		alloc := atomic.LoadUint64(&lane.allocCursor)
		read := atomic.LoadUint64(&lane.readCursor)
		if alloc-read >= localQuantaStreamUsable {
			metrics.LocalQuotaStreamDropTotal.Inc()
			if wrap != nil {
				codec.ByteSliceValuePool.Put(wrap)
			}
			if bufPtr != nil {
				codec.ByteBufPool.Put(bufPtr)
			}
			return false
		}
		if !atomic.CompareAndSwapUint64(&lane.allocCursor, alloc, alloc+1) {
			continue
		}
		idx := alloc & localQuantaStreamMask
		slot := &lane.slots[idx]
		if slot.ready.Load() != 0 {
			metrics.LocalQuotaStreamDropTotal.Inc()
			if wrap != nil {
				codec.ByteSliceValuePool.Put(wrap)
			}
			if bufPtr != nil {
				codec.ByteBufPool.Put(bufPtr)
			}
			return false
		}
		fillLocalQuantaStreamSlot(slot, shard, evt, camp, amountMicro, data, wrap, bufPtr)
		atomic.StoreUint64(&lane.writeCursor, alloc+1)
		return true
	}
}

func (p *LocalQuantaStreamPublisher) DrainBench() {
	if p == nil {
		return
	}
	p.ensureLanes()
	for i := range p.lanes {
		lane := &p.lanes[i]
		r := atomic.LoadUint64(&lane.readCursor)
		w := atomic.LoadUint64(&lane.writeCursor)
		for j := r; j < w; j++ {
			idx := j & localQuantaStreamMask
			slot := &lane.slots[idx]
			if slot.ready.Load() == 1 {
				if slot.wrap != nil {
					codec.ByteSliceValuePool.Put(slot.wrap)
					slot.wrap = nil
				}
				if slot.bufPtr != nil {
					codec.ByteBufPool.Put(slot.bufPtr)
					slot.bufPtr = nil
				}
				slot.data = nil
				slot.ready.Store(0)
			}
		}
		atomic.StoreUint64(&lane.readCursor, w)
		atomic.StoreUint64(&lane.allocCursor, w)
	}
}

func (p *LocalQuantaStreamPublisher) Pending() uint64 {
	p.ensureLanes()
	var total uint64
	for i := range p.lanes {
		lane := &p.lanes[i]
		write := atomic.LoadUint64(&lane.writeCursor)
		read := atomic.LoadUint64(&lane.readCursor)
		if write > read {
			total += (write - read)
		}
	}
	return total
}

func (p *LocalQuantaStreamPublisher) WaitDrained(timeout time.Duration) bool {
	if p == nil {
		return true
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if p.Pending() == 0 {
			return true
		}
		time.Sleep(time.Millisecond)
	}
	return p.Pending() == 0
}

func (p *LocalQuantaStreamPublisher) Close() {
	if p == nil {
		return
	}
	close(p.stopCh)
	p.wg.Wait()
}

// laneWorker drains readCursor..writeCursor on a ticker; Redis I/O stays off Tier B.
func (p *LocalQuantaStreamPublisher) laneWorker(shard int) {
	defer p.wg.Done()
	ticker := time.NewTicker(localQuantaStreamFlush)
	defer ticker.Stop()
	for {
		select {
		case <-p.stopCh:
			p.drainLane(shard, true)
			return
		case <-ticker.C:
			p.drainLane(shard, false)
		}
	}
}

func (p *LocalQuantaStreamPublisher) drainLane(shard int, final bool) {
	lane := &p.lanes[shard]
	batch := make([]*localQuantaStreamSlot, 0, localQuantaStreamBatch)
	for {
		read := atomic.LoadUint64(&lane.readCursor)
		write := atomic.LoadUint64(&lane.writeCursor)
		if read >= write {
			break
		}
		idx := read & localQuantaStreamMask
		slot := &lane.slots[idx]
		if slot.ready.Load() != 1 {
			break
		}
		batch = append(batch, slot)
		atomic.StoreUint64(&lane.readCursor, read+1)
		if len(batch) >= localQuantaStreamBatch {
			break
		}
	}
	if len(batch) > 0 {
		p.flushLaneBatch(shard, batch)
	}
	if final {
		for atomic.LoadUint64(&lane.writeCursor) != atomic.LoadUint64(&lane.readCursor) {
			p.drainLane(shard, false)
		}
	}
}

func (p *LocalQuantaStreamPublisher) appendIdemKey(scratch []byte, clickLen int, slot *localQuantaStreamSlot) string {
	const prefix = "idempotency:click:"
	n := copy(scratch, prefix)
	n += copy(scratch[n:], slot.clickID[:clickLen])
	return unsafeString(scratch[:n])
}

func (p *LocalQuantaStreamPublisher) flushLaneBatch(shard int, batch []*localQuantaStreamSlot) {
	ctx, cancel := context.WithTimeout(context.Background(), p.writeTimeout)
	defer cancel()

	flushed := p.flushShardPipeline(ctx, shard, batch)
	for _, slot := range batch {
		slot.wrap = nil
		slot.bufPtr = nil
		slot.data = nil
		slot.ready.Store(0)
	}
	if flushed > 0 {
		metrics.LocalQuotaStreamFlushTotal.Add(float64(flushed))
	}
}

type streamPipelineItem struct {
	slot     *localQuantaStreamSlot
	wrap     *ByteSliceValue
	bufPtr   *[]byte
	idemKey  string
	hasClick bool
}

// flushShardPipeline: (1) SetNX idempotency:click:* when click_id present, (2) Pipeline XADD unless
// stream is fcap:ignored (deferred to StreamProducer), (3) budget:sync INCRBY + fcap INCR side effects.
func (p *LocalQuantaStreamPublisher) flushShardPipeline(ctx context.Context, shard int, slots []*localQuantaStreamSlot) int {
	if shard < 0 || shard >= len(p.redisShards) {
		return 0
	}
	redisClient := p.redisShards[shard]
	if redisClient == nil {
		metrics.LocalQuotaStreamWriteErrorTotal.Add(float64(len(slots)))
		return 0
	}

	items := make([]streamPipelineItem, 0, len(slots))
	defer func() {
		for i := range items {
			if items[i].wrap != nil {
				codec.ByteSliceValuePool.Put(items[i].wrap)
			}
			if items[i].bufPtr != nil {
				codec.ByteBufPool.Put(items[i].bufPtr)
			}
		}
	}()

	for _, slot := range slots {
		if slot.wrap == nil {
			metrics.LocalQuotaStreamWriteErrorTotal.Inc()
			continue
		}
		item := streamPipelineItem{slot: slot, wrap: slot.wrap, bufPtr: slot.bufPtr}
		if slot.clickLen > 0 {
			var scratch [localQuantaSlotClickMax + 20]byte
			item.idemKey = p.appendIdemKey(scratch[:], int(slot.clickLen), slot)
			item.hasClick = true
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		return 0
	}

	accepted := make([]streamPipelineItem, 0, len(items))
	needIdem := false
	for i := range items {
		if items[i].hasClick {
			needIdem = true
			break
		}
	}
	if needIdem {
		idemPipe := redisClient.Pipeline()
		idemCmds := make([]*redis.BoolCmd, len(items))
		for i := range items {
			if items[i].hasClick {
				idemCmds[i] = idemPipe.SetNX(ctx, items[i].idemKey, "1", p.idemTTL)
			}
		}
		if _, err := idemPipe.Exec(ctx); err != nil {
			metrics.LocalQuotaStreamWriteErrorTotal.Add(float64(len(items)))
			return 0
		}
		for i, item := range items {
			if item.hasClick {
				ok, err := idemCmds[i].Result()
				if err != nil || !ok {
					continue
				}
			}
			accepted = append(accepted, item)
		}
	} else {
		accepted = items
	}
	if len(accepted) == 0 {
		return 0
	}

	type syncKey struct {
		camp uuid.UUID
		cust uuid.UUID
	}
	syncTotals := make(map[syncKey]int64, 8)
	flushed := 0

	if p.stream != "fcap:ignored" && p.stream != "" {
		// Async XADD batch; field "d" holds vtproto bytes (same layout as StreamProducer.flushBatch).
		xaddPipe := redisClient.Pipeline()
		xaddCmds := make([]*redis.StringCmd, len(accepted))
		for i, item := range accepted {
			xaddCmds[i] = xaddPipe.XAdd(ctx, &redis.XAddArgs{
				Stream: p.stream,
				MaxLen: p.maxLen,
				Approx: true,
				Values: []any{"d", item.wrap},
			})
		}
		if _, err := xaddPipe.Exec(ctx); err != nil {
			metrics.LocalQuotaStreamWriteErrorTotal.Add(float64(len(accepted)))
			return 0
		}
		for i, item := range accepted {
			if err := xaddCmds[i].Err(); err != nil {
				metrics.LocalQuotaStreamWriteErrorTotal.Inc()
				continue
			}
			flushed++
			if item.slot.amountMicro <= 0 {
				continue
			}
			var campID, custID uuid.UUID
			copy(campID[:], item.slot.campaignID[:])
			copy(custID[:], item.slot.customerID[:])
			if custID == uuid.Nil {
				continue
			}
			syncTotals[syncKey{camp: campID, cust: custID}] += item.slot.amountMicro
		}
	} else {
		flushed = len(accepted)
		for _, item := range accepted {
			if item.slot.amountMicro <= 0 {
				continue
			}
			var campID, custID uuid.UUID
			copy(campID[:], item.slot.campaignID[:])
			copy(custID[:], item.slot.customerID[:])
			if custID == uuid.Nil {
				continue
			}
			syncTotals[syncKey{camp: campID, cust: custID}] += item.slot.amountMicro
		}
	}

	fcapUpdates := false
	for _, item := range accepted {
		if item.slot.freqLimit > 0 && item.slot.userLen > 0 && item.slot.fcapLen > 0 {
			fcapUpdates = true
			break
		}
	}

	if len(syncTotals) > 0 || fcapUpdates {
		syncPipe := redisClient.Pipeline()
		for key, amt := range syncTotals {
			if amt <= 0 {
				continue
			}
			campSync := "budget:sync:campaign:" + key.camp.String()
			custSync := "budget:sync:customer:" + key.cust.String()
			syncPipe.IncrBy(ctx, campSync, amt)
			syncPipe.IncrBy(ctx, custSync, amt)
			syncPipe.SAdd(ctx, "budget:dirty_campaigns", key.camp.String())
			syncPipe.SAdd(ctx, "budget:dirty_customers", key.cust.String())
		}
		for _, item := range accepted {
			if item.slot.freqLimit > 0 && item.slot.userLen > 0 && item.slot.fcapLen > 0 {
				fcapKey := string(item.slot.fcapPrefix[:item.slot.fcapLen]) + string(item.slot.userID[:item.slot.userLen])
				syncPipe.Incr(ctx, fcapKey)
				syncPipe.Expire(ctx, fcapKey, time.Duration(item.slot.freqWindow)*time.Second)
			}
		}
		if _, err := syncPipe.Exec(ctx); err != nil {
			metrics.LocalQuotaStreamWriteErrorTotal.Add(float64(len(syncTotals)))
		}
	}
	return flushed
}
