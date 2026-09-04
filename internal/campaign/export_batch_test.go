package campaign

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCampaignExportIDs_rejectsEmptyAndTooMany(t *testing.T) {
	t.Parallel()
	_, err := ParseCampaignExportIDs("")
	require.Error(t, err)

	ids := make([]string, CampaignExportBatchMaxIDs+1)
	for i := range ids {
		ids[i] = uuid.New().String()
	}
	_, err = ParseCampaignExportIDs(joinCSV(ids))
	require.Error(t, err)
}

func TestParseCampaignExportIDs_dedupesAndParses(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	got, err := ParseCampaignExportIDs(id.String() + "," + id.String())
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, id, got[0])
}

func joinCSV(parts []string) string {
	out := parts[0]
	for _, part := range parts[1:] {
		out += "," + part
	}
	return out
}
