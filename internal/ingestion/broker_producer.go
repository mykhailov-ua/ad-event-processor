package ingestion

import (
	"context"
	"encoding/binary"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/ingestion/pb"
	"github.com/bidshard/ad-event-processor/internal/metrics"
	"github.com/bidshard/ad-event-processor/internal/telemetry"
	"github.com/bidshard/ad-event-processor/pkg/broker/client"
	"github.com/bidshard/ad-event-processor/pkg/iogate"

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
	sequence      uint64
	createdAtUnix int64
	fraudScore    uint32
	clickIDLen    uint8
	typeLen       uint8
	userIDLen     uint8
	ipLen         uint8
	uaLen         uint8
	fraudResLen   uint8
	ghostEvent    bool

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
		Capacity:      8192,
		BatchSize:     128,
		FlushInterval: 10 * time.Millisecond,
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
	for i := 0; i < capPow2; i++ {
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
	tail := bp.tail
	return int(head - tail)
}

func (bp *BrokerProducer) QueueCapacity() int {
	return int(bp.mask) + 1
}

func (bp *BrokerProducer) QueuePressurePct() int {
	cap := int(bp.mask) + 1
	if cap == 0 {
		return 0
	}
	occupied := int(bp.occupied())
	if occupied < 0 {
		return 100
	}
	return occupied * 100 / cap
}

func (bp *BrokerProducer) admissionLimit(admissionPct int) uint64 {
	cap := bp.mask + 1
	if admissionPct <= 0 {
		return cap
	}
	if admissionPct > 100 {
		admissionPct = 100
	}
	limit := cap * uint64(admissionPct) / 100
	if limit == 0 {
		return 1
	}
	return limit
}

func (bp *BrokerProducer) occupied() uint64 {
	head := atomic.LoadUint64(&bp.head)
	tail := bp.tail
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
	slot.ghostEvent = evt.GhostEvent

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
	slot.ghostEvent = evt.GhostEvent

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
		_ = dst[n-1] // BCE hint
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
		_ = dst[n-1] // BCE hint
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
			tail := bp.tail
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
				evt.GhostEvent = slot.ghostEvent

				evt.ClickId = slot.clickID[:slot.clickIDLen]
				evt.EventType = slot.eventType[:slot.typeLen]
				evt.UserId = slot.userID[:slot.userIDLen]
				evt.Ip = slot.ip[:slot.ipLen]
				evt.Ua = slot.ua[:slot.uaLen]
				evt.FraudReason = slot.fraudReason[:slot.fraudResLen]

				atomic.StoreUint64(&slot.sequence, tail+mask+1)
				bp.tail = tail + 1
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
