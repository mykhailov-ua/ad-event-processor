package ingestion

import (
	"context"
	"testing"

	"espx/internal/licensing"

	"github.com/stretchr/testify/assert"
)

func TestSyncEntitlements_noPoolNoOp(t *testing.T) {
	registry := NewRegistry(nil)
	assert.NoError(t, registry.SyncEntitlements(context.Background()))
	state, ent := registry.GetLicenseState()
	assert.Equal(t, licensing.StateExpired, state)
	assert.Equal(t, uint64(0), ent.Limits.MaxRPS)
}
