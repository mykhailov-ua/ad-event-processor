package ingestion

import (
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
)

func cgnatIPPolicyActive(globalBypass bool, camp *domain.Campaign) bool {
	if globalBypass {
		return true
	}
	return camp != nil && camp.CgnatIPPolicyEnabled
}

func shouldBypassCGNATIPVelocity(
	globalBypass bool,
	camp *domain.Campaign,
	carrierTable *MobileCarrierASNTable,
	lookup ASNLookup,
	ip string,
	signal string,
) bool {
	if !cgnatIPPolicyActive(globalBypass, camp) {
		return false
	}
	if carrierTable == nil || lookup == nil || ip == "" {
		return false
	}
	asn, ok := lookup.LookupASN(ip)
	if !ok || asn == 0 || !carrierTable.IsMobileCarrier(asn) {
		return false
	}
	if signal != "" {
		metrics.CGNATIPBypassTotal.WithLabelValues(signal).Inc()
	}
	return true
}

func cgnatBypassForCampaign(
	globalBypass bool,
	registry domain.CampaignRegistry,
	campaignID uuid.UUID,
	carrierTable *MobileCarrierASNTable,
	lookup ASNLookup,
	ip string,
	signal string,
) bool {
	var camp *domain.Campaign
	if registry != nil && campaignID != uuid.Nil {
		camp, _ = registry.GetCampaign(campaignID)
	}
	return shouldBypassCGNATIPVelocity(globalBypass, camp, carrierTable, lookup, ip, signal)
}

func asnLookupFromGeo(geo GeoProvider) ASNLookup {
	if geo == nil {
		return nil
	}
	lookup, ok := geo.(ASNLookup)
	if !ok {
		return nil
	}
	return lookup
}
