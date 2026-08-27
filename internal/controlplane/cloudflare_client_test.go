package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCloudflareClient_ListZones(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/zones", r.URL.Path)
		require.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"result": []map[string]string{
				{"id": "zone-1", "name": "example.com"},
			},
		})
	}))
	defer srv.Close()

	client := NewCloudflareClient("test-token", srv.URL)
	zones, err := client.ListZones(context.Background())
	require.NoError(t, err)
	require.Len(t, zones, 1)
	require.Equal(t, "zone-1", zones[0].ID)
	require.Equal(t, "example.com", zones[0].Name)
}

func TestCloudflareClient_CreateDNSRecord(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/zones/zone-abc/dns_records", r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "A", body["type"])
		require.Equal(t, "track.example.com", body["name"])
		require.Equal(t, "203.0.113.10", body["content"])
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"result":  map[string]string{"id": "rec-456"},
		})
	}))
	defer srv.Close()

	client := NewCloudflareClient("token", srv.URL)
	id, err := client.CreateDNSRecord(context.Background(), "zone-abc", "track.example.com", "A", "203.0.113.10", true)
	require.NoError(t, err)
	require.Equal(t, "rec-456", id)
}

func TestCloudflareClient_ZoneSSLStatus(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/zones/zone-abc/settings/ssl", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"result":  map[string]string{"value": "full"},
		})
	}))
	defer srv.Close()

	client := NewCloudflareClient("token", srv.URL)
	status, err := client.ZoneSSLStatus(context.Background(), "zone-abc")
	require.NoError(t, err)
	require.Equal(t, "full", status)
}

func TestCloudflareRecordTypeForTarget(t *testing.T) {
	t.Parallel()
	require.Equal(t, "A", cloudflareRecordTypeForTarget("203.0.113.1"))
	require.Equal(t, "CNAME", cloudflareRecordTypeForTarget("ingress.example.com"))
	require.Equal(t, "AAAA", cloudflareRecordTypeForTarget("2001:db8::1"))
}

func TestCloudflareClient_RejectsOversizedBody(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		payload := strings.Repeat("x", cloudflareMaxJSONBytes+1)
		_, _ = w.Write([]byte(`{"success":true,"result":[` + payload))
	}))
	defer srv.Close()

	client := NewCloudflareClient("token", srv.URL)
	_, err := client.ListZones(context.Background())
	require.Error(t, err)
}

func TestNewCloudflareClient_nilWithoutToken(t *testing.T) {
	t.Parallel()
	require.Nil(t, NewCloudflareClient("", ""))
}
