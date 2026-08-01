package telemetry

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorker_disabledWithoutOptIn(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
	}))
	defer srv.Close()

	w := NewWorker(Config{
		OptIn: false,
		URL:   srv.URL,
		Metadata: func(context.Context) (Metadata, error) {
			return Metadata{DeploymentID: "dep", BinaryVersion: "1"}, nil
		},
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.Start(ctx)
	time.Sleep(50 * time.Millisecond)
	cancel()
	assert.Equal(t, 0, calls)
}

func TestWorker_uploadsPulse(t *testing.T) {
	ResetForTest()
	RecordAccepted()
	RecordTrack()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, ValidatePayloadJSON(body))
		var got PulsePayload
		require.NoError(t, json.Unmarshal(body, &got))
		assert.Equal(t, "dep-1", got.DeploymentID)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	w := NewWorker(Config{
		OptIn:      true,
		URL:        srv.URL,
		Interval:   time.Hour,
		WindowSec:  3600,
		HTTPClient: srv.Client(),
		Metadata: func(context.Context) (Metadata, error) {
			return Metadata{DeploymentID: "dep-1", BinaryVersion: "9.9.9", SKU: "ingest_pro"}, nil
		},
	})
	require.NoError(t, w.validateEndpoints())
	w.tick(context.Background())
}

func TestWorker_rejectsSameURLAsLicenseServer(t *testing.T) {
	w := NewWorker(Config{
		OptIn:            true,
		URL:              "https://license.bidshard.com/v1/pulse",
		LicenseServerURL: "https://license.bidshard.com",
	})
	require.Error(t, w.validateEndpoints())
}

func TestPulsePayload_matchesSchemaV1(t *testing.T) {
	root := filepath.Join("..", "..", "docs", "telemetry", "schema_v1.json")
	rawSchema, err := os.ReadFile(root)
	require.NoError(t, err)

	var schema struct {
		Required   []string       `json:"required"`
		Properties map[string]any `json:"properties"`
	}
	require.NoError(t, json.Unmarshal(rawSchema, &schema))

	payload, err := json.Marshal(PulsePayload{
		SchemaVersion:  1,
		DeploymentID:   "dep",
		BinaryVersion:  "1.0.0",
		WindowSec:      3600,
		AcceptedEvents: 1,
		RejectedEvents: 2,
		PeakRPS:        3,
	})
	require.NoError(t, err)
	require.NoError(t, ValidatePayloadJSON(payload))

	var obj map[string]any
	require.NoError(t, json.Unmarshal(payload, &obj))
	for _, key := range schema.Required {
		_, ok := obj[key]
		require.True(t, ok, "missing required field %q", key)
	}
	for key := range obj {
		_, ok := schema.Properties[key]
		require.True(t, ok, "unexpected field %q", key)
	}
}
