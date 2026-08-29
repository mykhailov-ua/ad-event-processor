package rtb

import (
	"context"
	"encoding/binary"
	"hash/crc32"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var ErrInvalidRtbBudgetAuthority = domain.ErrInvalidRtbBudgetAuthority

func BudgetAuthorityFromSettings(cfg *config.Config, setting string) BudgetAuthority {
	if cfg == nil || !cfg.RtbEnabled() {
		return BudgetAuthorityShadow
	}
	if !cfg.RtbLiveSelectsCampaign() {
		return BudgetAuthorityShadow
	}
	raw := strings.TrimSpace(setting)
	if raw == "" {
		raw = cfg.RtbBudgetAuthority
	}
	switch strings.ToLower(raw) {
	case "rtb":
		return BudgetAuthorityRTB
	case "lua", "redis", "":
		return BudgetAuthorityRedis
	default:
		return BudgetAuthorityRedis
	}
}

func RtbSkipLuaBudgetDebit(cfg *config.Config, setting string) bool {
	return BudgetAuthorityFromSettings(cfg, setting) == BudgetAuthorityRTB
}

func NormalizeRtbBudgetAuthoritySetting(v string) (string, error) {
	return domain.NormalizeRtbBudgetAuthoritySetting(v)
}

func BoostPPMFromUint8(boost uint8) uint32 {
	if boost == 0 {
		return CTRPPMUnit
	}
	return CTRPPMUnit + uint32(boost)*10_000
}

type BudgetAuthority uint8

const (
	BudgetAuthorityRedis BudgetAuthority = iota
	BudgetAuthorityRTB
	BudgetAuthorityShadow
)

type RtbCampaignInput struct {
	BidMicro         int64
	CTRPPM           uint32
	ReserveMicro     int64
	DailyBudgetMicro int64
	PacingOpen       uint8
	CustomerID       CustomerID
	CustomerBudget   int64
	DeviceMask       uint8
	CategoryMask     uint64
	GeoHash          uint32
	Weight           uint32
	BoostPPM         uint32
}

type RtbTargetingInput struct {
	GeoHash             uint32
	DeviceType          uint8
	CategoryMask        uint64
	PublisherFloorMicro int64
	DealIDLen           uint8
	DealIDBuf           [64]byte
	SeatCount           uint8
	DeadlineMono        int64
	DealBlock           NoBidReason
	Schain              SchainNodes
	SchainCount         uint8
	FcapUserHash        uint64
	ConnectionType      uint8
	ViewabilityPPM      uint32
	PMPPrivate          uint8
	DeviceLMT           uint8
	BlockedCatMask      uint64
}

func CampaignIDFromUUID(id uuid.UUID) CampaignID {
	return CampaignID(binary.BigEndian.Uint64(id[:8]))
}

func GeoHashFromCountry(country string) uint32 {
	if country == "" {
		return 0
	}
	return crc32.ChecksumIEEE([]byte(country))
}

func GeoHashFromCountryBytes(country []byte) uint32 {
	if len(country) == 0 {
		return 0
	}
	return crc32.ChecksumIEEE(country)
}

func DeviceMaskFromType(deviceType []byte) uint8 {
	switch len(deviceType) {
	case 6:
		if deviceType[0] == 'm' && deviceType[1] == 'o' && deviceType[2] == 'b' &&
			deviceType[3] == 'i' && deviceType[4] == 'l' && deviceType[5] == 'e' {
			return 2
		}
		if deviceType[0] == 't' && deviceType[1] == 'a' && deviceType[2] == 'b' &&
			deviceType[3] == 'l' && deviceType[4] == 'e' && deviceType[5] == 't' {
			return 4
		}
	case 7:
		if deviceType[0] == 'd' && deviceType[1] == 'e' && deviceType[2] == 's' &&
			deviceType[3] == 'k' && deviceType[4] == 't' && deviceType[5] == 'o' &&
			deviceType[6] == 'p' {
			return 1
		}
	}
	return 1
}

func BidRequestFromEvent(evt *domain.Event, targeting RtbTargetingInput) BidRequest {
	fcapUserHash := targeting.FcapUserHash
	if fcapUserHash == 0 && evt != nil && evt.UserID != "" {
		fcapUserHash = hashUserID(evt.UserID)
	}
	return BidRequest{
		CategoryMask:   targeting.CategoryMask,
		MinBid:         targeting.PublisherFloorMicro,
		GeoHash:        targeting.GeoHash,
		DeviceType:     targeting.DeviceType,
		DeadlineMono:   targeting.DeadlineMono,
		DealBlock:      targeting.DealBlock,
		NowUnix:        time.Now().UTC().Unix(),
		FcapUserHash:   fcapUserHash,
		BlockedCatMask: targeting.BlockedCatMask,
	}
}

func hashUserID(userID string) uint64 {
	if userID == "" {
		return 0
	}
	return HashBytes64(unsafe.Slice(unsafe.StringData(userID), len(userID)))
}

func hashUserIDBytes(userID []byte) uint64 {
	if len(userID) == 0 {
		return 0
	}
	return HashBytes64(userID)
}

func CampaignDataFromDomain(camp *domain.Campaign, input RtbCampaignInput) CampaignData {
	remaining := camp.BudgetLimit - camp.CurrentSpend
	if remaining < 0 {
		remaining = 0
	}
	daypartMask, tzOffset, startUnix, endUnix := scheduleFieldsFromCampaign(camp)
	var freqLimit uint32
	if camp.FreqLimit > 0 {
		freqLimit = uint32(camp.FreqLimit)
	}
	var fcapPrefixHash uint64
	if camp.FcapKeyPrefix != "" {
		fcapPrefixHash = HashBytes64([]byte(camp.FcapKeyPrefix))
	}
	return CampaignData{
		ID:             CampaignIDFromUUID(camp.ID),
		Bid:            input.BidMicro,
		CTRPPM:         input.CTRPPM,
		Reserve:        input.ReserveMicro,
		DailyBudget:    input.DailyBudgetMicro,
		PacingOpen:     input.PacingOpen,
		CustomerID:     input.CustomerID,
		CustomerBudget: input.CustomerBudget,
		DeviceMask:     input.DeviceMask,
		CategoryMask:   input.CategoryMask,
		GeoHashVal:     input.GeoHash,
		Weight:         input.Weight,
		BoostPPM:       input.BoostPPM,
		Budget:         remaining,
		DaypartMask:    daypartMask,
		TZOffsetSec:    tzOffset,
		ScheduleStart:  startUnix,
		ScheduleEnd:    endUnix,
		FreqLimit:      freqLimit,
		FcapPrefixHash: fcapPrefixHash,
	}
}

func scheduleFieldsFromCampaign(camp *domain.Campaign) (mask uint32, tzOffset int32, startUnix, endUnix int64) {
	if camp == nil {
		return 0, 0, 0, 0
	}
	mask = DaypartMaskFromHours(camp.DaypartHours)
	now := time.Now().UTC()
	if camp.Location != nil {
		_, off := now.In(camp.Location).Zone()
		tzOffset = int32(off)
	}
	if camp.StartAt != nil {
		startUnix = camp.StartAt.Unix()
	}
	if camp.EndAt != nil {
		endUnix = camp.EndAt.Unix()
	}
	return mask, tzOffset, startUnix, endUnix
}

func BuildRtbCatalogRows(campaigns []*domain.Campaign, inputs map[uuid.UUID]RtbCampaignInput) []CampaignData {
	if len(campaigns) == 0 {
		return nil
	}
	out := make([]CampaignData, 0, len(campaigns))
	for _, camp := range campaigns {
		if camp == nil {
			continue
		}
		input, ok := inputs[camp.ID]
		if !ok {
			continue
		}
		out = append(out, fanOutRtbCatalogRows(camp, input)...)
	}
	return out
}

const (
	rtbBudgetMirrorRingCapacity = 4096
	rtbBudgetMirrorRingMask     = rtbBudgetMirrorRingCapacity - 1
	rtbBudgetMirrorRingUsable   = rtbBudgetMirrorRingCapacity - 1
	rtbBudgetMirrorFlushBatch   = 128
	rtbBudgetMirrorFlushEvery   = 2 * time.Second
)

type rtbBudgetMirrorSlot struct {
	ready      atomic.Uint32
	priceMicro int64
	campaignID CampaignID
}

type RtbBudgetMirrorWriter struct {
	_           [64]byte
	writeCursor uint64
	_           [64]byte
	allocCursor uint64
	_           [64]byte
	readCursor  uint64
	_           [64]byte
	slots       [rtbBudgetMirrorRingCapacity]rtbBudgetMirrorSlot

	catalog     *RtbCatalog
	registry    CampaignSource
	redisShards []redis.UniversalClient
	sharder     domain.Sharder

	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewRtbBudgetMirrorWriter(catalog *RtbCatalog, registry CampaignSource, redisShards []redis.UniversalClient, sharder domain.Sharder) *RtbBudgetMirrorWriter {
	if catalog == nil || registry == nil || len(redisShards) == 0 || sharder == nil {
		return nil
	}
	w := &RtbBudgetMirrorWriter{
		catalog:     catalog,
		registry:    registry,
		redisShards: redisShards,
		sharder:     sharder,
		stopCh:      make(chan struct{}),
	}
	w.wg.Add(1)
	go w.worker()
	SetBudgetSpendMirror(w)
	return w
}

func (w *RtbBudgetMirrorWriter) RecordSpend(campaignID CampaignID, _ uint32, priceMicro int64) {
	if w == nil || priceMicro <= 0 {
		return
	}
	for {
		alloc := atomic.LoadUint64(&w.allocCursor)
		read := atomic.LoadUint64(&w.readCursor)
		if alloc-read >= rtbBudgetMirrorRingUsable {
			return
		}
		if !atomic.CompareAndSwapUint64(&w.allocCursor, alloc, alloc+1) {
			continue
		}
		idx := alloc & rtbBudgetMirrorRingMask
		slot := &w.slots[idx]
		if slot.ready.Load() != 0 {
			return
		}
		slot.campaignID = campaignID
		slot.priceMicro = priceMicro
		slot.ready.Store(1)
		atomic.StoreUint64(&w.writeCursor, alloc+1)
		return
	}
}

func (w *RtbBudgetMirrorWriter) Close() {
	if w == nil {
		return
	}
	SetBudgetSpendMirror(nil)
	close(w.stopCh)
	w.wg.Wait()
}

func (w *RtbBudgetMirrorWriter) worker() {
	defer w.wg.Done()
	ticker := time.NewTicker(rtbBudgetMirrorFlushEvery)
	defer ticker.Stop()
	for {
		select {
		case <-w.stopCh:
			w.flush(context.Background())
			return
		case <-ticker.C:
			w.flush(context.Background())
		}
	}
}

func (w *RtbBudgetMirrorWriter) flush(ctx context.Context) {
	batch := 0
	for batch < rtbBudgetMirrorFlushBatch {
		read := atomic.LoadUint64(&w.readCursor)
		write := atomic.LoadUint64(&w.writeCursor)
		if read >= write {
			return
		}
		idx := read & rtbBudgetMirrorRingMask
		slot := &w.slots[idx]
		if slot.ready.Load() != 1 {
			return
		}
		w.applyDebit(ctx, slot.campaignID, slot.priceMicro)
		slot.ready.Store(0)
		atomic.StoreUint64(&w.readCursor, read+1)
		batch++
	}
}

func (w *RtbBudgetMirrorWriter) applyDebit(ctx context.Context, campID CampaignID, priceMicro int64) {
	uid, ok := w.catalog.UUIDForWinner(campID)
	if !ok {
		return
	}
	camp, ok := w.registry.GetCampaign(uid)
	if !ok || camp == nil || camp.BudgetCampaignKey == "" {
		return
	}
	shard := w.sharder.GetShard(uid)
	if shard < 0 || shard >= len(w.redisShards) {
		return
	}
	redisClient := w.redisShards[shard]
	if redisClient == nil {
		return
	}
	_ = redisClient.DecrBy(ctx, camp.BudgetCampaignKey, priceMicro).Err()
}

var _ BudgetSpendMirror = (*RtbBudgetMirrorWriter)(nil)
