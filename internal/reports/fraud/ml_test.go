package fraud

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateMLReportRoute_knownKeys(t *testing.T) {
	for _, key := range mlReportKeys() {
		require.NoError(t, validateMLReportRoute(key))
	}
}

func TestValidateMLReportRoute_rejectsUnknown(t *testing.T) {
	err := validateMLReportRoute("ml/unknown")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown ml report")
}

func TestMLReportKeys_coverRegisterRoutes(t *testing.T) {
	keys := mlReportKeys()
	assert.Len(t, keys, 3)
	assert.Contains(t, keys, "ml/score-distribution")
	assert.Contains(t, keys, "ml/shadow-delta")
	assert.Contains(t, keys, "ml/feature-spikes")
}
