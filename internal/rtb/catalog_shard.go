package rtb

const (
	geoShardCount       = 64
	geoShardMask        = geoShardCount - 1
	legacyGeoShardCount = 16
)

type CampaignAuctionRegistry struct {
	Count                 int
	CampaignIDs           []CampaignID
	Bids                  []int64
	CTRPPM                []uint32
	Reserves              []int64
	DailyBudgets          []int64
	PacingOpen            []uint8
	DeviceMasks           []uint8
	CategoryMasks         []uint64
	GeoHashes             []uint32
	Weights               []uint32
	BoostPPM              []uint32
	BudgetIndices         []uint32
	CustomerBudgetIndices []uint32
	DaypartMasks          []uint32
	TZOffsetSec           []int32
	ScheduleStart         []int64
	ScheduleEnd           []int64
	FreqLimits            []uint32
	FcapPrefixHash        []uint64

	GeoBucketCount int
	GeoBucketHash  []uint32
	GeoBucketStart []uint32
	GeoBucketSoA   candidateBucketSoA

	TargetBucketCount int
	TargetBucketKey   []uint64
	TargetBucketStart []uint32
	TargetBucketSoA   candidateBucketSoA

	CreativeCache         creativeCacheSoA
	CampaignCreativeStart []uint32
}

type CampaignData struct {
	ID             CampaignID
	Bid            int64
	CTRPPM         uint32
	Reserve        int64
	DailyBudget    int64
	PacingOpen     uint8
	CustomerID     CustomerID
	CustomerBudget int64
	DeviceMask     uint8
	CategoryMask   uint64
	GeoHashVal     uint32
	Weight         uint32
	BoostPPM       uint32
	Budget         int64
	DaypartMask    uint32
	TZOffsetSec    int32
	ScheduleStart  int64
	ScheduleEnd    int64
	FreqLimit      uint32
	FcapPrefixHash uint64
}

type catalogSnapshot struct {
	shards [geoShardCount]*CampaignAuctionRegistry
}
