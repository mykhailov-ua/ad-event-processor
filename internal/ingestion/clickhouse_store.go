package ingestion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/metrics"
	"github.com/bidshard/ad-event-processor/pkg/piihash"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type tgEventPayload struct {
	TgUserID   string `json:"tg_user_id"`
	StartParam string `json:"start_param"`
	ChatType   string `json:"chat_type"`
	IsPremium  bool   `json:"is_premium"`
	Motivated  bool   `json:"motivated"`
	WidgetID   string `json:"widget_id"`
	BotID      uint64 `json:"bot_id"`
}

func isTelegramEvent(e *domain.Event) bool {
	if e == nil {
		return false
	}
	if e.Type == "tg_click" || e.Type == "tg_impression" || e.Type == "tg_conversion" {
		return true
	}
	if len(e.Payload) > 0 {
		if bytes.Contains(e.Payload, []byte("tg_user_id")) || bytes.Contains(e.Payload, []byte("bot_id")) {
			return true
		}
	}
	return false
}

func isFraudTelemetry(e *domain.Event) bool {
	if e == nil {
		return false
	}
	if e.Type == fraudAggregateEventType {
		return false
	}
	return e.GhostEvent || e.FraudReason != "" || e.FraudScore > 0
}

const fraudAggregateEventType = "fraud_aggregate"

func isFraudAggregateSpike(e *domain.Event) bool {
	return e != nil && e.Type == fraudAggregateEventType
}

func fraudGhostFlag(e *domain.Event) uint8 {
	if e.GhostEvent {
		return 1
	}
	return 0
}

func fraudAggregateFields(e *domain.Event) (uint64, uint32) {
	var count uint64
	var windowMs uint32
	if e.ClickID != "" {
		if n, err := strconv.ParseUint(e.ClickID, 10, 64); err == nil {
			count = n
		}
	}
	if e.UserID != "" {
		if n, err := strconv.ParseUint(e.UserID, 10, 32); err == nil {
			windowMs = uint32(n)
		}
	}
	return count, windowMs
}

var slicePool = sync.Pool{
	New: func() any {
		s := make([]*domain.Event, 0, 50000)
		return &s
	},
}

type ClickHouseStore struct {
	conn          driver.Conn
	writeTimeout  time.Duration
	spool         *CHSpool
	chGate        *ProcessorChGate
	piiHasher     *piihash.Hasher
	ctx           context.Context
	cancel        context.CancelFunc
	replayWg      sync.WaitGroup
	replayRunning atomic.Bool
}

func NewClickHouseStore(conn driver.Conn, writeTimeout time.Duration, spoolDir string, spoolCfg CHSpoolConfig, chGate *ProcessorChGate) *ClickHouseStore {
	ctx, cancel := context.WithCancel(context.Background())
	chStore := &ClickHouseStore{
		conn:         conn,
		writeTimeout: writeTimeout,
		chGate:       chGate,
		ctx:          ctx,
		cancel:       cancel,
	}
	if spoolDir != "" {
		spool, err := OpenCHSpoolWithConfig(spoolDir, spoolCfg)
		if err != nil {
			slog.Error("failed to open clickhouse spool", "error", err, "dir", spoolDir)
		} else {
			chStore.spool = spool
			spool.StartAsyncFlusher(20 * time.Millisecond)
			chStore.startSpoolReplayer()
		}
	}
	return chStore
}

func (chStore *ClickHouseStore) StoreBatch(ctx context.Context, events []*domain.Event) error {
	if len(events) == 0 {
		return nil
	}

	if chStore.chGate != nil {
		if err := chStore.chGate.Acquire(ctx); err != nil {
			return err
		}
		defer chStore.chGate.Release()
	}

	token := chStore.getDeduplicationToken(ctx, events)
	var err error
	waitTime := InitialWait

	for i := 0; i <= MaxRetries; i++ {
		dbCtx, cancel := context.WithTimeout(ctx, chStore.writeTimeout)
		err = chStore.insertToClickHouse(dbCtx, events)
		cancel()

		if err == nil {
			return nil
		}

		if i < MaxRetries {
			timer := time.NewTimer(waitTime)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
				waitTime *= 2
				if waitTime > MaxWait {
					waitTime = MaxWait
				}
			}
		}
	}

	if chStore.spool == nil {
		metrics.DBWriteErrors.WithLabelValues("clickhouse").Inc()
		return err
	}

	if spoolErr := chStore.spool.AppendDurably(token, events); spoolErr != nil {
		metrics.DBWriteErrors.WithLabelValues("clickhouse_spool").Inc()
		return fmt.Errorf("clickhouse write failed and spool append failed: ch=%w spool=%w", err, spoolErr)
	}

	metrics.CHSpoolAppendTotal.Inc()
	slog.Warn("clickhouse unavailable, batch spooled to mmap WAL", "events", len(events), "token", token)
	return nil
}

func (chStore *ClickHouseStore) startSpoolReplayer() {
	if chStore.spool == nil || chStore.replayRunning.Swap(true) {
		return
	}
	chStore.replayWg.Add(1)
	go func() {
		defer chStore.replayWg.Done()
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-chStore.ctx.Done():
				return
			case <-ticker.C:
				chStore.replaySpoolOnce()
			}
		}
	}()
}

func (chStore *ClickHouseStore) replaySpoolOnce() {
	if chStore.spool == nil {
		return
	}
	records, err := chStore.spool.Scan()
	if err != nil || len(records) == 0 {
		return
	}

	rec := records[0]
	ctx, cancel := context.WithTimeout(chStore.ctx, chStore.writeTimeout)
	ctx = context.WithValue(ctx, domain.DeduplicationTokenKey, rec.DedupToken)
	insertErr := chStore.insertToClickHouse(ctx, rec.Events)
	cancel()
	if insertErr != nil {
		for _, e := range rec.Events {
			domain.EventPool.Put(e)
		}
		return
	}
	for _, e := range rec.Events {
		domain.EventPool.Put(e)
	}
	if err := chStore.spool.ReleaseRecord(rec); err != nil {
		slog.Error("failed to release ch spool record", "error", err, "offset", rec.EndOffset)
		return
	}
	metrics.CHSpoolReplayTotal.Inc()
}

func (chStore *ClickHouseStore) RecoverSpool(ctx context.Context) error {
	if chStore.spool == nil {
		return nil
	}
	for {
		records, err := chStore.spool.Scan()
		if err != nil {
			return err
		}
		if len(records) == 0 {
			return nil
		}
		rec := records[0]
		replayCtx, cancel := context.WithTimeout(ctx, chStore.writeTimeout)
		replayCtx = context.WithValue(replayCtx, domain.DeduplicationTokenKey, rec.DedupToken)
		insertErr := chStore.insertToClickHouse(replayCtx, rec.Events)
		cancel()
		if insertErr != nil {
			for _, e := range rec.Events {
				domain.EventPool.Put(e)
			}
			return insertErr
		}
		for _, e := range rec.Events {
			domain.EventPool.Put(e)
		}
		if err := chStore.spool.ReleaseRecord(rec); err != nil {
			return err
		}
		metrics.CHSpoolReplayTotal.Inc()
	}
}

func (chStore *ClickHouseStore) getDeduplicationToken(ctx context.Context, events []*domain.Event) string {
	if token, ok := ctx.Value(domain.DeduplicationTokenKey).(string); ok && token != "" {
		return token
	}
	if len(events) == 0 {
		return ""
	}
	h := sha256.New()
	for _, e := range events {
		h.Write([]byte(e.ClickID))
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], uint64(e.CreatedAt.UnixNano()))
		h.Write(buf[:])
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (chStore *ClickHouseStore) insertTable(ctx context.Context, table string, evts []*domain.Event, isFraud bool) error {
	if !database.ValidClickHouseIdentifier(table) {
		return fmt.Errorf("invalid clickhouse table name: %q", table)
	}
	start := time.Now()

	token := chStore.getDeduplicationToken(ctx, evts)
	query := "INSERT INTO " + table
	if token != "" {
		if !database.ValidCHHexToken(token) {
			return fmt.Errorf("invalid clickhouse deduplication token")
		}
		query = query + " SETTINGS insert_deduplicate=1, insert_deduplication_token='" + token + "'"
	}

	batch, err := chStore.conn.PrepareBatch(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare batch %s: %w", table, err)
	}

	for _, e := range evts {
		pii := hashEventPII(chStore.piiHasher, e)
		switch {
		case table == "fraud_aggregate_spikes":
			count, windowMs := fraudAggregateFields(e)
			err = batch.Append(
				piihash.FixedString16(pii.subnetHash),
				e.PlacementID,
				e.FraudReason,
				count,
				windowMs,
				e.CreatedAt,
			)
		case isFraud:
			err = batch.Append(
				e.ClickID,
				e.CampaignID,
				piihash.FixedString16(pii.userIDHash),
				e.Type,
				piihash.FixedString16(pii.ipHash),
				piihash.FixedString16(pii.uaHash),
				pii.saltVersion,
				unsafeString(e.Payload),
				e.FraudReason,
				e.FraudScore,
				fraudGhostFlag(e),
				e.CreatedAt,
			)
		case table == "clicks":
			err = batch.Append(
				e.ClickID,
				e.CampaignID,
				e.PlacementID,
				piihash.FixedString16(pii.ipHash),
				piihash.FixedString16(pii.uaHash),
				pii.saltVersion,
				e.TLSHash,
				unsafeString(e.Payload),
				e.CreatedAt,
			)
		case table == "tg_events_raw":
			var p tgEventPayload
			if len(e.Payload) > 0 {
				_ = json.Unmarshal(e.Payload, &p)
			}
			var premium uint8
			if p.IsPremium {
				premium = 1
			}
			var motivated uint8
			if p.Motivated {
				motivated = 1
			}
			err = batch.Append(
				e.ClickID,
				e.CampaignID,
				p.TgUserID,
				p.StartParam,
				p.ChatType,
				premium,
				motivated,
				p.WidgetID,
				p.BotID,
				unsafeString(e.Payload),
				e.CreatedAt,
				e.Type,
			)
		default:
			err = batch.Append(
				e.ClickID,
				e.CampaignID,
				e.PlacementID,
				piihash.FixedString16(pii.ipHash),
				piihash.FixedString16(pii.uaHash),
				pii.saltVersion,
				unsafeString(e.Payload),
				e.CreatedAt,
			)
		}
		if err != nil {
			return fmt.Errorf("append %s: %w", table, err)
		}
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("send %s: %w", table, err)
	}

	duration := time.Since(start).Seconds()
	metrics.DBWriteDuration.WithLabelValues("clickhouse").Observe(duration)

	return nil
}

func (chStore *ClickHouseStore) insertToClickHouse(ctx context.Context, events []*domain.Event) error {
	start := time.Now()

	pImps := slicePool.Get().(*[]*domain.Event)
	pClicks := slicePool.Get().(*[]*domain.Event)
	pConvs := slicePool.Get().(*[]*domain.Event)
	pFraud := slicePool.Get().(*[]*domain.Event)
	pAgg := slicePool.Get().(*[]*domain.Event)
	pTgEvents := slicePool.Get().(*[]*domain.Event)

	defer func() {
		for i := range *pImps {
			(*pImps)[i] = nil
		}
		*pImps = (*pImps)[:0]

		for i := range *pClicks {
			(*pClicks)[i] = nil
		}
		*pClicks = (*pClicks)[:0]

		for i := range *pConvs {
			(*pConvs)[i] = nil
		}
		*pConvs = (*pConvs)[:0]

		for i := range *pFraud {
			(*pFraud)[i] = nil
		}
		*pFraud = (*pFraud)[:0]

		for i := range *pAgg {
			(*pAgg)[i] = nil
		}
		*pAgg = (*pAgg)[:0]

		for i := range *pTgEvents {
			(*pTgEvents)[i] = nil
		}
		*pTgEvents = (*pTgEvents)[:0]

		if cap(*pImps) <= 100000 {
			slicePool.Put(pImps)
		}
		if cap(*pClicks) <= 100000 {
			slicePool.Put(pClicks)
		}
		if cap(*pConvs) <= 100000 {
			slicePool.Put(pConvs)
		}
		if cap(*pFraud) <= 100000 {
			slicePool.Put(pFraud)
		}
		if cap(*pAgg) <= 100000 {
			slicePool.Put(pAgg)
		}
		if cap(*pTgEvents) <= 100000 {
			slicePool.Put(pTgEvents)
		}
	}()

	imps := *pImps
	clicks := *pClicks
	convs := *pConvs
	fraud := *pFraud
	agg := *pAgg
	tgEvents := *pTgEvents

	for i := range events {
		e := events[i]
		if isTelegramEvent(e) {
			tgEvents = append(tgEvents, e)
			continue
		}
		if isFraudAggregateSpike(e) {
			agg = append(agg, e)
			continue
		}
		if isFraudTelemetry(e) {
			fraud = append(fraud, e)
			continue
		}

		switch e.Type {
		case "impression":
			imps = append(imps, e)
		case "click":
			clicks = append(clicks, e)
		case "conversion":
			convs = append(convs, e)
		}
	}

	*pImps, *pClicks, *pConvs, *pFraud, *pAgg, *pTgEvents = imps, clicks, convs, fraud, agg, tgEvents

	var wg sync.WaitGroup
	var firstErr error
	var errMu sync.Mutex

	setErr := func(err error) {
		if err == nil {
			return
		}
		errMu.Lock()
		if firstErr == nil {
			firstErr = err
		}
		errMu.Unlock()
	}

	runInsert := func(table string, evts []*domain.Event, isFraud bool) {
		if len(evts) == 0 {
			return
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := chStore.insertTable(ctx, table, evts, isFraud)
			setErr(err)
		}()
	}

	runInsert("impressions", imps, false)
	runInsert("clicks", clicks, false)
	runInsert("conversions", convs, false)
	runInsert("fraud_events", fraud, true)
	runInsert("fraud_aggregate_spikes", agg, false)
	runInsert("tg_events_raw", tgEvents, false)

	wg.Wait()
	if firstErr != nil {
		return firstErr
	}

	duration := time.Since(start).Seconds()
	metrics.DBWriteDuration.WithLabelValues("clickhouse").Observe(duration)

	return nil
}

func (chStore *ClickHouseStore) Close() error {
	chStore.cancel()
	chStore.replayWg.Wait()
	if chStore.spool != nil {
		_ = chStore.spool.Close()
	}
	return chStore.conn.Close()
}

func (chStore *ClickHouseStore) SetPIIHasher(h *piihash.Hasher) {
	if chStore != nil {
		chStore.piiHasher = h
	}
}

func (chStore *ClickHouseStore) PIIHasher() *piihash.Hasher {
	if chStore == nil {
		return nil
	}
	return chStore.piiHasher
}

func (chStore *ClickHouseStore) SetChGate(gate *ProcessorChGate) {
	chStore.chGate = gate
}

func (chStore *ClickHouseStore) SetSpool(spool *CHSpool) {
	chStore.spool = spool
}

func (chStore *ClickHouseStore) Spool() *CHSpool {
	return chStore.spool
}
