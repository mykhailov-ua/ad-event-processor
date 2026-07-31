package controlplane

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"espx/internal/config"

	db "espx/internal/domain/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

var defaultProcessorPeerOrder = []string{"processor", "processor-1"}

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
			Role:       RoleProcessor,
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
