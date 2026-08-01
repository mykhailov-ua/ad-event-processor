package ingestion

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

const (
	rtbExchangeRingCapacity = 4096
	rtbExchangeRingMask     = rtbExchangeRingCapacity - 1
	rtbExchangeRingUsable   = rtbExchangeRingCapacity - 1
	rtbExchangeFlushBatch   = 128
	rtbExchangeRequestIDMax = 128
	rtbExchangeBidIDMax     = 48
	rtbExchangeDealIDMax    = 64
	rtbExchangeInventoryMax = 128
	rtbExchangeDeviceOSMax  = 16
	rtbExchangeSourceTIDMax = 128
	rtbExchangeEIDSourceMax = 64
	rtbExchangeAppVerMax    = 32
)

type rtbExchangeLogMeta struct {
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
	requestID      [rtbExchangeRequestIDMax]byte
	bidID          [rtbExchangeBidIDMax]byte
	dealID         [rtbExchangeDealIDMax]byte
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
	requestID      [rtbExchangeRequestIDMax]byte
	bidID          [rtbExchangeBidIDMax]byte
	dealID         [rtbExchangeDealIDMax]byte
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

func SetRtbExchangeLogWriter(w *RtbExchangeLogWriter) {
	globalRtbExchangeLogWriter.Store(w)
}

func NewRtbExchangeLogWriter(conn driver.Conn, flushEvery time.Duration) *RtbExchangeLogWriter {
	if conn == nil {
		return nil
	}
	if flushEvery <= 0 {
		flushEvery = 5 * time.Second
	}
	w := &RtbExchangeLogWriter{
		conn:       conn,
		flushEvery: flushEvery,
		stopCh:     make(chan struct{}),
	}
	w.wg.Add(1)
	go w.worker()
	return w
}

func (w *RtbExchangeLogWriter) Close() {
	if w == nil {
		return
	}
	close(w.stopCh)
	w.wg.Wait()
}

func recordRtbExchangeLog(requestID []byte, bidID []byte, dealID []byte, won bool, reason uint16, priceMicro int64, meta rtbExchangeLogMeta) {
	w := globalRtbExchangeLogWriter.Load()
	if w == nil {
		return
	}
	_ = w.Enqueue(requestID, bidID, dealID, won, reason, priceMicro, meta)
}

func (w *RtbExchangeLogWriter) Enqueue(requestID, bidID, dealID []byte, won bool, reason uint16, priceMicro int64, meta rtbExchangeLogMeta) bool {
	if w == nil {
		return true
	}
	for {
		alloc := atomic.LoadUint64(&w.allocCursor)
		read := atomic.LoadUint64(&w.readCursor)
		if alloc-read >= rtbExchangeRingUsable {
			return false
		}
		if !atomic.CompareAndSwapUint64(&w.allocCursor, alloc, alloc+1) {
			continue
		}
		idx := alloc & rtbExchangeRingMask
		slot := &w.slots[idx]
		if slot.ready.Load() != 0 {
			return false
		}
		reqLen := len(requestID)
		if reqLen > rtbExchangeRequestIDMax {
			reqLen = rtbExchangeRequestIDMax
		}
		bidLen := len(bidID)
		if bidLen > rtbExchangeBidIDMax {
			bidLen = rtbExchangeBidIDMax
		}
		dealLen := len(dealID)
		if dealLen > rtbExchangeDealIDMax {
			dealLen = rtbExchangeDealIDMax
		}
		slot.reqLen = uint8(reqLen)
		slot.bidLen = uint8(bidLen)
		slot.dealLen = uint8(dealLen)
		for i := 0; i < reqLen; i++ {
			slot.requestID[i] = requestID[i]
		}
		for i := 0; i < bidLen; i++ {
			slot.bidID[i] = bidID[i]
		}
		for i := 0; i < dealLen; i++ {
			slot.dealID[i] = dealID[i]
		}
		invLen := int(meta.inventoryLen)
		if invLen > rtbExchangeInventoryMax {
			invLen = rtbExchangeInventoryMax
		}
		osLen := int(meta.deviceOSLen)
		if osLen > rtbExchangeDeviceOSMax {
			osLen = rtbExchangeDeviceOSMax
		}
		slot.inventoryLen = uint8(invLen)
		slot.deviceOSLen = uint8(osLen)
		slot.mediaW = meta.mediaW
		slot.mediaH = meta.mediaH
		slot.viewabilityPPM = meta.viewabilityPPM
		slot.connectionType = meta.connectionType
		slot.pmpPrivate = meta.pmpPrivate
		slot.deviceLMT = meta.deviceLMT
		slot.geoCountry = meta.geoCountry
		tidLen := int(meta.sourceTIDLen)
		if tidLen > rtbExchangeSourceTIDMax {
			tidLen = rtbExchangeSourceTIDMax
		}
		eidLen := int(meta.eidSourceLen)
		if eidLen > rtbExchangeEIDSourceMax {
			eidLen = rtbExchangeEIDSourceMax
		}
		verLen := int(meta.appVerLen)
		if verLen > rtbExchangeAppVerMax {
			verLen = rtbExchangeAppVerMax
		}
		slot.sourceTIDLen = uint8(tidLen)
		slot.eidSourceLen = uint8(eidLen)
		slot.appVerLen = uint8(verLen)
		for i := 0; i < invLen; i++ {
			slot.inventory[i] = meta.inventory[i]
		}
		for i := 0; i < osLen; i++ {
			slot.deviceOS[i] = meta.deviceOS[i]
		}
		for i := 0; i < tidLen; i++ {
			slot.sourceTID[i] = meta.sourceTID[i]
		}
		for i := 0; i < eidLen; i++ {
			slot.eidSource[i] = meta.eidSource[i]
		}
		for i := 0; i < verLen; i++ {
			slot.appVer[i] = meta.appVer[i]
		}
		if won {
			slot.won = 1
		}
		slot.noBidReason = reason
		slot.priceMicro = priceMicro
		slot.createdAt = time.Now().UTC().UnixMilli()
		slot.ready.Store(1)
		atomic.StoreUint64(&w.writeCursor, alloc+1)
		return true
	}
}

func (w *RtbExchangeLogWriter) worker() {
	defer w.wg.Done()
	ticker := time.NewTicker(w.flushEvery)
	defer ticker.Stop()
	batch := make([]rtbExchangeRow, 0, rtbExchangeFlushBatch)
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

func (w *RtbExchangeLogWriter) collectBatch(batch *[]rtbExchangeRow) bool {
	read := atomic.LoadUint64(&w.readCursor)
	write := atomic.LoadUint64(&w.writeCursor)
	if read >= write {
		return false
	}
	for read < write && len(*batch) < rtbExchangeFlushBatch {
		idx := read & rtbExchangeRingMask
		slot := &w.slots[idx]
		if slot.ready.Load() == 0 {
			break
		}
		*batch = append(*batch, rtbExchangeRow{
			reqLen:         slot.reqLen,
			bidLen:         slot.bidLen,
			dealLen:        slot.dealLen,
			won:            slot.won,
			noBidReason:    slot.noBidReason,
			priceMicro:     slot.priceMicro,
			createdAt:      slot.createdAt,
			requestID:      slot.requestID,
			bidID:          slot.bidID,
			dealID:         slot.dealID,
			inventory:      slot.inventory,
			deviceOS:       slot.deviceOS,
			sourceTID:      slot.sourceTID,
			eidSource:      slot.eidSource,
			appVer:         slot.appVer,
			geoCountry:     slot.geoCountry,
			inventoryLen:   slot.inventoryLen,
			deviceOSLen:    slot.deviceOSLen,
			sourceTIDLen:   slot.sourceTIDLen,
			eidSourceLen:   slot.eidSourceLen,
			appVerLen:      slot.appVerLen,
			connectionType: slot.connectionType,
			pmpPrivate:     slot.pmpPrivate,
			deviceLMT:      slot.deviceLMT,
			mediaW:         slot.mediaW,
			mediaH:         slot.mediaH,
			viewabilityPPM: slot.viewabilityPPM,
		})
		slot.ready.Store(0)
		read++
	}
	atomic.StoreUint64(&w.readCursor, read)
	return len(*batch) > 0
}

func (w *RtbExchangeLogWriter) drainBatch(batch *[]rtbExchangeRow) {
	for w.collectBatch(batch) {
		w.flushBatch(*batch)
		*batch = (*batch)[:0]
	}
}

func (w *RtbExchangeLogWriter) flushBatch(batch []rtbExchangeRow) {
	if len(batch) == 0 || w.conn == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	chBatch, err := w.conn.PrepareBatch(ctx, `INSERT INTO rtb_exchange_log (request_id, bid_id, won, no_bid_reason, price_micro, deal_id, inventory, geo_country, device_os, media_w, media_h, source_tid, connection_type, pmp_private, device_lmt, viewability_ppm, eid_source, app_ver, created_at)`)
	if err != nil {
		return
	}
	for i := range batch {
		row := &batch[i]
		created := time.UnixMilli(row.createdAt).UTC()
		if err := chBatch.Append(
			string(row.requestID[:row.reqLen]),
			string(row.bidID[:row.bidLen]),
			row.won,
			row.noBidReason,
			row.priceMicro,
			string(row.dealID[:row.dealLen]),
			string(row.inventory[:row.inventoryLen]),
			string(row.geoCountry[:]),
			string(row.deviceOS[:row.deviceOSLen]),
			row.mediaW,
			row.mediaH,
			string(row.sourceTID[:row.sourceTIDLen]),
			row.connectionType,
			row.pmpPrivate,
			row.deviceLMT,
			row.viewabilityPPM,
			string(row.eidSource[:row.eidSourceLen]),
			string(row.appVer[:row.appVerLen]),
			created,
		); err != nil {
			_ = chBatch.Abort()
			return
		}
	}
	_ = chBatch.Send()
}
