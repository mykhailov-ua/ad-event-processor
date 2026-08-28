package controlplane

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/flow"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestHostedLanderZip_uploadServeRoundTrip_integration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: hosted lander zip upload and serve")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	storeRoot := t.TempDir()
	cfg := &config.Config{
		LanderStoreRoot:     storeRoot,
		LanderPublicBaseURL: "https://trk.example",
	}
	svc := newBareService(t, pool, nil, cfg)

	ctx := context.Background()
	landerID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO landers (id, name, url) VALUES ($1, 'Hosted LP', '')`, landerID)
	require.NoError(t, err)

	body := "<html><body>hosted-lp-e2e</body></html>"
	zipBytes, zipSize := hostedLanderZipBytes(t, map[string]string{"index.html": body})

	_, err = svc.UploadHostedLanderZip(ctx, landerID, bytes.NewReader(zipBytes), zipSize)
	require.NoError(t, err)

	fh := &flow.HTTPHandlers{Service: svc}
	mux := http.NewServeMux()
	fh.RegisterHostedLanderRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/lp/"+landerID.String()+"/index.html", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, body, rec.Body.String())

	sum := sha256.Sum256([]byte(body))
	require.Equal(t, hex.EncodeToString(sum[:]), sha256Hex(rec.Body.Bytes()))
}

func hostedLanderZipBytes(t *testing.T, files map[string]string) ([]byte, int64) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	b := buf.Bytes()
	return b, int64(len(b))
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
