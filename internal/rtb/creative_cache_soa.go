package rtb

type creativeCacheSoA struct {
	Count       int
	CreativeIDs []CreativeID
	CampaignIdx []uint32
	Bids        []int64
	CTRPPM      []uint32
	Weights     []uint32
	MediaTypes  []uint8
	DurationSec []uint32
	VASTWire    [][]byte
}

func resetCreativeCacheSoA(soa *creativeCacheSoA) {
	if soa == nil {
		return
	}
	soa.CreativeIDs = soa.CreativeIDs[:0]
	soa.CampaignIdx = soa.CampaignIdx[:0]
	soa.Bids = soa.Bids[:0]
	soa.CTRPPM = soa.CTRPPM[:0]
	soa.Weights = soa.Weights[:0]
	soa.MediaTypes = soa.MediaTypes[:0]
	soa.DurationSec = soa.DurationSec[:0]
	soa.VASTWire = soa.VASTWire[:0]
	soa.Count = 0
}
