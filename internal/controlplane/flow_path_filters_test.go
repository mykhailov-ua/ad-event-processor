package controlplane

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateFlowPathFilters_holdout(t *testing.T) {
	t.Parallel()
	require.NoError(t, validateFlowPathFilters(0, &FlowPathFiltersDTO{
		Countries: []string{"us", "DE"},
		Devices:   []string{"mobile"},
		OS:        []string{"android"},
		Languages: []string{"en"},
	}))
	err := validateFlowPathFilters(0, &FlowPathFiltersDTO{
		Devices: []string{"bot"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid device")
}
