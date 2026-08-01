package fraud

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"espx/internal/database"
	"espx/internal/edge/fingerprint"
	"espx/pkg/piihash"

	"github.com/redis/go-redis/v9"
)

const tcpEdgeCorrelationQuery = `
SELECT
    ip_hash,
    any(tls_hash) AS ja3,
    any(toString(campaign_id)) AS campaign_id
FROM clicks
WHERE created_at >= now() - toIntervalSecond(?)
  AND ip_hash IN (?)
GROUP BY ip_hash`

type tcpEdgeCorrelationRule struct {
	q   *database.CHQuery
	rdb redis.Cmdable
	cfg AnalyzerConfig
}

func (r *tcpEdgeCorrelationRule) Name() string { return "tcp_edge_correlation" }

func (r *tcpEdgeCorrelationRule) Find(ctx context.Context) ([]SuspiciousIP, error) {
	if r == nil || r.q == nil || r.rdb == nil {
		return nil, nil
	}
	entries, err := fingerprint.ListRecent(ctx, r.rdb, 128)
	if err != nil {
		return nil, fmt.Errorf("list tcp fingerprints: %w", err)
	}
	if len(entries) == 0 {
		return nil, nil
	}

	ips := make([]string, 0, len(entries))
	hashArgs := make([]string, 0, len(entries))
	seenIP := make(map[string]struct{}, len(entries))
	for _, e := range entries {
		if e.IP == "" {
			continue
		}
		if _, ok := seenIP[e.IP]; ok {
			continue
		}
		seenIP[e.IP] = struct{}{}
		ips = append(ips, e.IP)
		hashArgs = append(hashArgs, piihash.FixedString16(hashIPForCH(e.IP)))
	}
	if len(hashArgs) == 0 {
		return nil, nil
	}

	windowSec := database.ClampCHWindowSeconds(int64(r.cfg.Window / time.Second))

	rows, err := r.q.Query(ctx, tcpEdgeCorrelationQuery, windowSec, hashArgs)
	if err != nil {
		return nil, fmt.Errorf("tcp edge correlation query: %w", err)
	}
	defer rows.Close()

	ipByHash := make(map[string]string, len(ips))
	for i, ip := range ips {
		ipByHash[hashArgs[i]] = ip
	}

	var out []SuspiciousIP
	for rows.Next() {
		var ipHash []byte
		var ja3, campaignID string
		if err := rows.Scan(&ipHash, &ja3, &campaignID); err != nil {
			return nil, fmt.Errorf("scan tcp edge row: %w", err)
		}
		if ja3 == "" || !IsSuspiciousJA3(ja3) {
			continue
		}
		ipKey := hex.EncodeToString(ipHash)
		rawIP := ipByHash[ipKey]
		if rawIP == "" {
			rawIP = ipKey
		}
		out = append(out, SuspiciousIP{
			IP:         rawIP,
			CampaignID: campaignID,
			Reason:     "ivt_tcp_edge_correlation",
			Score:      70,
			Action:     "ghost",
			TTLSeconds: 3600,
		})
	}
	return out, rows.Err()
}
