package controlplane

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSavedViewInput_holdout(t *testing.T) {
	t.Parallel()

	spec := json.RawMessage(`{"from":"2026-08-01T00:00:00Z","to":"2026-08-08T00:00:00Z","compare":"previous"}`)
	require.NoError(t, validateSavedViewInput("Daily placements", "placements", spec))

	err := validateSavedViewInput("", "placements", spec)
	require.Error(t, err)

	err = validateSavedViewInput("Bad key", "not-a-report", spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported report_key")

	err = validateSavedViewSpec(json.RawMessage(`{"secret_token":"x"}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported spec key")
}

func TestValidateSavedViewSpec_range(t *testing.T) {
	t.Parallel()

	spec := json.RawMessage(`{"from":"2026-08-01T00:00:00Z","to":"2026-08-08T00:00:00Z"}`)
	require.NoError(t, validateSavedViewSpec(spec))

	badRange := json.RawMessage(`{"from":"2026-08-10T00:00:00Z","to":"2026-08-01T00:00:00Z"}`)
	err := validateSavedViewSpec(badRange)
	require.Error(t, err)
}
