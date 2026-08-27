package ingestion

import (
	"sync/atomic"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/rtb"

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

func (c *RtbCatalog) Registry() *rtb.Registry {
	return c.registry
}

func (c *RtbCatalog) Authority() BudgetAuthority {
	return c.authority
}

func (c *RtbCatalog) SetAuthority(authority BudgetAuthority) {
	c.authority = authority
}

func (c *RtbCatalog) SetPrebidIVT(enabled bool) {
	c.prebidIVT.Store(enabled)
}

func (c *RtbCatalog) SetSupplyChainAllowlist(snap *SupplyChainAllowlistSnapshot) {
	if snap == nil {
		c.schainAllow.Store(nil)
		return
	}
	c.schainAllow.Store(snap)
}

func (c *RtbCatalog) ConfigureRtbGates(watcher *SettingsWatcher, geo GeoProvider) {
	if c == nil {
		return
	}
	c.settingsWatcher = watcher
	c.ingestGeo = geo
}

func (c *RtbCatalog) SetDealFloors(cache *DealFloorCache) {
	c.dealFloors = cache
}

func (c *RtbCatalog) SyncActiveCampaigns(campaigns []*domain.Campaign, inputs map[uuid.UUID]RtbCampaignInput) {
	rows := BuildRtbCatalogRows(campaigns, inputs)
	c.registry.UpdateCampaigns(rows)
	c.rebuildWinnerUUID(rows, campaigns)
}

func (c *RtbCatalog) rebuildWinnerUUID(rows []rtb.CampaignData, campaigns []*domain.Campaign) {
	if len(rows) == 0 {
		empty := make(map[rtb.CampaignID]uuid.UUID)
		c.winnerUUID.Store(&empty)
		return
	}
	m := make(map[rtb.CampaignID]uuid.UUID, len(rows))
	for _, camp := range campaigns {
		if camp == nil {
			continue
		}
		m[CampaignIDFromUUID(camp.ID)] = camp.ID
	}
	c.winnerUUID.Store(&m)
}

func (c *RtbCatalog) LookupCreativeADM(geoHash uint32, campaignID rtb.CampaignID, creativeID rtb.CreativeID) ([]byte, uint8, bool) {
	if c == nil || c.registry == nil {
		return nil, 0, false
	}
	return c.registry.LookupCreativeWire(geoHash, campaignID, creativeID)
}

func (c *RtbCatalog) UUIDForWinner(id rtb.CampaignID) (uuid.UUID, bool) {
	ptr := c.winnerUUID.Load()
	if ptr == nil {
		return uuid.Nil, false
	}
	uid, ok := (*ptr)[id]
	return uid, ok
}

func (c *RtbCatalog) SyncCampaignRows(campaigns []*domain.Campaign, rows []rtb.CampaignData) {
	c.registry.UpdateCampaigns(rows)
	c.rebuildWinnerUUID(rows, campaigns)
}

func (c *RtbCatalog) SyncFromRegistry(registry *Registry, inputs map[uuid.UUID]RtbCampaignInput) {
	if registry == nil {
		c.registry.UpdateCampaigns(nil)
		return
	}
	c.SyncActiveCampaigns(registry.ActiveCampaigns(), inputs)
}

func (c *RtbCatalog) SetClearingMode(mode rtb.ClearingMode) {
	c.registry.SetClearingMode(mode)
}

func (c *RtbCatalog) UpdateDeals(deals []rtb.DealData) {
	if c.dealIndex == nil {
		c.dealIndex = rtb.NewDealIndex()
	}
	c.dealIndex.UpdateDeals(deals)
}

func (c *RtbCatalog) DealCount() int {
	if c.dealIndex == nil {
		return 0
	}
	return c.dealIndex.Len()
}

func (c *RtbCatalog) LookupDeal(dealID string) (rtb.DealData, bool) {
	if c.dealIndex == nil {
		return rtb.DealData{}, false
	}
	return c.dealIndex.Lookup(dealID)
}

func (c *RtbCatalog) AllDeals() []rtb.DealData {
	if c.dealIndex == nil {
		return nil
	}
	return c.dealIndex.All()
}

func (c *RtbCatalog) EvaluateAuction(evt *domain.Event, targeting RtbTargetingInput) (rtb.AuctionResult, rtb.NoBidReason) {
	if c == nil || c.registry == nil {
		return rtb.AuctionResult{}, rtb.NoBidInvalidRequest
	}
	if reason := rtbPrefilterReject(c.settingsWatcher, c, targeting); reason != rtb.NoBidNone {
		return rtb.AuctionResult{}, reason
	}
	targeting = c.enrichTargetingDeal(targeting)
	if c.settingsWatcher != nil {
		c.registry.SetFcapSnapshot(c.settingsWatcher.GetFcapRtbSnapshot())
	}
	req := BidRequestFromEvent(evt, targeting)
	return c.registry.RunAuctionEval(&req)
}

func (c *RtbCatalog) RunAuction(evt *domain.Event, targeting RtbTargetingInput) (rtb.AuctionResult, rtb.NoBidReason) {
	if c == nil || c.registry == nil {
		return rtb.AuctionResult{}, rtb.NoBidInvalidRequest
	}
	if c.authority != BudgetAuthorityShadow {
		if reason := rtbPrefilterReject(c.settingsWatcher, c, targeting); reason != rtb.NoBidNone {
			return rtb.AuctionResult{}, reason
		}
		if c.prebidIVT.Load() {
			if reason := rtbPrebidIVTReject(true, c.ingestGeo, evt); reason != rtb.NoBidNone {
				return rtb.AuctionResult{}, reason
			}
		}
		if targeting.SchainCount > 0 {
			allow := c.schainAllow.Load()
			if allow != nil && !ValidateSchainNodes(targeting.Schain, allow) {
				return rtb.AuctionResult{}, rtb.NoBidSchainInvalid
			}
		}
	}
	targeting = c.enrichTargetingDeal(targeting)
	if c.settingsWatcher != nil {
		c.registry.SetFcapSnapshot(c.settingsWatcher.GetFcapRtbSnapshot())
	}
	req := BidRequestFromEvent(evt, targeting)
	if c.authority == BudgetAuthorityShadow {
		return c.registry.RunAuctionEval(&req)
	}
	res, reason := c.registry.RunAuction(&req)
	if reason.OK() && evt != nil {
		evt.ClearingPriceMicro = res.Price
	}
	return res, reason
}

func (c *RtbCatalog) enrichTargetingDeal(targeting RtbTargetingInput) RtbTargetingInput {
	if c == nil || c.dealIndex == nil {
		return targeting
	}
	var deal rtb.DealData
	var ok bool
	if targeting.DealIDLen > 0 {
		deal, ok = c.dealIndex.LookupBytes(targeting.DealIDBuf[:targeting.DealIDLen])
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

func (c *RtbCatalog) LookupDealBytes(dealID []byte) (rtb.DealData, bool) {
	if c == nil || c.dealIndex == nil {
		return rtb.DealData{}, false
	}
	return c.dealIndex.LookupBytes(dealID)
}
