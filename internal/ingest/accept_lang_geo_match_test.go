package ingest

import (
	"context"
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Holdout: inverted match logic rejects valid de-DE/DE and pt-BR/BR pairs (fail-closed on signal).
func TestAcceptLangGeo_holdoutMatchingLangPasses(t *testing.T) {
	assert.False(t, acceptLangGeoMismatch("de-DE,de;q=0.9,en;q=0.8", "DE"))
	assert.False(t, acceptLangGeoMismatch("pt-BR,pt;q=0.9", "BR"))
	assert.False(t, acceptLangGeoMismatch("en-US,en;q=0.9", "US"))
}

// Holdout: revert skips fraud signal on pt-BR vs DE and en-US vs DE (fail-closed on obvious mismatch).
func TestAcceptLangGeo_holdoutObviousMismatchFails(t *testing.T) {
	assert.True(t, acceptLangGeoMismatch("pt-BR,pt;q=0.9,en;q=0.8", "DE"))
	assert.True(t, acceptLangGeoMismatch("en-US,en;q=0.9", "DE"))
}

// Holdout: revert treats empty Accept-Language, empty geo, or non-ISO geo as mismatch (must fail-open).
func TestAcceptLangGeo_holdoutMissingInputsFailOpen(t *testing.T) {
	assert.False(t, acceptLangGeoMismatch("", "DE"))
	assert.False(t, acceptLangGeoMismatch("de-DE", ""))
	assert.False(t, acceptLangGeoMismatch("de-DE", "DEU"))
}

// Holdout: revert flags unknown ISO ZZ; primary-lang table miss must fail-open, not hard reject.
func TestAcceptLangGeo_holdoutUnknownCountryFailOpen(t *testing.T) {
	assert.False(t, acceptLangGeoMismatch("en-US,en;q=0.9", "ZZ"))
}

func TestGeoFilter_acceptLangGeo(t *testing.T) {
	reg := &Registry{}
	campID := uuid.New()
	reg.SeedCampaignForTest(&domain.Campaign{
		ID:                   campID,
		AcceptLangGeoEnabled: true,
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
	assert.True(t, acc.Has(FraudReasonAcceptLangGeoMismatch))
}

func TestGeoFilter_acceptLangGeoDisabledByCampaign(t *testing.T) {
	reg := &Registry{}
	campID := uuid.New()
	reg.SeedCampaignForTest(&domain.Campaign{ID: campID})

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
	assert.False(t, acc.Has(FraudReasonAcceptLangGeoMismatch))
}

// Holdout: CGNAT policy must not skip accept-lang/geo check when campaign enables both gates.
func TestGeoFilter_acceptLangGeo_holdoutCGNATDoesNotBypass(t *testing.T) {
	reg := &Registry{}
	campID := uuid.New()
	reg.SeedCampaignForTest(&domain.Campaign{
		ID:                   campID,
		AcceptLangGeoEnabled: true,
		CgnatIPPolicyEnabled: true,
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
	assert.True(t, acc.Has(FraudReasonAcceptLangGeoMismatch))
}
