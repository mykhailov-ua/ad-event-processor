package fraudadmin

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"ad-event-processor/internal/filter"
	"ad-event-processor/pkg/probecluster"
)

type ProbeClusterSummaryDTO struct {
	ClusterID     string `json:"cluster_id"`
	SessionCount  int64  `json:"session_count"`
	CampaignCount int64  `json:"campaign_count"`
	VerifyCount   int64  `json:"verify_count"`
	AvgProbeScore uint8  `json:"avg_probe_score"`
	JA3           string `json:"ja3,omitempty"`
	JA4           string `json:"ja4,omitempty"`
	TCPSig        string `json:"tcp_sig,omitempty"`
	WebGL         string `json:"webgl,omitempty"`
	ExportDone    bool   `json:"export_done"`
}

type ProbeCluster struct {
	host  ProbeClusterHost
	store *filter.ProbeClusterStore
}

func NewProbeCluster(host ProbeClusterHost, store *filter.ProbeClusterStore) *ProbeCluster {
	if host == nil || store == nil {
		return nil
	}
	return &ProbeCluster{host: host, store: store}
}

func (p *ProbeCluster) GetSummary(ctx context.Context, clusterIDHex string) (ProbeClusterSummaryDTO, error) {
	if p == nil || p.store == nil {
		return ProbeClusterSummaryDTO{}, fmt.Errorf("probe cluster store not configured")
	}
	clusterIDHex = strings.TrimSpace(clusterIDHex)
	if len(clusterIDHex) != 32 {
		return ProbeClusterSummaryDTO{}, ValidationError("invalid cluster_id")
	}
	if _, err := hex.DecodeString(clusterIDHex); err != nil {
		return ProbeClusterSummaryDTO{}, ValidationError("invalid cluster_id")
	}
	summary, ok, err := p.store.GetSummary(ctx, clusterIDHex)
	if err != nil {
		return ProbeClusterSummaryDTO{}, err
	}
	if !ok {
		return ProbeClusterSummaryDTO{}, ErrProbeClusterNotFound
	}
	return mapProbeClusterSummary(summary), nil
}

func mapProbeClusterSummary(summary filter.ProbeClusterSummary) ProbeClusterSummaryDTO {
	return ProbeClusterSummaryDTO{
		ClusterID:     summary.ClusterID,
		SessionCount:  summary.SessionCount,
		CampaignCount: summary.CampaignCount,
		VerifyCount:   summary.VerifyCount,
		AvgProbeScore: summary.AvgProbeScore,
		JA3:           summary.JA3,
		JA4:           summary.JA4,
		TCPSig:        summary.TCPSig,
		WebGL:         summary.WebGL,
		ExportDone:    summary.ExportDone,
	}
}

var ErrProbeClusterNotFound = fmt.Errorf("probe cluster not found")

func (p *ProbeCluster) ExportHotClusters(ctx context.Context, policy probecluster.Policy) (int, error) {
	if p == nil || p.host == nil || p.store == nil || p.host.ModeratorCorpusPool() == nil {
		return 0, nil
	}
	if !p.host.ProbeClusterExportEnabled() {
		return 0, nil
	}
	corpus := NewModeratorCorpus(p.host)
	if corpus == nil {
		return 0, fmt.Errorf("moderator corpus not configured")
	}
	exported := 0
	var cursor uint64
	for {
		keys, next, err := p.store.ScanMetaKeys(ctx, cursor, 64)
		if err != nil {
			return exported, err
		}
		for _, key := range keys {
			clusterIDHex := probeClusterIDFromMetaKey(key)
			if clusterIDHex == "" {
				continue
			}
			summary, ok, err := p.store.GetSummary(ctx, clusterIDHex)
			if err != nil || !ok || summary.ExportDone {
				continue
			}
			state := probecluster.State{
				SessionCount:  summary.SessionCount,
				CampaignCount: summary.CampaignCount,
				VerifyCount:   summary.VerifyCount,
				AvgProbeScore: summary.AvgProbeScore,
			}
			if !probecluster.ShouldRouteSandbox(state, policy) {
				continue
			}
			note := "probe_cluster:" + clusterIDHex
			_, err = corpus.UpsertTuple(ctx, ModeratorCorpusUpsertRequest{
				JA3:           summary.JA3,
				JA4:           summary.JA4,
				TCPSig:        summary.TCPSig,
				WebGLRenderer: summary.WebGL,
				Note:          note,
				Source:        "probe_cluster",
			})
			if err != nil {
				return exported, err
			}
			if err := p.store.MarkExported(ctx, clusterIDHex); err != nil {
				return exported, err
			}
			exported++
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	if exported > 0 {
		if err := p.host.RefreshModeratorCorpusFeed(ctx); err != nil {
			return exported, err
		}
	}
	return exported, nil
}

func probeClusterIDFromMetaKey(key string) string {
	const prefix = "probe:cluster:"
	const suffix = ":meta"
	if !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, suffix) {
		return ""
	}
	hexID := strings.TrimPrefix(key, prefix)
	hexID = strings.TrimSuffix(hexID, suffix)
	if len(hexID) != 32 {
		return ""
	}
	return hexID
}
