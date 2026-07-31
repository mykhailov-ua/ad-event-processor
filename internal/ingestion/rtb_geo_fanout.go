package ingestion

import (
	"sort"

	"espx/internal/domain"
	"espx/internal/rtb"
)

func sortedTargetCountries(camp *domain.Campaign) []string {
	if camp == nil || len(camp.TargetCountries) == 0 {
		return nil
	}
	out := make([]string, 0, len(camp.TargetCountries))
	for c := range camp.TargetCountries {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

func fanOutRtbCatalogRows(camp *domain.Campaign, base RtbCampaignInput) []rtb.CampaignData {
	countries := sortedTargetCountries(camp)
	if len(countries) == 0 {
		return []rtb.CampaignData{CampaignDataFromDomain(camp, base)}
	}
	out := make([]rtb.CampaignData, 0, len(countries))
	for _, country := range countries {
		inp := base
		inp.GeoHash = GeoHashFromCountry(country)
		out = append(out, CampaignDataFromDomain(camp, inp))
	}
	return out
}
