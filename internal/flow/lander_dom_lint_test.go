package flow

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/landerhost"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type landerDomLintHost struct {
	store *landerhost.Store
}

func (h *landerDomLintHost) HostedLanderPool() *pgxpool.Pool         { return nil }
func (h *landerDomLintHost) HostedLanderStore() *landerhost.Store    { return h.store }
func (h *landerDomLintHost) LanderPublicBase(context.Context) string { return "https://lp.test" }
func (h *landerDomLintHost) LanderPreviewSecret() []byte             { return nil }
func (h *landerDomLintHost) LanderManagementURL() string             { return "" }
func (h *landerDomLintHost) LanderMaxZipBytes() int64                { return landerhost.DefaultMaxZipBytes }
func (h *landerDomLintHost) LanderCSPEnabled() bool                  { return false }

func TestLanderDomLint_holdoutRejectsDeceptiveFixture(t *testing.T) {
	before := testutil.ToFloat64(metrics.LanderDomLintRejectTotal.WithLabelValues(landerhost.DomLintRuleMetaRefresh))
	root := t.TempDir()
	st, err := landerhost.NewStore(root)
	require.NoError(t, err)
	landerID := uuid.New()
	version := 1
	dir := st.VersionDir(landerID, version)
	require.NoError(t, os.MkdirAll(dir, 0o750))
	deceptive := `<!doctype html><html><head><meta http-equiv="refresh" content="0;url=https://evil.test"></head><body>x</body></html>`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte(deceptive), 0o640))

	host := &landerDomLintHost{store: st}
	err = lintHostedVersion(context.Background(), host, landerID, version)
	require.Error(t, err)
	var lintErr *landerhost.DomLintError
	require.ErrorAs(t, err, &lintErr)
	assert.Equal(t, landerhost.DomLintRuleMetaRefresh, lintErr.Result.Violations[0].Rule)
	assert.GreaterOrEqual(t, testutil.ToFloat64(metrics.LanderDomLintRejectTotal.WithLabelValues(landerhost.DomLintRuleMetaRefresh)), before+1)
}

func TestLanderDomLint_holdoutAllowsCleanFixture(t *testing.T) {
	root := t.TempDir()
	st, err := landerhost.NewStore(root)
	require.NoError(t, err)
	landerID := uuid.New()
	version := 1
	dir := st.VersionDir(landerID, version)
	require.NoError(t, os.MkdirAll(dir, 0o750))
	clean := `<!doctype html><html><head><title>ok</title></head><body><main>offer</main></body></html>`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte(clean), 0o640))

	host := &landerDomLintHost{store: st}
	err = lintHostedVersion(context.Background(), host, landerID, version)
	require.NoError(t, err)
}
