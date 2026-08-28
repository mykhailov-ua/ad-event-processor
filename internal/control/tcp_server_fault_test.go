package control

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/pkg/faultproof"

	"github.com/stretchr/testify/require"
)

func requestTCPSnapshotACK(t *testing.T, ctx context.Context, addr string, secret []byte, trackerID uint32, sh *domain.StaticSlotSharder) {
	t.Helper()
	dialer := net.Dialer{Timeout: 2 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	require.NoError(t, err)
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))

	var reqHdr domain.TCPControlHeader
	reqHdr.MsgType = domain.TCPMsgSnapshotRequest
	reqHdr.TrackerID = trackerID
	if sh != nil {
		reqHdr.SlotMapVersion = sh.ActiveVersion()
		reqHdr.RoutingEpoch = sh.Snapshot().MigrationGen
	}
	var reqBuf [domain.TCPControlHeaderSize]byte
	_, err = domain.EncodeTCPControlFrame(reqBuf[:], secret, &reqHdr, nil)
	require.NoError(t, err)
	_, err = conn.Write(reqBuf[:])
	require.NoError(t, err)

	var frame [4096]byte
	n, err := io.ReadAtLeast(conn, frame[:], domain.TCPControlHeaderSize)
	require.NoError(t, err)
	for n < cap(frame) {
		m, rerr := conn.Read(frame[n:])
		n += m
		if rerr != nil {
			if errors.Is(rerr, io.EOF) {
				break
			}
			require.NoError(t, rerr)
		}
		if m == 0 {
			break
		}
	}
	var hdr domain.TCPControlHeader
	_, err = domain.DecodeTCPControlFrame(frame[:n], secret, &hdr)
	require.NoError(t, err)
	require.Equal(t, domain.TCPMsgSnapshot, hdr.MsgType)

	ack := domain.TCPAckPayload{
		TrackerID:      trackerID,
		AppliedEpoch:   hdr.RoutingEpoch,
		AppliedSlotVer: hdr.SlotMapVersion,
	}
	var body [16]byte
	require.NotZero(t, domain.EncodeTCPAckPayload(body[:], &ack))
	var ackHdr domain.TCPControlHeader
	ackHdr.MsgType = domain.TCPMsgAck
	ackHdr.TrackerID = trackerID
	ackHdr.RoutingEpoch = hdr.RoutingEpoch
	ackHdr.SlotMapVersion = hdr.SlotMapVersion
	var ackFrame [80]byte
	ackN, err := domain.EncodeTCPControlFrame(ackFrame[:], secret, &ackHdr, body[:])
	require.NoError(t, err)
	_, err = conn.Write(ackFrame[:ackN])
	require.NoError(t, err)
}

func TestFault_TCP_SnapshotHMACACK(t *testing.T) {
	secret := []byte("tcp-hmac-secret")
	cfg := &config.Config{
		TCPControlEnabled:    true,
		TCPControlHMACSecret: config.Secret(secret),
	}
	sh := domain.NewStaticSlotSharder(2)
	srv := NewTCPControlServer(cfg, nil, sh, 2)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	srv.ln = ln

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go srv.acceptLoop(ctx)

	requestTCPSnapshotACK(t, ctx, ln.Addr().String(), secret, 1, sh)

	faultproof.Log(t, "tcp_snapshot_hmac_ack", map[string]string{
		"subsystem": "tcp_control",
		"ack":       "true",
	})
}
