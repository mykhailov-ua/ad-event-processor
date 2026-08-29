package broker

import (
	"context"
	"encoding/binary"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/ingest/pb"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/telemetry"
	"ad-event-processor/pkg/broker/client"
	"ad-event-processor/pkg/iogate"

	"github.com/google/uuid"
	"golang.org/x/sys/cpu"
)

var (
	ErrRingBufferFull = errors.New("broker producer ring buffer full")
	ErrProducerClosed = errors.New("broker producer closed")
)

type BrokerClient interface {
	Produce(ctx context.Context, topic string, partition uint16, payload []byte) (uint64, error)
	Close() error
}

type brokerProducerSlot struct {
	sequence          uint64
	createdAtUnix     int64
	fraudScore        uint32
	clickIDLen        uint8
	typeLen           uint8
	userIDLen         uint8
	ipLen             uint8
	uaLen             uint8
	fraudResLen       uint8
	silentRejectEvent bool

	campaignID  [16]byte
	clickID     [36]byte
	eventType   [16]byte
	userID      [36]byte
	ip          [45]byte
	ua          [128]byte
	fraudReason [64]byte
}

type BrokerProducerConfig struct {
	Topic         string
	BrokerAddr    string
	Partition     uint16
	Capacity      int
	BatchSize     int
	FlushInterval time.Duration
	Timeout       time.Duration
	Client        BrokerClient
	DiskGate      *iogate.DiskWriteGate
}

func DefaultBrokerProducerConfig() BrokerProducerConfig {
	return BrokerProducerConfig{
		Topic:         "ad-events",
		Partition:     0,
		Capacity:      32768,
		BatchSize:     512,
		FlushInterval: 2 * time.Millisecond,
		Timeout:       5 * time.Second,
	}
}

type BrokerProducer struct {
	_             cpu.CacheLinePad
	head          uint64
	_             cpu.CacheLinePad
	tail          uint64
	_             cpu.CacheLinePad
	mask          uint64
	ring          []brokerProducerSlot
	client        BrokerClient
	gate          *iogate.DiskWriteGate
	topic         string
	partition     uint16
	batchSize     int
	flushInterval time.Duration

	closed   atomic.Bool
	done     chan struct{}
	wg       sync.WaitGroup
	dropped  atomic.Uint64
	reserved atomic.Uint64
}

func NewBrokerProducer(cfg BrokerProducerConfig) (*BrokerProducer, error) {
	if cfg.Capacity <= 0 {
		cfg.Capacity = 8192
	}
	capPow2 := 1
	for capPow2 < cfg.Capacity {
		capPow2 <<= 1
	}

	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 128
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 10 * time.Millisecond
	}
	if cfg.Topic == "" {
		cfg.Topic = "ad-events"
	}

	ring := make([]brokerProducerSlot, capPow2)
	for i := range capPow2 {
		ring[i].sequence = uint64(i)
	}

	cli := cfg.Client
	if cli == nil && cfg.BrokerAddr != "" {
		cli = client.NewClient(cfg.BrokerAddr, cfg.Timeout)
	}

	bp := &BrokerProducer{
		mask:          uint64(capPow2 - 1),
		ring:          ring,
		client:        cli,
		gate:          cfg.DiskGate,
		topic:         cfg.Topic,
		partition:     cfg.Partition,
		batchSize:     cfg.BatchSize,
		flushInterval: cfg.FlushInterval,
		done:          make(chan struct{}),
	}

	bp.wg.Add(1)
	go bp.workerLoop()

	return bp, nil
}

func (bp *BrokerProducer) Enqueue(evt *domain.Event) error {
	if bp.closed.Load() {
		return ErrProducerClosed
	}

	mask := bp.mask
	ring := bp.ring

	for {
		head := atomic.LoadUint64(&bp.head)
		idx := head & mask
		slot := &ring[idx]
		seq := atomic.LoadUint64(&slot.sequence)
		diff := int64(seq) - int64(head)

		if diff == 0 {
			if atomic.CompareAndSwapUint64(&bp.head, head, head+1) {
				bp.fillSlot(slot, evt)
				atomic.StoreUint64(&slot.sequence, head+1)
				return nil
			}
		} else if diff < 0 {
			bp.dropped.Add(1)
			metrics.EventsDropped.Inc()
			telemetry.RecordRejected()
			return ErrRingBufferFull
		}
	}
}

func (bp *BrokerProducer) PendingCount() int {
	head := atomic.LoadUint64(&bp.head)
	tail := atomic.LoadUint64(&bp.tail)
	return int(head - tail)
}

func (bp *BrokerProducer) QueueCapacity() int {
	return int(bp.mask) + 1
}

func (bp *BrokerProducer) QueuePressurePct() int {
	queueCap := int(bp.mask) + 1
	if queueCap == 0 {
		return 0
	}
	occupied := int(bp.occupied())
	if occupied < 0 {
		return 100
	}
	return occupied * 100 / queueCap
}

func (bp *BrokerProducer) admissionLimit(admissionPct int) uint64 {
	ringCap := bp.mask + 1
	if admissionPct <= 0 {
		return ringCap
	}
	if admissionPct > 100 {
		admissionPct = 100
	}
	limit := ringCap * uint64(admissionPct) / 100
	if limit == 0 {
		return 1
	}
	return limit
}

func (bp *BrokerProducer) occupied() uint64 {
	head := atomic.LoadUint64(&bp.head)
	tail := atomic.LoadUint64(&bp.tail)
	pending := head - tail
	return pending + bp.reserved.Load()
}

func (bp *BrokerProducer) TryReserve(admissionPct int) bool {
	limit := bp.admissionLimit(admissionPct)
	for {
		if bp.occupied() >= limit {
			return false
		}
		r := bp.reserved.Load()
		if !bp.reserved.CompareAndSwap(r, r+1) {
			continue
		}
		if bp.occupied() > limit {
			bp.ReleaseReserve()
			return false
		}
		return true
	}
}

func (bp *BrokerProducer) ReleaseReserve() {
	bp.reserved.Add(^uint64(0))
}

func (bp *BrokerProducer) EnqueueReserved(evt *domain.Event) error {
	defer bp.ReleaseReserve()
	return bp.Enqueue(evt)
}

func (bp *BrokerProducer) EnqueueStreamEvent(evt *pb.AdStreamEvent) error {
	if bp.closed.Load() {
		return ErrProducerClosed
	}

	mask := bp.mask
	ring := bp.ring

	for {
		head := atomic.LoadUint64(&bp.head)
		idx := head & mask
		slot := &ring[idx]
		seq := atomic.LoadUint64(&slot.sequence)
		diff := int64(seq) - int64(head)

		if diff == 0 {
			if atomic.CompareAndSwapUint64(&bp.head, head, head+1) {
				bp.fillSlotFromStream(slot, evt)
				atomic.StoreUint64(&slot.sequence, head+1)
				return nil
			}
		} else if diff < 0 {
			bp.dropped.Add(1)
			metrics.EventsDropped.Inc()
			telemetry.RecordRejected()
			return ErrRingBufferFull
		}
	}
}

func (bp *BrokerProducer) fillSlot(slot *brokerProducerSlot, evt *domain.Event) {
	slot.campaignID = evt.CampaignID
	slot.createdAtUnix = evt.CreatedAt.Unix()
	slot.fraudScore = evt.FraudScore
	slot.silentRejectEvent = evt.SilentRejectEvent

	slot.clickIDLen = copyStrToFixed(slot.clickID[:], evt.ClickID)
	slot.typeLen = copyStrToFixed(slot.eventType[:], evt.Type)
	slot.userIDLen = copyStrToFixed(slot.userID[:], evt.UserID)
	slot.ipLen = copyStrToFixed(slot.ip[:], evt.IP)
	slot.uaLen = copyStrToFixed(slot.ua[:], evt.UA)
	slot.fraudResLen = copyStrToFixed(slot.fraudReason[:], evt.FraudReason)
}

func (bp *BrokerProducer) fillSlotFromStream(slot *brokerProducerSlot, evt *pb.AdStreamEvent) {
	if len(evt.CampaignId) >= 16 {
		copy(slot.campaignID[:], evt.CampaignId[:16])
	}
	slot.createdAtUnix = evt.CreatedAtUnix
	slot.fraudScore = evt.FraudScore
	slot.silentRejectEvent = evt.SilentRejectEvent

	slot.clickIDLen = copyBytesToFixed(slot.clickID[:], evt.ClickId)
	slot.typeLen = copyBytesToFixed(slot.eventType[:], evt.EventType)
	slot.userIDLen = copyBytesToFixed(slot.userID[:], evt.UserId)
	slot.ipLen = copyBytesToFixed(slot.ip[:], evt.Ip)
	slot.uaLen = copyBytesToFixed(slot.ua[:], evt.Ua)
	slot.fraudResLen = copyBytesToFixed(slot.fraudReason[:], evt.FraudReason)
}

func copyStrToFixed(dst []byte, src string) uint8 {
	n := len(src)
	if n > len(dst) {
		n = len(dst)
	}
	if n > 0 {
		_ = dst[n-1]
		copy(dst[:n], src)
	}
	return uint8(n)
}

func copyBytesToFixed(dst []byte, src []byte) uint8 {
	n := len(src)
	if n > len(dst) {
		n = len(dst)
	}
	if n > 0 {
		_ = dst[n-1]
		copy(dst[:n], src)
	}
	return uint8(n)
}

func (bp *BrokerProducer) workerLoop() {
	defer bp.wg.Done()

	ticker := time.NewTicker(bp.flushInterval)
	defer ticker.Stop()

	batch := make([]pb.AdStreamEvent, bp.batchSize)
	buf := make([]byte, 0, 64*1024)

	for {
		select {
		case <-bp.done:
			bp.flushPending(batch, &buf)
			return
		case <-ticker.C:
			bp.flushPending(batch, &buf)
		}
	}
}

func (bp *BrokerProducer) flushPending(batch []pb.AdStreamEvent, buf *[]byte) {
	mask := bp.mask
	ring := bp.ring

	for {
		count := 0
		for count < len(batch) {
			tail := atomic.LoadUint64(&bp.tail)
			idx := tail & mask
			slot := &ring[idx]
			seq := atomic.LoadUint64(&slot.sequence)
			diff := int64(seq) - int64(tail+1)

			if diff == 0 {
				evt := &batch[count]
				evt.Reset()

				evt.CampaignId = slot.campaignID[:]
				evt.CreatedAtUnix = slot.createdAtUnix
				evt.FraudScore = slot.fraudScore
				evt.SilentRejectEvent = slot.silentRejectEvent

				evt.ClickId = slot.clickID[:slot.clickIDLen]
				evt.EventType = slot.eventType[:slot.typeLen]
				evt.UserId = slot.userID[:slot.userIDLen]
				evt.Ip = slot.ip[:slot.ipLen]
				evt.Ua = slot.ua[:slot.uaLen]
				evt.FraudReason = slot.fraudReason[:slot.fraudResLen]

				atomic.StoreUint64(&slot.sequence, tail+mask+1)
				atomic.StoreUint64(&bp.tail, tail+1)
				count++
			} else {
				break
			}
		}

		if count == 0 {
			return
		}

		bp.dispatchBatch(batch[:count], buf)
	}
}

func (bp *BrokerProducer) dispatchBatch(events []pb.AdStreamEvent, bufPtr *[]byte) {
	if bp.client == nil {
		metrics.BrokerProducedEventsTotal.WithLabelValues("dropped_no_client").Add(float64(len(events)))
		return
	}

	start := time.Now()

	if bp.gate != nil {
		if err := bp.gate.AcquireAppend(context.Background(), iogate.TierHigh); err != nil {
			metrics.BrokerProducedEventsTotal.WithLabelValues("gate_shed").Add(float64(len(events)))
			return
		}
		defer bp.gate.ReleaseAppend(iogate.TierHigh)
	}

	buf := (*bufPtr)[:0]
	var lenBuf [binary.MaxVarintLen64]byte
	for i := range events {
		evt := &events[i]
		size := evt.SizeVT()
		nLen := binary.PutUvarint(lenBuf[:], uint64(size))
		offset := len(buf)
		buf = append(buf, lenBuf[:nLen]...)
		startIdx := len(buf)
		if cap(buf) < startIdx+size {
			newCap := (startIdx + size) * 2
			newBuf := make([]byte, len(buf), newCap)
			copy(newBuf, buf)
			buf = newBuf
		}
		buf = buf[:startIdx+size]
		n, err := evt.MarshalToSizedBufferVT(buf[startIdx:])
		if err != nil {
			buf = buf[:offset]
			metrics.BrokerProducedEventsTotal.WithLabelValues("marshal_error").Add(1)
			continue
		}
		if n < size {
			copy(buf[startIdx:], buf[startIdx+size-n:startIdx+size])
			buf = buf[:startIdx+n]
		}
	}
	*bufPtr = buf

	_, err := bp.client.Produce(context.Background(), bp.topic, bp.partition, buf)
	dur := time.Since(start).Seconds()

	metrics.BrokerWriteDuration.WithLabelValues(bp.topic).Observe(dur)

	if bp.gate != nil {
		if bp.gate.NoteAppend() {
			fsyncStart := time.Now()
			if fErr := bp.gate.AcquireFsync(context.Background()); fErr == nil {
				bp.gate.ReleaseFsync(time.Since(fsyncStart))
			}
		}
	}

	if err != nil {
		metrics.BrokerProducedEventsTotal.WithLabelValues("error").Add(float64(len(events)))
	} else {
		metrics.BrokerProducedEventsTotal.WithLabelValues("ok").Add(float64(len(events)))
		metrics.EventsProcessed.Add(float64(len(events)))
		telemetry.RecordAccepted()
	}
}

func (bp *BrokerProducer) DroppedCount() uint64 {
	return bp.dropped.Load()
}

func (bp *BrokerProducer) Close() error {
	if !bp.closed.CompareAndSwap(false, true) {
		return nil
	}

	close(bp.done)
	bp.wg.Wait()

	if bp.client != nil {
		return bp.client.Close()
	}
	return nil
}

type BrokerProducerSet struct {
	producers []*BrokerProducer
	sharder   domain.Sharder
}

func NewBrokerProducerSet(producers []*BrokerProducer) *BrokerProducerSet {
	n := 0
	for _, p := range producers {
		if p != nil {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	compact := make([]*BrokerProducer, 0, n)
	for _, p := range producers {
		if p != nil {
			compact = append(compact, p)
		}
	}
	return &BrokerProducerSet{
		producers: compact,
		sharder:   domain.NewStaticSlotSharder(len(compact)),
	}
}

func (s *BrokerProducerSet) Len() int {
	if s == nil {
		return 0
	}
	return len(s.producers)
}

func (s *BrokerProducerSet) Pick(campaignID uuid.UUID) (int, *BrokerProducer) {
	if s == nil || len(s.producers) == 0 {
		return 0, nil
	}
	if len(s.producers) == 1 {
		return 0, s.producers[0]
	}
	idx := s.sharder.GetShard(campaignID)
	if idx < 0 || idx >= len(s.producers) {
		return 0, s.producers[0]
	}
	return idx, s.producers[idx]
}

func (s *BrokerProducerSet) Close() error {
	if s == nil {
		return nil
	}
	var first error
	for _, p := range s.producers {
		if p == nil {
			continue
		}
		if err := p.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}
