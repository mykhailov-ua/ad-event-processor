package filter

import (
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/shard"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

var DefaultCampaignRedisKeyCatalog = domain.DefaultCampaignRedisKeyCatalog

func budgetCampaignKey(id uuid.UUID) string {
	return domain.BudgetCampaignKey(id)
}

func campaignSyncKey(id uuid.UUID) string {
	return domain.CampaignSyncKey(id)
}

func customerSyncKey(campaignID, customerID uuid.UUID) string {
	return domain.CustomerSyncKey(campaignID, customerID)
}

func budgetQuotaKey(id uuid.UUID) string {
	return domain.BudgetQuotaKey(id)
}

func fcapKeyPrefix(campaignID uuid.UUID, brandFcapKey string) string {
	return domain.FcapKeyPrefix(campaignID, brandFcapKey)
}

func dailySpendKeyPrefix(campaignID uuid.UUID) string {
	return domain.DailySpendKeyPrefix(campaignID)
}

var (
	MigrationFenceKeyPrefix = domain.MigrationFenceKeyPrefix
	BudgetFrozenKeyPrefix   = domain.BudgetFrozenKeyPrefix
)

func appendCampaignHashTag(dst []byte, id uuid.UUID) []byte {
	return domain.AppendCampaignHashTag(dst, id)
}

func campaignHashTag(id uuid.UUID) string {
	return domain.CampaignHashTag(id)
}

func crc32Castagnoli(data *uuid.UUID) uint32 {
	return domain.CRC32Castagnoli(data)
}

func BudgetCampaignKey(id uuid.UUID) string {
	return domain.BudgetCampaignKey(id)
}

func CampaignSyncKey(id uuid.UUID) string {
	return domain.CampaignSyncKey(id)
}

func PlacementBlacklistKey(campaignID uuid.UUID) string {
	return domain.PlacementBlacklistKey(campaignID)
}

func RedisClusterSlot(key string) int {
	return domain.RedisClusterSlot(key)
}

func FilterRedisOptions(addrs []string, password string, poolSize, filterTimeoutMs int) *redis.UniversalOptions {
	opts := &redis.UniversalOptions{
		Addrs:    addrs,
		Password: password,
		PoolSize: poolSize,
	}
	if filterTimeoutMs > 0 {
		d := time.Duration(filterTimeoutMs) * time.Millisecond
		opts.ReadTimeout = d
		opts.WriteTimeout = d
	}
	return opts
}

func FilterRedisReadTimeoutMs(filterTimeoutMs int) int {
	if filterTimeoutMs <= 0 {
		return 0
	}
	return filterTimeoutMs
}

type (
	Sharder           = domain.Sharder
	StaticSlotSharder = domain.StaticSlotSharder
	JumpHashSharder   = domain.JumpHashSharder
	SlotMapSnapshot   = domain.SlotMapSnapshot
)

func NewStaticSlotSharder(numBuckets int) *StaticSlotSharder {
	return domain.NewStaticSlotSharder(numBuckets)
}

func NewJumpHashSharder(numBuckets int) *JumpHashSharder {
	return domain.NewJumpHashSharder(numBuckets)
}

type slotTable = domain.SlotTable

func buildSlotTable(numBuckets int) *slotTable {
	return domain.BuildSlotTable(numBuckets)
}

const CampaignEpochKey = shard.CampaignEpochKey

var CRC32Castagnoli = domain.CRC32Castagnoli

const (
	SlotMask  = domain.SlotMask
	SlotCount = domain.SlotCount
)

var (
	CampaignSlotIndex       = domain.CampaignSlotIndex
	FilterCampaignIDsBySlot = domain.FilterCampaignIDsBySlot
)

var (
	LoadActiveSlotMap            = domain.LoadActiveSlotMap
	ReloadStaticSlotMapIfChanged = domain.ReloadStaticSlotMapIfChanged
	EncodeSlotMapReloadMessage   = domain.EncodeSlotMapReloadMessage
	DecodeSlotMapReloadMessage   = domain.DecodeSlotMapReloadMessage
)
