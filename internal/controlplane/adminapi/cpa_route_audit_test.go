package adminapi

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// cpaProductGap documents API/UI debt tracked in MILESTONE.md §1.2 (CPA-M0.1).
type cpaProductGap struct {
	Area        string
	API         string // "METHOD path"; empty when not implemented yet
	APIRequired bool   // when true, route must exist in adminapi catalog
	UI          string // client route; empty when N/A
	UIMissing   bool   // when true, UI route must not exist yet
	Phase       string
}

var cpaProductGaps = []cpaProductGap{
	{Area: "Customer billing forecast", API: "GET /api/v1/customers/{id}/billing/forecast", APIRequired: true, UI: "/customers/:id", UIMissing: false, Phase: "CPA-M2"},
	{Area: "Invoice ledger lines", API: "GET /api/v1/billing/invoices/{id}/ledger-lines", APIRequired: true, UI: "/billing/invoices/:id", UIMissing: false, Phase: "CPA-M2"},
	{Area: "Postback/CAPI DLQ (buyer)", API: "GET /api/v1/postbacks/dlq", APIRequired: true, UI: "/integrations/postbacks", UIMissing: false, Phase: "CPA-M4"},
	{Area: "Team invite / assign", API: "POST /api/v1/team/members", APIRequired: true, UI: "/team", UIMissing: false, Phase: "CPA-M5"},
	{Area: "Publisher dashboard", API: "GET /api/v1/publisher/dashboard", APIRequired: true, UI: "/publisher", UIMissing: false, Phase: "CPA-M6"},
	{Area: "Self-serve portal", API: "GET /api/v1/selfserve/templates", APIRequired: true, UI: "/selfserve", UIMissing: false, Phase: "CPA-M7"},
	{Area: "Unified DLQ inbox", API: "GET /api/v1/ops/dlq/inbox", APIRequired: true, UI: "/ops/dlq", UIMissing: false, Phase: "CPA-M8"},
	{Area: "Consent proof browser", API: "GET /api/v1/ops/consent/proofs", APIRequired: true, UI: "/ops/consent", UIMissing: false, Phase: "CPA-M8"},
	{Area: "Support feedback form", API: "POST /api/v1/support/feedback", APIRequired: true, UI: "/support/feedback", UIMissing: false, Phase: "CPA-M8"},
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, parent, dir, "go.mod not found")
		dir = parent
	}
}

func readRepoFile(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, rel))
	require.NoError(t, err)
	return string(b)
}

func catalogRouteSet() map[string]struct{} {
	out := make(map[string]struct{}, len(routeCatalog))
	for _, r := range routeCatalog {
		out[r.Method+" "+r.Path] = struct{}{}
	}
	return out
}

func extractLiveReportKeys(src string) []string {
	re := regexp.MustCompile(`\{\s*key:\s*'([^']+)'[^}]*live:\s*true`)
	matches := re.FindAllStringSubmatch(src, -1)
	keys := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		k := m[1]
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		keys = append(keys, k)
	}
	return keys
}

// TestCPA_LiveReports_haveAPIAndUIRoute ensures live REPORT_CATALOG entries are wired (CPA-M0.1).
func TestCPA_LiveReports_haveAPIAndUIRoute(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	reportTS := readRepoFile(t, root, "web/src/models/report.ts")
	appRoutes := readRepoFile(t, root, "web/src/app_routes.tsx")
	catalog := catalogRouteSet()

	for _, key := range extractLiveReportKeys(reportTS) {
		if key == "telegram" {
			require.Contains(t, appRoutes, "/reports/telegram", "telegram report route")
			continue
		}
		api := "GET /api/v1/reports/" + key
		_, ok := catalog[api]
		require.True(t, ok, "live report %q missing API catalog entry %s", key, api)
		require.Contains(t, appRoutes, "/reports/"+key, "live report %q missing app route", key)
	}
}

// TestCPA_DocumentedProductGaps_open verifies tracked debt: API exists, UI route still absent (CPA-M0.1).
func TestCPA_DocumentedProductGaps_open(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	appRoutes := readRepoFile(t, root, "web/src/app_routes.tsx")
	catalog := catalogRouteSet()

	for _, gap := range cpaProductGaps {
		if gap.APIRequired && gap.API != "" {
			_, ok := catalog[gap.API]
			require.True(t, ok, "gap %q: API %s must exist in catalog (phase %s)", gap.Area, gap.API, gap.Phase)
		}
		if !gap.UIMissing || gap.UI == "" {
			continue
		}
		require.NotContains(t, appRoutes, `path="`+gap.UI+`"`, "gap %q UI %s should stay open until %s", gap.Area, gap.UI, gap.Phase)
	}
}

// TestCPA_PatchCampaignRequest_parity ensures Go PATCH DTO exposes CPA-M1 fields (harness: campaign_patch_honest).
func TestCPA_PatchCampaignRequest_parity(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	campaignTS := readRepoFile(t, root, "web/src/types/api/campaign.ts")
	dtoGo := readRepoFile(t, root, "internal/controlplane/adminapi/campaign_dto.go")

	required := []string{"status", "budget_limit", "budget_limit_micro", "start_at", "end_at", "daypart_hours"}
	for _, field := range required {
		require.Contains(t, campaignTS, field+":", "CampaignPatchBody should document %s", field)
	}

	start := strings.Index(dtoGo, "type PatchCampaignRequest struct")
	require.GreaterOrEqual(t, start, 0)
	patchBlock := dtoGo[start:]
	if idx := regexp.MustCompile(`\ntype [A-Z]`).FindStringIndex(patchBlock[1:]); idx != nil {
		patchBlock = patchBlock[:idx[0]+1]
	}
	for _, field := range required {
		require.Contains(t, patchBlock, "`json:\""+field, "PatchCampaignRequest must expose %s (harness: %s)", field, HarnessCampaignPatchHonest)
	}
}

// TestCPA_SelfServeShell_forbidsOperatorNav ensures G4: no operator paths in selfserve shell (CPA-M7).
func TestCPA_SelfServeShell_forbidsOperatorNav(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	shell := readRepoFile(t, root, "web/src/components/selfserve_shell_layout.tsx")
	for _, forbidden := range []string{"/ops", "/customers", "/shards"} {
		require.NotContains(t, shell, forbidden, "selfserve shell must not link %s", forbidden)
	}
	require.Contains(t, shell, "data-testid=\"selfserve-shell\"")
}
