package rtb

type CreativeID uint64

type MediaType uint8

const (
	MediaTypeDisplay MediaType = 1 << iota
	MediaTypeVideo
)

type CreativeData struct {
	ID          CreativeID
	CampaignID  CampaignID
	Bid         int64
	CTRPPM      uint32
	Weight      uint32
	MediaType   MediaType
	DurationSec uint32
	VASTXML     []byte
	VASTWire    []byte
}
