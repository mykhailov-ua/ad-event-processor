package controlplane

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var licenseProductSurfaces = []struct {
	Area string
	API  string
	UI   string
}{
	{Area: "License status", API: "GET /api/v1/license/status", UI: "/settings/license"},
	{Area: "License apply", API: "POST /api/v1/license/apply", UI: "/settings/license"},
}

func skipIfAdminWebRemoved(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(repoRoot(t), "web", "src")); os.IsNotExist(err) {
		t.Skip("integration: admin UI rebuild - web/ removed")
	}
}

func TestLicense_RoutesInCatalogAndUI(t *testing.T) {
	t.Parallel()
	skipIfAdminWebRemoved(t)
	root := repoRoot(t)
	routes := catalogRouteSet()
	appRoutes := readRepoFile(t, root, "web/src/app_routes.tsx")

	for _, surface := range licenseProductSurfaces {
		require.Contains(t, routes, surface.API, "license API %s missing from routeCatalog", surface.API)
		require.Contains(t, appRoutes, surface.UI, "license UI route %s missing from app_routes", surface.UI)
	}
}

func TestLicense_StatusDTOFields_documentedInTypes(t *testing.T) {
	t.Parallel()
	skipIfAdminWebRemoved(t)
	root := repoRoot(t)
	licenseTS := readRepoFile(t, root, "web/src/types/license.ts")
	for _, field := range []string{"host_fingerprint", "hwid_v2", "hwid_match", "days_to_expiry"} {
		require.Contains(t, licenseTS, field, "LicenseStatusDTO missing %s", field)
	}
}

func TestLicense_VERIFYCatalog_coversBaselineProperties(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	verifyMD := readRepoFile(t, root, ".cursor/rules/LICENSING.mdc")
	for _, prop := range []string{"P-C2-01", "P-C3-03", "P-C4-03", "P-HWID-01"} {
		require.Contains(t, verifyMD, prop, "licensing.mdc missing %s", prop)
	}
}

func TestLicense_PilotDocReferencesStatusAndHostIdentity(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	pilot := readRepoFile(t, root, ".cursor/rules/LICENSING.mdc")
	require.Contains(t, pilot, "/api/v1/license/status")
	require.True(t, strings.Contains(pilot, "hwid_v2") || strings.Contains(pilot, "HWID v2"))
}

func TestLicense_AdminSmokeScriptsExist(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	for _, rel := range []string{
		"scripts/test/license_admin_smoke.sh",
		"scripts/test/release_hardening_smoke.sh",
		"scripts/test/sealed_bpf_xdp_smoke.sh",
	} {
		path := filepath.Join(root, rel)
		info, err := os.Stat(path)
		require.NoError(t, err, rel)
		require.False(t, info.IsDir(), rel)
	}
}
