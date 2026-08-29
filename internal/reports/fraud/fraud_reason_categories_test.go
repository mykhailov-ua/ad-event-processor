package fraud

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFraudReasonToCategory_holdoutKnownSignals(t *testing.T) {
	t.Parallel()
	category, label := FraudReasonToCategory("tls_ja4_mismatch")
	assert.Equal(t, fraudCategoryInvalidDevice, category)
	assert.Equal(t, "Invalid device signals", label)

	category, label = FraudReasonToCategory("residential_proxy")
	assert.Equal(t, fraudCategoryProxyDatacenter, category)
	assert.Equal(t, "Proxy or datacenter traffic", label)
}

func TestFraudReasonToCategory_holdoutUnknown(t *testing.T) {
	t.Parallel()
	category, label := FraudReasonToCategory("not_a_real_signal_xyz")
	assert.Equal(t, fraudCategoryOther, category)
	assert.Equal(t, "Other", label)
}

func TestFraudReasonCategoriesFromField_dedupes(t *testing.T) {
	t.Parallel()
	got := FraudReasonCategoriesFromField("tls_ja4_mismatch,h2_settings_mismatch,datacenter_ip")
	assert.Equal(t, []string{fraudCategoryInvalidDevice, fraudCategoryProxyDatacenter}, got)
}

func TestIsWireSignalReason_holdout(t *testing.T) {
	t.Parallel()
	assert.True(t, isWireSignalReason("tls_ja4_mismatch,low_ttc"))
	assert.False(t, isWireSignalReason("low_ttc,missing_imp_ts"))
}
