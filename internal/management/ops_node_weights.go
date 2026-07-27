package management

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"espx/internal/config"

	db "espx/internal/ingestion/sqlc"

	"github.com/jackc/pgx/v5/pgxpool"
)

// OpsNodeWeightEntry is one tracker peer in the edge routing snapshot (§8).
type OpsNodeWeightEntry struct {
	NodeID     string  `json:"node_id"`
	PeerIndex  int     `json:"peer_index"`
	Weight     float64 `json:"weight"`
	Score      float64 `json:"score"`
	Provenance string  `json:"provenance"`
}

// OpsNodeWeightsResponse is returned by GET /ops/node-weights for edge Lua sync.
type OpsNodeWeightsResponse struct {
	Epoch    int64                `json:"epoch"`
	EpochLag int64                `json:"epoch_lag"`
	Nodes    []OpsNodeWeightEntry `json:"node_weights"`
}

var defaultTrackerPeerOrder = []string{"tracker-1", "tracker-2", "tracker-3", "tracker-4"}

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
			Role:       RoleTracker,
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
