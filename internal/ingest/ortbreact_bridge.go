package ingest

import (
	"ad-event-processor/internal/ingest/ortbreact"
	"ad-event-processor/internal/openrtb"
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

type wireTargeting = ortbreact.WireTargeting

func mapParsedToTargeting(hot *OpenRTB26Hot, cold *OpenRTB26Cold, geo GeoProvider, clientIP string) wireTargeting {
	return ortbreact.MapParsedToTargeting(hot, cold, geo, clientIP)
}

func mapImpSlotToTargeting(hot *OpenRTB26Hot, cold *OpenRTB26Cold, slot *OpenRTB26ImpSlot, geo GeoProvider, clientIP string) wireTargeting {
	return ortbreact.MapImpSlotToTargeting(hot, cold, slot, geo, clientIP)
}

func mapWireToTargeting(req openrtb.BidRequest, geo GeoProvider, clientIP string) wireTargeting {
	return ortbreact.MapWireToTargeting(req, geo, clientIP)
}

func impSlotFromHot(hot *OpenRTB26Hot) OpenRTB26ImpSlot {
	return ortbreact.ImpSlotFromHot(hot)
}

func seatAllowedInWSeat(slot *OpenRTB26ImpSlot, seat []byte) bool {
	return ortbreact.SeatAllowedInWSeat(slot, seat)
}
