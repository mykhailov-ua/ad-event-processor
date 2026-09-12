package campaign

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateBulkPatchFields_rejectsEmptyPatch_holdout(t *testing.T) {
	t.Parallel()
	err := ValidateBulkPatchFields(BulkPatchFields{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one field")
}

func TestValidateBulkPatchFields_acceptsTargetURL(t *testing.T) {
	t.Parallel()
	url := "https://example.com/offer"
	err := ValidateBulkPatchFields(BulkPatchFields{TargetURL: &url})
	require.NoError(t, err)
}

func TestValidateBulkPatchFields_rejectsInvalidStatus_holdout(t *testing.T) {
	t.Parallel()
	status := "bogus"
	err := ValidateBulkPatchFields(BulkPatchFields{Status: &status})
	require.Error(t, err)
}

func TestBulkPatchFieldsToPatchRequest_mapsCountries(t *testing.T) {
	t.Parallel()
	fields := BulkPatchFields{TargetCountries: []string{"US", "DE"}}
	req := BulkPatchFieldsToPatchRequest(fields)
	assert.Equal(t, []string{"US", "DE"}, req.TargetCountries)
}
