package ingestion

import (
	"sync/atomic"

	"espx/internal/domain"
	"espx/internal/rtb"

	"github.com/google/uuid"
)

type RtbCatalog struct {
	registry   *rtb.Registry
	dealIndex  *rtb.DealIndex
	dealFloors *DealFloorCache
	authority  BudgetAuthority
	winnerUUID atomic.Pointer[map[rtb.CampaignID]uuid.UUID]

	prebidIVT       atomic.Bool
	schainAllow     atomic.Pointer[SupplyChainAllowlistSnapshot]
	settingsWatcher *SettingsWatcher
	ingestGeo       GeoProvider
}

func NewRtbCatalog(store *rtb.BudgetStore, authority BudgetAuthority) *RtbCatalog {
	return &RtbCatalog{
		registry:  rtb.NewRegistry(store),
		dealIndex: rtb.NewDealIndex(),
		authority: authority,
	}
}

func (catalog *RtbCatalog) Registry() *rtb.Registry {
	return catalog.registry
}

func (catalog *RtbCatalog) Authority() BudgetAuthority {
	return catalog.authority
}

func (catalog *RtbCatalog) SetAuthority(authority BudgetAuthority) {
	catalog.authority = authority
}

func (catalog *RtbCatalog) SetPrebidIVT(enabled bool) {
	catalog.prebidIVT.Store(enabled)
}

func (catalog *RtbCatalog) SetSupplyChainAllowlist(snap *SupplyChainAllowlistSnapshot) {
	if snap == nil {
		catalog.schainAllow.Store(nil)
		return
	}
	catalog.schainAllow.Store(snap)
}

func (catalog *RtbCatalog) ConfigureRtbGates(watcher *SettingsWatcher, geo GeoProvider) {
	if catalog == nil {
		return
	}
	catalog.settingsWatcher = watcher
	catalog.ingestGeo = geo
}

func (catalog *RtbCatalog) SetDealFloors(cache *DealFloorCache) {
	catalog.dealFloors = cache
}

func (catalog *RtbCatalog) SyncActiveCampaigns(campaigns []*domain.Campaign, inputs map[uuid.UUID]RtbCampaignInput) {
	rows := BuildRtbCatalogRows(campaigns, inputs)
	catalog.registry.UpdateCampaigns(rows)
	catalog.rebuildWinnerUUID(rows, campaigns)
}

func (catalog *RtbCatalog) rebuildWinnerUUID(rows []rtb.CampaignData, campaigns []*domain.Campaign) {
	if len(rows) == 0 {
		empty := make(map[rtb.CampaignID]uuid.UUID)
		catalog.winnerUUID.Store(&empty)
		return
	}
	m := make(map[rtb.CampaignID]uuid.UUID, len(rows))
	for _, camp := range campaigns {
		if camp == nil {
			continue
		}
		m[CampaignIDFromUUID(camp.ID)] = camp.ID
	}
	catalog.winnerUUID.Store(&m)
}

func (catalog *RtbCatalog) UUIDForWinner(id rtb.CampaignID) (uuid.UUID, bool) {
	ptr := catalog.winnerUUID.Load()
	if ptr == nil {
		return uuid.Nil, false
	}
	uid, ok := (*ptr)[id]
	return uid, ok
}

func (catalog *RtbCatalog) SyncCampaignRows(campaigns []*domain.Campaign, rows []rtb.CampaignData) {
	catalog.registry.UpdateCampaigns(rows)
	catalog.rebuildWinnerUUID(rows, campaigns)
}

func (catalog *RtbCatalog) SyncFromRegistry(registry *Registry, inputs map[uuid.UUID]RtbCampaignInput) {
	if registry == nil {
		catalog.registry.UpdateCampaigns(nil)
		return
	}
	catalog.SyncActiveCampaigns(registry.ActiveCampaigns(), inputs)
}

func (catalog *RtbCatalog) SetClearingMode(mode rtb.ClearingMode) {
	catalog.registry.SetClearingMode(mode)
}

func (catalog *RtbCatalog) UpdateDeals(deals []rtb.DealData) {
	if catalog.dealIndex == nil {
		catalog.dealIndex = rtb.NewDealIndex()
	}
	catalog.dealIndex.UpdateDeals(deals)
}

func (catalog *RtbCatalog) DealCount() int {
	if catalog.dealIndex == nil {
		return 0
	}
	return catalog.dealIndex.Len()
}

func (catalog *RtbCatalog) LookupDeal(dealID string) (rtb.DealData, bool) {
	if catalog.dealIndex == nil {
		return rtb.DealData{}, false
	}
	return catalog.dealIndex.Lookup(dealID)
}

func (catalog *RtbCatalog) AllDeals() []rtb.DealData {
	if catalog.dealIndex == nil {
		return nil
	}
	return catalog.dealIndex.All()
}

func (catalog *RtbCatalog) RunAuction(evt *domain.Event, targeting RtbTargetingInput) (rtb.AuctionResult, rtb.NoBidReason) {
	if catalog.authority != BudgetAuthorityShadow {
		if reason := rtbPrefilterReject(catalog.settingsWatcher, catalog, targeting); reason != rtb.NoBidNone {
			return rtb.AuctionResult{}, reason
		}
		if catalog.prebidIVT.Load() {
			if reason := rtbPrebidIVTReject(true, catalog.ingestGeo, evt); reason != rtb.NoBidNone {
				return rtb.AuctionResult{}, reason
			}
		}
		if targeting.SchainCount > 0 {
			allow := catalog.schainAllow.Load()
			if allow != nil && !ValidateSchainNodes(targeting.Schain, allow) {
				return rtb.AuctionResult{}, rtb.NoBidSchainInvalid
			}
		}
	}
	targeting = catalog.enrichTargetingDeal(targeting)
	if catalog.settingsWatcher != nil {
		catalog.registry.SetFcapSnapshot(catalog.settingsWatcher.GetFcapRtbSnapshot())
	}
	req := BidRequestFromEvent(evt, targeting)
	if catalog.authority == BudgetAuthorityShadow {
		return catalog.registry.RunAuctionEval(&req)
	}
	res, reason := catalog.registry.RunAuction(&req)
	if reason.OK() && evt != nil {
		evt.ClearingPriceMicro = res.Price
	}
	return res, reason
}

func (catalog *RtbCatalog) enrichTargetingDeal(targeting RtbTargetingInput) RtbTargetingInput {
	if catalog == nil || catalog.dealIndex == nil {
		return targeting
	}
	var deal rtb.DealData
	var ok bool
	if targeting.DealIDLen > 0 {
		deal, ok = catalog.dealIndex.LookupBytes(targeting.DealIDBuf[:targeting.DealIDLen])
	} else if targeting.DealID != "" {
		deal, ok = catalog.LookupDeal(targeting.DealID)
	}
	if !ok {
		return targeting
	}
	if deal.PacingOpen == rtb.PacingClosed {
		targeting.DealBlock = rtb.NoBidPacingClosed
		return targeting
	}
	geoBit := rtb.GeoBitFromHash(targeting.GeoHash)
	if (deal.GeoMask&geoBit) == 0 || (deal.CatMask&targeting.CategoryMask) == 0 {
		targeting.DealBlock = rtb.NoBidDealMismatch
		return targeting
	}
	if deal.Seats > 0 && int32(targeting.SeatCount) < deal.Seats {
		targeting.DealBlock = rtb.NoBidDealMismatch
		return targeting
	}
	return targeting
}

func (catalog *RtbCatalog) LookupDealBytes(dealID []byte) (rtb.DealData, bool) {
	if catalog == nil || catalog.dealIndex == nil {
		return rtb.DealData{}, false
	}
	return catalog.dealIndex.LookupBytes(dealID)
}
