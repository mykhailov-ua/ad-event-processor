package campaign

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ad-event-processor/internal/migrationsource"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type migrationPullCampaignStub struct {
	patchRevisionCampaignStub
	previewFn     func(context.Context, PullMigrationPreviewSpec) (migrationsource.PreviewResult, error)
	importFn      func(context.Context, PullMigrationImportSpec) (ImportMigrationResult, error)
	startImportFn func(context.Context, PullMigrationImportSpec) error
}

func (s migrationPullCampaignStub) PreviewMigrationPull(ctx context.Context, spec PullMigrationPreviewSpec) (migrationsource.PreviewResult, error) {
	if s.previewFn != nil {
		return s.previewFn(ctx, spec)
	}
	return migrationsource.PreviewResult{}, nil
}

func (s migrationPullCampaignStub) ImportMigrationPull(ctx context.Context, spec PullMigrationImportSpec) (ImportMigrationResult, error) {
	if s.importFn != nil {
		return s.importFn(ctx, spec)
	}
	return ImportMigrationResult{}, nil
}

func (s migrationPullCampaignStub) StartMigrationPullImport(ctx context.Context, spec PullMigrationImportSpec) error {
	if s.startImportFn != nil {
		return s.startImportFn(ctx, spec)
	}
	go func() {
		_, _ = s.ImportMigrationPull(context.Background(), spec)
	}()
	return nil
}

func migrationPullTestMux(stub *migrationPullCampaignStub) *http.ServeMux {
	h := &CampaignsHTTPHandlers{Campaigns: stub}
	mux := http.NewServeMux()
	h.registerMigrationRoutes(mux, func(next http.HandlerFunc) http.HandlerFunc { return next }, func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next })
	return mux
}

func TestMigrationPullHandlers_preview_holdout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"id":1,"name":"Pulled","alias":"p","domain":"https://trk.example"}]`))
	}))
	defer server.Close()

	mux := migrationPullTestMux(&migrationPullCampaignStub{
		previewFn: func(ctx context.Context, spec PullMigrationPreviewSpec) (migrationsource.PreviewResult, error) {
			payload, err := migrationsource.FetchRemotePayload(ctx, migrationsource.PullSpec{
				SourceKind: spec.SourceKind,
				BaseURL:    spec.BaseURL,
				APIToken:   spec.APIToken,
			})
			require.NoError(t, err)
			return migrationsource.Preview(spec.SourceKind, payload, nil)
		},
	})

	body, err := json.Marshal(map[string]string{
		"source_kind": string(migrationsource.SourceKindKeitaroAdminAPI),
		"base_url":    server.URL,
		"api_token":   "secret-key",
	})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/migrate/pull/preview", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.NotContains(t, strings.ToLower(rec.Body.String()), "secret-key")
	assert.NotContains(t, strings.ToLower(rec.Body.String()), "api_token")
}

func TestMigrationPullHandlers_importPullFailureNoPartial_holdout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	var importCalls int
	mux := migrationPullTestMux(&migrationPullCampaignStub{
		importFn: func(_ context.Context, spec PullMigrationImportSpec) (ImportMigrationResult, error) {
			importCalls++
			_, err := migrationsource.FetchRemotePayload(context.Background(), migrationsource.PullSpec{
				SourceKind: spec.SourceKind,
				BaseURL:    spec.BaseURL,
				APIToken:   spec.APIToken,
			})
			if err != nil {
				return ImportMigrationResult{}, errValidation(err.Error())
			}
			return ImportMigrationResult{}, nil
		},
	})

	body, err := json.Marshal(map[string]string{
		"source_kind": string(migrationsource.SourceKindKeitaroAdminAPI),
		"base_url":    server.URL,
		"api_token":   "secret-key",
		"customer_id": uuid.New().String(),
	})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/migrate/pull/import", bytes.NewReader(body))
	req.Header.Set("Idempotency-Key", "pull-fail-idem")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusAccepted, rec.Code)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && importCalls == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	assert.Equal(t, 1, importCalls)
	assert.NotContains(t, strings.ToLower(rec.Body.String()), "secret-key")
}

func TestMigrationPullHandlers_importAsyncAccepted_holdout(t *testing.T) {
	gate := make(chan struct{})
	importStarted := make(chan struct{}, 1)
	mux := migrationPullTestMux(&migrationPullCampaignStub{
		startImportFn: func(_ context.Context, _ PullMigrationImportSpec) error {
			importStarted <- struct{}{}
			<-gate
			return nil
		},
	})

	body, err := json.Marshal(map[string]string{
		"source_kind": string(migrationsource.SourceKindKeitaroAdminAPI),
		"base_url":    "https://trk.example",
		"api_token":   "secret-key",
		"customer_id": uuid.New().String(),
	})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/migrate/pull/import", bytes.NewReader(body))
	req.Header.Set("Idempotency-Key", "pull-async-idem")
	start := time.Now()
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusAccepted, rec.Code)
	require.Less(t, time.Since(start), 200*time.Millisecond)
	select {
	case <-importStarted:
	case <-time.After(time.Second):
		t.Fatal("background import did not start")
	}
	close(gate)
}

func TestMigrationPullHandlers_unconfiguredService(t *testing.T) {
	h := &CampaignsHTTPHandlers{}
	mux := http.NewServeMux()
	h.registerMigrationPullRoutes(mux, func(next http.HandlerFunc) http.HandlerFunc { return next }, func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next })

	body, err := json.Marshal(map[string]string{
		"source_kind": string(migrationsource.SourceKindKeitaroAdminAPI),
		"base_url":    "https://trk.example",
		"api_token":   "x",
	})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/migrate/pull/preview", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}
