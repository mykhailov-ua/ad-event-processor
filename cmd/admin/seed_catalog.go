// Seed catalog: deterministic display names and UUID helpers for db seed subcommand.
// Canonical fixture strings per ui.mdc; no trash tokens or round KPIs.
package main

import (
	"fmt"

	"github.com/google/uuid"
)

var seedCustomerNames = []string{
	"Horizon Media Group",
	"Pacific Ads Studio",
	"Nordic Performance Co",
	"Atlas Buying Desk",
	"Summit Traffiq",
	"Bluewave Partners",
	"Velocity Affiliates",
	"Prime Reach Agency",
	"Lumen Digital",
	"Crestline Media",
	"Meridian Performance",
	"Vantage Growth Labs",
	"Redwood Acquisition",
	"Kite & Compass Media",
	"Northgate Buying",
	"Silverline Performance",
	"Harborfront Ads",
	"Quartzlane Partners",
	"Everpeak Media",
	"Bridgeport Digital",
}

var seedCampaignNames = []string{
	"US Summer Surge",
	"EU Retargeting V2",
	"LATAM Mobile App",
	"Global Video Reach",
	"APAC Crypto Swap",
	"Nordic Ecom Promo",
	"DACH High Intent",
	"UK Search Ads Q3",
	"US Gaming Install",
	"SA Ecom Flash",
	"SaaS Leads Global",
	"Fintech Acquisition",
	"B2B Enterprise EU",
	"APAC Direct Sales",
	"US Display Retarget",
	"Crypto Exchange VIP",
	"Mobile Gaming Tier1",
	"EU Ecom Sales",
	"US Performance Push",
	"Global Brand Lift",
	"DE Finance Leads",
	"BR Nutra Push",
	"JP Mobile Subs",
	"CA Insurance CPL",
	"AU Solar Quotes",
	"MX Remittance App",
	"IN UPI Onboarding",
	"PL Ecom Remarketing",
	"IT Travel Meta",
	"ES Telco Prepaid",
}

var seedBrandNames = []string{
	"Velox Checkout",
	"Northstar Finance",
	"Pulse Health",
	"Orbit Travel",
	"Nova SaaS",
	"Harbor Insurance",
	"Kite Mobility",
	"Summit Ecom",
	"Lumen EdTech",
	"Crestline VPN",
}

var seedUserLocalParts = []string{
	"ops", "media.buyer", "finance", "growth", "traffic", "analytics", "partnerships", "dev", "campaigns", "billing",
}

var seedCustomerRegionLabels = []string{
	"US East", "US West", "EU North", "APAC", "LATAM",
}

var seedCampaignGeoTags = []string{
	"US", "GB", "CA", "UA", "DE", "FR", "JP",
}

var seedCampaignDeskTags = []string{
	"Alpha desk", "Bravo desk", "Cedar desk", "Delta desk", "Echo desk",
}

func seedEntityUUID(seq int) uuid.UUID {
	id, err := uuid.Parse(fmt.Sprintf("00000000-0000-0000-0000-%012x", seq))
	if err != nil {
		panic(err)
	}
	return id
}

func seedCustomerName(seq int) string {
	idx := seq - 1
	base := seedCustomerNames[idx%len(seedCustomerNames)]
	region := seedCustomerRegionLabels[(idx/len(seedCustomerNames))%len(seedCustomerRegionLabels)]
	return fmt.Sprintf("%s — %s", base, region)
}

func seedCampaignName(seq int) string {
	idx := seq - 1
	base := seedCampaignNames[idx%len(seedCampaignNames)]
	geo := seedCampaignGeoTags[(idx/len(seedCampaignNames))%len(seedCampaignGeoTags)]
	desk := seedCampaignDeskTags[(idx/(len(seedCampaignNames)*len(seedCampaignGeoTags)))%len(seedCampaignDeskTags)]
	return fmt.Sprintf("%s · %s · %s", base, geo, desk)
}

func seedBrandName(seq int) string {
	return seedBrandNames[(seq-1)%len(seedBrandNames)]
}

func seedCustomerBalanceMicro(seq int) int64 {
	base := int64(2_400_000_000)
	step := int64(2_817_431)
	spread := int64(281_600_000_000)
	return base + ((int64(seq)*step)%spread + int64(seq%7)*97_000_000)
}

func seedUserEmail(seq int) string {
	domain := []string{
		"horizon-media.io",
		"pacific-ads.studio",
		"nordic-performance.co",
		"atlas-buying.com",
		"summit-traffiq.net",
	}[seq%5]
	local := seedUserLocalParts[seq%len(seedUserLocalParts)]
	return fmt.Sprintf("%s+%d@%s", local, 100+seq, domain)
}
