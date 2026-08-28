package opsadmin

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/nodeadmin"
	"ad-event-processor/pkg/lifecycle"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

type OpsNodeWeightEntry struct {
	NodeID     string  `json:"node_id"`
	PeerIndex  int     `json:"peer_index"`
	Weight     float64 `json:"weight"`
	Score      float64 `json:"score"`
	Provenance string  `json:"provenance"`
}

type OpsNodeWeightsResponse struct {
	Epoch    int64                `json:"epoch"`
	EpochLag int64                `json:"epoch_lag"`
	Nodes    []OpsNodeWeightEntry `json:"node_weights"`
}

type PlatformRoutesDeps struct {
	Pool         *pgxpool.Pool
	RedisShards  []redis.UniversalClient
	Config       *config.Config
	LicenseReady func() bool
}

var defaultProcessorPeerOrder = []string{"processor", "processor-1"}

var defaultTrackerPeerOrder = []string{"tracker-1", "tracker-2", "tracker-3", "tracker-4"}

func RegisterOpsRoutes(ctx context.Context, mux *http.ServeMux, deps PlatformRoutesDeps) {
	pool := deps.Pool
	redisShards := deps.RedisShards
	cfg := deps.Config

	live := &lifecycle.Liveness{}
	ready := &lifecycle.ReadinessProbe{}
	ready.StartBackground(ctx, 2*time.Second, func(ctx context.Context) bool {
		if err := pool.Ping(ctx); err != nil {
			return false
		}
		if !pingConnectedRedisShards(ctx, redisShards) {
			return false
		}
		if deps.LicenseReady != nil && !deps.LicenseReady() {
			return false
		}
		return true
	})
	lifecycle.Register(mux, live, ready)
	prometheus.MustRegister(database.NewPgTableStatsCollector(pool))
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		ready.ServeReadyz(w, r)
	})
	mux.Handle("GET /metrics", promhttp.Handler())
	mux.HandleFunc("GET /ops/shards/slot-map", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		repo := domain.NewSlotMapRepo(pool)
		active, err := repo.GetActiveVersion(ctx)
		if err != nil {
			http.Error(w, "slot map meta unavailable", http.StatusServiceUnavailable)
			return
		}
		meta, err := repo.GetSlotMapMeta(ctx)
		if err != nil {
			http.Error(w, "slot map meta unavailable", http.StatusServiceUnavailable)
			return
		}
		rows, err := repo.ListVersion(ctx, active)
		if err != nil {
			http.Error(w, "slot map unavailable", http.StatusServiceUnavailable)
			return
		}
		slots, err := domain.SlotMapShardTable(rows)
		if err != nil {
			http.Error(w, "slot map incomplete", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(domain.OpsSlotMapResponse{
			Version:       active,
			ActiveVersion: active,
			RoutingEpoch:  meta.RoutingEpoch,
			Slots:         slots,
		}); err != nil {
			slog.Error("ops slot map encode failed", "error", err)
		}
	})
	registerOpsNodeWeights(mux, pool, cfg)
	registerOpsProcessorWeights(mux, pool, cfg)
}

func pingConnectedRedisShards(ctx context.Context, redisShards []redis.UniversalClient) bool {
	if len(redisShards) == 0 {
		return true
	}
	checked := 0
	for i, redisClient := range redisShards {
		if redisClient == nil {
			continue
		}
		checked++
		if err := redisClient.Ping(ctx).Err(); err != nil {
			slog.Warn("redis shard ping failed", "shard", i, "error", err)
			return false
		}
	}
	return checked > 0
}

func peerIndexForProcessor(nodeID string) int {
	for i, id := range defaultProcessorPeerOrder {
		if id == nodeID {
			return i
		}
	}
	return -1
}

func registerOpsProcessorWeights(mux *http.ServeMux, pool *pgxpool.Pool, cfg *config.Config) {
	if mux == nil || pool == nil {
		return
	}
	mux.HandleFunc("GET /ops/processor-weights", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		region := int16(0)
		if cfg != nil {
			region = int16(cfg.RegionCode)
		}

		q := db.New(pool)
		rows, err := q.ListNodeCapacityScoresByRegionRole(ctx, db.ListNodeCapacityScoresByRegionRoleParams{
			RegionCode: region,
			Role:       nodeadmin.RoleProcessor,
		})
		if err != nil {
			http.Error(w, "processor weights unavailable", http.StatusServiceUnavailable)
			return
		}

		var maxEpoch int64
		nodes := make([]OpsNodeWeightEntry, 0, len(rows))
		for _, row := range rows {
			idx := peerIndexForProcessor(row.NodeID)
			if idx < 0 {
				continue
			}
			if row.EpochID > maxEpoch {
				maxEpoch = row.EpochID
			}
			nodes = append(nodes, OpsNodeWeightEntry{
				NodeID:     row.NodeID,
				PeerIndex:  idx,
				Weight:     row.Weight,
				Score:      row.Score,
				Provenance: row.Provenance,
			})
		}
		sort.Slice(nodes, func(i, j int) bool {
			return nodes[i].PeerIndex < nodes[j].PeerIndex
		})

		var controlEpoch int64
		_ = pool.QueryRow(ctx, `SELECT COALESCE(MAX(epoch_id), 0) FROM control_plane_epochs`).Scan(&controlEpoch)

		epochLag := int64(0)
		if controlEpoch > maxEpoch {
			epochLag = controlEpoch - maxEpoch
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(OpsNodeWeightsResponse{
			Epoch:    maxEpoch,
			EpochLag: epochLag,
			Nodes:    nodes,
		}); err != nil {
			slog.Error("ops processor weights encode failed", "error", err)
		}
	})
}

func peerIndexForNode(nodeID string) int {
	for i, id := range defaultTrackerPeerOrder {
		if id == nodeID {
			return i
		}
	}
	return -1
}

func registerOpsNodeWeights(mux *http.ServeMux, pool *pgxpool.Pool, cfg *config.Config) {
	if mux == nil || pool == nil {
		return
	}
	mux.HandleFunc("GET /ops/node-weights", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		region := int16(0)
		if cfg != nil {
			region = int16(cfg.RegionCode)
		}

		q := db.New(pool)
		rows, err := q.ListNodeCapacityScoresByRegionRole(ctx, db.ListNodeCapacityScoresByRegionRoleParams{
			RegionCode: region,
			Role:       nodeadmin.RoleTracker,
		})
		if err != nil {
			http.Error(w, "node weights unavailable", http.StatusServiceUnavailable)
			return
		}

		var maxEpoch int64
		nodes := make([]OpsNodeWeightEntry, 0, len(rows))
		for _, row := range rows {
			idx := peerIndexForNode(row.NodeID)
			if idx < 0 {
				continue
			}
			if row.EpochID > maxEpoch {
				maxEpoch = row.EpochID
			}
			nodes = append(nodes, OpsNodeWeightEntry{
				NodeID:     row.NodeID,
				PeerIndex:  idx,
				Weight:     row.Weight,
				Score:      row.Score,
				Provenance: row.Provenance,
			})
		}
		sort.Slice(nodes, func(i, j int) bool {
			return nodes[i].PeerIndex < nodes[j].PeerIndex
		})

		var controlEpoch int64
		_ = pool.QueryRow(ctx, `SELECT COALESCE(MAX(epoch_id), 0) FROM control_plane_epochs`).Scan(&controlEpoch)

		epochLag := int64(0)
		if controlEpoch > maxEpoch {
			epochLag = controlEpoch - maxEpoch
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(OpsNodeWeightsResponse{
			Epoch:    maxEpoch,
			EpochLag: epochLag,
			Nodes:    nodes,
		}); err != nil {
			slog.Error("ops node weights encode failed", "error", err)
		}
	})
}
