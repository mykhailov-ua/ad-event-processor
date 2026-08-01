package rtb

type CampaignID uint64

type CustomerID uint64

type BidRequest struct {
	CategoryMask   uint64
	MinBid         int64
	GeoHash        uint32
	DeviceType     uint8
	MediaTypeMask  uint8
	MaxDurationSec uint32
	DeadlineMono   int64
	DealBlock      NoBidReason
	NowUnix        int64
	FcapUserHash   uint64
	BlockedCatMask uint64
}

type AuctionResult struct {
	CampaignID CampaignID
	CreativeID CreativeID
	Price      int64
}
