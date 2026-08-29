package filter

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/pkg/landerhost"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	redis "github.com/redis/go-redis/v9"
)

type campaignFlowRegistrySnapshot struct {
	byCampaign map[uuid.UUID]FlowPathSnapshot
}

type CampaignFlowRegistrySnapshot = campaignFlowRegistrySnapshot

func NewCampaignFlowRegistrySnapshot(byCampaign map[uuid.UUID]FlowPathSnapshot) *CampaignFlowRegistrySnapshot {
	return &campaignFlowRegistrySnapshot{byCampaign: byCampaign}
}

type CampaignFlowTable struct {
	active atomic.Pointer[campaignFlowRegistrySnapshot]
}

func NewCampaignFlowTable() *CampaignFlowTable {
	return &CampaignFlowTable{}
}

func (t *CampaignFlowTable) Publish(snap *CampaignFlowRegistrySnapshot) {
	if t == nil || snap == nil {
		return
	}
	t.active.Store(snap)
}

func (t *CampaignFlowTable) Ready() bool {
	return t != nil && t.active.Load() != nil
}

func (t *CampaignFlowTable) Select(campaignID uuid.UUID, userID []byte, ctx FlowSelectContext) (sel FlowSelection, landerURL []byte, ok bool) {
	if t == nil || campaignID == uuid.Nil || len(userID) == 0 {
		return FlowSelection{}, nil, false
	}
	snap := t.active.Load()
	if snap == nil {
		return FlowSelection{}, nil, false
	}
	flow, ok := snap.byCampaign[campaignID]
	if !ok {
		return FlowSelection{}, nil, false
	}
	return SelectSnapshot(&flow, userID, ctx)
}

func (t *CampaignFlowTable) SelectForEvent(campaignID uuid.UUID, userID []byte, evt *domain.Event) (sel FlowSelection, landerURL []byte, ok bool) {
	return t.Select(campaignID, userID, flowSelectContextFromEvent(evt))
}

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
	defer func() { _ = pubsub.Close() }()
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

const (
	flowDeviceDesktop uint8 = 1 << iota
	flowDeviceMobile
	flowDeviceTablet

	flowOSAndroid uint8 = 1 << iota
	flowOSIOS
	flowOSWindows
	flowOSMacOS
	flowOSLinux
)

type FlowPathFilters struct {
	Countries [][2]byte
	Devices   uint8
	OSMask    uint8
	Languages [][2]byte
}

type FlowSelectContext struct {
	Country    [2]byte
	DeviceMask uint8
	OSMask     uint8
	Language   [2]byte
}

type flowPathFiltersJSON struct {
	Countries []string `json:"countries"`
	Devices   []string `json:"devices"`
	OS        []string `json:"os"`
	Languages []string `json:"languages"`
}

func compileFlowPathFilters(raw *flowPathFiltersJSON) FlowPathFilters {
	if raw == nil {
		return FlowPathFilters{}
	}
	out := FlowPathFilters{}
	for _, code := range raw.Countries {
		code = strings.ToUpper(strings.TrimSpace(code))
		if len(code) != 2 {
			continue
		}
		out.Countries = append(out.Countries, [2]byte{code[0], code[1]})
	}
	for _, device := range raw.Devices {
		switch strings.ToLower(strings.TrimSpace(device)) {
		case "desktop":
			out.Devices |= flowDeviceDesktop
		case "mobile":
			out.Devices |= flowDeviceMobile
		case "tablet":
			out.Devices |= flowDeviceTablet
		}
	}
	for _, osName := range raw.OS {
		switch strings.ToLower(strings.TrimSpace(osName)) {
		case "android":
			out.OSMask |= flowOSAndroid
		case "ios":
			out.OSMask |= flowOSIOS
		case "windows":
			out.OSMask |= flowOSWindows
		case "macos":
			out.OSMask |= flowOSMacOS
		case "linux":
			out.OSMask |= flowOSLinux
		}
	}
	for _, lang := range raw.Languages {
		lang = strings.ToLower(strings.TrimSpace(lang))
		if len(lang) != 2 {
			continue
		}
		out.Languages = append(out.Languages, [2]byte{lang[0], lang[1]})
	}
	return out
}

func flowPathFiltersMatch(filters FlowPathFilters, ctx FlowSelectContext) bool {
	if len(filters.Countries) > 0 {
		if ctx.Country[0] == 0 || !flowCountryAllowed(ctx.Country, filters.Countries) {
			return false
		}
	}
	if filters.Devices != 0 {
		if ctx.DeviceMask == 0 || (filters.Devices&ctx.DeviceMask) == 0 {
			return false
		}
	}
	if filters.OSMask != 0 {
		if ctx.OSMask == 0 || (filters.OSMask&ctx.OSMask) == 0 {
			return false
		}
	}
	if len(filters.Languages) > 0 {
		if ctx.Language[0] == 0 || !flowLanguageAllowed(ctx.Language, filters.Languages) {
			return false
		}
	}
	return true
}

func flowCountryAllowed(country [2]byte, allowed [][2]byte) bool {
	for i := range allowed {
		if allowed[i] == country {
			return true
		}
	}
	return false
}

func flowLanguageAllowed(lang [2]byte, allowed [][2]byte) bool {
	for i := range allowed {
		if allowed[i] == lang {
			return true
		}
	}
	return false
}

func flowSelectContextFromEvent(evt *domain.Event) FlowSelectContext {
	if evt == nil {
		return FlowSelectContext{}
	}
	return FlowSelectContext{
		Country:    flowCountryBytes(evt.GeoCountry),
		DeviceMask: flowDeviceMaskFromUA(evt.UA),
		OSMask:     flowOSMaskFromUA(evt.UA),
		Language:   flowLanguageBytes(evt.AcceptLang),
	}
}

func flowCountryBytes(country string) [2]byte {
	country = strings.ToUpper(strings.TrimSpace(country))
	if len(country) != 2 {
		return [2]byte{}
	}
	return [2]byte{country[0], country[1]}
}

func flowLanguageBytes(acceptLang string) [2]byte {
	acceptLang = strings.TrimSpace(acceptLang)
	if acceptLang == "" {
		return [2]byte{}
	}
	if i := strings.IndexByte(acceptLang, ','); i >= 0 {
		acceptLang = acceptLang[:i]
	}
	if i := strings.IndexByte(acceptLang, '-'); i >= 0 {
		acceptLang = acceptLang[:i]
	}
	if i := strings.IndexByte(acceptLang, ';'); i >= 0 {
		acceptLang = acceptLang[:i]
	}
	acceptLang = strings.ToLower(strings.TrimSpace(acceptLang))
	if len(acceptLang) != 2 {
		return [2]byte{}
	}
	return [2]byte{acceptLang[0], acceptLang[1]}
}

func flowDeviceMaskFromUA(ua string) uint8 {
	uaLower := strings.ToLower(ua)
	if uaLower == "" {
		return 0
	}
	if strings.Contains(uaLower, "ipad") || strings.Contains(uaLower, "tablet") {
		return flowDeviceTablet
	}
	if strings.Contains(uaLower, "mobile") || strings.Contains(uaLower, "iphone") || strings.Contains(uaLower, "android") {
		return flowDeviceMobile
	}
	return flowDeviceDesktop
}

func flowOSMaskFromUA(ua string) uint8 {
	uaLower := strings.ToLower(ua)
	if uaLower == "" {
		return 0
	}
	switch {
	case strings.Contains(uaLower, "android"):
		return flowOSAndroid
	case strings.Contains(uaLower, "iphone"), strings.Contains(uaLower, "ipad"), strings.Contains(uaLower, "ios"):
		return flowOSIOS
	case strings.Contains(uaLower, "windows"):
		return flowOSWindows
	case strings.Contains(uaLower, "mac os"), strings.Contains(uaLower, "macintosh"):
		return flowOSMacOS
	case strings.Contains(uaLower, "linux"):
		return flowOSLinux
	default:
		return 0
	}
}

type FlowLanderEntry struct {
	LanderID uuid.UUID
	Weight   int32
	URL      []byte
}

type FlowOfferEntry struct {
	OfferID uuid.UUID
	Weight  int32
	URL     []byte
	Capped  bool
}

type FlowPath struct {
	Weight  int32
	Filters FlowPathFilters
	Landers []FlowLanderEntry
	Offers  []FlowOfferEntry
}

type FlowPathSnapshot struct {
	Paths []FlowPath
}

type FlowRouter struct {
	active atomic.Pointer[FlowPathSnapshot]
}

func NewFlowRouter() *FlowRouter {
	return &FlowRouter{}
}

func (r *FlowRouter) Publish(s *FlowPathSnapshot) {
	r.active.Store(s)
}

func (r *FlowRouter) Ready() bool {
	return r.active.Load() != nil
}

type FlowSelection struct {
	PathIdx   int
	LanderIdx int
	OfferIdx  int
	LanderID  uuid.UUID
	OfferID   uuid.UUID
}

func (r *FlowRouter) Select(userID []byte) (sel FlowSelection, ok bool) {
	snap := r.active.Load()
	sel, _, ok = SelectSnapshot(snap, userID, FlowSelectContext{})
	return sel, ok
}

func BanditSelect(snap *FlowPathSnapshot, userID []byte) (sel FlowSelection, landerURL []byte, ok bool) {
	return SelectSnapshot(snap, userID, FlowSelectContext{})
}

func (r *FlowRouter) BanditSelect(userID []byte) (sel FlowSelection, landerURL []byte, ok bool) {
	return BanditSelect(r.active.Load(), userID)
}

func SelectSnapshot(snap *FlowPathSnapshot, userID []byte, ctx FlowSelectContext) (sel FlowSelection, landerURL []byte, ok bool) {
	if snap == nil || len(snap.Paths) == 0 {
		return FlowSelection{}, nil, false
	}
	pathIdx, path, pathOK := selectWeightedFlowFiltered(snap.Paths, ctx, fnv1a32(userID))
	if !pathOK {
		return FlowSelection{}, nil, false
	}
	landerIdx, lander := selectWeightedLander(path.Landers, fnv1a32Salted(userID, 'l'))
	if landerIdx < 0 || len(lander.URL) == 0 {
		return FlowSelection{}, nil, false
	}
	offerIdx, offer := selectWeightedOffer(path.Offers, fnv1a32Salted(userID, 'o'))
	if offerIdx < 0 {
		return FlowSelection{}, nil, false
	}
	return FlowSelection{
		PathIdx:   pathIdx,
		LanderIdx: landerIdx,
		OfferIdx:  offerIdx,
		LanderID:  lander.LanderID,
		OfferID:   offer.OfferID,
	}, lander.URL, true
}

func selectWeightedFlowFiltered(paths []FlowPath, ctx FlowSelectContext, bucket uint32) (int, FlowPath, bool) {
	var total int32
	for i := range paths {
		if !flowPathFiltersMatch(paths[i].Filters, ctx) {
			continue
		}
		total += paths[i].Weight
	}
	if total <= 0 {
		return -1, FlowPath{}, false
	}
	target := int32(bucket % uint32(total))
	var acc int32
	for i := range paths {
		if !flowPathFiltersMatch(paths[i].Filters, ctx) {
			continue
		}
		acc += paths[i].Weight
		if target < acc {
			return i, paths[i], true
		}
	}
	for i := len(paths) - 1; i >= 0; i-- {
		if flowPathFiltersMatch(paths[i].Filters, ctx) && paths[i].Weight > 0 {
			return i, paths[i], true
		}
	}
	return -1, FlowPath{}, false
}

func selectWeightedLander(landers []FlowLanderEntry, bucket uint32) (int, FlowLanderEntry) {
	if len(landers) == 0 {
		return -1, FlowLanderEntry{}
	}
	if len(landers) == 1 {
		return 0, landers[0]
	}
	var total int32
	for i := range landers {
		total += landers[i].Weight
	}
	if total <= 0 {
		return 0, landers[0]
	}
	target := int32(bucket % uint32(total))
	var acc int32
	for i := range landers {
		acc += landers[i].Weight
		if target < acc {
			return i, landers[i]
		}
	}
	last := len(landers) - 1
	return last, landers[last]
}

func selectWeightedOffer(offers []FlowOfferEntry, bucket uint32) (int, FlowOfferEntry) {
	if len(offers) == 0 {
		return -1, FlowOfferEntry{}
	}
	var total int32
	eligible := 0
	for i := range offers {
		if offers[i].Capped || offers[i].Weight <= 0 {
			continue
		}
		total += offers[i].Weight
		eligible++
	}
	if eligible == 0 {
		return -1, FlowOfferEntry{}
	}
	if eligible == 1 {
		for i := range offers {
			if !offers[i].Capped && offers[i].Weight > 0 {
				return i, offers[i]
			}
		}
	}
	if total <= 0 {
		for i := range offers {
			if !offers[i].Capped && offers[i].Weight > 0 {
				return i, offers[i]
			}
		}
	}
	target := int32(bucket % uint32(total))
	var acc int32
	for i := range offers {
		if offers[i].Capped || offers[i].Weight <= 0 {
			continue
		}
		acc += offers[i].Weight
		if target < acc {
			return i, offers[i]
		}
	}
	for i := len(offers) - 1; i >= 0; i-- {
		if !offers[i].Capped && offers[i].Weight > 0 {
			return i, offers[i]
		}
	}
	return -1, FlowOfferEntry{}
}

func fnv1a32(b []byte) uint32 {
	const (
		offset32 = 2166136261
		prime32  = 16777619
	)
	h := uint32(offset32)
	for _, c := range b {
		h ^= uint32(c)
		h *= prime32
	}
	return h
}

func fnv1a32Salted(userID []byte, salt byte) uint32 {
	const (
		offset32 = 2166136261
		prime32  = 16777619
	)
	h := uint32(offset32)
	for _, c := range userID {
		h ^= uint32(c)
		h *= prime32
	}
	h ^= uint32(salt)
	h *= prime32
	return h
}
