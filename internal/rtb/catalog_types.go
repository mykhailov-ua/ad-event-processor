package rtb

// CampaignID is a fixed-width campaign identifier used on the bid hot path.
type CampaignID uint64

// CustomerID is a fixed-width advertiser key for shared customer budget pools.
type CustomerID uint64

// BidRequest carries targeting fields for a single bid in cache-friendly field order.
type BidRequest struct {
	CategoryMask   uint64
	MinBid         int64
	GeoHash        uint32
	DeviceType     uint8
	MediaTypeMask  uint8
	MaxDurationSec uint32
	DeadlineMono   int64
	DealBlock      NoBidReason // pre-validated PMP gate; NoBidNone when absent or passing
	NowUnix        int64       // cached UTC wall time for schedule gates
	FcapUserHash   uint64      // FNV-1a of user_id; 0 skips freq-cap pre-check
}

// AuctionResult carries the clearing outcome without heap allocation on the hot path.
type AuctionResult struct {
	CampaignID CampaignID
	CreativeID CreativeID
	Price      int64
}
