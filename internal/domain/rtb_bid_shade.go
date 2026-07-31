package domain

type RtbBidShadeInput struct {
	GeoHash      uint32
	DeviceType   uint8
	CategoryMask uint64
	MinBidMicro  int64
}

type RtbBidShadeOutput struct {
	HasBid              bool
	CampaignID          string
	ClearingPriceMicro  int64
	RecommendedBidMicro int64
	ShadeDeltaMicro     int64
	NoBidReason         string
	SecondPriceDeltaPct float64
}
