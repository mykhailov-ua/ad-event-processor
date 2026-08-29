package rtb

import (
	"time"
	"unsafe"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/openrtb"

	"github.com/google/uuid"
)

type CampaignSource interface {
	ActiveCampaigns() []*domain.Campaign
	GetCampaign(id uuid.UUID) (*domain.Campaign, bool)
}

type FcapSnapshotProvider interface {
	GetFcapRtbSnapshot() *FcapSnapshot
	RTBEmergencyBreaker() bool
	FraudBoosts() *FraudBoostSnapshot
}

type GeoAnonLookup interface {
	IsAnonymous(ip string) (bool, error)
	GetCountry(ip string) (string, error)
}

type CampaignWeighter interface {
	WeightFor(id uuid.UUID) uint32
	UpdateCampaigns(metas []*CampaignMeta, secondsElapsed, totalSeconds int64)
}

type FraudBoostSnapshot struct {
	Boosts map[uuid.UUID]uint8
}

type CampaignMeta struct {
	ID                uuid.UUID
	BidMicro          int64
	CTR               float64
	RemainingBudget   int64
	TotalBudget       int64
	PeakTrafficFactor float64
}

func remainingBudgetMicro(c *domain.Campaign) int64 {
	if c == nil {
		return 0
	}
	if c.BudgetLimit <= 0 {
		return 0
	}
	rem := c.BudgetLimit - c.CurrentSpend
	if rem < 0 {
		return 0
	}
	return rem
}

func appendDate(dst []byte, t time.Time) []byte {
	y, m, d := t.Date()
	dst = append(dst, byte('0'+y/1000%10), byte('0'+y/100%10), byte('0'+y/10%10), byte('0'+y%10))
	dst = append(dst, byte('0'+int(m)/10), byte('0'+int(m)%10))
	dst = append(dst, byte('0'+d/10), byte('0'+d%10))
	return dst
}

func UnsafeString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func ExchangeLogMetaFromSplit(h openrtb.OpenRTB26Hot, c openrtb.OpenRTB26Cold) RtbExchangeLogMeta {
	var m RtbExchangeLogMeta
	if c.SiteDomainLen > 0 {
		m.inventoryLen = uint8(copy(m.inventory[:], c.SiteDomain[:c.SiteDomainLen]))
	} else if c.AppBundleLen > 0 {
		m.inventoryLen = uint8(copy(m.inventory[:], c.AppBundle[:c.AppBundleLen]))
	}
	if h.GeoCountryLen >= 2 {
		copy(m.geoCountry[:], h.GeoCountry[:2])
	}
	if c.DeviceOSLen > 0 {
		m.deviceOSLen = uint8(copy(m.deviceOS[:], c.DeviceOS[:c.DeviceOSLen]))
	}
	if c.SourceTIDLen > 0 {
		m.sourceTIDLen = uint8(copy(m.sourceTID[:], c.SourceTID[:c.SourceTIDLen]))
	}
	if c.EIDSourceLen > 0 {
		m.eidSourceLen = uint8(copy(m.eidSource[:], c.EIDSource[:c.EIDSourceLen]))
	}
	if c.AppVerLen > 0 {
		m.appVerLen = uint8(copy(m.appVer[:], c.AppVer[:c.AppVerLen]))
	}
	m.connectionType = h.ConnectionType
	m.pmpPrivate = h.PMPPrivate
	m.deviceLMT = h.DeviceLMT
	m.viewabilityPPM = h.MetricValuePPM
	if h.Flags&openrtb.OpenRTB26FlagVideo != 0 {
		m.mediaW = h.VideoW
		m.mediaH = h.VideoH
	} else {
		m.mediaW = h.BannerW
		m.mediaH = h.BannerH
	}
	return m
}
