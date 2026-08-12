package ingestion

import "github.com/bidshard/ad-event-processor/internal/openrtb"

const (
	openrtb26RequestIDMax    = 64
	openrtb26ImpIDMax        = 32
	openrtb26DealIDMax       = 32
	openrtb26IFAMax          = 36
	openrtb26SourceTIDMax    = 128
	openrtb26SitePageMax     = 256
	openrtb26AppVerMax       = 32
	openrtb26EIDSourceMax    = 64
	openrtb26EIDUIDMax       = 128
	openrtb26MetricTypeMax   = 24
	openrtb26MetricVendorMax = 32
	openrtb26BlocklistMax    = 4
	openrtb26BCatItemMax     = 8
	openrtb26BAdvItemMax     = 64
	openrtb26ImpMax          = 10
	openrtb26SeatIDMax       = 16
)

const (
	impSlotFlagBanner uint8 = 1 << iota
	impSlotFlagVideo
	impSlotFlagAudio
	impSlotFlagNative
	impSlotFlagSecure
)

type OpenRTB26ImpSlot struct {
	ImpID             [openrtb26ImpIDMax]byte
	ImpIDLen          uint8
	BidFloorMicro     int64
	DealID            [openrtb26DealIDMax]byte
	DealIDLen         uint8
	DealBidFloorMicro int64
	BannerW           uint16
	BannerH           uint16
	VideoW            uint16
	VideoH            uint16
	MaxDurationSec    uint32
	Flags             uint8
	WSeat             [openrtb26SeatMax][openrtb26SeatIDMax]byte
	WSeatLen          [openrtb26SeatMax]uint8
	WSeatCount        uint8
}

type OpenRTB26Hot struct {
	Flags             uint64
	BidFloorMicro     int64
	DealBidFloorMicro int64
	DeviceType        uint8
	CategoryMask      uint64
	SeatCount         uint8
	TmaxMs            int32
	MaxDurationSec    uint32
	ImpCount          uint8
	RequestID         [openrtb26RequestIDMax]byte
	RequestIDLen      uint8
	ImpID             [openrtb26ImpIDMax]byte
	ImpIDLen          uint8
	GeoCountry        [openrtb26GeoCountryMax]byte
	GeoCountryLen     uint8
	BannerW           uint16
	BannerH           uint16
	VideoW            uint16
	VideoH            uint16
	DealID            [openrtb26DealIDMax]byte
	DealIDLen         uint8
	FcapUserHash      uint64
	ConnectionType    uint8
	PMPPrivate        uint8
	DeviceLMT         uint8
	MetricValuePPM    uint32
	EIDCount          uint8
	OK                bool
}

type OpenRTB26Cold struct {
	UserID          [openrtb26UserIDMax]byte
	UserIDLen       uint8
	GeoRegion       [openrtb26RegionMax]byte
	GeoRegionLen    uint8
	SiteDomain      [openrtb26DomainMax]byte
	SiteDomainLen   uint8
	AppBundle       [openrtb26BundleMax]byte
	AppBundleLen    uint8
	DeviceOS        [openrtb26OSMax]byte
	DeviceOSLen     uint8
	DeviceLang      [openrtb26LangMax]byte
	DeviceLangLen   uint8
	BuyerUID        [openrtb26BuyerUIDMax]byte
	BuyerUIDLen     uint8
	DeviceIFA       [openrtb26IFAMax]byte
	DeviceIFALen    uint8
	SourceTID       [openrtb26SourceTIDMax]byte
	SourceTIDLen    uint8
	SitePage        [openrtb26SitePageMax]byte
	SitePageLen     uint8
	AppVer          [openrtb26AppVerMax]byte
	AppVerLen       uint8
	EIDSource       [openrtb26EIDSourceMax]byte
	EIDSourceLen    uint8
	EIDUID          [openrtb26EIDUIDMax]byte
	EIDUIDLen       uint8
	MetricType      [openrtb26MetricTypeMax]byte
	MetricTypeLen   uint8
	MetricVendor    [openrtb26MetricVendorMax]byte
	MetricVendorLen uint8
	BCat            [openrtb26BlocklistMax][openrtb26BCatItemMax]byte
	BCatLen         [openrtb26BlocklistMax]uint8
	BCatCount       uint8
	BAdv            [openrtb26BlocklistMax][openrtb26BAdvItemMax]byte
	BAdvLen         [openrtb26BlocklistMax]uint8
	BAdvCount       uint8
	BApp            [openrtb26BlocklistMax][openrtb26BundleMax]byte
	BAppLen         [openrtb26BlocklistMax]uint8
	BAppCount       uint8
	BCatMask        uint64
	BSeat           [openrtb26SeatMax][openrtb26SeatIDMax]byte
	BSeatLen        [openrtb26SeatMax]uint8
	BSeatCount      uint8
	Imps            [openrtb26ImpMax]OpenRTB26ImpSlot
	ImpSlots        uint8
	Schain          SchainNodes
}

type OpenRTB26Parsed struct {
	OpenRTB26Hot
	OpenRTB26Cold
}

func (p OpenRTB26Parsed) ExchangeReady(cfg openrtb.ExchangeConfig) bool {
	return exchangeReady(&p.OpenRTB26Hot, &p.OpenRTB26Cold, cfg)
}

func (h OpenRTB26Hot) ExchangeReady(cfg openrtb.ExchangeConfig) bool {
	return exchangeReady(&h, nil, cfg)
}

func exchangeReady(h *OpenRTB26Hot, cold *OpenRTB26Cold, cfg openrtb.ExchangeConfig) bool {
	if h == nil || !h.OK || h.RequestIDLen == 0 || h.ImpCount == 0 {
		return false
	}
	if cfg.MultiImpMax > 0 && int(h.ImpCount) > cfg.MultiImpMax {
		return false
	}
	if int(h.ImpCount) > openrtb26ImpMax {
		return false
	}
	inv := 0
	if h.Flags&openrtb26FlagSite != 0 {
		inv++
	}
	if h.Flags&openrtb26FlagApp != 0 {
		inv++
	}
	if h.Flags&openrtb26FlagDOOH != 0 {
		return false
	}
	if inv != 1 {
		return false
	}
	hasIP := h.Flags&(openrtb26FlagDeviceIP|openrtb26FlagDeviceIPv6) != 0
	if !hasIP || h.Flags&openrtb26FlagDeviceUA == 0 {
		return false
	}
	if cold != nil && cold.ImpSlots > 0 {
		if int(cold.ImpSlots) < int(h.ImpCount) {
			return false
		}
		for i := 0; i < int(h.ImpCount); i++ {
			if !impSlotExchangeReady(&cold.Imps[i]) {
				return false
			}
		}
		return true
	}
	if h.ImpIDLen == 0 {
		return false
	}
	if h.Flags&(openrtb26FlagBanner|openrtb26FlagVideo) == 0 {
		return false
	}
	if h.Flags&(openrtb26FlagAudio|openrtb26FlagNative) != 0 {
		return false
	}
	if h.BidFloorMicro < 0 || h.DealBidFloorMicro < 0 {
		return false
	}
	return true
}

func impSlotExchangeReady(s *OpenRTB26ImpSlot) bool {
	if s == nil || s.ImpIDLen == 0 {
		return false
	}
	if s.Flags&(impSlotFlagBanner|impSlotFlagVideo) == 0 {
		return false
	}
	if s.Flags&(impSlotFlagAudio|impSlotFlagNative) != 0 {
		return false
	}
	if s.BidFloorMicro < 0 || s.DealBidFloorMicro < 0 {
		return false
	}
	return true
}
