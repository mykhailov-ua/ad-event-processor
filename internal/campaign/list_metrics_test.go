package campaign

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCampaignListMetricsIDs_rejectsEmpty(t *testing.T) {
	t.Parallel()
	_, err := ParseCampaignListMetricsIDs("")
	require.Error(t, err)
}

func TestParseCampaignListMetricsIDs_parsesUnique(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	got, err := ParseCampaignListMetricsIDs(id.String() + "," + id.String())
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, id, got[0])
}
