package landerhost_test

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"ad-event-processor/pkg/landerhost"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractZipSafe_holdoutPathTraversal(t *testing.T) {
	landerID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	root := t.TempDir()
	st, err := landerhost.NewStore(root)
	require.NoError(t, err)

	buf, size := zipBytes(t, map[string]string{
		"../evil.html": "<html></html>",
	})
	_, err = st.ExtractZip(landerID, 1, bytes.NewReader(buf), size)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "traverses")
}

func TestExtractZipPublish_roundTrip(t *testing.T) {
	landerID := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	st, err := landerhost.NewStore(t.TempDir())
	require.NoError(t, err)

	body := "<html><body>hosted</body></html>"
	buf, size := zipBytes(t, map[string]string{"index.html": body})
	count, err := st.ExtractZip(landerID, 1, bytes.NewReader(buf), size)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	require.NoError(t, st.PublishVersion(landerID, 1))

	rc, info, err := st.OpenLiveFile(landerID, "")
	require.NoError(t, err)
	defer rc.Close()
	assert.False(t, info.IsDir())
	got := make([]byte, len(body))
	_, err = rc.Read(got)
	require.NoError(t, err)
	assert.Equal(t, body, string(got))

	url := landerhost.PublicURL("https://trk.example.com", landerID)
	assert.Equal(t, "https://trk.example.com/lp/"+landerID.String()+"/", url)
}

func TestExtractZip_singleTopLevelFolder(t *testing.T) {
	landerID := uuid.New()
	st, err := landerhost.NewStore(t.TempDir())
	require.NoError(t, err)

	buf, size := zipBytes(t, map[string]string{"bundle/index.html": "<html>ok</html>"})
	_, err = st.ExtractZip(landerID, 1, bytes.NewReader(buf), size)
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(st.VersionDir(landerID, 1), "index.html"))
	require.NoError(t, err)
}

func zipBytes(t *testing.T, files map[string]string) ([]byte, int64) {
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
