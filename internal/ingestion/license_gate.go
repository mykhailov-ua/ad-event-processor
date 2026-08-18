package ingestion

import (
	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/licensing"
)

type LicenseStateReader interface {
	GetLicenseState() (licensing.LicenseState, licensing.Entitlements)
}

func openRTBLicenseAllowed(reg domain.CampaignRegistry) bool {
	if reg == nil {
		return true
	}
	reader, ok := reg.(LicenseStateReader)
	if !ok {
		return true
	}
	state, ent := reader.GetLicenseState()
	if !licensing.SeedGateOpenRTB(ent) {
		return false
	}
	return licensing.OpenRTBAllowed(state, ent)
}
