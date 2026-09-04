// Seed catalog: deterministic display names and UUID helpers for db seed subcommand.
// Canonical fixture strings per ui.mdc; no trash tokens or round KPIs.
package main

import (
	"fmt"
	"strings"

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

const seedUUIDNamespaceDNS = "ad-event-processor.local.seed"

var seedUUIDNamespace = uuid.NewSHA1(uuid.NameSpaceDNS, []byte(seedUUIDNamespaceDNS))

func seedDeterministicUUID(entityKind string, seq int) uuid.UUID {
	return uuid.NewSHA1(seedUUIDNamespace, []byte(fmt.Sprintf("%s:%d", entityKind, seq)))
}

func seedCustomerUUID(seq int) uuid.UUID {
	return seedDeterministicUUID("customer", seq)
}

func seedBrandUUID(seq int) uuid.UUID {
	return seedDeterministicUUID("brand", seq)
}

func seedCreativeUUID(seq int) uuid.UUID {
	return seedDeterministicUUID("creative", seq)
}

func seedCampaignUUID(seq int) uuid.UUID {
	return seedDeterministicUUID("campaign", seq)
}

func seedDeploymentUUID() uuid.UUID {
	return seedDeterministicUUID("deployment", 1)
}

func seedLicenseRecordUUID() uuid.UUID {
	return seedDeterministicUUID("license", 1)
}

// seedEntityUUID is the campaign id helper used by UI demo seed and legacy callers.
func seedEntityUUID(seq int) uuid.UUID {
	return seedCampaignUUID(seq)
}

func seedCustomerName(seq int) string {
	idx := seq - 1
	base := seedCustomerNames[idx%len(seedCustomerNames)]
	region := seedCustomerRegionLabels[(idx*2+seq/3)%len(seedCustomerRegionLabels)]
	var name string
	switch idx % 4 {
	case 0:
		name = base
	case 1:
		name = fmt.Sprintf("%s — %s", base, region)
	case 2:
		name = fmt.Sprintf("%s (%s)", region, base)
	default:
		name = fmt.Sprintf("%s · %s desk", base, region)
	}
	if idx >= len(seedCustomerNames) {
		name = fmt.Sprintf("%s · group %d", name, 1+(seq%41))
	}
	return name
}

func seedCampaignName(seq int) string {
	idx := seq - 1
	base := seedCampaignNames[idx%len(seedCampaignNames)]
	geo := seedCampaignGeoTags[(idx*3+seq)%len(seedCampaignGeoTags)]
	desk := seedCampaignDeskTags[(idx*5+seq/3)%len(seedCampaignDeskTags)]
	wave := 1 + (seq % 53)

	var name string
	switch idx % 6 {
	case 0:
		name = base
	case 1:
		name = fmt.Sprintf("%s · %s", base, geo)
	case 2:
		name = fmt.Sprintf("%s (%s)", geo, base)
	case 3:
		goal := []string{"Install", "Lead gen", "Checkout", "Signup", "Trial", "LAL"}[(idx+seq)%6]
		name = fmt.Sprintf("%s — %s", base, goal)
	case 4:
		period := []string{"Q1", "Q2", "Q3", "Q4", "H2", "FY"}[(idx+seq/7)%6]
		name = fmt.Sprintf("%s %s %s", period, base, geo)
	default:
		name = fmt.Sprintf("%s / %s / %s", base, geo, desk)
	}
	if seq > len(seedCampaignNames) {
		name = fmt.Sprintf("%s · wave %d", name, wave)
	}
	return name
}

func seedBrandName(seq int) string {
	return seedBrandNames[(seq-1)%len(seedBrandNames)]
}

var seedCreativeNames = []string{
	"Hero carousel",
	"Video pre-roll",
	"Static banner",
	"Native card",
	"Interstitial",
	"Playable unit",
	"Rich media",
	"Search text",
	"Product feed",
	"Story placement",
	"Audio spot",
	"CTV bumper",
}

func seedCreativeDisplayName(seq int) string {
	idx := seq - 1
	geo := seedCampaignGeoTags[(idx/12)%len(seedCampaignGeoTags)]
	desk := seedCampaignDeskTags[(idx/(12*len(seedCampaignGeoTags)))%len(seedCampaignDeskTags)]
	return fmt.Sprintf("%s - %s - %s", seedCreativeNames[idx%len(seedCreativeNames)], geo, desk)
}

func seedBrandDisplayName(seq int) string {
	idx := seq - 1
	region := seedCustomerRegionLabels[(idx/10)%len(seedCustomerRegionLabels)]
	return fmt.Sprintf("%s - %s", seedBrandName(seq), region)
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

var seedUiDemoCountryCodes = []string{"US", "GB", "DE", "CA", "UA", "FR", "JP", "AU", "BR", "MX"}

func seedUiDemoTargetCountries(seq int) []string {
	count := 1 + (seq % 4)
	out := make([]string, count)
	for i := 0; i < count; i++ {
		out[i] = seedUiDemoCountryCodes[(seq+i)%len(seedUiDemoCountryCodes)]
	}
	return out
}

func formatPostgresTextArray(values []string) string {
	if len(values) == 0 {
		return "{}"
	}
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
	}
	return "{" + strings.Join(parts, ",") + "}"
}
