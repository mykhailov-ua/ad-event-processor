package ingestion

import (
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/rtb"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type staticGeoProvider struct {
	country   string
	anonymous bool
}

func (s *staticGeoProvider) GetCountry(ip string) (string, error) {
	return s.country, nil
}

func (s *staticGeoProvider) IsAnonymous(ip string) (bool, error) {
	return s.anonymous, nil
}

func (s *staticGeoProvider) Close() error { return nil }

func TestApplyRtbAuction_liveSelectsWinner(t *testing.T) {
	store := rtb.NewBudgetStore()
	catalog := NewRtbCatalog(store, BudgetAuthorityRTB)

	winnerID := uuid.New()
	geo := GeoHashFromCountry("US")
	catalog.SyncActiveCampaigns(
		[]*domain.Campaign{{ID: winnerID, BudgetLimit: 5000, TargetCountries: map[string]struct{}{"US": {}}}},
		map[uuid.UUID]RtbCampaignInput{
			winnerID: {BidMicro: 100, DeviceMask: 1, CategoryMask: 1, GeoHash: geo, Weight: 1},
		},
	)

	proc := trackProcessor{
		rtbCatalog: catalog,
		rtbMode:    rtbModeLive,
		ingestGeo:  &staticGeoProvider{country: "US"},
	}
	clientID := uuid.New()
	evt := &domain.Event{CampaignID: clientID, IP: "8.8.8.8"}
	ensureIngestGeo(proc.ingestGeo, evt)

	out, handled := applyRtbAuction(proc, evt, []byte("desktop"))
	assert.False(t, handled)
	assert.Equal(t, winnerID, evt.CampaignID)
	assert.Equal(t, trackOutcome{}, out)
}

func TestApplyRtbAuction_shadowKeepsClientCampaign(t *testing.T) {
	store := rtb.NewBudgetStore()
	catalog := NewRtbCatalog(store, BudgetAuthorityShadow)

	id := uuid.New()
	geo := GeoHashFromCountry("US")
	catalog.SyncActiveCampaigns(
		[]*domain.Campaign{{ID: id, BudgetLimit: 5000}},
		map[uuid.UUID]RtbCampaignInput{
			id: {BidMicro: 100, DeviceMask: 1, CategoryMask: 1, GeoHash: geo, Weight: 1},
		},
	)

	clientID := uuid.New()
	evt := &domain.Event{CampaignID: clientID, IP: "8.8.8.8"}
	proc := trackProcessor{
		rtbCatalog: catalog,
		rtbMode:    rtbModeShadow,
		ingestGeo:  &staticGeoProvider{country: "US"},
	}
	ensureIngestGeo(proc.ingestGeo, evt)

	_, handled := applyRtbAuction(proc, evt, nil)
	assert.False(t, handled)
	assert.Equal(t, clientID, evt.CampaignID)
}

func TestApplyRtbAuction_liveNoBidRejects(t *testing.T) {
	store := rtb.NewBudgetStore()
	catalog := NewRtbCatalog(store, BudgetAuthorityRTB)
	catalog.SyncActiveCampaigns(nil, nil)

	proc := trackProcessor{rtbCatalog: catalog, rtbMode: rtbModeLive}
	evt := &domain.Event{CampaignID: uuid.New()}

	out, handled := applyRtbAuction(proc, evt, nil)
	require.True(t, handled)
	assert.Equal(t, trackStatusRejected, out.Status)
	assert.Equal(t, filterRejectBidFloor, out.RejectKind)
}

func TestConfigureTrackRtb_skipLuaBudget(t *testing.T) {
	cfg := &config.Config{RtbMode: "live", RtbBudgetAuthority: "rtb"}
	catalog := NewRtbCatalog(rtb.NewBudgetStore(), BudgetAuthorityRTB)
	proc := trackProcessor{}
	uf := NewUnifiedFilter(nil, nil, nil, nil, 0, 0, 0, 0, 0, 0, "", 0)
	ConfigureTrackRtb(&proc, cfg, catalog, nil, uf, nil)
	assert.Equal(t, rtbModeLive, proc.rtbMode)
	assert.Equal(t, oneAny, uf.skipBudgetDebitAny)
}

func TestBuildRtbTargeting_OpenRTB3AndLegacy(t *testing.T) {
	openrtbPayload := []byte(`{
 "openrtb": {
 "ver": "3.0",
 "domainspec": "adcom",
 "domainver": "1.0",
 "request": {
 "id": "req-123456789",
 "item": [
 {
 "id": "item-1",
 "flr": 1.50
 }
 ],
 "context": {
 "device": {
 "type": 4
 }
 }
 }
 },
 "category_mask": 8
}`)

	evtOpenRTB := &domain.Event{
		Payload:           openrtbPayload,
		IngestGeoResolved: true,
		GeoHash:           12345,
	}

	targetingOpenRTB := buildRtbTargeting(evtOpenRTB, []byte("desktop"), 0, nil)
	assert.Equal(t, uint32(12345), targetingOpenRTB.GeoHash)
	assert.Equal(t, uint8(2), targetingOpenRTB.DeviceType)
	assert.Equal(t, uint64(8), targetingOpenRTB.CategoryMask)
	assert.Equal(t, int64(1500000), targetingOpenRTB.PublisherFloorMicro)

	legacyPayload := []byte(`{"category_mask":4,"bid_micro":100}`)
	evtLegacy := &domain.Event{
		Payload:           legacyPayload,
		IngestGeoResolved: true,
		GeoHash:           12345,
	}

	targetingLegacy := buildRtbTargeting(evtLegacy, []byte("mobile"), 0, nil)
	assert.Equal(t, uint32(12345), targetingLegacy.GeoHash)
	assert.Equal(t, uint8(2), targetingLegacy.DeviceType)
	assert.Equal(t, uint64(4), targetingLegacy.CategoryMask)
	assert.Equal(t, int64(100), targetingLegacy.PublisherFloorMicro)

	allocs := testing.AllocsPerRun(1000, func() {
		_ = buildRtbTargeting(evtOpenRTB, []byte("desktop"), 0, nil)
	})
	assert.Equal(t, float64(0), allocs)
}

func BenchmarkBuildRtbTargeting_OpenRTB3(b *testing.B) {
	openrtbPayload := []byte(`{
 "openrtb": {
 "ver": "3.0",
 "domainspec": "adcom",
 "domainver": "1.0",
 "request": {
 "id": "req-123456789",
 "item": [
 {
 "id": "item-1",
 "flr": 1.50
 }
 ],
 "context": {
 "device": {
 "type": 4
 }
 }
 }
 },
 "category_mask": 8
}`)
	evt := &domain.Event{
		Payload:           openrtbPayload,
		IngestGeoResolved: true,
		GeoHash:           12345,
	}
	slot := acquireOpenRTBScratchSlot()
	parseOpenRTB3FSMInto(&slot.parsed, openrtbPayload)
	attachOpenRTB3Scratch(evt, slot)
	b.Cleanup(func() { releaseOpenRTB3Scratch(evt) })
	b.ReportAllocs()
	for b.Loop() {
		_ = buildRtbTargeting(evt, []byte("desktop"), 0, nil)
	}
}
