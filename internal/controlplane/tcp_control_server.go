package controlplane

import (
	"context"
	"log/slog"
	"net"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TCPControlServer struct {
	cfg       *config.Config
	pool      *pgxpool.Pool
	sharder   domain.Sharder
	secret    []byte
	numShards int
	ln        net.Listener
	connSem   chan struct{}
}

const tcpControlMaxConcurrent = 64

func NewTCPControlServer(cfg *config.Config, pool *pgxpool.Pool, sharder domain.Sharder, numShards int) *TCPControlServer {
	return &TCPControlServer{
		cfg:       cfg,
		pool:      pool,
		sharder:   sharder,
		secret:    []byte(cfg.TCPControlHMACSecret),
		numShards: numShards,
		connSem:   make(chan struct{}, tcpControlMaxConcurrent),
	}
}

func (s *TCPControlServer) Start(ctx context.Context) error {
	if s == nil || s.cfg == nil || !s.cfg.TCPControlEnabled {
		return nil
	}
	bind := s.cfg.TCPControlBindAddr
	if bind == "" {
		bind = ":8192"
	}
	ln, err := net.Listen("tcp", bind)
	if err != nil {
		return err
	}
	s.ln = ln
	s.connSem = make(chan struct{}, tcpControlMaxConcurrent)
	slog.Info("management tcp control started", "bind", bind)
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()
	go s.acceptLoop(ctx)
	return nil
}

func (s *TCPControlServer) Close() error {
	if s != nil && s.ln != nil {
		return s.ln.Close()
	}
	return nil
}

func (s *TCPControlServer) PublishSnapshot(ctx context.Context, routingEpoch int64, slotVersion int32) {
	if s == nil || !s.cfg.TCPControlEnabled {
		return
	}
	for _, addr := range s.cfg.TCPTrackerAddrs {
		if err := s.pushSnapshot(ctx, addr, routingEpoch, slotVersion); err != nil {
			slog.Warn("tcp snapshot push failed", "tracker", addr, "error", err)
		}
	}
}

func (s *TCPControlServer) acceptLoop(ctx context.Context) {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}
		select {
		case s.connSem <- struct{}{}:
			go func() {
				defer func() { <-s.connSem }()
				s.handleConn(ctx, conn)
			}()
		case <-ctx.Done():
			_ = conn.Close()
			return
		default:
			metrics.TCPControlSnapshotErrorsTotal.Inc()
			_ = conn.Close()
		}
	}
}

func (s *TCPControlServer) handleConn(ctx context.Context, conn net.Conn) {
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	var buf [4096]byte
	n, err := conn.Read(buf[:])
	if err != nil || n < domain.TCPControlHeaderSize {
		return
	}
	var hdr domain.TCPControlHeader
	payload, err := domain.DecodeTCPControlFrame(buf[:n], s.secret, &hdr)
	if err != nil {
		metrics.TCPControlSnapshotErrorsTotal.Inc()
		return
	}
	switch hdr.MsgType {
	case domain.TCPMsgSnapshotRequest:
		epoch, slotVer := s.currentEpoch(ctx)
		if err := s.writeSnapshot(conn, epoch, slotVer); err != nil {
			metrics.TCPControlSnapshotErrorsTotal.Inc()
			return
		}
		metrics.TCPControlSnapshotSentTotal.Inc()
	case domain.TCPMsgAck:
		var ack domain.TCPAckPayload
		if domain.DecodeTCPAckPayload(payload, &ack) {
			metrics.TCPControlAckReceivedTotal.Inc()
			slog.Debug("tcp routing ack", "tracker", ack.TrackerID, "epoch", ack.AppliedEpoch, "slot_version", ack.AppliedSlotVer)
		}
	}
}

func (s *TCPControlServer) currentEpoch(ctx context.Context) (int64, int32) {
	slotVer := int32(0)
	if sh, ok := s.sharder.(*domain.StaticSlotSharder); ok {
		snap := sh.Snapshot()
		slotVer = snap.Version
		if snap.MigrationGen > 0 {
			return snap.MigrationGen, slotVer
		}
	}
	if s.pool != nil {
		row, err := domain.NewCampaignRoutingRepo(s.pool).GetGlobalRoutingEpoch(ctx)
		if err == nil {
			return row.RoutingEpoch, row.ActiveVersion
		}
	}
	return 0, slotVer
}

func (s *TCPControlServer) writeSnapshot(conn net.Conn, routingEpoch int64, slotVersion int32) error {
	limits := s.buildLimits()
	var payload [domain.UDPMaxControlShards * 8]byte
	pl := domain.EncodeTCPLimitsPayload(payload[:], limits)
	var hdr domain.TCPControlHeader
	hdr.MsgType = domain.TCPMsgSnapshot
	hdr.RoutingEpoch = routingEpoch
	hdr.SlotMapVersion = slotVersion
	hdr.NumShards = limits.NumShards
	var frame [512]byte
	n, err := domain.EncodeTCPControlFrame(frame[:], s.secret, &hdr, payload[:pl])
	if err != nil {
		return err
	}
	_, err = conn.Write(frame[:n])
	return err
}

func (s *TCPControlServer) pushSnapshot(ctx context.Context, addr string, routingEpoch int64, slotVersion int32) error {
	dialer := net.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	return s.writeSnapshot(conn, routingEpoch, slotVersion)
}

func (s *TCPControlServer) buildLimits() *domain.UDPControlLimits {
	n := s.numShards
	if n <= 0 {
		n = 1
	}
	if n > domain.UDPMaxControlShards {
		n = domain.UDPMaxControlShards
	}
	limits := &domain.UDPControlLimits{NumShards: uint8(n)}
	fallback := uint64(50_000)
	if s.cfg != nil && s.cfg.UDPDefaultShardRPS > 0 {
		fallback = s.cfg.UDPDefaultShardRPS
	}
	for i := 0; i < n; i++ {
		limits.Limits[i] = fallback
	}
	return limits
}
