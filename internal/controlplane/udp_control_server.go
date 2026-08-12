package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"sync/atomic"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/domain"
	db "github.com/bidshard/ad-event-processor/internal/domain/db"
	"github.com/bidshard/ad-event-processor/internal/metrics"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UDPControlServer struct {
	cfg       *config.Config
	pool      *pgxpool.Pool
	sharder   domain.Sharder
	conn      *net.UDPConn
	epoch     atomic.Int64
	numShards int
	trackers  []*net.UDPAddr
}

func NewUDPControlServer(cfg *config.Config, pool *pgxpool.Pool, sharder domain.Sharder, numShards int) *UDPControlServer {
	s := &UDPControlServer{
		cfg:       cfg,
		pool:      pool,
		sharder:   sharder,
		numShards: numShards,
	}
	for _, raw := range cfg.UDPTrackerAddrs {
		if addr, err := net.ResolveUDPAddr("udp", raw); err == nil {
			s.trackers = append(s.trackers, addr)
		}
	}
	return s
}

func (s *UDPControlServer) Start(ctx context.Context) error {
	if s == nil || s.cfg == nil || !s.cfg.UDPControlEnabled {
		return nil
	}
	addr, err := net.ResolveUDPAddr("udp", s.cfg.UDPControlBindAddr)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	s.conn = conn
	go s.recvLoop(ctx)
	go s.publishLoop(ctx)
	slog.Info("management udp control started", "bind", addr.String(), "trackers", len(s.trackers))
	return nil
}

func (s *UDPControlServer) Close() error {
	if s != nil && s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

func (s *UDPControlServer) recvLoop(ctx context.Context) {
	buf := make([]byte, 2048)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		_ = s.conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, remote, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				continue
			}
			if ctx.Err() != nil {
				return
			}
			continue
		}
		var hdr domain.UDPHeader
		if !domain.DecodeUDPHeader(buf[:n], &hdr) {
			continue
		}
		if hdr.MsgType != domain.UDPMsgConfigRequest {
			continue
		}
		payload := buf[domain.UDPHeaderSize:n]
		var req domain.UDPConfigRequestPayload
		if !domain.DecodeUDPConfigRequest(payload, &req) {
			continue
		}
		slog.Debug("udp config request", "tracker", req.TrackerID, "last_epoch", req.LastEpoch, "remote", remote)
		s.sendSnapshotBurst(ctx, remote, 5)
	}
}

func (s *UDPControlServer) publishLoop(ctx context.Context) {
	interval := time.Duration(s.cfg.UDPSyncIntervalMs) * time.Millisecond
	if interval <= 0 {
		interval = 10 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	s.publishEpoch(ctx, false)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.publishEpoch(ctx, false)
		}
	}
}

func (s *UDPControlServer) publishEpoch(ctx context.Context, snapshot bool) {
	limits := s.buildLimits()
	weights := s.buildNodeWeights(ctx)
	epoch := s.epoch.Add(1)
	slotVersion := int32(0)
	if sh, ok := s.sharder.(*domain.StaticSlotSharder); ok {
		slotVersion = sh.SnapshotVersion()
	}
	hash := domain.ComputeUDPConfigHashWithWeights(epoch, slotVersion, limits, weights)
	if err := s.persistEpoch(ctx, epoch, hash, slotVersion, limits, weights); err != nil {
		slog.Warn("control_plane_epochs insert failed", "error", err)
	}
	msgType := domain.UDPMsgQuotaEpoch
	flags := uint16(0)
	if snapshot {
		msgType = domain.UDPMsgConfigSnapshot
		flags = domain.UDPFlagSnapshot
	}
	hdr := &domain.UDPHeader{
		CoarseTimeNs:   time.Now().UnixNano(),
		EpochID:        epoch,
		ConfigHash:     hash,
		SlotMapVersion: slotVersion,
		Flags:          flags,
	}
	pkt := make([]byte, 4096)
	n := domain.EncodeQuotaEpochDatagramWithWeights(pkt, msgType, hdr, limits, weights)
	if n == 0 {
		return
	}
	for _, taddr := range s.trackers {
		for i := 0; i < 3; i++ {
			_, _ = s.conn.WriteToUDP(pkt[:n], taddr)
		}
	}
	if bcast, err := net.ResolveUDPAddr("udp", "127.0.0.1:8191"); err == nil {
		_, _ = s.conn.WriteToUDP(pkt[:n], bcast)
	}
	metrics.UDPControlPublishTotal.Inc()
}

func (s *UDPControlServer) sendSnapshotBurst(ctx context.Context, addr *net.UDPAddr, count int) {
	limits := s.buildLimits()
	weights := s.buildNodeWeights(ctx)
	epoch := s.epoch.Load()
	if epoch == 0 {
		s.publishEpoch(ctx, true)
		epoch = s.epoch.Load()
	}
	slotVersion := int32(0)
	if sh, ok := s.sharder.(*domain.StaticSlotSharder); ok {
		slotVersion = sh.SnapshotVersion()
	}
	hash := domain.ComputeUDPConfigHashWithWeights(epoch, slotVersion, limits, weights)
	hdr := &domain.UDPHeader{
		CoarseTimeNs:   time.Now().UnixNano(),
		EpochID:        epoch,
		ConfigHash:     hash,
		SlotMapVersion: slotVersion,
		Flags:          domain.UDPFlagSnapshot,
	}
	pkt := make([]byte, 4096)
	n := domain.EncodeQuotaEpochDatagramWithWeights(pkt, domain.UDPMsgConfigSnapshot, hdr, limits, weights)
	if n == 0 {
		return
	}
	for i := 0; i < count; i++ {
		_, _ = s.conn.WriteToUDP(pkt[:n], addr)
	}
	metrics.UDPControlPublishTotal.Add(float64(count))
}

func (s *UDPControlServer) buildLimits() *domain.UDPControlLimits {
	n := s.numShards
	if n <= 0 {
		n = 1
	}
	if n > domain.UDPMaxControlShards {
		n = domain.UDPMaxControlShards
	}
	limits := &domain.UDPControlLimits{NumShards: uint8(n)}

	var maxRPS uint64
	var maxRPD uint64
	if s.pool != nil {
		var entitlementsJSON []byte
		err := s.pool.QueryRow(context.Background(), `
			SELECT entitlements_json FROM billing.license_status LIMIT 1`).Scan(&entitlementsJSON)
		if err == nil && len(entitlementsJSON) > 0 {
			var ent struct {
				Limits struct {
					MaxRPS            uint64 `json:"max_rps"`
					MaxRequestsPerDay uint64 `json:"max_requests_per_day"`
					MaxRegions        uint64 `json:"max_regions"`
				} `json:"limits"`
				Features struct {
					MultiRegion bool `json:"multi_region"`
				} `json:"features"`
			}
			if json.Unmarshal(entitlementsJSON, &ent) == nil {
				maxRPS = ent.Limits.MaxRPS
				maxRPD = ent.Limits.MaxRequestsPerDay
			}
		}
	}

	if maxRPS > 0 {
		shardRPS := maxRPS / uint64(n)
		if shardRPS == 0 {
			shardRPS = 1
		}
		for i := 0; i < n; i++ {
			limits.Limits[i] = shardRPS
		}
	} else {
		fallback := s.cfg.UDPDefaultShardRPS
		if fallback == 0 {
			fallback = 50_000
		}
		for i := 0; i < n; i++ {
			limits.Limits[i] = fallback
		}
	}

	if s.cfg != nil && s.cfg.MultiRegionEnabled && maxRPD > 0 && s.pool != nil {
		var regionCount int64
		if err := s.pool.QueryRow(context.Background(), `
			SELECT COUNT(*) FROM regions WHERE active = TRUE`).Scan(&regionCount); err == nil && regionCount > 0 {
			limits.MaxRPD = maxRPD / uint64(regionCount)
		}
	}
	return limits
}

func (s *UDPControlServer) buildNodeWeights(ctx context.Context) []domain.UDPNodeWeight {
	if s == nil || s.pool == nil || s.cfg == nil || !s.cfg.MultiRegionEnabled {
		return nil
	}
	q := db.New(s.pool)
	rows, err := q.ListNodeCapacityScoresByRegion(ctx, int16(s.cfg.RegionCode))
	if err != nil || len(rows) == 0 {
		return nil
	}
	out := make([]domain.UDPNodeWeight, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.UDPNodeWeight{
			NodeID:     row.NodeID,
			Role:       row.Role,
			Weight:     row.Weight,
			Score:      row.Score,
			Provenance: domain.ProvenanceToUDPCode(row.Provenance),
		})
	}
	return out
}

func (s *UDPControlServer) persistEpoch(ctx context.Context, epoch int64, hash [16]byte, slotVersion int32, limits *domain.UDPControlLimits, weights []domain.UDPNodeWeight) error {
	if s.pool == nil {
		return nil
	}
	payload, err := domain.MarshalEpochPayload(slotVersion, limits, weights)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO control_plane_epochs (epoch_id, config_hash, payload_json)
		VALUES ($1, $2, $3::jsonb)
		ON CONFLICT (epoch_id) DO NOTHING`,
		epoch, hash[:], json.RawMessage(payload))
	return err
}
