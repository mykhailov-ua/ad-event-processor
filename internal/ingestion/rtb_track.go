package ingestion

import (
	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
)

const (
	rtbModeOff uint8 = iota
	rtbModeShadow
	rtbModeLive
)

func rtbModeFromConfig(cfg *config.Config) uint8 {
	if cfg == nil {
		return rtbModeOff
	}
	switch config.ParseRtbMode(cfg.RtbMode) {
	case config.RtbModeShadow:
		return rtbModeShadow
	case config.RtbModeLive:
		return rtbModeLive
	default:
		return rtbModeOff
	}
}

func ConfigureTrackRtb(proc *trackProcessor, cfg *config.Config, catalog *RtbCatalog, geo GeoProvider, unified *UnifiedFilter, watcher *SettingsWatcher) {
	if proc == nil || cfg == nil || catalog == nil || !cfg.RtbEnabled() {
		return
	}
	if !openRTBLicenseAllowed(proc.registry) {
		proc.rtbMode = rtbModeOff
		return
	}
	proc.rtbCatalog = catalog
	proc.rtbMode = rtbModeFromConfig(cfg)
	proc.settingsWatcher = watcher
	if watcher != nil {
		proc.rtbMode = RtbModeFromSetting(watcher.Get().RtbMode, cfg)
		watcher.AddChangeListener(func(dc *DynamicConfig) {
			if proc != nil {
				proc.rtbMode = RtbModeFromSetting(dc.RtbMode, cfg)
			}
		})
	}
	proc.ingestGeo = geo
	catalog.ConfigureRtbGates(watcher, geo)
	if cfg.RtbPrebidIVTEnabled() {
		catalog.SetPrebidIVT(true)
	}
	if unified != nil {
		setting := ""
		if watcher != nil {
			setting = watcher.Get().RtbBudgetAuthority
		}
		unified.SetSkipBudgetDebit(RtbSkipLuaBudgetDebit(cfg, setting))
	}
}

func ConfigureIngestGeo(proc *trackProcessor, geo GeoProvider) {
	if proc != nil {
		proc.ingestGeo = geo
	}
}

func (h *AdsPacketHandler) ConfigureRtb(catalog *RtbCatalog, geo GeoProvider, unified *UnifiedFilter, watcher *SettingsWatcher) {
	if h == nil {
		return
	}
	ConfigureTrackRtb(&h.trackProc, h.cfg, catalog, geo, unified, watcher)
}

func (h *AdsPacketHandler) ConfigureIngestGeo(geo GeoProvider) {
	if h == nil {
		return
	}
	ConfigureIngestGeo(&h.trackProc, geo)
}

func buildRtbTargeting(evt *domain.Event, deviceType []byte, floorMicro int64, catalog *RtbCatalog) RtbTargetingInput {
	geoHash := uint32(0)
	if evt != nil && evt.IngestGeoResolved {
		geoHash = evt.GeoHash
	}

	out := RtbTargetingInput{GeoHash: geoHash}

	if evt != nil && len(evt.Payload) > 0 {
		var parsed OpenRTB3Parsed
		var haveParsed bool
		if cached, ok := openRTB3ParsedFromScratch(evt); ok {
			parsed = *cached
			haveParsed = true
		} else {
			parsed = parseOpenRTB3FSM(evt.Payload)
			haveParsed = parsed.IsOpenRTB
		}
		if haveParsed {
			if floorMicro <= 0 {
				floorMicro = parsed.MinBid
			}
			if parsed.DealIDLen > 0 {
				out.DealIDLen = parsed.DealIDLen
				src := ortbSlice(evt.Payload, parsed.DealIDOff, parsed.DealIDLen)
				copy(out.DealIDBuf[:], src)
			}
			floorMicro = EffectiveDealFloorBytes(catalog, catalogDealFloors(catalog), out.DealIDBuf[:out.DealIDLen], floorMicro)
			out.DeviceType = parsed.DeviceType
			out.CategoryMask = parsed.CategoryMask
			out.PublisherFloorMicro = floorMicro
			return out
		}
	}

	if evt != nil && len(evt.Payload) > 0 {
		incIngressLegacyJSON()
	}
	if floorMicro <= 0 && evt != nil {
		floorMicro = parseBidMicro(evt.Payload)
	}
	categoryMask := uint64(1)
	if evt != nil {
		if parsed := parseCategoryMask(evt.Payload); parsed != 0 {
			categoryMask = parsed
		}
		if n := ParseDealIDBytes(evt.Payload, out.DealIDBuf[:]); n > 0 {
			out.DealIDLen = uint8(n)
		}
	}
	if out.DealIDLen > 0 {
		floorMicro = EffectiveDealFloorBytes(catalog, catalogDealFloors(catalog), out.DealIDBuf[:out.DealIDLen], floorMicro)
	}
	out.DeviceType = DeviceMaskFromType(deviceType)
	out.CategoryMask = categoryMask
	out.PublisherFloorMicro = floorMicro
	return out
}

func catalogDealFloors(catalog *RtbCatalog) *DealFloorCache {
	if catalog == nil {
		return nil
	}
	return catalog.dealFloors
}

func applyRtbAuction(proc trackProcessor, evt *domain.Event, deviceType []byte) (trackOutcome, bool) {
	if proc.rtbCatalog == nil || proc.rtbMode == rtbModeOff || evt == nil {
		return trackOutcome{}, false
	}
	if !openRTBLicenseAllowed(proc.registry) {
		return trackOutcome{}, false
	}

	targeting := buildRtbTargeting(evt, deviceType, 0, proc.rtbCatalog)
	payloadBidMicro := targeting.PublisherFloorMicro
	res, reason := proc.rtbCatalog.RunAuction(evt, targeting)

	if proc.rtbMode == rtbModeShadow {
		recordRtbShadowAuction(proc.rtbCatalog, evt, res, reason, payloadBidMicro)
		recordRtbDealOutcomeBytes(targeting.DealIDBuf[:], targeting.DealIDLen, payloadBidMicro, res, reason)
		return trackOutcome{}, false
	}

	recordRtbDealOutcomeBytes(targeting.DealIDBuf[:], targeting.DealIDLen, payloadBidMicro, res, reason)

	if !reason.OK() {
		return trackOutcome{Status: trackStatusRejected, RejectKind: noBidToRejectKind(reason)}, true
	}

	uid, ok := proc.rtbCatalog.UUIDForWinner(res.CampaignID)
	if !ok {
		return trackOutcome{Status: trackStatusRejected, RejectKind: filterRejectCampaignNotFound}, true
	}
	evt.CampaignID = uid
	evt.ClearingPriceMicro = res.Price
	return trackOutcome{}, false
}
