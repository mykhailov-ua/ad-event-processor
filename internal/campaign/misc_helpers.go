package campaign

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/flow"
	"ad-event-processor/internal/integrationschema"
	"ad-event-processor/internal/migrationsource"
	"ad-event-processor/internal/postback"
	"ad-event-processor/internal/reportjob"
	"ad-event-processor/internal/reports"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"gopkg.in/yaml.v3"
)

func computeCampaignAllowedActions(ctx context.Context, status string) ([]string, map[string]string) {
	snap, ok := authz.SnapshotFromContext(ctx)
	if !ok {
		return nil, nil
	}
	actions := make([]string, 0, 8)
	denied := make(map[string]string)

	add := func(action string) {
		actions = append(actions, action)
	}

	if snap.Has(authz.PermCampaignsWrite) || snap.Has(authz.PermCampaignsWriteMask) {
		if status != "ARCHIVED" {
			add("edit_general")
		}
	}
	if snap.Has(authz.PermCampaignsPause) {
		if status == "ACTIVE" {
			add("pause")
		}
		if status == "PAUSED" {
			add("resume")
		}
	}
	if snap.Has(authz.PermCampaignsWrite) {
		add("clone")
		add("edit_fraud")
		add("edit_budget")
		add("export")
	} else {
		switch {
		case snap.Has(authz.PermCampaignsWriteMask):
			denied["edit_fraud"] = "requires_campaigns_write"
			denied["edit_budget"] = "requires_campaigns_write"
			denied["clone"] = "requires_campaigns_write"
		default:
			denied["edit_general"] = "requires_campaigns_write"
			denied["edit_fraud"] = "requires_campaigns_write"
			denied["edit_budget"] = "requires_campaigns_write"
			denied["clone"] = "requires_campaigns_write"
			denied["export"] = "requires_campaigns_read"
		}
	}
	if !snap.Has(authz.PermCampaignsRead) && !snap.Has(authz.PermCampaignsReadMasked) {
		return nil, denied
	}
	if snap.Has(authz.PermCampaignsRead) || snap.Has(authz.PermCampaignsReadMasked) {
		if !containsString(actions, "export") && snap.Has(authz.PermCampaignsRead) {
			add("export")
		}
	}
	return actions, denied
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func AttachCampaignPresentation(ctx context.Context, dto *CampaignDTO) {
	attachCampaignPresentation(ctx, dto)
}

func attachCampaignPresentation(ctx context.Context, dto *CampaignDTO) {
	if dto == nil {
		return
	}
	dto.StatusLabel = campaignStatusLabel(dto.Status)
	dto.StatusTone = campaignStatusTone(dto.Status)
	attachCampaignMoneyDisplay(dto)
	actions, denied := computeCampaignAllowedActions(ctx, dto.Status)
	dto.AllowedActions = actions
	if len(denied) > 0 {
		dto.DeniedReasons = denied
	}
}

func MaskLevelFromContext(ctx context.Context) authz.MaskLevel {
	return maskLevelFromContext(ctx)
}

func maskLevelFromContext(ctx context.Context) authz.MaskLevel {
	snap, ok := authz.SnapshotFromContext(ctx)
	if !ok {
		return authz.MaskMasked
	}
	return snap.Mask
}

type CampaignEventTimelineItemDTO struct {
	TitleLabel        string `json:"title_label"`
	ActorLabel        string `json:"actor_label"`
	ChangeSummary     string `json:"change_summary"`
	SectionID         string `json:"section_id,omitempty"`
	OccurredAtDisplay string `json:"occurred_at_display"`
}

type CampaignEventTimelineDayDTO struct {
	Day    string                         `json:"day"`
	Events []CampaignEventTimelineItemDTO `json:"events"`
}

type CampaignEventTimelineResponseDTO struct {
	Days []CampaignEventTimelineDayDTO `json:"days"`
}

func buildCampaignEventTimeline(items []CampaignEventDTO, masked bool) CampaignEventTimelineResponseDTO {
	if len(items) == 0 {
		return CampaignEventTimelineResponseDTO{Days: nil}
	}
	byDay := make(map[string][]CampaignEventTimelineItemDTO)
	for _, item := range items {
		day := timelineDayKey(item.CreatedAt)
		byDay[day] = append(byDay[day], campaignEventTimelineItem(item, masked))
	}
	days := make([]CampaignEventTimelineDayDTO, 0, len(byDay))
	for day, events := range byDay {
		days = append(days, CampaignEventTimelineDayDTO{Day: day, Events: events})
	}
	sortTimelineDays(days)
	return CampaignEventTimelineResponseDTO{Days: days}
}

func timelineDayKey(createdAt string) string {
	createdAt = strings.TrimSpace(createdAt)
	if createdAt == "" {
		return "unknown"
	}
	if ts, err := time.Parse(time.RFC3339, createdAt); err == nil {
		return ts.UTC().Format("2006-01-02")
	}
	if len(createdAt) >= 10 {
		return createdAt[:10]
	}
	return createdAt
}

func campaignEventTimelineItem(item CampaignEventDTO, masked bool) CampaignEventTimelineItemDTO {
	title := eventTypeTitleLabel(item.EventType)
	summary := eventTypeSummary(item)
	actor := strings.TrimSpace(item.UserID)
	if masked && actor != "" {
		actor = maskActorLabel(actor)
	}
	display := createdAtDisplay(item.CreatedAt)
	return CampaignEventTimelineItemDTO{
		TitleLabel:        title,
		ActorLabel:        actor,
		ChangeSummary:     summary,
		SectionID:         eventTypeSectionID(item.EventType),
		OccurredAtDisplay: display,
	}
}

func eventTypeTitleLabel(eventType string) string {
	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case "click":
		return "Click recorded"
	case "conversion":
		return "Conversion recorded"
	case "impression":
		return "Impression recorded"
	default:
		if eventType == "" {
			return "Event recorded"
		}
		return strings.ToUpper(eventType[:1]) + eventType[1:] + " recorded"
	}
}

func eventTypeSummary(item CampaignEventDTO) string {
	if len(item.Payload) == 0 {
		return item.EventType
	}
	var raw map[string]any
	if err := json.Unmarshal(item.Payload, &raw); err != nil {
		return item.EventType
	}
	if placement, ok := raw["placement_id"].(string); ok && placement != "" {
		return "placement " + placement
	}
	return item.EventType
}

func eventTypeSectionID(eventType string) string {
	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case "click", "impression":
		return "tracking"
	case "conversion":
		return "postbacks"
	default:
		return "integrations"
	}
}

func createdAtDisplay(createdAt string) string {
	ts, err := time.Parse(time.RFC3339, strings.TrimSpace(createdAt))
	if err != nil {
		return createdAt
	}
	return ts.UTC().Format("2006-01-02 15:04 UTC")
}

func maskActorLabel(actor string) string {
	if len(actor) <= 4 {
		return "[masked]"
	}
	return actor[:2] + "***" + actor[len(actor)-2:]
}

func sortTimelineDays(days []CampaignEventTimelineDayDTO) {
	for i := range len(days) {
		for j := i + 1; j < len(days); j++ {
			if days[j].Day > days[i].Day {
				days[i], days[j] = days[j], days[i]
			}
		}
	}
}

const campaignGeoExpandMaxRows = 50

type CampaignGeoCountryRowDTO struct {
	Code  string `json:"code"`
	Label string `json:"label"`
	Kind  string `json:"kind"`
}

type CampaignGeoSummaryDTO struct {
	IncludedLabel   string                     `json:"included_label"`
	ExcludedLabel   string                     `json:"excluded_label"`
	ConflictWarning bool                       `json:"conflict_warning,omitempty"`
	Expanded        []CampaignGeoCountryRowDTO `json:"expanded,omitempty"`
	Truncated       bool                       `json:"truncated,omitempty"`
}

var isoCountryLabels = map[string]string{
	"US": "United States",
	"GB": "United Kingdom",
	"DE": "Germany",
	"FR": "France",
	"CA": "Canada",
	"AU": "Australia",
}

func BuildCampaignGeoSummary(camp CampaignDTO, expand bool) CampaignGeoSummaryDTO {
	return buildCampaignGeoSummary(camp, expand)
}

func buildCampaignGeoSummary(campaign CampaignDTO, expand bool) CampaignGeoSummaryDTO {
	include := normalizeCountryCodes(campaign.TargetCountries)
	out := CampaignGeoSummaryDTO{
		IncludedLabel: geoListLabel(include, "any country"),
		ExcludedLabel: "none",
	}
	if !expand {
		return out
	}
	rows := make([]CampaignGeoCountryRowDTO, 0, len(include))
	for _, code := range include {
		rows = append(rows, CampaignGeoCountryRowDTO{Code: code, Label: countryLabel(code), Kind: "include"})
	}
	if len(rows) > campaignGeoExpandMaxRows {
		out.Truncated = true
		rows = rows[:campaignGeoExpandMaxRows]
	}
	out.Expanded = rows
	return out
}

func normalizeCountryCodes(codes []string) []string {
	out := make([]string, 0, len(codes))
	seen := make(map[string]struct{}, len(codes))
	for _, raw := range codes {
		code := strings.ToUpper(strings.TrimSpace(raw))
		if code == "" {
			continue
		}
		if _, dup := seen[code]; dup {
			continue
		}
		seen[code] = struct{}{}
		out = append(out, code)
	}
	return out
}

func geoListLabel(codes []string, emptyLabel string) string {
	if len(codes) == 0 {
		return emptyLabel
	}
	if len(codes) == 1 {
		return countryLabel(codes[0])
	}
	return fmt.Sprintf("%d countries", len(codes))
}

func countryLabel(code string) string {
	if label, ok := isoCountryLabels[code]; ok {
		return label
	}
	return code
}

type campaignLifecycleOutboxPayload struct {
	CampaignID  string `json:"campaign_id"`
	BudgetLimit int64  `json:"budget_limit,omitempty"`
}

func EmitCampaignLifecycleOutbox(ctx context.Context, q db.Querier, campaignID uuid.UUID, status db.CampaignStatusType, budgetLimit int64) error {
	switch status {
	case db.CampaignStatusTypeACTIVE:
		payload, err := coldpath.MarshalOutbox(campaignLifecycleOutboxPayload{CampaignID: campaignID.String(), BudgetLimit: budgetLimit})
		if err != nil {
			return fmt.Errorf("marshal create campaign outbox payload: %w", err)
		}
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{EventType: "CREATE_CAMPAIGN", Payload: payload})
		return err
	case db.CampaignStatusTypePAUSED:
		payload, err := coldpath.MarshalOutbox(campaignLifecycleOutboxPayload{CampaignID: campaignID.String()})
		if err != nil {
			return fmt.Errorf("marshal pause campaign outbox payload: %w", err)
		}
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{EventType: "PAUSE_CAMPAIGN", Payload: payload})
		return err
	default:
		return nil
	}
}

func TransitionCampaignStatus(ctx context.Context, fx Effects, q db.Querier, campaignID uuid.UUID, old, newStatus db.CampaignStatusType, reason string, budget int64) error {
	_, err := q.UpdateCampaignStatus(ctx, db.UpdateCampaignStatusParams{
		ID:     domain.ToUUID(campaignID),
		Status: newStatus,
	})
	if err != nil {
		return err
	}
	err = q.CreateStatusHistory(ctx, db.CreateStatusHistoryParams{
		CampaignID: domain.ToUUID(campaignID),
		OldStatus:  db.NullCampaignStatusType{CampaignStatusType: old, Valid: true},
		NewStatus:  newStatus,
		Reason:     pgtype.Text{String: reason, Valid: true},
	})
	if err != nil {
		return err
	}
	if fx != nil {
		return fx.EmitCampaignLifecycleOutbox(ctx, q, campaignID, newStatus, budget)
	}
	return EmitCampaignLifecycleOutbox(ctx, q, campaignID, newStatus, budget)
}

type ListSortDTO struct {
	Field string `json:"field"`
	Order string `json:"order"`
}

type ListEnvelope[T any] struct {
	Items          []T               `json:"items"`
	Total          int64             `json:"total"`
	Limit          int32             `json:"limit"`
	Offset         int32             `json:"offset"`
	Freshness      DataFreshnessDTO  `json:"freshness,omitempty"`
	FiltersApplied map[string]string `json:"filters_applied,omitempty"`
	Sort           *ListSortDTO      `json:"sort,omitempty"`
}

type AssignCampaignOwnerRequest struct {
	UserID string `json:"user_id"`
}

func parseListSort(r *http.Request, allowed map[string]struct{}, defaultField string) (field, order string, err error) {
	field = strings.TrimSpace(r.URL.Query().Get("sort"))
	if field == "" {
		field = defaultField
	}
	if _, ok := allowed[field]; !ok {
		return "", "", invalidQueryError("invalid sort")
	}
	order = strings.ToLower(strings.TrimSpace(r.URL.Query().Get("order")))
	if order == "" {
		order = "desc"
	}
	if order != "asc" && order != "desc" {
		return "", "", invalidQueryError("invalid order")
	}
	return field, order, nil
}

func filtersAppliedFromQuery(r *http.Request, keys ...string) map[string]string {
	out := make(map[string]string, len(keys))
	q := r.URL.Query()
	for _, key := range keys {
		if v := strings.TrimSpace(q.Get(key)); v != "" {
			out[key] = v
		}
	}
	return out
}

func invalidQueryError(msg string) error {
	return fmt.Errorf("%w: %s", ErrInvalidQuery, msg)
}

var ErrInvalidQuery = errors.New("invalid query")

const PacingLookbackDays = 90

func UniformHourWeights() [24]float64 {
	var weights [24]float64
	for i := range weights {
		weights[i] = 1.0 / 24.0
	}
	return weights
}

func SmartPacingExpectedRatio(weights [24]float64, daypart []int16, localNow time.Time) float64 {
	daypartSet := make(map[int16]struct{}, len(daypart))
	for _, h := range daypart {
		daypartSet[h] = struct{}{}
	}
	useDaypart := len(daypartSet) > 0

	currentHour := localNow.Hour()
	minuteFrac := (float64(localNow.Minute()) + float64(localNow.Second())/60.0) / 60.0

	var totalWeight, elapsedWeight float64
	for h := range 24 {
		if useDaypart {
			if _, ok := daypartSet[int16(h)]; !ok {
				continue
			}
		}
		w := weights[h]
		if w <= 0 {
			w = 1.0 / 24.0
		}
		totalWeight += w
		switch {
		case h < currentHour:
			elapsedWeight += w
		case h == currentHour:
			elapsedWeight += w * minuteFrac
		}
	}
	if totalWeight <= 0 {
		startOfDay := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, localNow.Location())
		elapsed := localNow.Sub(startOfDay).Seconds()
		if elapsed < 0 {
			elapsed = 0
		}
		ratio := elapsed / 86400.0
		if ratio > 1.0 {
			ratio = 1.0
		}
		return ratio
	}
	ratio := elapsedWeight / totalWeight
	if ratio > 1.0 {
		ratio = 1.0
	}
	if ratio < 0 {
		ratio = 0
	}
	return ratio
}

type PauseCampaignWouldChange struct {
	CampaignID  string `json:"campaign_id"`
	Status      string `json:"status,omitempty"`
	Noop        bool   `json:"noop,omitempty"`
	StatusFrom  string `json:"status_from,omitempty"`
	StatusTo    string `json:"status_to,omitempty"`
	OutboxEvent string `json:"outbox_event,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

type ResumeCampaignWouldChange struct {
	CampaignID  string `json:"campaign_id"`
	StatusFrom  string `json:"status_from"`
	StatusTo    string `json:"status_to"`
	OutboxEvent string `json:"outbox_event"`
	Reason      string `json:"reason"`
}

func NewMutationPreview(action string, change any) (MutationPreviewDTO, error) {
	raw, err := json.Marshal(change)
	if err != nil {
		return MutationPreviewDTO{}, err
	}
	return MutationPreviewDTO{DryRun: true, Action: action, WouldChange: raw}, nil
}

func PreviewPauseCampaign(ctx context.Context, pool *pgxpool.Pool, campaignID uuid.UUID, reason string) (MutationPreviewDTO, error) {
	if pool == nil {
		return MutationPreviewDTO{}, errServiceUnavailable()
	}
	camp, err := db.New(pool).GetCampaign(ctx, domain.ToUUID(campaignID))
	if err != nil {
		return MutationPreviewDTO{}, mapCampaignStoreError(err)
	}
	if camp.Status == db.CampaignStatusTypePAUSED {
		return NewMutationPreview("PAUSE_CAMPAIGN", PauseCampaignWouldChange{
			CampaignID: campaignID.String(),
			Status:     string(camp.Status),
			Noop:       true,
		})
	}
	if camp.Status != db.CampaignStatusTypeACTIVE {
		return MutationPreviewDTO{}, fmt.Errorf("%w in status %s", ErrCampaignCannotBePaused, camp.Status)
	}
	return NewMutationPreview("PAUSE_CAMPAIGN", PauseCampaignWouldChange{
		CampaignID:  campaignID.String(),
		StatusFrom:  string(camp.Status),
		StatusTo:    string(db.CampaignStatusTypePAUSED),
		OutboxEvent: "PAUSE_CAMPAIGN",
		Reason:      reason,
	})
}

func PreviewResumeCampaign(ctx context.Context, pool *pgxpool.Pool, fx Effects, campaignID uuid.UUID, reason string) (MutationPreviewDTO, error) {
	if pool == nil {
		return MutationPreviewDTO{}, errServiceUnavailable()
	}
	camp, err := db.New(pool).GetCampaign(ctx, domain.ToUUID(campaignID))
	if err != nil {
		return MutationPreviewDTO{}, mapCampaignStoreError(err)
	}
	if camp.Status != db.CampaignStatusTypePAUSED {
		return MutationPreviewDTO{}, ErrCampaignNotPaused
	}
	now := time.Now()
	var startAt, endAt *time.Time
	if camp.StartAt.Valid {
		startAt = &camp.StartAt.Time
	}
	if camp.EndAt.Valid {
		endAt = &camp.EndAt.Time
	}
	if ResolveScheduleStatus(now, startAt, endAt) != db.CampaignStatusTypeACTIVE {
		return MutationPreviewDTO{}, ErrCampaignOutsideSchedule
	}
	if fx != nil {
		if err := fx.EnforceCampaignPublishGate(ctx, campaignID, camp, false); err != nil {
			return MutationPreviewDTO{}, err
		}
	}
	return NewMutationPreview("RESUME_CAMPAIGN", ResumeCampaignWouldChange{
		CampaignID:  campaignID.String(),
		StatusFrom:  string(camp.Status),
		StatusTo:    string(db.CampaignStatusTypeACTIVE),
		OutboxEvent: "RESUME_CAMPAIGN",
		Reason:      reason,
	})
}

func CampaignOwnerUserFilter(ctx context.Context) pgtype.UUID {
	u, ok := authz.GetUser(ctx)
	if !ok || u.UserID == uuid.Nil {
		return pgtype.UUID{}
	}
	if authz.NormalizeRole(u.Role) == authz.RoleMediaBuyer {
		return domain.ToUUID(u.UserID)
	}
	return pgtype.UUID{}
}

func mediaBuyerOwnsCampaign(u authz.AuthenticatedUser, camp db.Campaign) bool {
	if authz.NormalizeRole(u.Role) != authz.RoleMediaBuyer {
		return true
	}
	if !camp.OwnerUserID.Valid {
		return false
	}
	return uuid.UUID(camp.OwnerUserID.Bytes) == u.UserID
}

func AssertMediaBuyerCampaignAccess(ctx context.Context, camp db.Campaign) error {
	u, ok := authz.GetUser(ctx)
	if !ok {
		return nil
	}
	if !mediaBuyerOwnsCampaign(u, camp) {
		return ErrForbidden
	}
	return nil
}

func campaignOwnerUserFilter(ctx context.Context) pgtype.UUID {
	return CampaignOwnerUserFilter(ctx)
}

func assertMediaBuyerCampaignAccess(ctx context.Context, camp db.Campaign) error {
	return AssertMediaBuyerCampaignAccess(ctx, camp)
}

type TemplateCatalogHost interface {
	TrackingDomain(ctx context.Context, override string) string
	PublishCampaignUpdate(ctx context.Context, campaignID string)
	PostbackEncryptionKey() []byte
}

type TemplateCatalog struct {
	pool *pgxpool.Pool
	host TemplateCatalogHost
}

func NewTemplateCatalog(pool *pgxpool.Pool, host TemplateCatalogHost) *TemplateCatalog {
	return &TemplateCatalog{pool: pool, host: host}
}

func (tc *TemplateCatalog) ListBundledTemplates(_ context.Context) []integrationschema.TemplateCatalogEntry {
	return integrationschema.BundledIntegrationTemplateCatalog
}

func (tc *TemplateCatalog) ImportBundledTemplates(ctx context.Context, names []string) ([]IntegrationSchemaDTO, error) {
	if tc == nil || tc.pool == nil {
		return nil, fmt.Errorf("service unavailable")
	}
	wantAll := len(names) == 0
	want := make(map[string]struct{}, len(names))
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n != "" {
			want[n] = struct{}{}
		}
	}
	var out []IntegrationSchemaDTO
	for _, entry := range integrationschema.BundledIntegrationTemplateCatalog {
		if !wantAll {
			if _, ok := want[entry.Name]; !ok {
				continue
			}
		}
		dto, err := tc.importBundledTemplate(ctx, entry)
		if err != nil {
			return nil, err
		}
		out = append(out, dto)
	}
	return out, nil
}

func (tc *TemplateCatalog) ImportCatalogEntry(ctx context.Context, entry integrationschema.TemplateCatalogEntry) (IntegrationSchemaDTO, error) {
	return tc.importBundledTemplate(ctx, entry)
}

func (tc *TemplateCatalog) importBundledTemplate(ctx context.Context, entry integrationschema.TemplateCatalogEntry) (IntegrationSchemaDTO, error) {
	raw, kind, parsed, err := integrationschema.LoadBundledTemplate(entry)
	if err != nil {
		return IntegrationSchemaDTO{}, err
	}
	jsonBody, err := json.Marshal(parsed)
	if err != nil {
		return IntegrationSchemaDTO{}, err
	}
	_ = raw
	var id uuid.UUID
	err = tc.pool.QueryRow(ctx, `
		INSERT INTO integration_schemas (name, version, kind, body)
		VALUES ($1, $2, $3, $4::jsonb)
		ON CONFLICT (name, version) DO UPDATE
		SET kind = EXCLUDED.kind, body = EXCLUDED.body, updated_at = NOW()
		RETURNING id`,
		entry.Name, entry.Version, string(kind), jsonBody,
	).Scan(&id)
	if err != nil {
		return IntegrationSchemaDTO{}, err
	}
	return tc.getIntegrationSchemaDTO(ctx, id)
}

func (tc *TemplateCatalog) getIntegrationSchemaDTO(ctx context.Context, id uuid.UUID) (IntegrationSchemaDTO, error) {
	var dto IntegrationSchemaDTO
	var kind string
	var body []byte
	var created, updated time.Time
	err := tc.pool.QueryRow(ctx, `
		SELECT id, name, version, kind, body, created_at, updated_at
		FROM integration_schemas WHERE id = $1`, id).Scan(
		&id, &dto.Name, &dto.Version, &kind, &body, &created, &updated,
	)
	if err != nil {
		return IntegrationSchemaDTO{}, err
	}
	dto.ID = id.String()
	dto.Kind = kind
	dto.Schema = json.RawMessage(body)
	dto.CreatedAt = created
	dto.UpdatedAt = updated
	return dto, nil
}

func (tc *TemplateCatalog) resolveSchemaIDByName(ctx context.Context, name string) (uuid.UUID, error) {
	var id uuid.UUID
	err := tc.pool.QueryRow(ctx, `
		SELECT id FROM integration_schemas WHERE name = $1 ORDER BY version DESC LIMIT 1`, name,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, fmt.Errorf("schema %q not imported", name)
		}
		return uuid.Nil, err
	}
	return id, nil
}

func (tc *TemplateCatalog) ApplyCampaignTemplates(ctx context.Context, campaignID uuid.UUID, req ApplyCampaignTemplatesRequest) (ApplyCampaignTemplatesResult, error) {
	if tc == nil || tc.pool == nil {
		return ApplyCampaignTemplatesResult{}, fmt.Errorf("service unavailable")
	}
	if campaignID == uuid.Nil {
		return ApplyCampaignTemplatesResult{}, fmt.Errorf("campaign id required")
	}
	result := ApplyCampaignTemplatesResult{CampaignID: campaignID.String()}

	trackingDomain := strings.TrimSpace(req.TrackingDomain)
	if trackingDomain == "" && tc.host != nil {
		trackingDomain = tc.host.TrackingDomain(ctx, "")
	}

	if src := strings.TrimSpace(req.TrafficSource); src != "" {
		schemaID, err := tc.resolveSchemaIDByName(ctx, src)
		if err != nil {
			return result, err
		}
		applied, err := tc.applyIntegrationSchema(ctx, campaignID, schemaID, trackingDomain)
		if err != nil {
			return result, err
		}
		result.TrafficSource = applied
	}

	if net := strings.TrimSpace(req.AffiliateNetwork); net != "" {
		outID, err := tc.resolveSchemaIDByName(ctx, net)
		if err != nil {
			return result, err
		}
		applied, err := tc.applyIntegrationSchema(ctx, campaignID, outID, trackingDomain)
		if err != nil {
			return result, err
		}
		result.AffiliatePostback = applied
		if statusName, ok := integrationschema.AffiliateStatusTemplateName(net); ok {
			statusID, err := tc.resolveSchemaIDByName(ctx, statusName)
			if err != nil {
				return result, err
			}
			statusApplied, err := tc.applyIntegrationSchema(ctx, campaignID, statusID, trackingDomain)
			if err != nil {
				return result, err
			}
			result.AffiliateStatus = statusApplied
		}
	}

	return result, nil
}

func (tc *TemplateCatalog) applyIntegrationSchema(ctx context.Context, campaignID, schemaID uuid.UUID, trackingDomain string) (map[string]string, error) {
	var kind string
	var schemaBody []byte
	err := tc.pool.QueryRow(ctx, `SELECT kind, body FROM integration_schemas WHERE id = $1`, schemaID).Scan(&kind, &schemaBody)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("schema not found")
		}
		return nil, err
	}

	q := db.New(tc.pool)
	if _, err := q.GetCampaign(ctx, pgtype.UUID{Bytes: campaignID, Valid: true}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("campaign not found")
		}
		return nil, err
	}

	tx, err := tc.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	applied := map[string]string{"schema_id": schemaID.String(), "kind": kind}
	key := []byte("postback-encryption-secret-key32")
	if tc.host != nil {
		if k := tc.host.PostbackEncryptionKey(); len(k) > 0 {
			key = k
		}
	}
	switch integrationschema.Kind(kind) {
	case integrationschema.KindInboundTokens:
		parsedKind, parsed, err := integrationschema.ParseDocument(schemaBody)
		if err != nil || parsedKind != integrationschema.KindInboundTokens {
			return nil, fmt.Errorf("invalid inbound schema")
		}
		inbound := parsed.(*integrationschema.InboundTokensSchema)
		trackingURL := integrationschema.BuildInboundTrackingURL(trackingDomain, inbound)
		if _, err := tx.Exec(ctx, `
			UPDATE campaigns
			SET integration_schema_id = $2, target_url = $3, updated_at = NOW()
			WHERE id = $1`, campaignID, schemaID, trackingURL); err != nil {
			return nil, err
		}
		applied["target_url"] = trackingURL
	case integrationschema.KindOutboundPostback:
		tpl, err := integrationschema.OutboundURLTemplateFromBody(schemaBody)
		if err != nil {
			return nil, err
		}
		encToken, err := postback.EncryptAESGCM([]byte("integration-schema"), key)
		if err != nil {
			return nil, fmt.Errorf("encryption failed")
		}
		txQ := db.New(tx)
		if err := txQ.UpsertPostbackConfig(ctx, db.UpsertPostbackConfigParams{
			CampaignID:        pgtype.UUID{Bytes: campaignID, Valid: true},
			Provider:          "webhook",
			UrlTemplate:       tpl,
			ApiTokenEncrypted: encToken,
			TargetEvent:       "conversion",
		}); err != nil {
			return nil, err
		}
		applied["url_template"] = tpl
	case integrationschema.KindAffiliateReceivePostback:
		parsedKind, parsed, err := integrationschema.ParseDocument(schemaBody)
		if err != nil || parsedKind != integrationschema.KindAffiliateReceivePostback {
			return nil, fmt.Errorf("invalid affiliate receive schema")
		}
		recv := parsed.(*integrationschema.AffiliateReceivePostbackSchema)
		panelURL := integrationschema.BuildAffiliateReceivePanelURL(trackingDomain, recv)
		applied["panel_postback_url"] = panelURL
		if suffix := strings.TrimSpace(recv.OfferURLSuffix); suffix != "" {
			applied["offer_url_suffix"] = suffix
		}
	case integrationschema.KindStatusMapping:
		if _, err := tx.Exec(ctx, `
			UPDATE campaigns SET status_integration_schema_id = $2, updated_at = NOW() WHERE id = $1`,
			campaignID, schemaID); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported schema kind %q", kind)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	if tc.host != nil {
		tc.host.PublishCampaignUpdate(ctx, campaignID.String())
	}
	return applied, nil
}

func CampaignLocation(cache *sync.Map, timezone string) *time.Location {
	if cache != nil {
		if cached, found := cache.Load(timezone); found {
			return cached.(*time.Location)
		}
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	if cache != nil {
		cache.Store(timezone, loc)
	}
	return loc
}

func validateDaypartHours(hours []int16) error {
	for _, h := range hours {
		if h < 0 || h > 23 {
			return fmt.Errorf("daypart hour must be 0-23, got %d", h)
		}
	}
	return nil
}

func validateSchedule(startAt, endAt *time.Time) error {
	if startAt != nil && endAt != nil && !endAt.After(*startAt) {
		return fmt.Errorf("end_at must be after start_at")
	}
	return nil
}

func CountriesOrEmpty(c []string) []string {
	return countriesOrEmpty(c)
}

func countriesOrEmpty(c []string) []string {
	if c == nil {
		return []string{}
	}
	return c
}

func ResolveScheduleStatus(now time.Time, startAt, endAt *time.Time) db.CampaignStatusType {
	if startAt != nil && now.Before(*startAt) {
		return db.CampaignStatusTypePAUSED
	}
	if endAt != nil && !now.Before(*endAt) {
		return db.CampaignStatusTypePAUSED
	}
	return db.CampaignStatusTypeACTIVE
}

func resolveScheduleStatus(now time.Time, startAt, endAt *time.Time) db.CampaignStatusType {
	return ResolveScheduleStatus(now, startAt, endAt)
}

func ValidateDaypartHours(hours []int16) error {
	return validateDaypartHours(hours)
}

func ValidateSchedule(startAt, endAt *time.Time) error {
	return validateSchedule(startAt, endAt)
}

func DaypartOrEmpty(hours []int16) []int16 {
	if hours == nil {
		return []int16{}
	}
	return hours
}

type OnboardingFlowDefault struct {
	FlowName string                 `json:"flow_name"`
	Lander   CampaignWizardAssetRef `json:"lander"`
	Offer    CampaignWizardAssetRef `json:"offer"`
}

type OnboardingTemplate struct {
	Key                   string                `json:"key"`
	Title                 string                `json:"title"`
	Description           string                `json:"description"`
	TrafficFamily         string                `json:"traffic_family"`
	DefaultFlow           OnboardingFlowDefault `json:"default_flow"`
	IntegrationSchemaRefs []string              `json:"integration_schema_refs"`
	SampleMacros          map[string]string     `json:"sample_macros"`
}

type onboardingTemplateWizardYAML struct {
	TrafficTemplateID string            `yaml:"traffic_template_id"`
	IntegrationSchema string            `yaml:"integration_schema"`
	ClickQueryParams  map[string]string `yaml:"click_query_params"`
	CampaignName      string            `yaml:"campaign_name"`
	FlowName          string            `yaml:"flow_name"`
	LanderName        string            `yaml:"lander_name"`
	LanderURL         string            `yaml:"lander_url"`
	OfferName         string            `yaml:"offer_name"`
	OfferURL          string            `yaml:"offer_url"`
	BudgetLimitMicro  int64             `yaml:"budget_limit_micro"`
	Timezone          string            `yaml:"timezone"`
	TargetCountries   []string          `yaml:"target_countries"`
}

type onboardingTemplateYAML struct {
	Key                   string                       `yaml:"key"`
	Title                 string                       `yaml:"title"`
	Description           string                       `yaml:"description"`
	TrafficFamily         string                       `yaml:"traffic_family"`
	IntegrationSchemaRefs []string                     `yaml:"integration_schema_refs"`
	SampleMacros          map[string]string            `yaml:"sample_macros"`
	Wizard                onboardingTemplateWizardYAML `yaml:"wizard"`
}

type onboardingCatalogYAML struct {
	Version   int                      `yaml:"version"`
	Templates []onboardingTemplateYAML `yaml:"templates"`
}

var (
	onboardingCatalogOnce sync.Once
	onboardingCatalogErr  error
	onboardingCatalog     []onboardingTemplateDef
)

type onboardingTemplateDef struct {
	OnboardingTemplate
	wizard onboardingTemplateWizardYAML
}

func ListOnboardingTemplates() ([]OnboardingTemplate, error) {
	defs, err := loadOnboardingCatalog()
	if err != nil {
		return nil, err
	}
	out := make([]OnboardingTemplate, 0, len(defs))
	for _, def := range defs {
		out = append(out, def.OnboardingTemplate)
	}
	return out, nil
}

func OnboardingTemplateKeys() ([]string, error) {
	defs, err := loadOnboardingCatalog()
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(defs))
	for _, def := range defs {
		out = append(out, def.Key)
	}
	return out, nil
}

func ApplyOnboardingTemplate(key string) (CampaignWizardStored, error) {
	key = strings.TrimSpace(key)
	defs, err := loadOnboardingCatalog()
	if err != nil {
		return CampaignWizardStored{}, err
	}
	keys, _ := OnboardingTemplateKeys()
	for _, def := range defs {
		if def.Key != key {
			continue
		}
		w := def.wizard
		stored := CampaignWizardStored{
			TrafficSource: CampaignWizardTrafficSourceStep{
				Name:              strings.TrimSpace(w.CampaignName),
				TrafficTemplateID: strings.TrimSpace(w.TrafficTemplateID),
				ClickQueryParams:  w.ClickQueryParams,
			},
			IntegrationTemplate: CampaignWizardIntegrationTemplateStep{
				IntegrationSchema: strings.TrimSpace(w.IntegrationSchema),
			},
			FlowSkeleton: CampaignWizardFlowSkeletonStep{
				FlowName: strings.TrimSpace(w.FlowName),
				Lander: CampaignWizardAssetRef{
					Name: strings.TrimSpace(w.LanderName),
					URL:  strings.TrimSpace(w.LanderURL),
				},
				Offer: CampaignWizardAssetRef{
					Name: strings.TrimSpace(w.OfferName),
					URL:  strings.TrimSpace(w.OfferURL),
				},
			},
			Budget: CampaignWizardBudgetStep{
				BudgetLimitMicro: w.BudgetLimitMicro,
				Timezone:         strings.TrimSpace(w.Timezone),
				TargetCountries:  append([]string(nil), w.TargetCountries...),
			},
		}
		if err := validateOnboardingWizardStored(stored); err != nil {
			return CampaignWizardStored{}, err
		}
		return stored, nil
	}
	return CampaignWizardStored{}, errValidation(fmt.Sprintf("unknown template_key %q; valid keys: %s", key, strings.Join(keys, ", ")))
}

func validateOnboardingWizardStored(stored CampaignWizardStored) error {
	if err := ValidateWizardTrafficSourceStep(stored.TrafficSource); err != nil {
		return err
	}
	if err := ValidateWizardIntegrationTemplateStep(stored.IntegrationTemplate); err != nil {
		return err
	}
	if err := ValidateWizardFlowSkeletonStep(stored.FlowSkeleton); err != nil {
		return err
	}
	if err := ValidateWizardBudgetStep(stored.Budget); err != nil {
		return err
	}
	return nil
}

func loadOnboardingCatalog() ([]onboardingTemplateDef, error) {
	onboardingCatalogOnce.Do(func() {
		path, err := resolveOnboardingCatalogPath()
		if err != nil {
			onboardingCatalogErr = err
			return
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			onboardingCatalogErr = fmt.Errorf("read onboarding catalog: %w", err)
			return
		}
		var parsed onboardingCatalogYAML
		if err := yaml.Unmarshal(raw, &parsed); err != nil {
			onboardingCatalogErr = fmt.Errorf("parse onboarding catalog: %w", err)
			return
		}
		if parsed.Version != 1 {
			onboardingCatalogErr = fmt.Errorf("unsupported onboarding catalog version %d", parsed.Version)
			return
		}
		defs := make([]onboardingTemplateDef, 0, len(parsed.Templates))
		for _, row := range parsed.Templates {
			key := strings.TrimSpace(row.Key)
			if key == "" {
				onboardingCatalogErr = fmt.Errorf("onboarding template missing key")
				return
			}
			defs = append(defs, onboardingTemplateDef{
				OnboardingTemplate: OnboardingTemplate{
					Key:           key,
					Title:         strings.TrimSpace(row.Title),
					Description:   strings.TrimSpace(row.Description),
					TrafficFamily: strings.TrimSpace(row.TrafficFamily),
					DefaultFlow: OnboardingFlowDefault{
						FlowName: strings.TrimSpace(row.Wizard.FlowName),
						Lander: CampaignWizardAssetRef{
							Name: strings.TrimSpace(row.Wizard.LanderName),
							URL:  strings.TrimSpace(row.Wizard.LanderURL),
						},
						Offer: CampaignWizardAssetRef{
							Name: strings.TrimSpace(row.Wizard.OfferName),
							URL:  strings.TrimSpace(row.Wizard.OfferURL),
						},
					},
					IntegrationSchemaRefs: append([]string(nil), row.IntegrationSchemaRefs...),
					SampleMacros:          row.SampleMacros,
				},
				wizard: row.Wizard,
			})
		}
		onboardingCatalog = defs
	})
	return onboardingCatalog, onboardingCatalogErr
}

func resolveOnboardingCatalogPath() (string, error) {
	name := filepath.Join("onboarding", "catalog.v1.yaml")
	if root := strings.TrimSpace(os.Getenv("REPO_ROOT")); root != "" {
		candidate := filepath.Join(root, "deploy", "schemas", name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	candidates := []string{
		filepath.Join("deploy", "schemas", name),
		filepath.Join("..", "..", "deploy", "schemas", name),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("onboarding catalog not found")
}

func resetOnboardingCatalogForTest() {
	onboardingCatalogOnce = sync.Once{}
	onboardingCatalogErr = nil
	onboardingCatalog = nil
}

type ingressCostPatchHost interface {
	PublishCampaignUpdate(ctx context.Context, campaignID string)
}

func ApplyIngressCostPatch(
	ctx context.Context,
	host ingressCostPatchHost,
	pool *pgxpool.Pool,
	campaignID uuid.UUID,
	cfg IngressCostConfigDTO,
) error {
	if pool == nil || host == nil {
		return errServiceUnavailable()
	}
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		return applyCampaignIngressCostPatchTx(ctx, db.New(tx), campaignID, cfg)
	})
	if err != nil {
		return err
	}
	host.PublishCampaignUpdate(ctx, campaignID.String())
	return nil
}

type brandAssignAuditChange struct {
	BrandID string `json:"brand_id"`
}

func AssignCampaignBrand(ctx context.Context, fx Effects, pool *pgxpool.Pool, campaignID, brandID uuid.UUID) error {
	if fx == nil || pool == nil {
		return errServiceUnavailable()
	}
	if campaignID == uuid.Nil {
		return errValidation("campaign id required")
	}
	camp, err := fx.GetCampaignRow(ctx, campaignID)
	if err != nil {
		return mapCampaignStoreError(err)
	}
	customerID := uuid.UUID(camp.CustomerID.Bytes)

	brandFcapKey := "fcap:c:" + campaignID.String()
	brandArg := brandIDOrNil(uuid.Nil)
	auditBrandID := ""
	if brandID != uuid.Nil {
		q := db.New(pool)
		brand, err := q.GetBrand(ctx, domain.ToUUID(brandID))
		if err != nil {
			return mapCampaignNotFound(err, ErrBrandNotFound)
		}
		if uuid.UUID(brand.CustomerID.Bytes) != customerID {
			return ErrBrandBelongsToAnotherCustomer
		}
		brandFcapKey = "fcap:b:" + brandID.String()
		brandArg = brandID
		auditBrandID = brandID.String()
	}

	tag, err := pool.Exec(ctx,
		`UPDATE campaigns SET brand_id = $2, brand_fcap_key = $3, updated_at = now() WHERE id = $1 AND deleted_at IS NULL`,
		campaignID, brandArg, brandFcapKey,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrCampaignNotFound
	}

	adminID := uuid.Nil
	if u, ok := authz.GetUser(ctx); ok {
		adminID = u.UserID
	}
	fx.AuditLog(ctx, db.New(pool), adminID, "PATCH_CAMPAIGN", "campaign", &campaignID, brandAssignAuditChange{
		BrandID: auditBrandID,
	}, nil)

	fx.PublishCampaignUpdate(ctx, campaignID.String())
	return nil
}

func brandIDOrNil(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}

type PostbackHTTPHandlers struct {
	Pool              *pgxpool.Pool
	EncryptionKey     []byte
	ApplyRateLimit    func(http.HandlerFunc) http.HandlerFunc
	RequirePermission func(string, http.HandlerFunc) http.HandlerFunc
}

func (h *PostbackHTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil {
		return
	}
	limit := h.ApplyRateLimit
	perm := h.RequirePermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}

	mux.HandleFunc("GET /api/v1/postbacks/config", limit(perm("campaigns:read", h.getPostbacksConfig)))
	mux.HandleFunc("PUT /api/v1/postbacks/config/{campaign_id}", limit(perm("campaigns:write", h.updatePostbackConfig)))
	mux.HandleFunc("GET /api/v1/postbacks/dlq", limit(perm("campaigns:read", h.getDLQ)))
	mux.HandleFunc("POST /api/v1/postbacks/dlq/{id}/retry", limit(perm("campaigns:write", h.retryDLQ)))
	mux.HandleFunc("GET /api/v1/postbacks/campaign-status", limit(perm("campaigns:read", h.getCampaignStatus)))
	mux.HandleFunc("POST /api/v1/postbacks/config/{campaign_id}/test", limit(perm("campaigns:write", h.testPostbackConfig)))
}

type PostbackConfigDTO struct {
	CampaignID    string `json:"campaign_id"`
	Provider      string `json:"provider"`
	URLTemplate   string `json:"url_template"`
	TargetEvent   string `json:"target_event"`
	TestEventCode string `json:"test_event_code,omitempty"`
	HasAPIToken   bool   `json:"has_api_token"`
}

func (h *PostbackHTTPHandlers) getPostbacksConfig(w http.ResponseWriter, r *http.Request) {
	q := db.New(h.Pool)
	configs, err := q.ListPostbackConfigs(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	dtos := make([]PostbackConfigDTO, 0, len(configs))
	for _, c := range configs {
		var campaignIDStr string
		if c.CampaignID.Valid {
			campaignIDStr = ingestionUUIDToString(c.CampaignID)
		}
		dtos = append(dtos, PostbackConfigDTO{
			CampaignID:    campaignIDStr,
			Provider:      c.Provider,
			URLTemplate:   c.UrlTemplate,
			TargetEvent:   c.TargetEvent,
			TestEventCode: c.TestEventCode,
			HasAPIToken:   len(c.ApiTokenEncrypted) > 0,
		})
	}

	httpresponse.JSON(w, http.StatusOK, dtos)
}

type UpdatePostbackConfigRequest struct {
	Provider      string `json:"provider"`
	URLTemplate   string `json:"url_template"`
	APIToken      string `json:"api_token"`
	TargetEvent   string `json:"target_event"`
	TestEventCode string `json:"test_event_code"`
}

func (h *PostbackHTTPHandlers) updatePostbackConfig(w http.ResponseWriter, r *http.Request) {
	campaignIDStr := r.PathValue("campaign_id")
	if campaignIDStr == "" {
		campaignIDStr = r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	}
	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
		return
	}

	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	var req UpdatePostbackConfigRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body")
		return
	}

	if req.Provider == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "provider is required")
		return
	}
	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	switch provider {
	case "webhook", "facebook", "google", "tiktok", "taboola", "outbrain", "microsoft_ads":
	default:
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "unsupported provider")
		return
	}
	if strings.TrimSpace(req.URLTemplate) == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "url_template is required")
		return
	}

	q := db.New(h.Pool)
	_, err = q.GetCampaign(r.Context(), pgtype.UUID{Bytes: campaignID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "campaign not found")
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	var encryptedToken []byte
	existing, existingErr := q.GetPostbackConfig(r.Context(), pgtype.UUID{Bytes: campaignID, Valid: true})
	if existingErr != nil && !errors.Is(existingErr, pgx.ErrNoRows) {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", existingErr.Error())
		return
	}
	if req.APIToken != "" {
		key := h.EncryptionKey
		if len(key) == 0 {
			key = []byte("postback-encryption-secret-key32")
		}
		encryptedToken, err = postback.EncryptAESGCM([]byte(req.APIToken), key)
		if err != nil {
			httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "encryption failed: "+err.Error())
			return
		}
	} else if existingErr == nil {
		encryptedToken = existing.ApiTokenEncrypted
	}
	if postback.ProviderRequiresToken(provider) && len(encryptedToken) == 0 {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "api_token is required for CAPI providers")
		return
	}
	if encryptedToken == nil {
		encryptedToken = []byte{}
	}

	targetEv := "conversion"
	if req.TargetEvent != "" {
		targetEv = req.TargetEvent
	}

	err = q.UpsertPostbackConfig(r.Context(), db.UpsertPostbackConfigParams{
		CampaignID:        pgtype.UUID{Bytes: campaignID, Valid: true},
		Provider:          provider,
		UrlTemplate:       strings.TrimSpace(req.URLTemplate),
		ApiTokenEncrypted: encryptedToken,
		TargetEvent:       targetEv,
		TestEventCode:     strings.TrimSpace(req.TestEventCode),
	})
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	httpresponse.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type PostbackDlqDTO struct {
	ID            int64           `json:"id"`
	OutboxEventID int64           `json:"outbox_event_id"`
	CampaignID    string          `json:"campaign_id"`
	ClickID       string          `json:"click_id"`
	EventType     string          `json:"event_type"`
	Payload       json.RawMessage `json:"payload"`
	FailuresCount int32           `json:"failures_count"`
	LastError     string          `json:"last_error,omitempty"`
	Status        string          `json:"status"`
}

func (h *PostbackHTTPHandlers) getDLQ(w http.ResponseWriter, r *http.Request) {
	q := db.New(h.Pool)
	dlqs, err := q.ListPostbackDLQ(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	dtos := make([]PostbackDlqDTO, 0, len(dlqs))
	for _, d := range dlqs {
		dtos = append(dtos, PostbackDlqDTO{
			ID:            d.ID,
			OutboxEventID: d.OutboxEventID,
			CampaignID:    ingestionUUIDToString(d.CampaignID),
			ClickID:       d.ClickID,
			EventType:     d.EventType,
			Payload:       json.RawMessage(d.Payload),
			FailuresCount: d.FailuresCount,
			LastError:     d.LastError.String,
			Status:        d.Status,
		})
	}

	httpresponse.JSON(w, http.StatusOK, dtos)
}

func (h *PostbackHTTPHandlers) retryDLQ(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.PathValue("id")
	if idStr == "" {
		idStr = r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		if idStr == "retry" {
			parts := strings.Split(r.URL.Path, "/")
			if len(parts) >= 3 {
				idStr = parts[len(parts)-2]
			}
		}
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id")
		return
	}

	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := db.New(tx)
	dlq, err := q.GetPostbackDLQ(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "dlq entry not found")
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	if dlq.Status == "RETRIED" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "already retried")
		return
	}

	_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType: "SEND_POSTBACK",
		Payload:   dlq.Payload,
	})
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	err = q.UpdatePostbackDLQ(ctx, db.UpdatePostbackDLQParams{
		ID:            dlq.ID,
		FailuresCount: dlq.FailuresCount,
		LastError:     pgtype.Text{String: "Manual retry triggered", Valid: true},
		Status:        "RETRIED",
	})
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	if err := tx.Commit(ctx); err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	httpresponse.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type PostbackCampaignStatusDTO struct {
	CampaignID      string     `json:"campaign_id"`
	Provider        string     `json:"provider"`
	LastSuccessAt   *time.Time `json:"last_success_at,omitempty"`
	DLQPendingCount int64      `json:"dlq_pending_count"`
}

func (h *PostbackHTTPHandlers) getCampaignStatus(w http.ResponseWriter, r *http.Request) {
	rows, err := db.New(h.Pool).ListPostbackCampaignStatus(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	out := make([]PostbackCampaignStatusDTO, 0, len(rows))
	for _, row := range rows {
		dto := PostbackCampaignStatusDTO{
			CampaignID:      uuid.UUID(row.CampaignID.Bytes).String(),
			Provider:        row.Provider,
			DLQPendingCount: row.DlqPendingCount,
		}
		if row.LastSuccessAt.Valid {
			t := row.LastSuccessAt.Time
			dto.LastSuccessAt = &t
		}
		out = append(out, dto)
	}
	httpresponse.JSON(w, http.StatusOK, out)
}

func (h *PostbackHTTPHandlers) testPostbackConfig(w http.ResponseWriter, r *http.Request) {
	campaignIDStr := r.PathValue("campaign_id")
	if campaignIDStr == "" {
		campaignIDStr = strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/postbacks/config/"), "/test")
	}
	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
		return
	}

	q := db.New(h.Pool)
	cfg, err := q.GetPostbackConfig(r.Context(), pgtype.UUID{Bytes: campaignID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "postback config not found")
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	key := h.EncryptionKey
	if len(key) == 0 {
		key = []byte("postback-encryption-secret-key32")
	}
	token := ""
	if len(cfg.ApiTokenEncrypted) > 0 {
		plain, decErr := postback.DecryptAESGCM(cfg.ApiTokenEncrypted, key)
		if decErr != nil {
			httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "decrypt failed")
			return
		}
		token = string(plain)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	result := postback.DryRunConfig(ctx, cfg.Provider, cfg.UrlTemplate, token, cfg.TargetEvent, cfg.TestEventCode, campaignID)
	status := http.StatusOK
	if !result.OK {
		status = http.StatusUnprocessableEntity
	}
	httpresponse.JSON(w, status, result)
}

func ingestionUUIDToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	id, err := uuid.FromBytes(u.Bytes[:])
	if err != nil {
		return ""
	}
	return id.String()
}

const maxSubIDs = 30

const (
	previewClickID = "preview-click-id"
	previewUserID  = "preview-user-id"
)

type PreviewRequest struct {
	Sub1    string
	Country string
	ClickID string
	UserID  string
	FBCLID  string
	GCLID   string
	TTCLID  string
}

type Context struct {
	CampaignID string
	ClickID    string
	UserID     string
	Country    string
	Subs       [maxSubIDs]string
	FBCLID     string
	GCLID      string
	TTCLID     string
}

func PreviewContext(campaignID string, req PreviewRequest) Context {
	clickID := strings.TrimSpace(req.ClickID)
	if clickID == "" {
		clickID = previewClickID
	}
	userID := strings.TrimSpace(req.UserID)
	if userID == "" {
		userID = previewUserID
	}
	ctx := Context{
		CampaignID: strings.TrimSpace(campaignID),
		ClickID:    clickID,
		UserID:     userID,
		Country:    strings.TrimSpace(req.Country),
		FBCLID:     strings.TrimSpace(req.FBCLID),
		GCLID:      strings.TrimSpace(req.GCLID),
		TTCLID:     strings.TrimSpace(req.TTCLID),
	}
	if sub1 := strings.TrimSpace(req.Sub1); sub1 != "" {
		ctx.Subs[0] = sub1
	}
	return ctx
}

func Expand(raw string, ctx Context) (string, []string) {
	if raw == "" {
		return "", nil
	}
	var unresolved []string
	out := expandRedirectMacros(raw, ctx, &unresolved)
	out = expandDoubleBrace(out, ctx, &unresolved)
	return out, unresolved
}

func expandDoubleBrace(raw string, ctx Context, unresolved *[]string) string {
	var b strings.Builder
	b.Grow(len(raw))
	i := 0
	for i < len(raw) {
		if i+1 < len(raw) && raw[i] == '{' && raw[i+1] == '{' {
			end := strings.Index(raw[i+2:], "}}")
			if end < 0 {
				b.WriteByte(raw[i])
				i++
				continue
			}
			key := strings.TrimSpace(raw[i+2 : i+2+end])
			token := raw[i : i+2+end+2]
			value, ok := doubleBraceValue(key, ctx)
			if !ok || value == "" {
				*unresolved = append(*unresolved, token)
				b.WriteString(token)
			} else {
				b.WriteString(value)
			}
			i += 2 + end + 2
			continue
		}
		b.WriteByte(raw[i])
		i++
	}
	return b.String()
}

func doubleBraceValue(key string, ctx Context) (string, bool) {
	switch strings.ToLower(key) {
	case "campaign.id", "campaign_id":
		return ctx.CampaignID, true
	case "country":
		return ctx.Country, true
	case "fbclid":
		return ctx.FBCLID, true
	case "gclid":
		return ctx.GCLID, true
	case "ttclid":
		return ctx.TTCLID, true
	case "click_id":
		return ctx.ClickID, true
	case "user_id":
		return ctx.UserID, true
	default:
		if strings.HasPrefix(strings.ToLower(key), "sub") {
			idx, ok := parseSubIndex(key[3:])
			if ok && idx >= 1 && idx <= maxSubIDs {
				return ctx.Subs[idx-1], true
			}
		}
		return "", false
	}
}

func parseSubIndex(raw string) (int, bool) {
	if raw == "" {
		return 0, false
	}
	n := 0
	for i := range len(raw) {
		c := raw[i]
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
		if n > maxSubIDs {
			return 0, false
		}
	}
	if n == 0 {
		return 0, false
	}
	return n, true
}

func expandRedirectMacros(raw string, ctx Context, unresolved *[]string) string {
	var b strings.Builder
	b.Grow(len(raw))
	i := 0
	for i < len(raw) {
		if raw[i] != '{' {
			b.WriteByte(raw[i])
			i++
			continue
		}
		if i+1 < len(raw) && raw[i+1] == '{' {
			b.WriteByte(raw[i])
			i++
			continue
		}
		if i > 0 && raw[i-1] == '{' {
			b.WriteByte(raw[i])
			i++
			continue
		}
		macroID, end := dispatchRedirectMacro(raw, i)
		switch macroID {
		case redirectMacroClickID:
			b.WriteString(ctx.ClickID)
			i = end
		case redirectMacroUserID:
			b.WriteString(ctx.UserID)
			i = end
		default:
			if macroID >= redirectMacroSub1 && macroID < redirectMacroSub1+redirectMacroID(maxSubIDs) {
				sub := ctx.Subs[macroID-redirectMacroSub1]
				if sub == "" {
					*unresolved = append(*unresolved, raw[i:end])
					b.WriteString(raw[i:end])
				} else {
					b.WriteString(sub)
				}
				i = end
				continue
			}
			b.WriteByte(raw[i])
			i++
		}
	}
	return b.String()
}

type redirectMacroID uint8

const (
	redirectMacroNone redirectMacroID = iota
	redirectMacroClickID
	redirectMacroUserID
	redirectMacroSub1
)

const (
	redirectMacroClickLen = 10
	redirectMacroUserLen  = 9
	redirectMacroSubLen   = 6
)

func dispatchRedirectMacro(base string, i int) (redirectMacroID, int) {
	n := len(base)
	if i >= n || base[i] != '{' || i+1 >= n {
		return redirectMacroNone, i
	}
	switch base[i+1] {
	case 'c':
		if i+redirectMacroClickLen <= n &&
			base[i+2] == 'l' && base[i+3] == 'i' && base[i+4] == 'c' && base[i+5] == 'k' &&
			base[i+6] == '_' && base[i+7] == 'i' && base[i+8] == 'd' && base[i+9] == '}' {
			return redirectMacroClickID, i + redirectMacroClickLen
		}
	case 'u':
		if i+redirectMacroUserLen <= n &&
			base[i+2] == 's' && base[i+3] == 'e' && base[i+4] == 'r' &&
			base[i+5] == '_' && base[i+6] == 'i' && base[i+7] == 'd' && base[i+8] == '}' {
			return redirectMacroUserID, i + redirectMacroUserLen
		}
	case 's':
		if i+redirectMacroSubLen <= n && base[i+2] == 'u' && base[i+3] == 'b' && base[i+5] == '}' {
			digit := base[i+4]
			if digit >= '1' && digit <= '9' {
				return redirectMacroID(redirectMacroSub1 + redirectMacroID(digit-'1')), i + redirectMacroSubLen
			}
		}
		if i+redirectMacroSubLen+1 <= n && base[i+2] == 'u' && base[i+3] == 'b' {
			d1, d2 := base[i+4], base[i+5]
			if d1 >= '1' && d1 <= '3' && d2 >= '0' && d2 <= '9' {
				idx := int(d1-'0')*10 + int(d2-'0')
				if idx >= 10 && idx <= maxSubIDs && base[i+6] == '}' {
					return redirectMacroID(redirectMacroSub1 + redirectMacroID(idx-1)), i + redirectMacroSubLen + 1
				}
			}
		}
	}
	return redirectMacroNone, i
}

const migrationMaxBodyBytes = migrationsource.MaxPayloadBytes

func (h *CampaignsHTTPHandlers) registerMigrationRoutes(
	mux *http.ServeMux,
	limit func(http.HandlerFunc) http.HandlerFunc,
	perm func([]string, http.HandlerFunc) http.HandlerFunc,
) {
	mux.HandleFunc("GET /api/v1/campaigns/migrate/sources", limit(perm([]string{"campaigns:read"}, h.listMigrationSources)))
	mux.HandleFunc("POST /api/v1/campaigns/migrate/preview", limit(perm([]string{"campaigns:write"}, h.previewMigration)))
	mux.HandleFunc("POST /api/v1/campaigns/import/validate", limit(perm([]string{"campaigns:write"}, h.previewMigration)))
	mux.HandleFunc("POST /api/v1/campaigns/import/validate/jobs", limit(perm([]string{"campaigns:write"}, h.postCampaignImportValidateJob)))
	mux.HandleFunc("GET /api/v1/campaigns/import/validate/jobs/{id}", limit(perm([]string{"campaigns:write"}, h.getCampaignImportValidateJob)))
	mux.HandleFunc("POST /api/v1/campaigns/migrate/import", limit(perm([]string{"campaigns:write"}, h.importMigration)))
	h.registerMigrationPullRoutes(mux, limit, perm)
}

func (h *CampaignsHTTPHandlers) listMigrationSources(w http.ResponseWriter, r *http.Request) {
	httpresponse.JSON(w, http.StatusOK, migrationsource.SourcesResponse{
		Sources:         migrationsource.ListSources(),
		MaxPayloadBytes: migrationsource.MaxPayloadBytes,
	})
}

func (h *CampaignsHTTPHandlers) previewMigration(w http.ResponseWriter, r *http.Request) {
	req, ok := coldpath.DecodeRequestOrBadRequest[MigratePreviewRequest](w, r, migrationMaxBodyBytes)
	if !ok {
		return
	}
	kind := migrationsource.SourceKind(strings.TrimSpace(req.SourceKind))
	if kind == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "source_kind is required")
		return
	}
	payload := bytesTrimSpaceJSON(req.Payload)
	if len(payload) == 0 {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "payload is required")
		return
	}
	if len(payload) > migrationMaxBodyBytes {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "payload too large")
		return
	}
	result, err := migrationsource.Preview(kind, payload, nil)
	if err != nil {
		if strings.Contains(err.Error(), "not implemented") || strings.Contains(err.Error(), "unsupported") {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

func (h *CampaignsHTTPHandlers) importMigration(w http.ResponseWriter, r *http.Request) {
	if h.Campaigns == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "campaign service unavailable")
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Idempotency-Key header is required")
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[MigrateImportRequest](w, r, migrationMaxBodyBytes)
	if !ok {
		return
	}
	customerID, err := uuid.Parse(strings.TrimSpace(req.CustomerID))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
		return
	}
	if h.ResolveCustomerID != nil {
		customerID, err = h.ResolveCustomerID(r, nonNilUUID(customerID))
		if err != nil {
			h.WriteHandlerError(w, err)
			return
		}
	}
	kind := migrationsource.SourceKind(strings.TrimSpace(req.SourceKind))
	if kind == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "source_kind is required")
		return
	}
	payload := bytesTrimSpaceJSON(req.Payload)
	if len(payload) == 0 {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "payload is required")
		return
	}
	if len(payload) > migrationMaxBodyBytes {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "payload too large")
		return
	}
	result, err := h.Campaigns.ImportMigrationCampaigns(r.Context(), ImportMigrationSpec{
		CustomerID:       customerID,
		IdempotencyKey:   idempotencyKey,
		SourceKind:       kind,
		Payload:          payload,
		NamePrefix:       req.NamePrefix,
		BudgetLimitMicro: req.BudgetLimitMicro,
	})
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusCreated, result)
}

func (h *CampaignsHTTPHandlers) postCampaignImportValidateJob(w http.ResponseWriter, r *http.Request) {
	if h.ReportJobs == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "import validation jobs not configured")
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[ImportValidateJobRequest](w, r, migrationMaxBodyBytes)
	if !ok {
		return
	}
	customerID, err := uuid.Parse(strings.TrimSpace(req.CustomerID))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
		return
	}
	if h.ResolveCustomerID != nil {
		customerID, err = h.ResolveCustomerID(r, nonNilUUID(customerID))
		if err != nil {
			h.WriteHandlerError(w, err)
			return
		}
	}
	kind := migrationsource.SourceKind(strings.TrimSpace(req.SourceKind))
	if kind == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "source_kind is required")
		return
	}
	payload := bytesTrimSpaceJSON(req.Payload)
	if len(payload) == 0 {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "payload is required")
		return
	}
	jobID, err := h.ReportJobs.CreateJob(r.Context(), reportjob.ReportJobSpec{
		CustomerID:       customerID.String(),
		ReportKey:        reportjob.CampaignImportValidationReportKey,
		Format:           "json",
		ImportSourceKind: string(kind),
		ImportPayload:    payload,
	}, strings.TrimSpace(r.Header.Get("Idempotency-Key")))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	status, _ := h.ReportJobs.GetJob(r.Context(), jobID)
	httpresponse.JSON(w, http.StatusCreated, status)
}

func (h *CampaignsHTTPHandlers) getCampaignImportValidateJob(w http.ResponseWriter, r *http.Request) {
	if h.ReportJobs == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "import validation jobs not configured")
		return
	}
	jobID := r.PathValue("id")
	status, ok := h.ReportJobs.GetJob(r.Context(), jobID)
	if !ok {
		httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "job not found")
		return
	}
	if status.ReportKey != reportjob.CampaignImportValidationReportKey {
		httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "job not found")
		return
	}
	if h.ResolveCustomerID != nil {
		customerID, err := uuid.Parse(status.CustomerID)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid job customer")
			return
		}
		if _, err := h.ResolveCustomerID(r, &customerID); err != nil {
			h.WriteHandlerError(w, err)
			return
		}
	}
	httpresponse.JSON(w, http.StatusOK, status)
}

func bytesTrimSpaceJSON(raw json.RawMessage) []byte {
	return []byte(strings.TrimSpace(string(raw)))
}

type migrationPullService interface {
	PreviewMigrationPull(ctx context.Context, spec PullMigrationPreviewSpec) (migrationsource.PreviewResult, error)
	ImportMigrationPull(ctx context.Context, spec PullMigrationImportSpec) (ImportMigrationResult, error)
}

func (h *CampaignsHTTPHandlers) registerMigrationPullRoutes(
	mux *http.ServeMux,
	limit func(http.HandlerFunc) http.HandlerFunc,
	perm func([]string, http.HandlerFunc) http.HandlerFunc,
) {
	write := []string{"campaigns:write"}
	mux.HandleFunc("POST /api/v1/campaigns/migrate/pull/preview", limit(perm(write, h.previewMigrationPull)))
	mux.HandleFunc("POST /api/v1/campaigns/migrate/pull/import", limit(perm(write, h.importMigrationPull)))
}

func (h *CampaignsHTTPHandlers) previewMigrationPull(w http.ResponseWriter, r *http.Request) {
	req, ok := coldpath.DecodeRequestOrBadRequest[MigratePullRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	svc, ok := h.Campaigns.(migrationPullService)
	if !ok {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "migration pull not configured")
		return
	}
	kind := migrationsource.SourceKind(strings.TrimSpace(req.SourceKind))
	if !migrationsource.PullSupported(kind) {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "source_kind does not support live pull")
		return
	}
	result, err := svc.PreviewMigrationPull(r.Context(), PullMigrationPreviewSpec{
		SourceKind: kind,
		BaseURL:    req.BaseURL,
		APIToken:   req.APIToken,
		PullPath:   req.PullPath,
	})
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

func (h *CampaignsHTTPHandlers) importMigrationPull(w http.ResponseWriter, r *http.Request) {
	if h.Campaigns == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "campaign service unavailable")
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Idempotency-Key header is required")
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[MigratePullRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	svc, ok := h.Campaigns.(migrationPullService)
	if !ok {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "migration pull not configured")
		return
	}
	customerID, err := uuid.Parse(strings.TrimSpace(req.CustomerID))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
		return
	}
	if h.ResolveCustomerID != nil {
		customerID, err = h.ResolveCustomerID(r, nonNilUUID(customerID))
		if err != nil {
			h.WriteHandlerError(w, err)
			return
		}
	}
	kind := migrationsource.SourceKind(strings.TrimSpace(req.SourceKind))
	if !migrationsource.PullSupported(kind) {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "source_kind does not support live pull")
		return
	}
	result, err := svc.ImportMigrationPull(r.Context(), PullMigrationImportSpec{
		PullMigrationPreviewSpec: PullMigrationPreviewSpec{
			SourceKind: kind,
			BaseURL:    req.BaseURL,
			APIToken:   req.APIToken,
			PullPath:   req.PullPath,
		},
		CustomerID:       customerID,
		IdempotencyKey:   idempotencyKey,
		NamePrefix:       req.NamePrefix,
		BudgetLimitMicro: req.BudgetLimitMicro,
	})
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusCreated, result)
}

func (h *CampaignsHTTPHandlers) writeCampaignRevisionConflict(w http.ResponseWriter, r *http.Request, campaignID uuid.UUID, current CampaignDTO, req PatchCampaignRequest) {
	if h.RecordRevisionConflict != nil && req.ExpectedRevision != nil {
		h.RecordRevisionConflict(r.Context(), campaignID, strings.TrimSpace(*req.ExpectedRevision))
	}
	httpresponse.JSON(w, http.StatusConflict, buildCampaignConflictResponse(current, req))
}

func ImportMigrationCampaigns(ctx context.Context, fx Effects, spec ImportMigrationSpec) (ImportMigrationResult, error) {
	host := fx.CampaignImportExportHost()
	if fx == nil || host == nil || host.Pool() == nil {
		return ImportMigrationResult{}, errServiceUnavailable()
	}
	if spec.CustomerID == uuid.Nil {
		return ImportMigrationResult{}, errValidation("customer_id is required")
	}
	batchKey := strings.TrimSpace(spec.IdempotencyKey)
	if batchKey == "" {
		return ImportMigrationResult{}, errValidation("idempotency key is required")
	}
	preview, err := migrationsource.Preview(spec.SourceKind, spec.Payload, nil)
	if err != nil {
		return ImportMigrationResult{}, errValidation(err.Error())
	}
	out := ImportMigrationResult{
		ImportBatchID: batchKey,
		Warnings:      preview.Warnings,
	}
	defaultBudget := migrationsource.DefaultMigrateBudgetMicro()
	if spec.BudgetLimitMicro != nil && *spec.BudgetLimitMicro > 0 {
		defaultBudget = *spec.BudgetLimitMicro
	}
	for i, mapped := range preview.MappedCampaigns {
		shape := migrationsource.MappedToExportShape(mapped, spec.NamePrefix, defaultBudget)
		bundle := exportBundleFromMigrationShape(shape)
		itemKey := migrationsource.ImportIdempotencyKey(batchKey, i)
		result, err := fx.ImportCampaign(ctx, ImportCampaignSpec{
			CustomerID:     spec.CustomerID,
			IdempotencyKey: itemKey,
			Bundle:         bundle,
		})
		if err != nil {
			out.Failed = append(out.Failed, ImportMigrationFailure{
				Ref:     mapped.Ref,
				Name:    mapped.Name,
				Message: err.Error(),
			})
			continue
		}
		out.Imported = append(out.Imported, result)
	}
	if len(out.Imported) == 0 && len(out.Failed) > 0 {
		return out, errValidation("no campaigns imported")
	}
	if len(out.Imported) > 0 {
		fx.AuditLog(ctx, nil, uuid.Nil, "MIGRATE_IMPORT", "migration_batch", nil, auditMigrateImportChange{
			SourceKind: string(spec.SourceKind),
			CustomerID: spec.CustomerID.String(),
			Imported:   len(out.Imported),
			Failed:     len(out.Failed),
		}, auditIdempotencyMeta{IdempotencyKey: batchKey})
	}
	return out, nil
}

func exportBundleFromMigrationShape(shape migrationsource.ExportCampaignShape) CampaignExportBundle {
	camp := CampaignExportCampaign{
		Name:              shape.Name,
		BudgetLimitMicro:  shape.BudgetLimitMicro,
		TargetURL:         shape.TargetURL,
		TrafficTemplateID: shape.TrafficTemplateID,
		ClickQueryParams:  shape.ClickQueryParams,
	}
	bundle := CampaignExportBundle{
		ExportVersion: CampaignExportVersion,
		Campaign:      camp,
	}
	if shape.IntegrationSchema != "" {
		bundle.IntegrationSchemaName = shape.IntegrationSchema
	}
	if shape.IngressCostParam != "" {
		camp.IngressCostConfig = &IngressCostConfigDTO{
			Param:  shape.IngressCostParam,
			Scale:  "decimal",
			Policy: "ignore",
		}
		bundle.Campaign = camp
	}
	if shape.PostbackURLTemplate != "" {
		bundle.PostbackConfig = &CampaignExportPostback{
			Provider:    "custom",
			URLTemplate: shape.PostbackURLTemplate,
			TargetEvent: "conversion",
		}
	}
	if shape.Flow != nil && len(shape.Flow.Paths) > 0 {
		bundle.Flow = &CampaignExportFlow{Name: shape.Flow.Name}
		landerByRef := make(map[string]CampaignExportLander, len(shape.Flow.Paths))
		offerByRef := make(map[string]CampaignExportOffer, len(shape.Flow.Paths))
		for _, path := range shape.Flow.Paths {
			bundle.Flow.Paths = append(bundle.Flow.Paths, CampaignExportFlowPath{
				Weight: path.Weight,
				Landers: []CampaignExportFlowLanderRef{{
					Ref:    path.LanderRef,
					Weight: 100,
				}},
				Offers: []CampaignExportFlowOfferRef{{
					Ref:    path.OfferRef,
					Weight: 100,
				}},
			})
			landerByRef[path.LanderRef] = CampaignExportLander{
				Ref:  path.LanderRef,
				Name: path.LanderName,
				URL:  path.LanderURL,
			}
			offerByRef[path.OfferRef] = CampaignExportOffer{
				Ref:  path.OfferRef,
				Name: path.OfferName,
				URL:  path.OfferURL,
			}
		}
		for _, lander := range landerByRef {
			bundle.Landers = append(bundle.Landers, lander)
		}
		for _, offer := range offerByRef {
			bundle.Offers = append(bundle.Offers, offer)
		}
	}
	return bundle
}

type auditMigrateImportChange struct {
	SourceKind string `json:"source_kind"`
	CustomerID string `json:"customer_id"`
	Imported   int    `json:"imported_count"`
	Failed     int    `json:"failed_count"`
}

func PreviewMigrationPull(ctx context.Context, fx Effects, spec PullMigrationPreviewSpec) (migrationsource.PreviewResult, error) {
	payload, err := migrationsource.FetchRemotePayload(ctx, migrationsource.PullSpec{
		SourceKind: spec.SourceKind,
		BaseURL:    spec.BaseURL,
		APIToken:   spec.APIToken,
		PullPath:   spec.PullPath,
	})
	if err != nil {
		return migrationsource.PreviewResult{}, errValidation(err.Error())
	}
	return migrationsource.Preview(spec.SourceKind, payload, nil)
}

func ImportMigrationPull(ctx context.Context, fx Effects, spec PullMigrationImportSpec) (ImportMigrationResult, error) {
	payload, err := migrationsource.FetchRemotePayload(ctx, migrationsource.PullSpec{
		SourceKind: spec.SourceKind,
		BaseURL:    spec.BaseURL,
		APIToken:   spec.APIToken,
		PullPath:   spec.PullPath,
	})
	if err != nil {
		return ImportMigrationResult{}, errValidation(err.Error())
	}
	return fx.ImportMigrationCampaigns(ctx, ImportMigrationSpec{
		CustomerID:       spec.CustomerID,
		IdempotencyKey:   spec.IdempotencyKey,
		SourceKind:       spec.SourceKind,
		Payload:          payload,
		NamePrefix:       spec.NamePrefix,
		BudgetLimitMicro: spec.BudgetLimitMicro,
	})
}

func mapServiceError(err error) (int, string, string) {
	if err == nil {
		return http.StatusInternalServerError, "INTERNAL", "internal error"
	}
	if errors.Is(err, ErrCampaignNotFound) {
		return http.StatusNotFound, "NOT_FOUND", err.Error()
	}
	if errors.Is(err, ErrForbidden) {
		return http.StatusForbidden, "FORBIDDEN", err.Error()
	}
	if errors.Is(err, ErrValidation) {
		return http.StatusBadRequest, "BAD_REQUEST", err.Error()
	}
	return http.StatusInternalServerError, "INTERNAL", err.Error()
}

type DataFreshnessDTO = reports.DataFreshnessDTO

type (
	FlowDTO                 = flow.DTO
	FlowPathDTO             = flow.PathDTO
	FlowPathLanderRef       = flow.PathLanderRef
	FlowPathOfferRef        = flow.PathOfferRef
	FlowPathFiltersDTO      = flow.PathFiltersDTO
	FlowPathErrorDTO        = flow.PathErrorDTO
	FlowValidateResponseDTO = flow.ValidateResponseDTO
)

func ValidateFlowPathShape(paths []FlowPathDTO) error {
	return flow.ValidatePathShape(paths)
}

func BuildCampaignFlowValidateResponse(paths []FlowPathDTO) FlowValidateResponseDTO {
	return flow.BuildValidateResponse(paths)
}

func ParseFlowPaths(raw json.RawMessage) ([]FlowPathDTO, error) {
	return flow.ParsePaths(raw)
}

func FormatFlowPathErrors(pathErrors []FlowPathErrorDTO) string {
	return flow.FormatPathErrors(pathErrors)
}

var (
	ErrForecastClickHouseTimeout = errors.New("forecast clickhouse query timed out")
	ErrForecastUnavailable       = errors.New("forecast service unavailable")
	ErrClickHouseNotConfigured   = errors.New("clickhouse not configured")
)

const forecastDefaultRetryAfterSec = 30

func ForecastRetryAfterSec() int {
	return forecastDefaultRetryAfterSec
}

type CampaignDTO struct {
	ID                         string                `json:"id"`
	Name                       string                `json:"name"`
	Status                     string                `json:"status"`
	BudgetLimit                string                `json:"budget_limit"`
	BudgetLimitDisplay         string                `json:"budget_limit_display,omitempty"`
	CurrentSpend               string                `json:"current_spend"`
	CurrentSpendDisplay        string                `json:"current_spend_display,omitempty"`
	CustomerID                 string                `json:"customer_id"`
	PacingMode                 string                `json:"pacing_mode"`
	DailyBudget                string                `json:"daily_budget"`
	DailyBudgetDisplay         string                `json:"daily_budget_display,omitempty"`
	Timezone                   string                `json:"timezone"`
	FreqLimit                  int32                 `json:"freq_limit"`
	FreqWindow                 int32                 `json:"freq_window"`
	TargetCountries            []string              `json:"target_countries"`
	TargetURL                  string                `json:"target_url,omitempty"`
	SafePageURL                string                `json:"safe_page_url,omitempty"`
	SafePageEnabled            bool                  `json:"safe_page_enabled"`
	AttestationEnabled         bool                  `json:"attestation_enabled"`
	AttestationMode            string                `json:"attestation_mode,omitempty"`
	AttestationTTLSec          int32                 `json:"attestation_ttl_sec"`
	DmrEnabled                 bool                  `json:"dmr_enabled"`
	CIDRBlockEnabled           bool                  `json:"cidr_block_enabled"`
	ProxyVPNBlockEnabled       bool                  `json:"proxy_vpn_block_enabled"`
	ModeratorIntelEnabled      bool                  `json:"moderator_intel_enabled"`
	ReviewTrafficAction        string                `json:"review_traffic_action,omitempty"`
	TLSFingerprintBlockEnabled bool                  `json:"tls_fingerprint_block_enabled"`
	ConnTypePolicy             string                `json:"conn_type_policy,omitempty"`
	LinkSigningEnabled         bool                  `json:"link_signing_enabled"`
	LinkSigningTTLSec          int32                 `json:"link_signing_ttl_sec"`
	ClickDelivery              string                `json:"click_delivery,omitempty"`
	ProxyUpstreamURL           string                `json:"proxy_upstream_url,omitempty"`
	ProxyRewriteAssets         bool                  `json:"proxy_rewrite_assets"`
	BrandID                    string                `json:"brand_id,omitempty"`
	CreativePayload            json.RawMessage       `json:"creative_payload,omitempty"`
	ReferrerFilter             string                `json:"referrer_filter,omitempty"`
	StartAt                    string                `json:"start_at,omitempty"`
	EndAt                      string                `json:"end_at,omitempty"`
	DaypartHours               []int16               `json:"daypart_hours"`
	FlowID                     string                `json:"flow_id,omitempty"`
	OwnerUserID                string                `json:"owner_user_id,omitempty"`
	IngressCostConfig          *IngressCostConfigDTO `json:"ingress_cost_config,omitempty"`
	TrafficTemplateID          string                `json:"traffic_template_id,omitempty"`
	ClickQueryParams           map[string]string     `json:"click_query_params,omitempty"`
	CreatedAt                  string                `json:"created_at"`
	UpdatedAt                  string                `json:"updated_at"`
	Revision                   string                `json:"revision,omitempty"`
	MarginBreach               bool                  `json:"margin_breach,omitempty"`
	StatusLabel                string                `json:"status_label,omitempty"`
	StatusTone                 string                `json:"status_tone,omitempty"`
	AllowedActions             []string              `json:"allowed_actions,omitempty"`
	DeniedReasons              map[string]string     `json:"denied_reasons,omitempty"`
	FieldsRedacted             []string              `json:"fields_redacted,omitempty"`
	EffectiveBudgetMicros      *int64                `json:"effective_budget_micros,omitempty"`
	PendingBudgetMicros        *int64                `json:"pending_budget_micros,omitempty"`
}

type BlacklistDTO struct {
	ID        int64  `json:"id"`
	IP        string `json:"ip"`
	Reason    string `json:"reason"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

type IngressCostConfigDTO struct {
	Param    string `json:"param"`
	Scale    string `json:"scale,omitempty"`
	MaxMicro int64  `json:"max_micro,omitempty"`
	Policy   string `json:"policy,omitempty"`
}

type CampaignForecastInput struct {
	CustomerID       *uuid.UUID
	BudgetLimitMicro int64
	TargetCountries  []string
	DaypartHours     []int16
	StartAt          time.Time
	EndAt            time.Time
	PacingMode       string
	Timezone         string
}

type SpendCurvePoint struct {
	Hour        string `json:"hour"`
	SpendMicro  int64  `json:"spend_micro"`
	Impressions int64  `json:"impressions"`
}

type ForecastAdvisory struct {
	Code            string `json:"code"`
	Message         string `json:"message"`
	SuggestedPacing string `json:"suggested_pacing"`
}

type CampaignForecastDTO struct {
	ImpressionsP50 int64             `json:"impressions_p50"`
	ImpressionsP90 int64             `json:"impressions_p90"`
	SpendCurve     []SpendCurvePoint `json:"spend_curve"`
	LowConfidence  bool              `json:"low_confidence"`
	Advisory       *ForecastAdvisory `json:"advisory,omitempty"`
}

type CampaignMetricsDTO struct {
	Impressions int64 `json:"impressions"`
	Clicks      int64 `json:"clicks"`
	Conversions int64 `json:"conversions"`
}

type CampaignHourlyBucketDTO struct {
	Hour        string `json:"hour"`
	Impressions int64  `json:"impressions"`
	Clicks      int64  `json:"clicks"`
	Conversions int64  `json:"conversions"`
}

type CampaignDailyBucketDTO struct {
	Day         string `json:"day"`
	Impressions int64  `json:"impressions"`
	Clicks      int64  `json:"clicks"`
	Conversions int64  `json:"conversions"`
}

type CampaignStatsDTO struct {
	CampaignID   string                    `json:"campaign_id"`
	CurrentSpend string                    `json:"current_spend"`
	Metrics      CampaignMetricsDTO        `json:"metrics"`
	Hourly       []CampaignHourlyBucketDTO `json:"hourly"`
	Daily        []CampaignDailyBucketDTO  `json:"daily,omitempty"`
	Granularity  string                    `json:"granularity"`
	From         string                    `json:"from"`
	To           string                    `json:"to"`
	Stale        bool                      `json:"stale"`
	Source       string                    `json:"source"`
	Consistency  string                    `json:"consistency"`
}

type BlacklistListResponse = ListResponse[BlacklistDTO]

type MutationPreviewDTO struct {
	DryRun      bool            `json:"dry_run"`
	Action      string          `json:"action"`
	WouldChange json.RawMessage `json:"would_change"`
}

type BalanceLedgerDTO struct {
	ID              int64  `json:"id"`
	CustomerID      string `json:"customer_id"`
	CampaignID      string `json:"campaign_id,omitempty"`
	Amount          string `json:"amount"`
	Type            string `json:"type"`
	IdempotencyHash string `json:"idempotency_hash,omitempty"`
	CreatedAt       string `json:"created_at"`
}

type CustomerBalanceDTO struct {
	CustomerID string             `json:"customer_id"`
	Balance    string             `json:"balance"`
	Currency   string             `json:"currency"`
	Ledger     []BalanceLedgerDTO `json:"ledger"`
}

type LedgerExportResult struct {
	NextCursor int64
	Truncated  bool
	Bytes      int
}

type UsageExportResult struct {
	NextCursor string
	Truncated  bool
	Bytes      int
}

type AuditLogDTO struct {
	ID         int64           `json:"id"`
	AdminID    string          `json:"admin_id,omitempty"`
	Action     string          `json:"action"`
	TargetType string          `json:"target_type"`
	TargetID   string          `json:"target_id,omitempty"`
	Changes    json.RawMessage `json:"changes"`
	Metadata   json.RawMessage `json:"metadata"`
	IsMasked   bool            `json:"is_masked"`
	CreatedAt  string          `json:"created_at"`
}

type AuditLogListResponse = ListResponse[AuditLogDTO]

type PatchCampaignRequest struct {
	Name                       *string               `json:"name,omitempty"`
	Status                     *string               `json:"status,omitempty"`
	BudgetLimitMicro           *int64                `json:"budget_limit_micro,omitempty"`
	BudgetLimit                *string               `json:"budget_limit,omitempty"`
	PacingMode                 *string               `json:"pacing_mode,omitempty"`
	DailyBudgetMicro           *int64                `json:"daily_budget_micro,omitempty"`
	Timezone                   *string               `json:"timezone,omitempty"`
	FreqLimit                  *int32                `json:"freq_limit,omitempty"`
	FreqWindow                 *int32                `json:"freq_window,omitempty"`
	TargetCountries            []string              `json:"target_countries,omitempty"`
	TargetURL                  *string               `json:"target_url,omitempty"`
	SafePageURL                *string               `json:"safe_page_url,omitempty"`
	SafePageEnabled            *bool                 `json:"safe_page_enabled,omitempty"`
	DmrEnabled                 *bool                 `json:"dmr_enabled,omitempty"`
	CIDRBlockEnabled           *bool                 `json:"cidr_block_enabled,omitempty"`
	ProxyVPNBlockEnabled       *bool                 `json:"proxy_vpn_block_enabled,omitempty"`
	ModeratorIntelEnabled      *bool                 `json:"moderator_intel_enabled,omitempty"`
	ReviewTrafficAction        *string               `json:"review_traffic_action,omitempty"`
	TLSFingerprintBlockEnabled *bool                 `json:"tls_fingerprint_block_enabled,omitempty"`
	ConnTypePolicy             *string               `json:"conn_type_policy,omitempty"`
	LinkSigningEnabled         *bool                 `json:"link_signing_enabled,omitempty"`
	LinkSigningTTLSec          *int32                `json:"link_signing_ttl_sec,omitempty"`
	AttestationEnabled         *bool                 `json:"attestation_enabled,omitempty"`
	AttestationMode            *string               `json:"attestation_mode,omitempty"`
	AttestationTTLSec          *int32                `json:"attestation_ttl_sec,omitempty"`
	ReferrerFilter             *string               `json:"referrer_filter,omitempty"`
	ClickDelivery              *string               `json:"click_delivery,omitempty"`
	ProxyUpstreamURL           *string               `json:"proxy_upstream_url,omitempty"`
	ProxyRewriteAssets         *bool                 `json:"proxy_rewrite_assets,omitempty"`
	StartAt                    *time.Time            `json:"start_at,omitempty"`
	EndAt                      *time.Time            `json:"end_at,omitempty"`
	DaypartHours               []int16               `json:"daypart_hours,omitempty"`
	FlowID                     *uuid.UUID            `json:"flow_id,omitempty"`
	BrandID                    *uuid.UUID            `json:"brand_id,omitempty"`
	IngressCostConfig          *IngressCostConfigDTO `json:"ingress_cost_config,omitempty"`
	TrafficTemplateID          *string               `json:"traffic_template_id,omitempty"`
	ClickQueryParams           *map[string]string    `json:"click_query_params,omitempty"`
	ExpectedRevision           *string               `json:"expected_revision,omitempty"`
	PublishForce               bool                  `json:"-"`
}

type CampaignEventDTO struct {
	ClickID   string          `json:"click_id"`
	EventType string          `json:"event_type"`
	UserID    string          `json:"user_id,omitempty"`
	IP        string          `json:"ip_address,omitempty"`
	UserAgent string          `json:"user_agent,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	CreatedAt string          `json:"created_at"`
}

type CampaignEventListResponse = ListResponse[CampaignEventDTO]
