package access

import (
	"sort"

	ctrlhttp "ad-event-processor/internal/control/http"
)

var capabilityPermissions = map[string][]string{
	"page.customers":              {ctrlhttp.PermCustomersRead},
	"page.campaigns":              {ctrlhttp.PermCampaignsReadMasked},
	"page.team":                   {"team:read"},
	"page.settings":               {ctrlhttp.PermSettingsRead},
	"page.exports":                {"exports:read"},
	"page.ops":                    {ctrlhttp.PermShardsRead},
	"page.audit":                  {ctrlhttp.PermAuditRead},
	"page.integrations":           {ctrlhttp.PermCampaignsRead},
	"page.integrations.postbacks": {"postbacks:read"},
	"page.billing":                {ctrlhttp.PermBillingRead},
	"page.rtb":                    {ctrlhttp.PermRtbRead},
	"data.campaigns.unmasked":     {ctrlhttp.PermCampaignsRead},
	"data.campaigns.masked_only":  {ctrlhttp.PermCampaignsReadMasked},
	"action.campaigns.create":     {ctrlhttp.PermCampaignsWrite},
	"action.campaigns.edit":       {ctrlhttp.PermCampaignsWrite},
	"action.campaigns.pause":      {ctrlhttp.PermCampaignsPause},
	"action.campaigns.archive":    {"campaigns:archive"},
	"action.campaigns.delete":     {"campaigns:delete"},
	"action.postbacks.view":       {"postbacks:read"},
	"action.postbacks.edit":       {"postbacks:write"},
	"action.exports.run":          {"exports:run"},
	"action.team.invite":          {"team:write"},
	"action.team.manage":          {"team:write"},
	"action.settings.edit":        {ctrlhttp.PermSettingsWrite},
	"action.blacklist.edit":       {ctrlhttp.PermBlacklistWrite},
	"action.billing.edit":         {ctrlhttp.PermBillingWrite},
	"action.access.read":          {ctrlhttp.PermAccessRead},
	"action.access.write":         {ctrlhttp.PermAccessWrite},
}

var capabilityCatalog = []CapabilityEntry{
	{ID: "page.customers", Label: "Customers", Group: "pages"},
	{ID: "page.campaigns", Label: "Campaigns", Group: "pages"},
	{ID: "page.team", Label: "Team", Group: "pages"},
	{ID: "page.settings", Label: "Settings", Group: "pages"},
	{ID: "page.exports", Label: "Exports", Group: "pages"},
	{ID: "page.ops", Label: "Ops", Group: "pages"},
	{ID: "page.audit", Label: "Audit", Group: "pages"},
	{ID: "page.integrations", Label: "Integrations hub", Group: "pages"},
	{ID: "page.integrations.postbacks", Label: "Postbacks / CAPI", Group: "pages"},
	{ID: "page.billing", Label: "Billing", Group: "pages"},
	{ID: "page.rtb", Label: "RTB", Group: "pages"},
	{ID: "data.campaigns.unmasked", Label: "Unmasked campaign data", Group: "data"},
	{ID: "data.campaigns.masked_only", Label: "Masked campaign data only", Group: "data"},
	{ID: "action.campaigns.create", Label: "Create campaigns", Group: "campaigns"},
	{ID: "action.campaigns.edit", Label: "Edit campaigns / flows", Group: "campaigns"},
	{ID: "action.campaigns.pause", Label: "Pause / resume", Group: "campaigns"},
	{ID: "action.campaigns.archive", Label: "Archive campaigns", Group: "campaigns"},
	{ID: "action.campaigns.delete", Label: "Delete campaigns", Group: "campaigns"},
	{ID: "action.postbacks.view", Label: "View postback config", Group: "integrations"},
	{ID: "action.postbacks.edit", Label: "Edit postback config", Group: "integrations"},
	{ID: "action.exports.run", Label: "Run exports", Group: "exports"},
	{ID: "action.team.invite", Label: "Invite team members", Group: "team"},
	{ID: "action.team.manage", Label: "Block / spend cap", Group: "team"},
	{ID: "action.settings.edit", Label: "Platform settings", Group: "admin"},
	{ID: "action.blacklist.edit", Label: "Fraud blacklist", Group: "admin"},
	{ID: "action.billing.edit", Label: "Billing mutations", Group: "admin"},
	{ID: "action.access.read", Label: "View access matrix", Group: "admin"},
	{ID: "action.access.write", Label: "Edit access matrix", Group: "admin"},
}

var permissionCatalog = []PermissionEntry{
	{ID: "*", Label: "Superuser wildcard", Internal: true},
	{ID: "customers:read", Label: "Customers read"},
	{ID: "customers:write", Label: "Customers write"},
	{ID: "campaigns:read", Label: "Campaigns read (full)"},
	{ID: "campaigns:read:masked", Label: "Campaigns read (masked)"},
	{ID: "campaigns:write", Label: "Campaigns write"},
	{ID: "campaigns:write:masked", Label: "Campaigns write (masked)"},
	{ID: "campaigns:pause", Label: "Campaigns pause/resume"},
	{ID: "campaigns:archive", Label: "Campaigns archive"},
	{ID: "campaigns:delete", Label: "Campaigns delete"},
	{ID: "postbacks:read", Label: "Postbacks read"},
	{ID: "postbacks:write", Label: "Postbacks write"},
	{ID: "exports:read", Label: "Exports read"},
	{ID: "exports:run", Label: "Exports run"},
	{ID: "team:read", Label: "Team read"},
	{ID: "team:write", Label: "Team write"},
	{ID: "brands:read", Label: "Brands read"},
	{ID: "brands:write", Label: "Brands write"},
	{ID: "billing:read", Label: "Billing read"},
	{ID: "billing:write", Label: "Billing write"},
	{ID: "settings:read", Label: "Settings read"},
	{ID: "settings:write", Label: "Settings write"},
	{ID: "blacklist:read", Label: "Blacklist read"},
	{ID: "blacklist:write", Label: "Blacklist write"},
	{ID: "audit:read", Label: "Audit read"},
	{ID: "users:write", Label: "Users write"},
	{ID: "shards:read", Label: "Shards read"},
	{ID: "shards:write", Label: "Shards write"},
	{ID: "ops:write", Label: "Ops write"},
	{ID: "rtb:read", Label: "RTB read"},
	{ID: "rtb:write", Label: "RTB write"},
	{ID: "supply:read:scoped", Label: "Supply read (scoped)"},
	{ID: "access:read", Label: "Access read"},
	{ID: "access:write", Label: "Access write"},
}

func Catalog() CatalogResponse {
	return CatalogResponse{
		Capabilities: capabilityCatalog,
		Permissions:  permissionCatalog,
	}
}

func KnownPermissions() map[string]struct{} {
	out := make(map[string]struct{}, len(permissionCatalog))
	for _, p := range permissionCatalog {
		out[p.ID] = struct{}{}
	}
	return out
}

func KnownCapabilities() map[string]struct{} {
	out := make(map[string]struct{}, len(capabilityCatalog))
	for _, c := range capabilityCatalog {
		out[c.ID] = struct{}{}
	}
	return out
}

func AllCatalogPermissionIDs() []string {
	ids := make([]string, 0, len(permissionCatalog))
	for _, p := range permissionCatalog {
		ids = append(ids, p.ID)
	}
	sort.Strings(ids)
	return ids
}
