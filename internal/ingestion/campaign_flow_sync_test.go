package ingestion

import (
	"encoding/json"
	"testing"

	"ad-event-processor/pkg/landerhost"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildFlowSnapshot_hostedLanderURL(t *testing.T) {
	t.Parallel()
	landerID := uuid.MustParse("00000000-0000-4000-8000-000000000011")
	offerID := uuid.MustParse("00000000-0000-4000-8000-000000000022")
	hostedURL := landerhost.PublicURL("https://trk.example.com", landerID)
	paths := []byte(`[{"weight":100,"landers":[{"lander_id":"00000000-0000-4000-8000-000000000011","weight":100}],"offers":[{"offer_id":"00000000-0000-4000-8000-000000000022","weight":100}]}]`)
	landerURLs := map[uuid.UUID][]byte{
		landerID: []byte(hostedURL),
	}
	offerURLs := map[uuid.UUID][]byte{
		offerID: []byte("https://offer.example/"),
	}
	snap, ok := buildFlowSnapshot(paths, landerURLs, offerURLs)
	require.True(t, ok)
	require.Len(t, snap.Paths, 1)
	require.Len(t, snap.Paths[0].Landers, 1)
	assert.Equal(t, hostedURL, string(snap.Paths[0].Landers[0].URL))
}

func TestBuildFlowSnapshot_skipsLanderWithoutURL(t *testing.T) {
	t.Parallel()
	landerID := uuid.New()
	paths, err := json.Marshal([]map[string]any{
		{
			"weight": 100,
			"landers": []map[string]any{
				{"lander_id": landerID.String(), "weight": 100},
			},
			"offers": []map[string]any{},
		},
	})
	require.NoError(t, err)
	_, ok := buildFlowSnapshot(paths, map[uuid.UUID][]byte{}, map[uuid.UUID][]byte{})
	assert.False(t, ok)
}
