package domains

import (
	"os"
	"path/filepath"
	"testing"

	"ad-event-processor/internal/config"

	"github.com/stretchr/testify/require"
)

func TestDeployWildcardIngressTLS_writesCertAndKey(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := &config.Config{}
	cfg.Management.IngressTLSDir = dir

	certPEM := []byte("-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----")
	keyPEM := []byte("-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----")

	err := deployWildcardIngressTLS(cfg, "trk.example.com", certPEM, keyPEM)
	require.NoError(t, err)

	certPath, keyPath, _ := wildcardIngressCertPaths(cfg, "trk.example.com")
	gotCert, err := os.ReadFile(certPath)
	require.NoError(t, err)
	require.Equal(t, certPEM, gotCert)

	gotKey, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	require.Equal(t, keyPEM, gotKey)

	info, err := os.Stat(keyPath)
	require.NoError(t, err)
	require.Equal(t, filepath.Base(keyPath), info.Name())
}
