package landerhost_test

import (
	"bytes"
	"testing"
	"time"

	"ad-event-processor/pkg/landerhost"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreviewToken_roundTrip(t *testing.T) {
	secret := []byte("preview-secret")
	landerID := uuid.New()
	now := time.Unix(1_700_000_000, 0)
	token, err := landerhost.MintPreviewToken(secret, landerID, 3, now)
	require.NoError(t, err)
	ver, ok := landerhost.VerifyPreviewToken(secret, token, landerID, now.Add(30*time.Minute))
	assert.True(t, ok)
	assert.Equal(t, 3, ver)
	_, ok = landerhost.VerifyPreviewToken(secret, token, landerID, now.Add(2*time.Hour))
	assert.False(t, ok)
}

func TestEditorCloneAndWrite(t *testing.T) {
	landerID := uuid.New()
	st, err := landerhost.NewStore(t.TempDir())
	require.NoError(t, err)
	buf, size := zipBytes(t, map[string]string{"index.html": "<html>v1</html>"})
	_, err = st.ExtractZip(landerID, 1, bytes.NewReader(buf), size)
	require.NoError(t, err)
	require.NoError(t, st.CloneVersion(landerID, 1, 2))
	require.NoError(t, st.WriteVersionTextFile(landerID, 2, "index.html", []byte("<html>v2</html>")))
	files, err := st.ListVersionFiles(landerID, 2)
	require.NoError(t, err)
	require.Len(t, files, 1)
	got, err := st.ReadVersionFile(landerID, 2, "index.html")
	require.NoError(t, err)
	assert.Equal(t, "<html>v2</html>", string(got))
}
