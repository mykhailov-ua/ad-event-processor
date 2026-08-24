package e2e_test

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ad-event-processor/pkg/broker/protocol"
	"ad-event-processor/pkg/iogate"
	rserver "ad-event-processor/pkg/regionproxy/server"
	"ad-event-processor/pkg/regionproxy/wal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_RegionProxyIngress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping region-proxy e2e")
	}

	dataDir := t.TempDir()
	gate := iogate.NewDiskWriteGate(iogate.Config{AppendCapacity: 16, GroupCommitRecords: 1})
	srv, err := rserver.NewServer("127.0.0.1:0", dataDir, gate)
	require.NoError(t, err)
	require.NoError(t, srv.Start())
	defer srv.Stop()
	time.Sleep(150 * time.Millisecond)

	conn, err := net.DialTimeout("tcp", srv.Addr(), 2*time.Second)
	require.NoError(t, err)
	defer conn.Close()

	writeBuf := make([]byte, 4096)
	readBuf := make([]byte, 4096)
	lenBuf := make([]byte, 4)

	reg := protocol.EncodeRegisterTopicRequest(writeBuf, 1, rserver.DefaultIngressTopic)
	_, err = conn.Write(reg)
	require.NoError(t, err)
	_, _, regPayload, err := protocol.ReadFrame(conn, readBuf, lenBuf)
	require.NoError(t, err)
	regStatus, topicID, err := protocol.DecodeRegisterTopicResponse(regPayload)
	require.NoError(t, err)
	require.Equal(t, byte(0), regStatus)

	var batch []byte
	batch = protocol.AppendBatchMessage(batch, topicID, []byte("proxy-batch-a"))
	batch = protocol.AppendBatchMessage(batch, topicID, []byte("proxy-batch-b"))
	req := protocol.EncodeProduceBatchRequest(writeBuf, 2, batch)
	_, err = conn.Write(req)
	require.NoError(t, err)
	_, _, batchResp, err := protocol.ReadFrame(conn, readBuf, lenBuf)
	require.NoError(t, err)
	status, offset, committed, err := protocol.DecodeProduceBatchResponse(batchResp)
	require.NoError(t, err)
	assert.Equal(t, byte(0), status)
	assert.Equal(t, uint32(2), committed)
	assert.Equal(t, uint64(1), offset)

	segPath := filepath.Join(dataDir, "wal.segment")
	fi, err := os.Stat(segPath)
	require.NoError(t, err)
	assert.Greater(t, fi.Size(), int64(0))

	reopened, err := wal.Open(dataDir, gate)
	require.NoError(t, err)
	defer reopened.Close()
	assert.Equal(t, uint64(2), reopened.NextSeq())
	hdr, payload, err := reopened.ReadRecord(0)
	require.NoError(t, err)
	assert.True(t, hdr.Has(wal.WalFlagAppended))
	assert.Equal(t, []byte("proxy-batch-a"), payload)
}
