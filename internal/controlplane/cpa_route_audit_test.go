package controlplane

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type cpaProductGap struct {
	Area        string
	API         string
	APIRequired bool
	UI          string
	UIMissing   bool
	Phase       string
}

var cpaProductGaps = []cpaProductGap{
	{Area: "Customer billing forecast", API: "GET /api/v1/customers/{id}/billing/forecast", APIRequired: true, UI: "/customers/:id", UIMissing: false, Phase: "customer_billing"},
	{Area: "Invoice ledger lines", API: "GET /api/v1/billing/invoices/{id}/ledger-lines", APIRequired: true, UI: "/billing/invoices/:id", UIMissing: false, Phase: "customer_billing"},
	{Area: "Postback/CAPI DLQ (buyer)", API: "GET /api/v1/postbacks/dlq", APIRequired: true, UI: "/integrations/postbacks", UIMissing: false, Phase: "postback_dlq"},
	{Area: "Team invite / assign", API: "POST /api/v1/team/members", APIRequired: true, UI: "/team", UIMissing: false, Phase: "team_workspace"},
	{Area: "Publisher dashboard", API: "GET /api/v1/publisher/dashboard", APIRequired: true, UI: "/publisher", UIMissing: false, Phase: "publisher_portal"},
	{Area: "Self-serve portal", API: "GET /api/v1/selfserve/templates", APIRequired: true, UI: "/selfserve", UIMissing: false, Phase: "selfserve_portal"},
	{Area: "Unified DLQ inbox", API: "GET /api/v1/ops/dlq/inbox", APIRequired: true, UI: "/ops/dlq", UIMissing: false, Phase: "ops_console"},
	{Area: "Consent proof browser", API: "GET /api/v1/ops/consent/proofs", APIRequired: true, UI: "/ops/consent", UIMissing: false, Phase: "ops_console"},
	{Area: "Support feedback form", API: "POST /api/v1/support/feedback", APIRequired: true, UI: "/support/feedback", UIMissing: false, Phase: "ops_console"},
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

func TestCPA_DocumentedProductGaps_open(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	catalog := catalogRouteSet()

	for _, gap := range cpaProductGaps {
		if gap.APIRequired && gap.API != "" {
			_, ok := catalog[gap.API]
			require.True(t, ok, "gap %q: API %s must exist in catalog (phase %s)", gap.Area, gap.API, gap.Phase)
		}
		_ = root
	}
}

func TestCPA_PatchCampaignRequest_parity(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	dtoGo := readRepoFile(t, root, "internal/campaign/dto.go")

	required := []string{"status", "budget_limit", "budget_limit_micro", "start_at", "end_at", "daypart_hours"}
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

func TestCPA_SelfServeShell_forbidsOperatorNav(t *testing.T) {
	t.Skip("integration: admin UI rebuild - web/ removed")
}
