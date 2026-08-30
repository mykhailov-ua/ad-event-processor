package commandpalette

import (
	"context"

	"ad-event-processor/internal/controlplane/authz"
)

var staticNavEntries = []navEntry{
	{ID: "nav:campaigns", Kind: "route", Label: "Campaigns", Href: "/campaigns", Permissions: []string{authz.PermCampaignsRead, authz.PermCampaignsReadMasked}, Group: "campaigns"},
	{ID: "nav:flows", Kind: "route", Label: "Flows", Href: "/flows", Permissions: []string{authz.PermCampaignsRead, authz.PermCampaignsReadMasked}, Group: "campaigns"},
	{ID: "nav:landers", Kind: "route", Label: "Landers", Href: "/landers", Permissions: []string{authz.PermCampaignsRead, authz.PermCampaignsReadMasked}, Group: "campaigns"},
	{ID: "nav:offers", Kind: "route", Label: "Offers", Href: "/offers", Permissions: []string{authz.PermCampaignsRead, authz.PermCampaignsReadMasked}, Group: "campaigns"},
	{ID: "nav:brands", Kind: "route", Label: "Brands", Href: "/brands", Permissions: []string{authz.PermCampaignsRead, authz.PermCampaignsReadMasked}, Group: "campaigns"},
	{ID: "nav:reports", Kind: "route", Label: "Reports", Href: "/reports", Permissions: []string{authz.PermCampaignsRead, authz.PermCampaignsReadMasked}, Group: "reports"},
	{ID: "nav:customers", Kind: "route", Label: "Customers", Href: "/customers", Permissions: []string{"customers:read"}, Group: "admin"},
	{ID: "nav:billing", Kind: "route", Label: "Billing", Href: "/billing", Permissions: []string{authz.PermBillingRead}, Group: "billing"},
	{ID: "nav:integrations", Kind: "route", Label: "Integrations", Href: "/integrations", Permissions: []string{authz.PermCampaignsRead, authz.PermCampaignsReadMasked}, Group: "integrations"},
	{ID: "nav:postbacks", Kind: "route", Label: "CAPI and postbacks", Href: "/integrations/postbacks", Permissions: []string{authz.PermCampaignsRead, authz.PermCampaignsReadMasked}, Group: "integrations"},
	{ID: "nav:cost-sync", Kind: "route", Label: "Cost sync", Href: "/integrations/cost-sync", Permissions: []string{authz.PermCampaignsRead, authz.PermCampaignsReadMasked}, Group: "integrations"},
	{ID: "nav:automation", Kind: "route", Label: "Automation rules", Href: "/integrations/automation", Permissions: []string{authz.PermCampaignsRead, authz.PermCampaignsReadMasked}, Group: "integrations"},
	{ID: "nav:smart-alerts", Kind: "route", Label: "Smart alerts", Href: "/integrations/smart-alerts", Permissions: []string{authz.PermCampaignsRead, authz.PermCampaignsReadMasked}, Group: "integrations"},
	{ID: "nav:margin-guard", Kind: "route", Label: "Margin guard", Href: "/integrations/margin-guard", Permissions: []string{authz.PermCampaignsRead, authz.PermCampaignsReadMasked}, Group: "integrations"},
	{ID: "nav:traffic-optimizer", Kind: "route", Label: "Traffic optimizer", Href: "/integrations/traffic-optimizer", Permissions: []string{authz.PermCampaignsRead, authz.PermCampaignsReadMasked}, Group: "integrations"},
	{ID: "nav:platform-campaigns", Kind: "route", Label: "Platform sync", Href: "/platform-campaigns", Permissions: []string{authz.PermCampaignsRead, authz.PermCampaignsReadMasked}, Group: "integrations", LicenseGated: true, FeatureKey: "ad_platform_campaign_api"},
	{ID: "nav:rtb", Kind: "route", Label: "RTB admin", Href: "/rtb", Permissions: []string{"rtb:read"}, Group: "rtb", LicenseGated: true, FeatureKey: "openrtb"},
	{ID: "nav:doctor", Kind: "route", Label: "Doctor", Href: "/ops/doctor", Permissions: []string{"ops:read"}, Group: "ops"},
	{ID: "action:new-campaign", Kind: "action", Label: "New campaign", Href: "/campaigns/new", Permissions: []string{authz.PermCampaignsWrite, authz.PermCampaignsWriteMask}, Group: "actions"},
	{ID: "action:migration-import", Kind: "action", Label: "Migration import", Href: "/campaigns/migrate", Permissions: []string{authz.PermCampaignsWrite, authz.PermCampaignsWriteMask}, Group: "actions"},
}

var RequiredLiveNavPaths = []string{
	"/campaigns",
	"/flows",
	"/landers",
	"/offers",
	"/integrations/postbacks",
	"/integrations/cost-sync",
	"/integrations/automation",
	"/integrations/smart-alerts",
	"/integrations/margin-guard",
	"/integrations/traffic-optimizer",
	"/billing",
	"/reports",
	"/customers",
	"/ops/doctor",
}

func NavCatalogEntries() []navEntry {
	out := make([]navEntry, 0, len(staticNavEntries)+len(reportNavEntries))
	out = append(out, staticNavEntries...)
	out = append(out, reportNavEntries...)
	return out
}

func FilterNavCatalog(ctx context.Context, entries []navEntry, licenseAllowed func(string) bool) []ItemDTO {
	snap, ok := authz.SnapshotFromContext(ctx)
	if !ok {
		return nil
	}
	out := make([]ItemDTO, 0, len(entries))
	for _, entry := range entries {
		if !navEntryAllowed(snap, entry, licenseAllowed) {
			continue
		}
		out = append(out, navEntryToItem(entry))
	}
	return out
}

func navEntryAllowed(snap authz.Snapshot, entry navEntry, licenseAllowed func(string) bool) bool {
	if len(entry.Permissions) > 0 && !snap.HasAny(entry.Permissions...) {
		return false
	}
	if entry.ReportKey == "fraud-evidence-pack" && snap.Mask == authz.MaskMasked {
		return false
	}
	if entry.LicenseGated && entry.FeatureKey != "" {
		if licenseAllowed == nil || !licenseAllowed(entry.FeatureKey) {
			return false
		}
	}
	return true
}

func navEntryToItem(entry navEntry) ItemDTO {
	return ItemDTO{
		ID:    entry.ID,
		Kind:  entry.Kind,
		Label: entry.Label,
		Href:  entry.Href,
		Meta:  entry.Meta,
		Group: entry.Group,
	}
}

func catalogHrefSet(entries []navEntry) map[string]struct{} {
	set := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		set[entry.Href] = struct{}{}
	}
	return set
}
