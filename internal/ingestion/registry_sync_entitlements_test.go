package ingestion

import (
	"context"
	"testing"

	"ad-event-processor/internal/licensing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncEntitlements_noPoolNoOp(t *testing.T) {
	registry := NewRegistry(nil)
	assert.NoError(t, registry.SyncEntitlements(context.Background()))
	state, ent := registry.GetLicenseState()
	assert.Equal(t, licensing.StateExpired, state)
	assert.Equal(t, uint64(0), ent.Limits.MaxRPS)
}

func TestParseLicenseEntitlementsJSON_holdout(t *testing.T) {
	t.Run("holdoutPositive_validJSON", func(t *testing.T) {
		raw := []byte(`{"limits":{"max_rps":5000},"features":{"openrtb_engine":true}}`)
		ent, err := parseLicenseEntitlementsJSON(raw)
		require.NoError(t, err)
		assert.Equal(t, uint64(5000), ent.Limits.MaxRPS)
		assert.True(t, ent.Features.OpenRTBEnabled())
	})

	t.Run("holdoutNegative_corruptJSON", func(t *testing.T) {
		_, err := parseLicenseEntitlementsJSON([]byte(`{"limits":`))
		require.Error(t, err)
	})

	t.Run("emptyBytes", func(t *testing.T) {
		ent, err := parseLicenseEntitlementsJSON(nil)
		require.NoError(t, err)
		assert.Equal(t, uint64(0), ent.Limits.MaxRPS)
	})
}
