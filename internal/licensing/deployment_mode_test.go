package licensing

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeDeploymentMode(t *testing.T) {
	t.Parallel()
	require.Equal(t, DeploymentModeOnPrem, NormalizeDeploymentMode(""))
	require.Equal(t, DeploymentModeOnPrem, NormalizeDeploymentMode("on_prem"))
	require.Equal(t, DeploymentModeManagedSaas, NormalizeDeploymentMode("managed_saas"))
	require.Equal(t, DeploymentModeOnPrem, NormalizeDeploymentMode("unknown"))
}

func TestValidateDeploymentMode(t *testing.T) {
	t.Parallel()
	require.NoError(t, ValidateDeploymentMode(DeploymentModeOnPrem))
	require.NoError(t, ValidateDeploymentMode(DeploymentModeManagedSaas))
	require.Error(t, ValidateDeploymentMode("cloud"))
}

func TestLoadSKUFile_managedSaasDeploymentMode(t *testing.T) {
	doc, err := LoadSKUFile(filepath.Join("..", "..", "deploy", "vendor", "sku.yaml"))
	require.NoError(t, err)
	sku, err := doc.GetSKU(SKUCodeManagedSaas)
	require.NoError(t, err)
	require.Equal(t, DeploymentModeManagedSaas, sku.DeploymentMode)

	claims := sku.BuildClaims(IssueLicenseInput{
		CustomerName: "SaaS Buyer",
		DeploymentID: "dep-saas",
		LicenseID:    "lic-saas",
	})
	require.Equal(t, DeploymentModeManagedSaas, claims.DeploymentMode)
}

func TestSKUBuildClaims_defaultsOnPrem(t *testing.T) {
	doc, err := LoadSKUFile(filepath.Join("..", "..", "deploy", "vendor", "sku.yaml"))
	require.NoError(t, err)
	sku, err := doc.GetSKU(SKUCodeStarter)
	require.NoError(t, err)
	claims := sku.BuildClaims(IssueLicenseInput{
		CustomerName: "Buyer",
		DeploymentID: "dep-1",
		LicenseID:    "lic-1",
	})
	require.Equal(t, DeploymentModeOnPrem, claims.DeploymentMode)
}
