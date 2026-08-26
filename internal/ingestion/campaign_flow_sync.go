package ingestion

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync/atomic"
	"time"

	"ad-event-processor/pkg/landerhost"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type flowPathJSON struct {
	Weight  int32                `json:"weight"`
	Filters *flowPathFiltersJSON `json:"filters"`
	Landers []struct {
		LanderID uuid.UUID `json:"lander_id"`
		Weight   int32     `json:"weight"`
	} `json:"landers"`
	Offers []struct {
		OfferID  uuid.UUID `json:"offer_id"`
		Weight   int32     `json:"weight"`
		CapDaily *int32    `json:"cap_daily"`
		CapTotal *int32    `json:"cap_total"`
	} `json:"offers"`
}

type campaignFlowSync struct {
	pool          *pgxpool.Pool
	table         *CampaignFlowTable
	interval      time.Duration
	gen           atomic.Uint64
	publicBase    string
	reloadChannel string
	redisShard    redis.UniversalClient
}

func NewCampaignFlowSync(pool *pgxpool.Pool, table *CampaignFlowTable, interval time.Duration, publicBase string, redisShard redis.UniversalClient, reloadChannel string) *campaignFlowSync {
	if pool == nil || table == nil {
		return nil
	}
	if interval <= 0 {
		interval = 30 * time.Second
	}
	if reloadChannel == "" {
		reloadChannel = "flow:reload"
	}
	return &campaignFlowSync{
		pool:          pool,
		table:         table,
		interval:      interval,
		publicBase:    publicBase,
		reloadChannel: reloadChannel,
		redisShard:    redisShard,
	}
}

func (s *campaignFlowSync) Start(ctx context.Context) {
	if s == nil {
		return
	}
	s.reloadOnce(ctx)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	if s.redisShard != nil {
		go s.runReloadSubscriber(ctx)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.reloadOnce(ctx)
		}
	}
}

func (s *campaignFlowSync) runReloadSubscriber(ctx context.Context) {
	pubsub := s.redisShard.Subscribe(ctx, s.reloadChannel)
	defer pubsub.Close()
	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if msg == nil {
				continue
			}
			s.reloadOnce(ctx)
		}
	}
}

func (s *campaignFlowSync) reloadOnce(ctx context.Context) {
	landerURLs, err := s.loadLanderURLMap(ctx)
	if err != nil {
		slog.Warn("campaign flow sync landers", "error", err)
		return
	}
	offerURLs, err := s.loadURLMap(ctx, "offers")
	if err != nil {
		slog.Warn("campaign flow sync offers", "error", err)
		return
	}
	offerCounts, err := loadOfferConversionCounts(ctx, s.pool)
	if err != nil {
		slog.Warn("campaign flow sync offer caps", "error", err)
		offerCounts = map[uuid.UUID]offerConversionCounts{}
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
		snap, ok := buildFlowSnapshot(raw, landerURLs, offerURLs, offerCounts)
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

func (s *campaignFlowSync) loadLanderURLMap(ctx context.Context) (map[uuid.UUID][]byte, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, COALESCE(url, ''), hosted_asset_id
		FROM landers`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[uuid.UUID][]byte)
	for rows.Next() {
		var id uuid.UUID
		var url string
		var hostedAssetID *uuid.UUID
		if err := rows.Scan(&id, &url, &hostedAssetID); err != nil {
			return nil, err
		}
		if url != "" {
			out[id] = []byte(url)
			continue
		}
		if hostedAssetID != nil && *hostedAssetID != uuid.Nil && s.publicBase != "" {
			if hosted := landerhost.PublicURL(s.publicBase, id); hosted != "" {
				out[id] = []byte(hosted)
			}
		}
	}
	return out, rows.Err()
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

func buildFlowSnapshot(raw []byte, landerURLs, offerURLs map[uuid.UUID][]byte, offerCounts map[uuid.UUID]offerConversionCounts) (FlowPathSnapshot, bool) {
	var paths []flowPathJSON
	if err := json.Unmarshal(raw, &paths); err != nil || len(paths) == 0 {
		return FlowPathSnapshot{}, false
	}
	out := FlowPathSnapshot{Paths: make([]FlowPath, 0, len(paths))}
	for _, p := range paths {
		if p.Weight <= 0 || len(p.Landers) == 0 {
			continue
		}
		fp := FlowPath{Weight: p.Weight, Filters: compileFlowPathFilters(p.Filters), Landers: make([]FlowLanderEntry, 0, len(p.Landers)), Offers: make([]FlowOfferEntry, 0, len(p.Offers))}
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
			fp.Offers = append(fp.Offers, FlowOfferEntry{
				OfferID: o.OfferID,
				Weight:  o.Weight,
				URL:     url,
				Capped:  offerIsCapped(o.OfferID, o.CapDaily, o.CapTotal, offerCounts),
			})
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
