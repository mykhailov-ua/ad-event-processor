package rtb

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/db"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/redis/go-redis/v9"
)

func (c *RtbCatalog) LookupDealBytes(dealID []byte) (DealData, bool) {
	if c == nil || c.dealIndex == nil {
		return DealData{}, false
	}
	return c.dealIndex.LookupBytes(dealID)
}

const rtbCatalogReloadDebounce = 100 * time.Millisecond

func runRtbCatalogReloadDebouncer(ctx context.Context, trigger <-chan struct{}, reload func(), debounce time.Duration) {
	if debounce <= 0 {
		debounce = rtbCatalogReloadDebounce
	}
	debounceTimer := time.NewTimer(time.Hour)
	if !debounceTimer.Stop() {
		select {
		case <-debounceTimer.C:
		default:
		}
	}
	for {
		select {
		case <-ctx.Done():
			if !debounceTimer.Stop() {
				select {
				case <-debounceTimer.C:
				default:
				}
			}
			return
		case <-trigger:
			if !debounceTimer.Stop() {
				select {
				case <-debounceTimer.C:
				default:
				}
			}
			debounceTimer.Reset(debounce)
		case <-debounceTimer.C:
			reload()
		}
	}
}

const (
	rtbOutcomeRingCapacity = 4096
	rtbOutcomeRingMask     = rtbOutcomeRingCapacity - 1
	rtbOutcomeRingUsable   = rtbOutcomeRingCapacity - 1
	rtbOutcomeFlushBatch   = 128
	rtbOutcomeDealIDMax    = 64
)

type rtbOutcomeSlot struct {
	ready      atomic.Uint32
	dealLen    uint8
	outcome    uint8
	_          [2]byte
	floorMicro int64
	createdAt  int64
	dealID     [rtbOutcomeDealIDMax]byte
}

type rtbOutcomeRow struct {
	dealLen    uint8
	outcome    uint8
	_          [2]byte
	floorMicro int64
	createdAt  int64
	dealID     [rtbOutcomeDealIDMax]byte
}

type RtbDealOutcomeWriter struct {
	_           [64]byte
	writeCursor uint64
	_           [64]byte
	allocCursor uint64
	_           [64]byte
	readCursor  uint64
	_           [64]byte
	slots       [rtbOutcomeRingCapacity]rtbOutcomeSlot

	conn       driver.Conn
	flushEvery time.Duration

	stopCh chan struct{}
	wg     sync.WaitGroup
}

var globalRtbOutcomeWriter atomic.Pointer[RtbDealOutcomeWriter]

func SetRtbDealOutcomeWriter(w *RtbDealOutcomeWriter) {
	globalRtbOutcomeWriter.Store(w)
}

func NewRtbDealOutcomeWriter(conn driver.Conn, flushEvery time.Duration) *RtbDealOutcomeWriter {
	if conn == nil {
		return nil
	}
	if flushEvery <= 0 {
		flushEvery = 5 * time.Second
	}
	w := &RtbDealOutcomeWriter{
		conn:       conn,
		flushEvery: flushEvery,
		stopCh:     make(chan struct{}),
	}
	w.wg.Add(1)
	go w.worker()
	return w
}

func (w *RtbDealOutcomeWriter) Close() {
	if w == nil {
		return
	}
	close(w.stopCh)
	w.wg.Wait()
}

func (w *RtbDealOutcomeWriter) Enqueue(dealID []byte, outcome uint8, floorMicro int64) bool {
	if w == nil {
		return true
	}
	for {
		alloc := atomic.LoadUint64(&w.allocCursor)
		read := atomic.LoadUint64(&w.readCursor)
		if alloc-read >= rtbOutcomeRingUsable {
			return false
		}
		if !atomic.CompareAndSwapUint64(&w.allocCursor, alloc, alloc+1) {
			continue
		}
		idx := alloc & rtbOutcomeRingMask
		slot := &w.slots[idx]
		for slot.ready.Load() != 0 {
			return false
		}
		ln := len(dealID)
		if ln > rtbOutcomeDealIDMax {
			ln = rtbOutcomeDealIDMax
		}
		slot.dealLen = uint8(ln)
		slot.outcome = outcome
		slot.floorMicro = floorMicro
		slot.createdAt = time.Now().UTC().UnixMilli()
		for i := range ln {
			slot.dealID[i] = dealID[i]
		}
		slot.ready.Store(1)
		atomic.StoreUint64(&w.writeCursor, alloc+1)
		return true
	}
}

func RecordRtbDealOutcome(dealID string, floorMicro int64, res AuctionResult, reason NoBidReason) {
	if dealID == "" {
		RecordRtbDealOutcomeBytes(nil, 0, floorMicro, res, reason)
		return
	}
	var buf [rtbOutcomeDealIDMax]byte
	n := copy(buf[:], dealID)
	RecordRtbDealOutcomeBytes(buf[:n], uint8(n), floorMicro, res, reason)
}

func RecordRtbDealOutcomeBytes(dealID []byte, dealLen uint8, floorMicro int64, res AuctionResult, reason NoBidReason) {
	w := globalRtbOutcomeWriter.Load()
	if w == nil {
		return
	}
	outcome := uint8(0)
	if reason.OK() {
		outcome = 1
	}
	var buf []byte
	if dealLen > 0 {
		buf = dealID[:dealLen]
	}
	_ = w.Enqueue(buf, outcome, floorMicro)
}

func (w *RtbDealOutcomeWriter) worker() {
	defer w.wg.Done()
	ticker := time.NewTicker(w.flushEvery)
	defer ticker.Stop()
	batch := make([]rtbOutcomeRow, 0, rtbOutcomeFlushBatch)
	for {
		select {
		case <-w.stopCh:
			w.drainBatch(&batch)
			return
		case <-ticker.C:
			w.drainBatch(&batch)
		default:
			if w.collectBatch(&batch) {
				w.flushBatch(batch)
				batch = batch[:0]
			} else {
				time.Sleep(time.Millisecond)
			}
		}
	}
}

func (w *RtbDealOutcomeWriter) collectBatch(batch *[]rtbOutcomeRow) bool {
	read := atomic.LoadUint64(&w.readCursor)
	write := atomic.LoadUint64(&w.writeCursor)
	if read >= write {
		return false
	}
	for read < write && len(*batch) < rtbOutcomeFlushBatch {
		idx := read & rtbOutcomeRingMask
		slot := &w.slots[idx]
		if slot.ready.Load() == 0 {
			break
		}
		*batch = append(*batch, rtbOutcomeRow{
			dealLen:    slot.dealLen,
			outcome:    slot.outcome,
			floorMicro: slot.floorMicro,
			createdAt:  slot.createdAt,
			dealID:     slot.dealID,
		})
		slot.ready.Store(0)
		read++
	}
	atomic.StoreUint64(&w.readCursor, read)
	return len(*batch) > 0
}

func (w *RtbDealOutcomeWriter) drainBatch(batch *[]rtbOutcomeRow) {
	for w.collectBatch(batch) {
		w.flushBatch(*batch)
		*batch = (*batch)[:0]
	}
}

func (w *RtbDealOutcomeWriter) flushBatch(batch []rtbOutcomeRow) {
	if len(batch) == 0 || w.conn == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	chBatch, err := w.conn.PrepareBatch(ctx, `INSERT INTO rtb_deal_outcomes (deal_id, outcome, floor_micro, created_at)`)
	if err != nil {
		return
	}
	for i := range batch {
		slot := &batch[i]
		dealID := string(slot.dealID[:slot.dealLen])
		created := time.UnixMilli(slot.createdAt).UTC()
		if err := chBatch.Append(dealID, slot.outcome, slot.floorMicro, created); err != nil {
			_ = chBatch.Abort()
			return
		}
	}
	_ = chBatch.Send()
}

func ReloadRtbCatalog(
	ctx context.Context,
	q *db.Queries,
	registry CampaignSource,
	catalog *RtbCatalog,
	cfg *config.Config,
	hybrid CampaignWeighter,
	budgetSync RtbBudgetSync,
	watcher FcapSnapshotProvider,
) error {
	if err := ReloadDeals(ctx, q, catalog); err != nil {
		return err
	}
	if registry != nil && catalog != nil && cfg != nil && cfg.RtbEnabled() {
		SyncRtbCatalog(ctx, registry, catalog, cfg, hybrid, budgetSync, watcher)
		if allow, err := LoadSupplyChainAllowlist(ctx, q); err == nil {
			catalog.SetSupplyChainAllowlist(allow)
		}
	}
	return nil
}

func StartRtbCatalogReloadWatch(
	ctx context.Context,
	q *db.Queries,
	redisClient redis.UniversalClient,
	channel string,
	registry CampaignSource,
	catalog *RtbCatalog,
	cfg *config.Config,
	hybrid CampaignWeighter,
	budgetSync RtbBudgetSync,
	watcher FcapSnapshotProvider,
) {
	if redisClient == nil || catalog == nil || q == nil {
		return
	}
	if channel == "" {
		channel = domain.DefaultRtbCatalogReloadChannel
	}

	reload := func() {
		reloadCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := ReloadRtbCatalog(reloadCtx, q, registry, catalog, cfg, hybrid, budgetSync, watcher); err != nil {
			slog.Error("rtb catalog reload failed", "error", err)
			return
		}
		slog.Info("rtb catalog reloaded via pubsub", "deals", catalog.DealCount())
	}

	go func() {
		pubsub := redisClient.Subscribe(ctx, channel)
		defer func() { _ = pubsub.Close() }()

		ch := pubsub.Channel(redis.WithChannelSize(64))
		trigger := make(chan struct{}, 1)
		go runRtbCatalogReloadDebouncer(ctx, trigger, reload, rtbCatalogReloadDebounce)

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					slog.Error("rtb catalog reload pubsub channel closed")
					return
				}
				if msg == nil {
					continue
				}
				select {
				case trigger <- struct{}{}:
				default:
				}
			}
		}
	}()
}

const (
	rtbExchangeRingCapacity = 4096
	rtbExchangeRingMask     = rtbExchangeRingCapacity - 1
	rtbExchangeRingUsable   = rtbExchangeRingCapacity - 1
	rtbExchangeFlushBatch   = 128
	RtbExchangeRequestIDMax = 128
	RtbExchangeBidIDMax     = 48
	RtbExchangeDealIDMax    = 64
	rtbExchangeInventoryMax = 128
	rtbExchangeDeviceOSMax  = 16
	rtbExchangeSourceTIDMax = 128
	rtbExchangeEIDSourceMax = 64
	rtbExchangeAppVerMax    = 32
)

type RtbExchangeLogMeta struct {
	inventoryLen   uint8
	deviceOSLen    uint8
	sourceTIDLen   uint8
	eidSourceLen   uint8
	appVerLen      uint8
	connectionType uint8
	pmpPrivate     uint8
	deviceLMT      uint8
	mediaW         uint16
	mediaH         uint16
	viewabilityPPM uint32
	geoCountry     [2]byte
	inventory      [rtbExchangeInventoryMax]byte
	deviceOS       [rtbExchangeDeviceOSMax]byte
	sourceTID      [rtbExchangeSourceTIDMax]byte
	eidSource      [rtbExchangeEIDSourceMax]byte
	appVer         [rtbExchangeAppVerMax]byte
}

type rtbExchangeSlot struct {
	ready          atomic.Uint32
	reqLen         uint8
	bidLen         uint8
	dealLen        uint8
	won            uint8
	noBidReason    uint16
	priceMicro     int64
	createdAt      int64
	requestID      [RtbExchangeRequestIDMax]byte
	bidID          [RtbExchangeBidIDMax]byte
	dealID         [RtbExchangeDealIDMax]byte
	inventory      [rtbExchangeInventoryMax]byte
	deviceOS       [rtbExchangeDeviceOSMax]byte
	sourceTID      [rtbExchangeSourceTIDMax]byte
	eidSource      [rtbExchangeEIDSourceMax]byte
	appVer         [rtbExchangeAppVerMax]byte
	geoCountry     [2]byte
	inventoryLen   uint8
	deviceOSLen    uint8
	sourceTIDLen   uint8
	eidSourceLen   uint8
	appVerLen      uint8
	connectionType uint8
	pmpPrivate     uint8
	deviceLMT      uint8
	mediaW         uint16
	mediaH         uint16
	viewabilityPPM uint32
}

type rtbExchangeRow struct {
	reqLen         uint8
	bidLen         uint8
	dealLen        uint8
	won            uint8
	noBidReason    uint16
	priceMicro     int64
	createdAt      int64
	requestID      [RtbExchangeRequestIDMax]byte
	bidID          [RtbExchangeBidIDMax]byte
	dealID         [RtbExchangeDealIDMax]byte
	inventory      [rtbExchangeInventoryMax]byte
	deviceOS       [rtbExchangeDeviceOSMax]byte
	sourceTID      [rtbExchangeSourceTIDMax]byte
	eidSource      [rtbExchangeEIDSourceMax]byte
	appVer         [rtbExchangeAppVerMax]byte
	geoCountry     [2]byte
	inventoryLen   uint8
	deviceOSLen    uint8
	sourceTIDLen   uint8
	eidSourceLen   uint8
	appVerLen      uint8
	connectionType uint8
	pmpPrivate     uint8
	deviceLMT      uint8
	mediaW         uint16
	mediaH         uint16
	viewabilityPPM uint32
}

type RtbExchangeLogWriter struct {
	_           [64]byte
	writeCursor uint64
	_           [64]byte
	allocCursor uint64
	_           [64]byte
	readCursor  uint64
	_           [64]byte
	slots       [rtbExchangeRingCapacity]rtbExchangeSlot

	conn       driver.Conn
	flushEvery time.Duration

	stopCh chan struct{}
	wg     sync.WaitGroup
}

var globalRtbExchangeLogWriter atomic.Pointer[RtbExchangeLogWriter]
