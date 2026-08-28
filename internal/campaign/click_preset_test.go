package campaign

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateClickQueryParams_holdout(t *testing.T) {
	t.Parallel()
	require.NoError(t, validateClickQueryParams(map[string]string{"sub2": "{{campaign.id}}"}))
	err := validateClickQueryParams(map[string]string{"bad_key": "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid key")
	long := strings.Repeat("a", maxClickQueryParamValueLen+1)
	err = validateClickQueryParams(map[string]string{"sub1": long})
	require.Error(t, err)
}

func TestNormalizeClickQueryParams_skipsEmptyValues(t *testing.T) {
	t.Parallel()
	out := normalizeClickQueryParams(map[string]string{"sub1": "x", "sub2": "", "sub3": "  "})
	assert.Equal(t, map[string]string{"sub1": "x"}, out)
}
