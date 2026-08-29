package ortbreact

import (
	"bytes"
	"strings"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"
	ingestcold "ad-event-processor/internal/ingest/cold"
	"ad-event-processor/internal/openrtb"
	"ad-event-processor/internal/rtb"
)

const (
	openrtb26ImpIDMax       = openrtb.OpenRTB26ImpIDMax
	openrtb26SeatMax        = 8
	openrtb26SeatIDMax      = 16
	impSlotFlagBanner       = openrtb.ImpSlotFlagBanner
	impSlotFlagVideo        = openrtb.ImpSlotFlagVideo
	impSlotFlagAudio        = openrtb.ImpSlotFlagAudio
	impSlotFlagNative       = openrtb.ImpSlotFlagNative
	impSlotFlagSecure       = openrtb.ImpSlotFlagSecure
	openrtb26FlagSite       = openrtb.OpenRTB26FlagSite
	openrtb26FlagApp        = openrtb.OpenRTB26FlagApp
	openrtb26FlagBanner     = openrtb.OpenRTB26FlagBanner
	openrtb26FlagVideo      = openrtb.OpenRTB26FlagVideo
	openrtb26FlagDeviceIP   = openrtb.OpenRTB26FlagDeviceIP
	openrtb26FlagDeviceUA   = openrtb.OpenRTB26FlagDeviceUA
	openrtb26FlagGeoCountry = openrtb.OpenRTB26FlagGeoCountry
	openrtb26FlagTest       = openrtb.OpenRTB26FlagTest
	openrtb26FlagEUR        = openrtb.OpenRTB26FlagEUR
	openrtb26FlagSecure     = openrtb.OpenRTB26FlagSecure
)

type WireTargeting struct {
	Input       rtb.RtbTargetingInput
	MediaMask   uint8
	MaxDuration uint32
	ImpIDBuf    [openrtb26ImpIDMax]byte
	ImpIDLen    uint8
	Test        bool
	CurrencyUSD bool
}

func MapParsedToTargeting(hot *openrtb.OpenRTB26Hot, cold *openrtb.OpenRTB26Cold, geo filter.GeoProvider, clientIP string) WireTargeting {
	out := WireTargeting{
		Test:        hot.Flags&openrtb26FlagTest != 0,
		CurrencyUSD: hot.Flags&openrtb26FlagEUR == 0,
	}
	out.Input.PublisherFloorMicro = hot.BidFloorMicro
	if hot.DealBidFloorMicro > out.Input.PublisherFloorMicro {
		out.Input.PublisherFloorMicro = hot.DealBidFloorMicro
	}
	if hot.ImpIDLen > 0 {
		out.ImpIDLen = hot.ImpIDLen
		copy(out.ImpIDBuf[:], hot.ImpID[:hot.ImpIDLen])
	} else {
		out.ImpIDBuf[0] = '1'
		out.ImpIDLen = 1
	}
	if hot.Flags&openrtb26FlagVideo != 0 {
		out.MediaMask = uint8(rtb.MediaTypeVideo)
		out.MaxDuration = hot.MaxDurationSec
	} else {
		out.MediaMask = uint8(rtb.MediaTypeDisplay)
	}
	if hot.DealIDLen > 0 {
		ln := copy(out.Input.DealIDBuf[:], hot.DealID[:hot.DealIDLen])
		out.Input.DealIDLen = uint8(ln)
	}
	out.Input.DeviceType = hot.DeviceType
	out.Input.CategoryMask = hot.CategoryMask
	out.Input.SeatCount = hot.SeatCount
	out.Input.DeadlineMono = openrtb.DeadlineMonoFromTmax(hot.TmaxMs)
	if cold != nil {
		out.Input.Schain = cold.Schain
		out.Input.SchainCount = cold.Schain.Count
	}
	out.Input.FcapUserHash = hot.FcapUserHash
	out.Input.ConnectionType = hot.ConnectionType
	out.Input.ViewabilityPPM = hot.MetricValuePPM
	out.Input.PMPPrivate = hot.PMPPrivate
	out.Input.DeviceLMT = hot.DeviceLMT
	if cold != nil {
		out.Input.BlockedCatMask = cold.BCatMask
	}
	if hot.Flags&openrtb26FlagGeoCountry != 0 && hot.GeoCountryLen > 0 {
		out.Input.GeoHash = rtb.GeoHashFromCountryBytes(hot.GeoCountry[:hot.GeoCountryLen])
	} else if geo != nil && clientIP != "" {
		var evt domain.Event
		evt.IP = clientIP
		ingestcold.EnsureIngestGeo(geo, &evt)
		out.Input.GeoHash = evt.GeoHash
	}
	return out
}

func MapImpSlotToTargeting(hot *openrtb.OpenRTB26Hot, cold *openrtb.OpenRTB26Cold, slot *openrtb.OpenRTB26ImpSlot, geo filter.GeoProvider, clientIP string) WireTargeting {
	out := MapParsedToTargeting(hot, cold, geo, clientIP)
	if slot == nil {
		return out
	}
	out.Input.PublisherFloorMicro = slot.BidFloorMicro
	if slot.DealBidFloorMicro > out.Input.PublisherFloorMicro {
		out.Input.PublisherFloorMicro = slot.DealBidFloorMicro
	}
	if slot.ImpIDLen > 0 {
		out.ImpIDLen = slot.ImpIDLen
		copy(out.ImpIDBuf[:], slot.ImpID[:slot.ImpIDLen])
	}
	out.Input.DealIDLen = 0
	if slot.DealIDLen > 0 {
		ln := copy(out.Input.DealIDBuf[:], slot.DealID[:slot.DealIDLen])
		out.Input.DealIDLen = uint8(ln)
	}
	if slot.Flags&impSlotFlagVideo != 0 {
		out.MediaMask = uint8(rtb.MediaTypeVideo)
		out.MaxDuration = slot.MaxDurationSec
	} else {
		out.MediaMask = uint8(rtb.MediaTypeDisplay)
		out.MaxDuration = 0
	}
	out.Input.SeatCount = impSlotSeatCount(slot, hot)
	return out
}

func ImpSlotFromHot(hot *openrtb.OpenRTB26Hot) openrtb.OpenRTB26ImpSlot {
	var slot openrtb.OpenRTB26ImpSlot
	if hot == nil {
		return slot
	}
	slot.ImpIDLen = hot.ImpIDLen
	copy(slot.ImpID[:], hot.ImpID[:hot.ImpIDLen])
	slot.BidFloorMicro = hot.BidFloorMicro
	slot.DealBidFloorMicro = hot.DealBidFloorMicro
	slot.DealIDLen = hot.DealIDLen
	copy(slot.DealID[:], hot.DealID[:hot.DealIDLen])
	slot.BannerW = hot.BannerW
	slot.BannerH = hot.BannerH
	slot.VideoW = hot.VideoW
	slot.VideoH = hot.VideoH
	slot.MaxDurationSec = hot.MaxDurationSec
	if hot.Flags&openrtb26FlagBanner != 0 {
		slot.Flags |= impSlotFlagBanner
	}
	if hot.Flags&openrtb26FlagVideo != 0 {
		slot.Flags |= impSlotFlagVideo
	}
	if hot.Flags&openrtb26FlagSecure != 0 {
		slot.Flags |= impSlotFlagSecure
	}
	return slot
}

func MapWireToTargeting(req openrtb.BidRequest, geo filter.GeoProvider, clientIP string) WireTargeting {
	out := WireTargeting{
		Test:        req.Test == 1,
		CurrencyUSD: true,
	}
	if len(req.Imp) > 0 {
		imp := req.Imp[0]
		out.Input.PublisherFloorMicro = openrtb.PriceToMicro(imp.BidFloor)
		out.MediaMask, out.MaxDuration = mediaFromImp(imp)
		id := strings.TrimSpace(imp.ID)
		if id == "" {
			out.ImpIDBuf[0] = '1'
			out.ImpIDLen = 1
		} else {
			ln := copy(out.ImpIDBuf[:], id)
			out.ImpIDLen = uint8(ln)
		}
		if imp.PMP != nil && len(imp.PMP.Deals) > 0 {
			deal := imp.PMP.Deals[0]
			dealID := strings.TrimSpace(deal.ID)
			ln := copy(out.Input.DealIDBuf[:], dealID)
			out.Input.DealIDLen = uint8(ln)
			if deal.BidFloor > 0 {
				floor := openrtb.PriceToMicro(deal.BidFloor)
				if floor > out.Input.PublisherFloorMicro {
					out.Input.PublisherFloorMicro = floor
				}
			}
		}
	} else {
		out.ImpIDBuf[0] = '1'
		out.ImpIDLen = 1
	}
	out.Input.DeviceType = deviceTypeFromWire(req.Device.DeviceType)
	out.Input.CategoryMask = categoryMaskFromWire(req)
	out.Input.DeadlineMono = openrtb.DeadlineMonoFromTmax(int32(req.Tmax))
	if req.Source != nil && req.Source.Ext != nil && req.Source.Ext.Schain != nil {
		out.Input.Schain = schainFromWire(req.Source.Ext.Schain)
		out.Input.SchainCount = out.Input.Schain.Count
	}
	if req.Device.Geo != nil && strings.TrimSpace(req.Device.Geo.Country) != "" {
		out.Input.GeoHash = rtb.GeoHashFromCountry(normalizeCountry(req.Device.Geo.Country))
	} else if geo != nil && strings.TrimSpace(clientIP) != "" {
		evt := &domain.Event{IP: clientIP}
		ingestcold.EnsureIngestGeo(geo, evt)
		out.Input.GeoHash = evt.GeoHash
	}
	if len(req.Cur) > 0 {
		c := strings.ToUpper(strings.TrimSpace(req.Cur[0]))
		if c == "EUR" {
			out.CurrencyUSD = false
		}
	}
	if out.CurrencyUSD && len(req.Imp) > 0 {
		c := strings.ToUpper(strings.TrimSpace(req.Imp[0].BidFloorCur))
		if c == "EUR" {
			out.CurrencyUSD = false
		}
	}
	return out
}

func SeatAllowedInWSeat(slot *openrtb.OpenRTB26ImpSlot, seat []byte) bool {
	if slot == nil || slot.WSeatCount == 0 || len(seat) == 0 {
		return true
	}
	for i := range int(slot.WSeatCount) {
		if slot.WSeatLen[i] == 0 {
			continue
		}
		if bytes.Equal(seat, slot.WSeat[i][:slot.WSeatLen[i]]) {
			return true
		}
	}
	return false
}

func impSlotSeatCount(slot *openrtb.OpenRTB26ImpSlot, hot *openrtb.OpenRTB26Hot) uint8 {
	if slot != nil && slot.WSeatCount > 0 {
		return slot.WSeatCount
	}
	if hot != nil && hot.SeatCount > 0 {
		return hot.SeatCount
	}
	return 1
}

func mediaFromImp(imp openrtb.Imp) (mask uint8, maxDur uint32) {
	if imp.Video != nil {
		mask = uint8(rtb.MediaTypeVideo)
		if imp.Video.MaxDuration > 0 {
			maxDur = uint32(imp.Video.MaxDuration)
		}
		return mask, maxDur
	}
	if imp.Banner != nil {
		return uint8(rtb.MediaTypeDisplay), 0
	}
	return uint8(rtb.MediaTypeDisplay), 0
}

func deviceTypeFromWire(dt int) uint8 {
	switch dt {
	case 1, 4:
		return 2
	case 2:
		return 1
	case 5:
		return 4
	default:
		return 1
	}
}

func categoryMaskFromWire(req openrtb.BidRequest) uint64 {
	var cats []string
	if req.Site != nil {
		cats = req.Site.Cat
	} else if req.App != nil {
		cats = req.App.Cat
	}
	if len(cats) == 0 {
		return 1
	}
	var mask uint64
	for _, c := range cats {
		c = strings.TrimSpace(c)
		if len(c) == 0 {
			continue
		}
		last := c[len(c)-1]
		if last >= '0' && last <= '9' {
			mask |= uint64(1) << uint64(last-'0')
		} else {
			mask |= 1
		}
	}
	if mask == 0 {
		return 1
	}
	return mask
}

func schainFromWire(s *openrtb.Schain) rtb.SchainNodes {
	var out rtb.SchainNodes
	if s == nil {
		return out
	}
	for i, n := range s.Nodes {
		if i >= openrtb.SchainNodeMax {
			break
		}
		node := rtb.SchainNode{}
		node.ASILen = uint8(copy(node.ASI[:], n.ASI))
		node.SIDLen = uint8(copy(node.SID[:], n.SID))
		out.Nodes[i] = node
		out.Count++
	}
	return out
}

func normalizeCountry(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) == 3 && code[0] == 'U' && code[1] == 'S' && code[2] == 'A' {
		return "US"
	}
	if len(code) >= 2 {
		return code[:2]
	}
	return code
}
