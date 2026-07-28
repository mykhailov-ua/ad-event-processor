package rtb

// CreativeID is a fixed-width creative identifier on the bid hot path.
type CreativeID uint64

// MediaType classifies inventory for video vs display auction filters.
type MediaType uint8

const (
	MediaTypeDisplay MediaType = 1 << iota
	MediaTypeVideo
)

const invalidCreativeIdx = ^uint32(0)

// CreativeData is the cold-path input used when management sync rebuilds the creative cache.
type CreativeData struct {
	ID          CreativeID
	CampaignID  CampaignID
	Bid         int64
	CTRPPM      uint32
	Weight      uint32
	MediaType   MediaType
	DurationSec uint32
	VASTXML     []byte // raw VAST 4.2; parsed on rebuild only
	VASTWire    []byte // optional pre-serialized vtproto; skips XML when set
}
