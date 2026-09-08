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
	"Summer checkout retarget",
	"Velox trial onboarding",
	"Horizon brand lift Q3",
	"Sportsbook install tier-1",
	"Insurance quote funnel",
	"Solar panel CPL west",
	"Fintech card signup",
	"Ecom cart abandoners",
	"Mobile game level-10",
	"B2B SaaS demo requests",
	"Travel meta search spring",
	"Telco prepaid acquisition",
	"Crypto exchange KYC",
	"Nutra sweepstakes LP",
	"Remittance app re-engagement",
	"UPI wallet funding",
	"Remarketing catalog sales",
	"Native article placements",
	"Push notification winback",
	"Search non-brand conquest",
	"Display prospecting broad",
	"Connected TV awareness",
	"Podcast host read spots",
	"Influencer whitelisting burst",
	"Affiliate coupon codes",
	"Lead gen whitepaper gate",
	"Webinar registration drive",
	"App store search boost",
	"Cross-sell existing buyers",
	"Loyalty tier upgrade",
	"Seasonal promo burst",
	"Black Friday warm-up",
	"Back-to-school supplies",
	"Holiday gift guides",
	"Valentine flash offers",
	"Mother's day gift lane",
	"Back to campus wifi",
	"Payday loan alternate",
	"Mortgage rate compare",
	"Auto insurance quotes",
	"Pet insurance trials",
	"Streaming trial starts",
	"Meal kit first box",
	"Fashion lookalike scale",
	"Beauty sample boxes",
	"Home security installs",
	"Smart thermostat leads",
	"VPN annual plans",
	"Password manager trials",
	"EdTech course enroll",
	"Language app premium",
	"Fitness app reactivation",
	"Meditation subscription",
	"Dating app installs",
	"Food delivery credits",
	"Ride share referrals",
	"Hotel booking meta",
	"Flight deal alerts",
	"Cruise package leads",
	"Real estate listings",
	"Rental apartment tours",
	"Moving services quotes",
	"Storage unit promos",
	"Legal consultation intake",
	"Tax prep early bird",
	"Payroll software trials",
	"Invoicing SMB signup",
	"CRM free tier upgrade",
	"Hosting migration offer",
	"Domain renewal nudge",
	"Email marketing trials",
	"Analytics SDK adoption",
	"Dev tools freemium",
	"Cloud credits campaign",
	"Cybersecurity audits",
	"Backup software DR",
	"Printer ink subscribe",
	"Office supplies bulk",
	"Wholesale marketplace",
	"Dropship supplier intro",
	"Marketplace seller onboarding",
	"POS hardware bundle",
	"Inventory sync SaaS",
	"Loyalty card wallet",
	"Gift card marketplace",
	"Cashback browser ext",
	"Comparison shopping feed",
	"Price drop alerts",
	"Review site sponsorship",
	"Forum community ads",
	"Discord server boosts",
	"Twitch stream overlays",
	"YouTube pre-roll tests",
	"Reddit conversation ads",
	"Pinterest shopping pins",
	"Snap AR lens trial",
	"TikTok spark posts",
	"Meta advantage+ scale",
	"Google PMax feed-only",
	"Microsoft audience network",
	"Taboola content recirc",
	"Outbrain premium pubs",
	"Revcontent native lane",
	"MGID widget rotation",
	"Propeller push subs",
	"RichAds popunder route",
	"ExoClick video bumper",
	"Adsterra multi-format",
	"Galaksion smartlink mix",
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

var seedCampaignGoalLabels = []string{
	"Install push", "Lead gen", "Checkout retarget", "Signup lift", "Trial start", "Lookalike scale",
}

var seedCampaignFlightLabels = []string{
	"Primary flight", "Secondary flight", "Holdout cell", "Scale pass", "Refresh run",
	"Lift test", "Winback push", "Prospect pass", "Remarket pass", "Evergreen run",
	"Seasonal push", "Promo burst", "Catalog test", "Offer test", "Audience pass",
	"Geo expansion", "Budget scale", "Bid floor test", "Creative pass", "Channel mix",
	"Partner run", "Direct pass", "Affiliate burst", "Retarget pass", "Acquisition pass",
	"Conversion lift", "Signup pass", "Trial pass", "Install pass", "Checkout pass",
	"Lead pass", "Loyalty pass", "Reactivation pass", "Upsell pass",
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

func seedCustomerName(seq int) string {
	idx := seq - 1
	base := seedCustomerNames[idx%len(seedCustomerNames)]
	if seq <= len(seedCustomerNames) {
		return base
	}
	cycle := (seq - 1) / len(seedCustomerNames)
	region := seedCustomerRegionLabels[(idx+cycle)%len(seedCustomerRegionLabels)]
	goal := seedCampaignGoalLabels[(idx+seq+cycle)%len(seedCampaignGoalLabels)]
	switch cycle % 3 {
	case 0:
		return fmt.Sprintf("%s - %s", base, region)
	case 1:
		return fmt.Sprintf("%s - %s", base, goal)
	default:
		return fmt.Sprintf("%s %s", region, base)
	}
}

func seedCampaignName(seq int) string {
	idx := seq - 1
	if seq <= len(seedCampaignNames) {
		return seedCampaignNames[idx]
	}
	variantIndex := idx - len(seedCampaignNames)
	baseCount := len(seedCampaignNames)
	goalCount := len(seedCampaignGoalLabels)
	baseIdx := variantIndex % baseCount
	remainder := variantIndex / baseCount
	goalIdx := remainder % goalCount
	flightIdx := remainder / goalCount
	base := seedCampaignNames[baseIdx]
	flight := seedCampaignFlightLabels[flightIdx%len(seedCampaignFlightLabels)]
	goal := seedCampaignGoalLabels[goalIdx]
	switch variantIndex % 4 {
	case 0:
		return fmt.Sprintf("%s - %s %s", base, flight, goal)
	case 1:
		return fmt.Sprintf("%s (%s, %s)", base, flight, goal)
	case 2:
		return fmt.Sprintf("%s, %s - %s", flight, base, goal)
	default:
		return fmt.Sprintf("%s / %s / %s", base, flight, goal)
	}
}

func loadTestSequentialUUID(seq int) string {
	return fmt.Sprintf("00000000-0000-0000-0000-%012x", seq)
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
	kind := seedCreativeNames[idx%len(seedCreativeNames)]
	if seq <= len(seedCreativeNames) {
		return kind
	}
	brand := seedBrandNames[(idx/len(seedCreativeNames))%len(seedBrandNames)]
	return fmt.Sprintf("%s - %s", kind, brand)
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

var seedUIDemoCountryCodes = []string{"US", "GB", "DE", "CA", "UA", "FR", "JP", "AU", "BR", "MX"}

func seedUIDemoTargetCountries(seq int) []string {
	count := 1 + (seq % 4)
	out := make([]string, count)
	for i := range count {
		out[i] = seedUIDemoCountryCodes[(seq+i)%len(seedUIDemoCountryCodes)]
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
