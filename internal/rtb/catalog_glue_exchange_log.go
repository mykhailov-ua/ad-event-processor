package rtb

import (
	"context"
	"encoding/binary"
	"log/slog"
	"math"
	"sort"
	"strconv"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/domain"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

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

func RecordRtbExchangeLog(requestID []byte, bidID []byte, dealID []byte, won bool, reason uint16, priceMicro int64, meta RtbExchangeLogMeta) {
	w := globalRtbExchangeLogWriter.Load()
	if w == nil {
		return
	}
	_ = w.Enqueue(requestID, bidID, dealID, won, reason, priceMicro, meta)
}

func (w *RtbExchangeLogWriter) Enqueue(requestID, bidID, dealID []byte, won bool, reason uint16, priceMicro int64, meta RtbExchangeLogMeta) bool {
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
		if reqLen > RtbExchangeRequestIDMax {
			reqLen = RtbExchangeRequestIDMax
		}
		bidLen := len(bidID)
		if bidLen > RtbExchangeBidIDMax {
			bidLen = RtbExchangeBidIDMax
		}
		dealLen := len(dealID)
		if dealLen > RtbExchangeDealIDMax {
			dealLen = RtbExchangeDealIDMax
		}
		slot.reqLen = uint8(reqLen)
		slot.bidLen = uint8(bidLen)
		slot.dealLen = uint8(dealLen)
		for i := range reqLen {
			slot.requestID[i] = requestID[i]
		}
		for i := range bidLen {
			slot.bidID[i] = bidID[i]
		}
		for i := range dealLen {
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
		for i := range invLen {
			slot.inventory[i] = meta.inventory[i]
		}
		for i := range osLen {
			slot.deviceOS[i] = meta.deviceOS[i]
		}
		for i := range tidLen {
			slot.sourceTID[i] = meta.sourceTID[i]
		}
		for i := range eidLen {
			slot.eidSource[i] = meta.eidSource[i]
		}
		for i := range verLen {
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

type DealFloorCache struct {
	redisClient redis.UniversalClient
	snap        atomic.Pointer[map[string]int64]
}

func NewDealFloorCache(redisClient redis.UniversalClient) *DealFloorCache {
	c := &DealFloorCache{redisClient: redisClient}
	empty := make(map[string]int64)
	c.snap.Store(&empty)
	return c
}

func (c *DealFloorCache) Get(dealID string) (int64, bool) {
	if dealID == "" {
		return 0, false
	}
	ptr := c.snap.Load()
	if ptr == nil {
		return 0, false
	}
	v, ok := (*ptr)[dealID]
	return v, ok
}

func (c *DealFloorCache) Refresh(ctx context.Context, dealIDs []string) {
	if c == nil || c.redisClient == nil || len(dealIDs) == 0 {
		return
	}
	keys := make([]string, len(dealIDs))
	for i, id := range dealIDs {
		keys[i] = domain.RtbFloorRedisKeyPrefix + id
	}
	vals, err := c.redisClient.MGet(ctx, keys...).Result()
	if err != nil {
		slog.Warn("deal floor cache refresh failed", "error", err)
		return
	}
	next := make(map[string]int64, len(dealIDs))
	for i, raw := range vals {
		if raw == nil {
			continue
		}
		s, ok := raw.(string)
		if !ok || s == "" {
			continue
		}
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil || v <= 0 {
			continue
		}
		next[dealIDs[i]] = v
	}
	c.snap.Store(&next)
}

func StartDealFloorRefresh(ctx context.Context, cache *DealFloorCache, catalog *RtbCatalog, interval time.Duration) {
	if cache == nil || catalog == nil || interval <= 0 {
		return
	}
	refresh := func() {
		deals := catalog.AllDeals()
		if len(deals) == 0 {
			return
		}
		ids := make([]string, len(deals))
		for i, d := range deals {
			ids[i] = d.DealID
		}
		rctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		cache.Refresh(rctx, ids)
		cancel()
	}
	refresh()
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				refresh()
			}
		}
	}()
}

func EffectiveDealFloor(catalog *RtbCatalog, floors *DealFloorCache, dealID string, publisherFloor int64) int64 {
	return EffectiveDealFloorBytes(catalog, floors, []byte(dealID), publisherFloor)
}

func EffectiveDealFloorBytes(catalog *RtbCatalog, floors *DealFloorCache, dealID []byte, publisherFloor int64) int64 {
	floor := publisherFloor
	if len(dealID) == 0 {
		return floor
	}
	if catalog != nil {
		if deal, ok := catalog.LookupDealBytes(dealID); ok && deal.FloorMicro > floor {
			floor = deal.FloorMicro
		}
	}
	if floors != nil {
		key := UnsafeString(dealID)
		if optimized, ok := floors.Get(key); ok && optimized > floor {
			floor = optimized
		}
	}
	return floor
}

func sortedTargetCountries(camp *domain.Campaign) []string {
	if camp == nil || len(camp.TargetCountries) == 0 {
		return nil
	}
	out := make([]string, 0, len(camp.TargetCountries))
	for c := range camp.TargetCountries {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

func fanOutRtbCatalogRows(camp *domain.Campaign, base RtbCampaignInput) []CampaignData {
	countries := sortedTargetCountries(camp)
	if len(countries) == 0 {
		return []CampaignData{CampaignDataFromDomain(camp, base)}
	}
	out := make([]CampaignData, 0, len(countries))
	for _, country := range countries {
		inp := base
		inp.GeoHash = GeoHashFromCountry(country)
		out = append(out, CampaignDataFromDomain(camp, inp))
	}
	return out
}

func CustomerIDFromCustomerUUID(id uuid.UUID) CustomerID {
	return CustomerID(binary.BigEndian.Uint64(id[:8]))
}

func PacingOpenFromManagement(externallyOpen bool) uint8 {
	if externallyOpen {
		return PacingOpen
	}
	return PacingClosed
}

func RtbCampaignInputFromHybrid(
	meta *CampaignMeta,
	geo uint32,
	deviceMask uint8,
	categoryMask uint64,
	weight uint32,
	pacingOpen uint8,
	customerID CustomerID,
	customerBudget int64,
	dailyBudget int64,
) RtbCampaignInput {
	ctrPPM := uint32(CTRPPMUnit)
	if meta != nil && meta.CTR > 0 {
		scaled := meta.CTR * float64(CTRPPMUnit)
		if scaled > float64(math.MaxUint32) {
			ctrPPM = math.MaxUint32
		} else {
			ctrPPM = uint32(scaled)
		}
	}
	bidMicro := int64(0)
	if meta != nil {
		bidMicro = meta.BidMicro
	}
	return RtbCampaignInput{
		BidMicro:         bidMicro,
		CTRPPM:           ctrPPM,
		DeviceMask:       deviceMask,
		CategoryMask:     categoryMask,
		GeoHash:          geo,
		Weight:           weight,
		PacingOpen:       pacingOpen,
		CustomerID:       customerID,
		CustomerBudget:   customerBudget,
		DailyBudgetMicro: dailyBudget,
	}
}

func BuildRtbCatalogRowsFromHybrid(
	campaigns []*domain.Campaign,
	metaByID map[uuid.UUID]*CampaignMeta,
	inputs map[uuid.UUID]RtbCampaignInput,
) []CampaignData {
	if len(campaigns) == 0 {
		return nil
	}
	out := make([]CampaignData, 0, len(campaigns))
	for _, camp := range campaigns {
		if camp == nil {
			continue
		}
		base, ok := inputs[camp.ID]
		if !ok {
			continue
		}
		meta := metaByID[camp.ID]
		if meta != nil {
			base.BidMicro = meta.BidMicro
			if meta.CTR > 0 {
				scaled := meta.CTR * float64(CTRPPMUnit)
				if scaled > math.MaxUint32 {
					base.CTRPPM = math.MaxUint32
				} else {
					base.CTRPPM = uint32(scaled)
				}
			}
		}
		out = append(out, fanOutRtbCatalogRows(camp, base)...)
	}
	return out
}

const (
	rtbLiveGateMinParityRate   = 0.95
	rtbLiveGateMinShadowEvals  = 100
	rtbLiveGateDefaultWindow   = time.Hour
	rtbLiveGateMismatchReason  = "shadow_winner_mismatch_rate_high"
	rtbLiveGateReconcileReason = "budget_reconcile_divergence_high"
	rtbLiveGateInsufficient    = "insufficient_shadow_evals"
)
