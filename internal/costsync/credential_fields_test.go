package costsync

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateExtraConfig_microsoftAdsRequired(t *testing.T) {
	err := ValidateExtraConfig("microsoft_ads", map[string]string{"customer_id": "123"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "developer_token")

	err = ValidateExtraConfig("microsoft_ads", map[string]string{
		"customer_id":     "123",
		"developer_token": "dev-token",
	})
	require.NoError(t, err)
}

func TestMergeExtraConfig_preservesSecretWhenIncomingEmpty(t *testing.T) {
	schema, ok := CredentialSchemaForNetwork("microsoft_ads")
	require.True(t, ok)
	merged := MergeExtraConfig(
		map[string]string{"customer_id": "1", "developer_token": "secret"},
		map[string]string{"customer_id": "2", "developer_token": ""},
		schema,
	)
	require.Equal(t, "2", merged["customer_id"])
	require.Equal(t, "secret", merged["developer_token"])
}

func TestMaskExtraConfigForResponse_hidesSecrets(t *testing.T) {
	visible, set := MaskExtraConfigForResponse("microsoft_ads", map[string]string{
		"customer_id":     "123",
		"developer_token": "secret",
	})
	require.Equal(t, map[string]string{"customer_id": "123"}, visible)
	require.Equal(t, map[string]bool{"developer_token": true}, set)
}
