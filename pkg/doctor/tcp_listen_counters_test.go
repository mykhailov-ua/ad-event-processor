package doctor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleNetstat = `TcpExt: SyncookiesSent SyncookiesRecv SyncookiesFailed EmbryonicRsts PruneCalled RcvPruned OfoPruned OutOfWindowIcmps LockDroppedIcmps ArpFilter TW TWRecycled TWKilled PAWSActive PAWSEstab BeyondWindow TSEcrRejected PAWSOldAck PAWSTimewait DelayedACKs DelayedACKLocked DelayedACKLost ListenOverflows ListenDrops TCPHPHits
TcpExt: 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 42 7 0
`

func TestParseTcpListenCounters(t *testing.T) {
	got, err := parseTcpListenCountersFromBytes([]byte(sampleNetstat))
	require.NoError(t, err)
	assert.Equal(t, uint64(42), got.ListenOverflows)
	assert.Equal(t, uint64(7), got.ListenDrops)
}

func TestTcpListenCounters_Delta(t *testing.T) {
	before := TcpListenCounters{ListenOverflows: 100, ListenDrops: 5}
	after := TcpListenCounters{ListenOverflows: 103, ListenDrops: 5}
	delta := before.Delta(after)
	assert.Equal(t, uint64(3), delta.ListenOverflows)
	assert.Equal(t, uint64(0), delta.ListenDrops)
}

func TestParseTcpListenCounters_missingBlock(t *testing.T) {
	_, err := parseTcpListenCountersFromBytes([]byte("IpExt: Foo\n"))
	require.Error(t, err)
}

func TestReadTcpListenCounters_live(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	got, err := ReadTcpListenCounters()
	require.NoError(t, err)
	t.Logf("ListenOverflows=%d ListenDrops=%d", got.ListenOverflows, got.ListenDrops)
}
