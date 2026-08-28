package regionproxy

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"ad-event-processor/pkg/broker/protocol"
	"ad-event-processor/pkg/iogate"
	"ad-event-processor/pkg/regionproxy/wal"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegionProxy_ProduceBatchIngress(t *testing.T) {
	srv, err := NewServer("127.0.0.1:0", t.TempDir(), iogate.NewDiskWriteGate(iogate.TestGateConfig()))
	require.NoError(t, err)
	require.NoError(t, srv.Start())
	defer srv.Stop()
	time.Sleep(100 * time.Millisecond)

	conn, err := net.DialTimeout("tcp", srv.Addr(), time.Second)
	require.NoError(t, err)
	defer conn.Close()

	writeBuf := make([]byte, 4096)
	readBuf := make([]byte, 4096)
	lenBuf := make([]byte, 4)

	reg := protocol.EncodeRegisterTopicRequest(writeBuf, 1, DefaultIngressTopic)
	_, err = conn.Write(reg)
	require.NoError(t, err)
	_, _, regPayload, err := protocol.ReadFrame(conn, readBuf, lenBuf)
	require.NoError(t, err)
	status, topicID, err := protocol.DecodeRegisterTopicResponse(regPayload)
	require.NoError(t, err)
	require.Equal(t, byte(0), status)

	var batch []byte
	batch = protocol.AppendBatchMessage(batch, topicID, []byte("evt-0"))
	batch = protocol.AppendBatchMessage(batch, topicID, []byte("evt-1"))
	req := protocol.EncodeProduceBatchRequest(writeBuf, 2, batch)
	_, err = conn.Write(req)
	require.NoError(t, err)
	_, _, batchResp, err := protocol.ReadFrame(conn, readBuf, lenBuf)
	require.NoError(t, err)
	produceStatus, offset, committed, err := protocol.DecodeProduceBatchResponse(batchResp)
	require.NoError(t, err)
	assert.Equal(t, byte(0), produceStatus)
	assert.Equal(t, uint32(2), committed)
	assert.Equal(t, uint64(1), offset)

	hdr, payload, err := srv.WAL().ReadRecord(0)
	require.NoError(t, err)
	assert.True(t, hdr.Has(wal.WalFlagAppended))
	assert.Equal(t, []byte("evt-0"), payload)
	hdr, payload, err = srv.WAL().ReadRecord(1)
	require.NoError(t, err)
	assert.True(t, hdr.Has(wal.WalFlagAppended))
	assert.Equal(t, []byte("evt-1"), payload)
}

func TestRegionProxy_BackpressureWhenDegraded(t *testing.T) {
	srv, err := NewServer("127.0.0.1:0", t.TempDir(), iogate.NewDiskWriteGate(iogate.TestGateConfig()))
	require.NoError(t, err)
	srv.Gate().SetDegraded(true)
	require.NoError(t, srv.Start())
	defer srv.Stop()
	time.Sleep(100 * time.Millisecond)

	conn, err := net.DialTimeout("tcp", srv.Addr(), time.Second)
	require.NoError(t, err)
	defer conn.Close()

	writeBuf := make([]byte, 4096)
	readBuf := make([]byte, 4096)
	lenBuf := make([]byte, 4)
	id, err := srv.registry.Register(DefaultIngressTopic)
	require.NoError(t, err)
	var batch []byte
	batch = protocol.AppendBatchMessage(batch, id, []byte("x"))
	req := protocol.EncodeProduceBatchRequest(writeBuf, 1, batch)
	_, err = conn.Write(req)
	require.NoError(t, err)
	_, _, resp, err := protocol.ReadFrame(conn, readBuf, lenBuf)
	require.NoError(t, err)
	status, _, committed, err := protocol.DecodeProduceBatchResponse(resp)
	require.NoError(t, err)
	assert.Equal(t, proxyBackpressure, status)
	assert.Equal(t, uint32(0), committed)
}

func TestRegionProxy_ReadyEndpoints(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	srv, err := NewServer("127.0.0.1:0", t.TempDir(), iogate.NewDiskWriteGate(iogate.Config{AppendCapacity: 4}))
	require.NoError(t, err)
	srv.SetHealthAddr("127.0.0.1:0")
	srv.SetReadyProbe(func(ctx context.Context) error {
		return redis.NewClient(&redis.Options{Addr: mr.Addr()}).Ping(ctx).Err()
	})
	require.NoError(t, srv.Start())
	defer srv.Stop()
	time.Sleep(100 * time.Millisecond)

	healthResp, err := http.Get("http://" + srv.HealthAddr() + "/health")
	require.NoError(t, err)
	_ = healthResp.Body.Close()
	assert.Equal(t, http.StatusOK, healthResp.StatusCode)

	readyResp, err := http.Get("http://" + srv.HealthAddr() + "/ready")
	require.NoError(t, err)
	_ = readyResp.Body.Close()
	assert.Equal(t, http.StatusOK, readyResp.StatusCode)

	srv.Gate().SetDegraded(true)
	readyResp, err = http.Get("http://" + srv.HealthAddr() + "/ready")
	require.NoError(t, err)
	_ = readyResp.Body.Close()
	assert.Equal(t, http.StatusServiceUnavailable, readyResp.StatusCode)
}
