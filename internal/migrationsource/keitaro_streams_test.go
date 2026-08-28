package migrationsource

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreviewKeitaro_streamsMultiPath_holdout(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "keitaro_facebook_streams.json"))
	require.NoError(t, err)
	maps, err := LoadMaps(migrationMapsDir(t))
	require.NoError(t, err)
	result, err := Preview(SourceKindKeitaroJSON, raw, maps)
	require.NoError(t, err)
	require.Len(t, result.MappedCampaigns, 1)
	flow := result.MappedCampaigns[0].Flow
	require.NotNil(t, flow)
	require.Len(t, flow.Paths, 2)
	assert.Equal(t, int32(60), flow.Paths[0].Weight)
	assert.Equal(t, int32(40), flow.Paths[1].Weight)
	assert.Equal(t, "rotation-main", flow.Name)

	var unmapped bool
	for _, w := range result.Warnings {
		if w.Slug == "stream_node_unmapped" {
			unmapped = true
			assert.Contains(t, w.Message, "filter:country")
		}
	}
	assert.True(t, unmapped, "expected stream_node_unmapped warning")
}

func TestPreviewKeitaro_streamsUnmappedNode_holdout(t *testing.T) {
	payload := []byte(`{"campaigns":[{"id":1,"name":"Warn","tracking_url":"https://trk.example/click","streams":[{"paths":[{"weight":100,"lander":{"name":"L","url":"https://l.example"},"offer":{"name":"O","url":"https://o.example"}}],"unmapped_nodes":["filter:device"]}]}]}`)
	maps, err := LoadMaps(migrationMapsDir(t))
	require.NoError(t, err)
	result, err := Preview(SourceKindKeitaroJSON, payload, maps)
	require.NoError(t, err)
	var found bool
	for _, w := range result.Warnings {
		if w.Slug == "stream_node_unmapped" && w.Message == "stream node not imported: filter:device" {
			found = true
		}
	}
	assert.True(t, found)
}

func TestFetchRemotePayload_keitaro_holdout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/admin_api/v1/campaigns", r.URL.Path)
		assert.Equal(t, "secret-key", r.Header.Get("Api-Key"))
		_, _ = w.Write([]byte(`[{"id":1,"name":"Pulled","alias":"pull","domain":"https://trk.example"}]`))
	}))
	defer server.Close()

	body, err := FetchRemotePayload(context.Background(), PullSpec{
		SourceKind: SourceKindKeitaroAdminAPI,
		BaseURL:    server.URL,
		APIToken:   "secret-key",
	})
	require.NoError(t, err)
	bundle, err := ParseKeitaroAdminAPI(body)
	require.NoError(t, err)
	require.Len(t, bundle.Campaigns, 1)
	assert.Equal(t, "Pulled", bundle.Campaigns[0].Name)
}

func TestFetchRemotePayload_failureNoBody_holdout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := FetchRemotePayload(context.Background(), PullSpec{
		SourceKind: SourceKindKeitaroAdminAPI,
		BaseURL:    server.URL,
		APIToken:   "secret-key",
	})
	require.Error(t, err)
}
