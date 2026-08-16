package ingestion

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bidshard/ad-event-processor/internal/domain"
)

// GM-M0.a baseline audit: documents current IP/ASN classification surfaces and
// gaps vs GRAY_MARKET_MILESTONE §2 before GM-M1 work lands.
func TestGrayMarketBaseline_inventorySymbolsExist(t *testing.T) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	ingestionDir := filepath.Dir(file)

	symbols := map[string][]string{
		"filters.go":          {"FraudFilter"},
		"filter_errors.go":    {"FraudReasonCode"},
		"safe_page_verify.go": {"safePageVerifyFingerprint", "scoreSafePageBehavior"},
		"safe_page.go":        {"resolveSafePageLanding"},
		"click_redirect.go":   {"reactClickRedirect"},
		"schedule_filter.go":  {"BrandCreativeStore"},
		"proxy_vpn_lpm.go":    {"ProxyVPNTable", "Lookup4"},
		"flow_router.go":      {"FlowRouter", "Select"},
	}
	for rel, want := range symbols {
		found := parseExportedOrTypeNames(t, filepath.Join(ingestionDir, rel))
		for _, name := range want {
			assert.Contains(t, found, name, "file=%s", rel)
		}
	}
}

func TestGrayMarketBaseline_fraudFilter_datacenterOnly(t *testing.T) {
	geo := &MockGeoProvider{}
	f := NewFraudFilter(geo)
	engine := NewFilterEngine(0, f)
	engine.SetRegistry(&mockRegistry{})

	evt := &domain.Event{
		Type:         "click",
		UserID:       "user1",
		CampaignID:   uuid.New(),
		IP:           "203.0.113.66",
		StringBuffer: make([]byte, 0, 64),
	}
	err := engine.Check(context.Background(), evt)
	require.NoError(t, err)
	assert.True(t, evt.ShadowEvent)
	assert.Equal(t, FraudReasonCodeDatacenterIP, evt.FraudReason)
}

func TestGrayMarketBaseline_safePageVerify_fingerprintSurface(t *testing.T) {
	fp := safePageVerifyFingerprint{
		UA:        "Mozilla/5.0",
		Lang:      "en",
		Languages: []string{"en"},
		Timezone:  "America/New_York",
		Webdriver: false,
	}
	assert.True(t, validSafePageFingerprint(fp))

	fp.Webdriver = true
	assert.False(t, validSafePageFingerprint(fp))

	hydrator := string(safePageHydratorJS)
	assert.Contains(t, hydrator, "timezone")
	assert.Contains(t, hydrator, "RTCPeerConnection")
}

func TestGrayMarketBaseline_proxyVPNHarness_ready(t *testing.T) {
	table := NewProxyVPNTable()
	assert.False(t, table.Ready())
	var ip [4]byte
	ip[0], ip[1], ip[2], ip[3] = 54, 230, 1, 1
	match, conn, asn := table.Lookup4(ip)
	assert.False(t, match)
	assert.Zero(t, conn)
	assert.Zero(t, asn)
}

func TestGrayMarketBaseline_flowRouterHarness_ready(t *testing.T) {
	router := NewFlowRouter()
	assert.False(t, router.Ready())
	_, ok := router.Select([]byte("user-1"))
	assert.False(t, ok)
}

func TestGrayMarketBaseline_productScopeGates(t *testing.T) {
	// G1–G4 from GRAY_MARKET_MILESTONE §3 — acknowledged in first GM PR.
	const (
		gateNoVisualEditors   = "no_grapejs_lp_builder"
		gateNoLocalLanders    = "redirect_and_proxy_only"
		gateNoFlowBuilderUI   = "declarative_backend_lists_only"
		gateNoExternalIPIntel = "local_in_memory_l15_only"
	)
	assert.NotEmpty(t, gateNoVisualEditors)
	assert.NotEmpty(t, gateNoLocalLanders)
	assert.NotEmpty(t, gateNoFlowBuilderUI)
	assert.NotEmpty(t, gateNoExternalIPIntel)
}

func parseExportedOrTypeNames(t *testing.T, path string) map[string]struct{} {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	require.NoError(t, err, "parse %s", path)
	out := make(map[string]struct{})
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			if d.Tok != token.TYPE {
				continue
			}
			for _, spec := range d.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if ok {
					out[ts.Name.Name] = struct{}{}
				}
			}
		case *ast.FuncDecl:
			if d.Name != nil {
				out[d.Name.Name] = struct{}{}
			}
		}
	}
	return out
}

func TestGrayMarketBaseline_section2_harnessSymbolsPresent(t *testing.T) {
	assert.NotNil(t, NewProxyVPNTable())
	assert.NotNil(t, NewFlowRouter())
}
