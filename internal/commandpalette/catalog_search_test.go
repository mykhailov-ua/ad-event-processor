package commandpalette

import (
	"context"
	"strings"
	"testing"

	"ad-event-processor/internal/controlplane/authz"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fullReadSnapshot() authz.Snapshot {
	return authz.Snapshot{
		Permissions: map[string]struct{}{
			authz.PermCampaignsRead: {},
			"billing:read":          {},
			"customers:read":        {},
			"ops:read":              {},
			"rtb:read":              {},
			"fraud:read":            {},
			"audit:read":            {},
		},
		Mask: authz.MaskFull,
	}
}

func TestCommandPalette_reportSearchParity_holdout(t *testing.T) {
	t.Parallel()
	ctx := authz.WithSnapshot(t.Context(), fullReadSnapshot())
	for _, entry := range reportNavEntries {
		word := firstSearchToken(entry.Label)
		if word == "" {
			continue
		}
		candidates := searchCatalog(ctx, word, 25, catalogKindSet{searchReports: true}, func(string) bool { return true })
		items := mergeSearchResults(25, candidates)
		found := false
		for _, item := range items {
			if item.ID == entry.ID {
				found = true
				break
			}
		}
		assert.True(t, found, "report %q not searchable by token %q", entry.ReportKey, word)
	}
}

func firstSearchToken(label string) string {
	for _, part := range strings.Fields(label) {
		part = strings.TrimSpace(part)
		if len(part) >= MinSearchQueryLen {
			return strings.ToLower(part)
		}
	}
	return ""
}

func TestSearchCatalog_costSyncRouteAndReport(t *testing.T) {
	t.Parallel()
	ctx := authz.WithSnapshot(t.Context(), fullReadSnapshot())
	candidates := searchCatalog(ctx, "cost sync", 25, catalogKindSet{searchReports: true, searchRoutes: true}, nil)
	items := mergeSearchResults(25, candidates)
	require.NotEmpty(t, items)
	var hasRoute bool
	var hasReport bool
	for _, item := range items {
		if item.Href == "/integrations/cost-sync" {
			hasRoute = true
		}
		if item.Href == "/reports/cost-sync-coverage" {
			hasReport = true
		}
	}
	assert.True(t, hasRoute)
	assert.True(t, hasReport)
}

func TestSearchCatalog_reportKindFilter(t *testing.T) {
	t.Parallel()
	ctx := authz.WithSnapshot(t.Context(), fullReadSnapshot())
	candidates := searchCatalog(ctx, "fraud", 25, catalogKindSet{searchReports: true}, nil)
	items := mergeSearchResults(25, candidates)
	require.NotEmpty(t, items)
	for _, item := range items {
		assert.Equal(t, "report", item.Kind)
	}
}

func TestCommandPalette_search_licenseGatedReport_holdout(t *testing.T) {
	t.Parallel()
	ctx := authz.WithSnapshot(t.Context(), authz.Snapshot{
		Permissions: map[string]struct{}{
			authz.PermCampaignsRead: {},
			"rtb:read":              {},
		},
		Mask: authz.MaskFull,
	})
	candidates := searchCatalog(ctx, "rtb", 25, catalogKindSet{searchReports: true, searchRoutes: true}, func(featureKey string) bool {
		return featureKey != "openrtb"
	})
	items := mergeSearchResults(25, candidates)
	for _, item := range items {
		assert.NotEqual(t, "/reports/rtb-overview", item.Href)
		assert.NotEqual(t, "/rtb", item.Href)
	}
}

func TestCommandPalette_action_requiresPermission_holdout(t *testing.T) {
	t.Parallel()
	ctx := authz.WithSnapshot(t.Context(), authz.Snapshot{
		Permissions: map[string]struct{}{
			authz.PermCampaignsRead: {},
		},
		Mask: authz.MaskFull,
	})
	candidates := searchCatalog(ctx, "campaign", 25, catalogKindSet{searchActions: true}, nil)
	items := mergeSearchResults(25, candidates)
	for _, item := range items {
		assert.NotEqual(t, "/campaigns/new", item.Href)
		assert.NotEqual(t, "/campaigns/migrate", item.Href)
	}
}

func TestService_Search_mergesCatalogAndEntities(t *testing.T) {
	t.Parallel()
	ctx := authz.WithSnapshot(t.Context(), fullReadSnapshot())
	svc := &Service{Store: &recordingSearcher{
		items: []ItemDTO{{
			ID:    "00000000-0000-4000-8000-000000000099",
			Kind:  "campaign",
			Label: "Camp Alpha",
			Href:  "/campaigns/00000000-0000-4000-8000-000000000099",
			Group: "campaigns",
		}},
	}}
	resp := svc.Search(ctx, uuid.MustParse("00000000-0000-4000-8000-000000000001"), "camp", 25, nil, nil)
	require.NotEmpty(t, resp.Items)
	var hasCampaign bool
	var hasRoute bool
	for _, item := range resp.Items {
		if item.Kind == "campaign" {
			hasCampaign = true
		}
		if item.Href == "/campaigns" {
			hasRoute = true
		}
	}
	assert.True(t, hasCampaign)
	assert.True(t, hasRoute)
}

func TestService_Search_catalogOnlySkipsStore(t *testing.T) {
	t.Parallel()
	ctx := authz.WithSnapshot(t.Context(), fullReadSnapshot())
	rec := &recordingSearcher{}
	svc := &Service{Store: rec}
	resp := svc.Search(ctx, uuid.MustParse("00000000-0000-4000-8000-000000000001"), "cost", 25, []string{"report"}, nil)
	require.NotEmpty(t, resp.Items)
	assert.False(t, rec.called)
}

func TestService_Search_pgFailureReturnsCatalogDegraded(t *testing.T) {
	t.Parallel()
	ctx := authz.WithSnapshot(t.Context(), fullReadSnapshot())
	svc := &Service{Store: failingSearcher{}}
	resp := svc.Search(ctx, uuid.MustParse("00000000-0000-4000-8000-000000000001"), "cost", 25, nil, nil)
	require.NotEmpty(t, resp.Items)
	assert.True(t, resp.Degraded)
}

type failingSearcher struct{}

func (failingSearcher) SearchEntities(context.Context, uuid.UUID, string, int, []string) ([]ItemDTO, error) {
	return nil, assert.AnError
}
