package ingestion

import (
	"context"
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcceptLangGeo_holdoutMatchingLangPasses(t *testing.T) {
	assert.False(t, acceptLangGeoMismatch("de-DE,de;q=0.9,en;q=0.8", "DE"))
	assert.False(t, acceptLangGeoMismatch("pt-BR,pt;q=0.9", "BR"))
	assert.False(t, acceptLangGeoMismatch("en-US,en;q=0.9", "US"))
}

func TestAcceptLangGeo_holdoutObviousMismatchFails(t *testing.T) {
	assert.True(t, acceptLangGeoMismatch("pt-BR,pt;q=0.9,en;q=0.8", "DE"))
	assert.True(t, acceptLangGeoMismatch("en-US,en;q=0.9", "DE"))
}

func TestAcceptLangGeo_holdoutMissingInputsFailOpen(t *testing.T) {
	assert.False(t, acceptLangGeoMismatch("", "DE"))
	assert.False(t, acceptLangGeoMismatch("de-DE", ""))
	assert.False(t, acceptLangGeoMismatch("de-DE", "DEU"))
}

func TestAcceptLangGeo_holdoutUnknownCountryFailOpen(t *testing.T) {
	assert.False(t, acceptLangGeoMismatch("en-US,en;q=0.9", "ZZ"))
}

func TestGeoFilter_acceptLangGeo(t *testing.T) {
	reg := &Registry{}
	campID := uuid.New()
	reg.manuallyAdded = map[uuid.UUID]bool{campID: true}
	reg.storeCampaignSnapshot(&campaignMapSnapshot{
		byID: map[uuid.UUID]campaignInfo{
			campID: {
				campaign: &domain.Campaign{
					ID:                   campID,
					AcceptLangGeoEnabled: true,
				},
			},
		},
	})

	f := NewGeoFilter(&MockGeoProvider{}, reg)
	f.SetAcceptLangGeoEnabled(true)

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	evt.CampaignID = campID
	evt.AcceptLang = "pt-BR,pt;q=0.9,en;q=0.8"
	evt.GeoCountry = "DE"
	evt.IngestGeoResolved = true
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)

	require.NoError(t, f.Check(context.Background(), evt))
	assert.True(t, acc.has(FraudReasonAcceptLangGeoMismatch))
}

func TestGeoFilter_acceptLangGeoDisabledByCampaign(t *testing.T) {
	reg := &Registry{}
	campID := uuid.New()
	reg.manuallyAdded = map[uuid.UUID]bool{campID: true}
	reg.storeCampaignSnapshot(&campaignMapSnapshot{
		byID: map[uuid.UUID]campaignInfo{
			campID: {
				campaign: &domain.Campaign{
					ID: campID,
				},
			},
		},
	})

	f := NewGeoFilter(&MockGeoProvider{}, reg)
	f.SetAcceptLangGeoEnabled(true)

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	evt.CampaignID = campID
	evt.AcceptLang = "pt-BR,pt;q=0.9"
	evt.GeoCountry = "DE"
	evt.IngestGeoResolved = true
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)

	require.NoError(t, f.Check(context.Background(), evt))
	assert.False(t, acc.has(FraudReasonAcceptLangGeoMismatch))
}

func TestGeoFilter_acceptLangGeo_holdoutCGNATDoesNotBypass(t *testing.T) {
	reg := &Registry{}
	campID := uuid.New()
	reg.manuallyAdded = map[uuid.UUID]bool{campID: true}
	reg.storeCampaignSnapshot(&campaignMapSnapshot{
		byID: map[uuid.UUID]campaignInfo{
			campID: {
				campaign: &domain.Campaign{
					ID:                   campID,
					AcceptLangGeoEnabled: true,
					CgnatIPPolicyEnabled: true,
				},
			},
		},
	})

	f := NewGeoFilter(&MockGeoProvider{}, reg)
	f.SetAcceptLangGeoEnabled(true)

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	evt.CampaignID = campID
	evt.AcceptLang = "pt-BR,pt;q=0.9"
	evt.GeoCountry = "DE"
	evt.IngestGeoResolved = true
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)

	require.NoError(t, f.Check(context.Background(), evt))
	assert.True(t, acc.has(FraudReasonAcceptLangGeoMismatch))
}
