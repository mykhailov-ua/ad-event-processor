package ingestion

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type flowPathJSON struct {
	Weight  int32 `json:"weight"`
	Landers []struct {
		LanderID uuid.UUID `json:"lander_id"`
		Weight   int32     `json:"weight"`
	} `json:"landers"`
	Offers []struct {
		OfferID uuid.UUID `json:"offer_id"`
		Weight  int32     `json:"weight"`
	} `json:"offers"`
}

type campaignFlowSync struct {
	pool     *pgxpool.Pool
	table    *CampaignFlowTable
	interval time.Duration
	gen      atomic.Uint64
}

// NewCampaignFlowSync builds the cold-path PG sync worker. Returns nil when disabled.
func NewCampaignFlowSync(pool *pgxpool.Pool, table *CampaignFlowTable, interval time.Duration) *campaignFlowSync {
	if pool == nil || table == nil {
		return nil
	}
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &campaignFlowSync{pool: pool, table: table, interval: interval}
}

func (s *campaignFlowSync) Start(ctx context.Context) {
	if s == nil {
		return
	}
	s.reloadOnce(ctx)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.reloadOnce(ctx)
		}
	}
}

func (s *campaignFlowSync) reloadOnce(ctx context.Context) {
	landerURLs, err := s.loadURLMap(ctx, "landers")
	if err != nil {
		slog.Warn("campaign flow sync landers", "error", err)
		return
	}
	offerURLs, err := s.loadURLMap(ctx, "offers")
	if err != nil {
		slog.Warn("campaign flow sync offers", "error", err)
		return
	}
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, f.paths
		FROM campaigns c
		JOIN flows f ON f.id = c.flow_id
		WHERE c.flow_id IS NOT NULL AND c.deleted_at IS NULL`)
	if err != nil {
		slog.Warn("campaign flow sync campaigns", "error", err)
		return
	}
	defer rows.Close()

	byCampaign := make(map[uuid.UUID]FlowPathSnapshot)
	for rows.Next() {
		var campaignID uuid.UUID
		var raw []byte
		if err := rows.Scan(&campaignID, &raw); err != nil {
			slog.Warn("campaign flow sync scan", "error", err)
			return
		}
		snap, ok := buildFlowSnapshot(raw, landerURLs, offerURLs)
		if !ok {
			continue
		}
		byCampaign[campaignID] = snap
	}
	if err := rows.Err(); err != nil {
		slog.Warn("campaign flow sync rows", "error", err)
		return
	}
	_ = s.gen.Add(1)
	s.table.Publish(&campaignFlowRegistrySnapshot{byCampaign: byCampaign})
}

func (s *campaignFlowSync) loadURLMap(ctx context.Context, table string) (map[uuid.UUID][]byte, error) {
	q := "SELECT id, url FROM " + table
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[uuid.UUID][]byte)
	for rows.Next() {
		var id uuid.UUID
		var url string
		if err := rows.Scan(&id, &url); err != nil {
			return nil, err
		}
		if url != "" {
			out[id] = []byte(url)
		}
	}
	return out, rows.Err()
}

func buildFlowSnapshot(raw []byte, landerURLs, offerURLs map[uuid.UUID][]byte) (FlowPathSnapshot, bool) {
	var paths []flowPathJSON
	if err := json.Unmarshal(raw, &paths); err != nil || len(paths) == 0 {
		return FlowPathSnapshot{}, false
	}
	out := FlowPathSnapshot{Paths: make([]FlowPath, 0, len(paths))}
	for _, p := range paths {
		if p.Weight <= 0 || len(p.Landers) == 0 {
			continue
		}
		fp := FlowPath{Weight: p.Weight, Landers: make([]FlowLanderEntry, 0, len(p.Landers)), Offers: make([]FlowOfferEntry, 0, len(p.Offers))}
		for _, l := range p.Landers {
			url := landerURLs[l.LanderID]
			if l.Weight <= 0 || len(url) == 0 {
				continue
			}
			fp.Landers = append(fp.Landers, FlowLanderEntry{LanderID: l.LanderID, Weight: l.Weight, URL: url})
		}
		for _, o := range p.Offers {
			url := offerURLs[o.OfferID]
			if o.Weight <= 0 {
				continue
			}
			fp.Offers = append(fp.Offers, FlowOfferEntry{OfferID: o.OfferID, Weight: o.Weight, URL: url})
		}
		if len(fp.Landers) == 0 {
			continue
		}
		if len(fp.Offers) == 0 {
			fp.Offers = []FlowOfferEntry{{OfferID: uuid.Nil, Weight: 100}}
		}
		out.Paths = append(out.Paths, fp)
	}
	if len(out.Paths) == 0 {
		return FlowPathSnapshot{}, false
	}
	return out, true
}
