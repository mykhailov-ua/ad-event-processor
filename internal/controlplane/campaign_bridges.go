package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ad-event-processor/internal/automation"
	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/flow"
	"ad-event-processor/internal/integrationschema"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/migrationsource"
	"ad-event-processor/internal/outbox"
	"ad-event-processor/internal/platformadmin"
	"ad-event-processor/internal/reports"
	"ad-event-processor/internal/shardadmin"
	"ad-event-processor/pkg/coldpath"
	db "ad-event-processor/internal/domain/db"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	_ campaign.DeliveryHost        = (*Service)(nil)
	_ campaign.Effects             = (*Service)(nil)
	_ campaign.LoopHost            = (*Service)(nil)
	_ campaign.VPPHost             = (*Service)(nil)
	_ campaign.DrainHost           = (*Service)(nil)
	_ campaign.WizardHost          = (*Service)(nil)
	_ campaign.TemplateCatalogHost = (*Service)(nil)
	_ automation.LicenseGate       = (*Service)(nil)
)

var ErrClickHouseNotConfigured = campaign.ErrClickHouseNotConfigured

const campaignExportVersion = campaign.CampaignExportVersion

type deliveryOutboxMerge = campaign.DeliveryOutboxMerge

const (
	outboxPriSyncBrandCreatives = campaign.OutboxPriSyncBrandCreatives
	outboxPriCreateCampaign     = campaign.OutboxPriCreateCampaign
	outboxPriPacing             = campaign.OutboxPriPacing
	outboxPriPause              = campaign.OutboxPriPause
)

var (
	errCampaignWizardSessionNotFound = campaign.ErrCampaignWizardSessionNotFound
	errCampaignWizardSessionExpired  = campaign.ErrCampaignWizardSessionExpired
	errCampaignWizardIncomplete      = campaign.ErrCampaignWizardIncomplete
)

const (
	wizardStepTrafficSource       = campaign.WizardStepTrafficSource
	wizardStepIntegrationTemplate = campaign.WizardStepIntegrationTemplate
	wizardStepFlowSkeleton        = campaign.WizardStepFlowSkeleton
	wizardStepBudget              = campaign.WizardStepBudget
	wizardStepReview              = campaign.WizardStepReview
)

func campaignOwnerUserFilter(ctx context.Context) pgtype.UUID {
	return campaign.CampaignOwnerUserFilter(ctx)
}

func assertMediaBuyerCampaignAccess(ctx context.Context, camp db.Campaign) error {
	return campaign.AssertMediaBuyerCampaignAccess(ctx, camp)
}

func campaignRevision(updatedAt string) string {
	return campaign.CampaignRevision(updatedAt)
}

func resolveScheduleStatus(now time.Time, startAt, endAt *time.Time) db.CampaignStatusType {
	return campaign.ResolveScheduleStatus(now, startAt, endAt)
}

func validateDaypartHours(hours []int16) error {
	return campaign.ValidateDaypartHours(hours)
}

func validateSchedule(startAt, endAt *time.Time) error {
	return campaign.ValidateSchedule(startAt, endAt)
}

func countriesOrEmpty(c []string) []string {
	return campaign.CountriesOrEmpty(c)
}

func daypartOrEmpty(hours []int16) []int16 {
	return campaign.DaypartOrEmpty(hours)
}

func defaultTimezone(raw string) string {
	return campaign.DefaultTimezone(raw)
}

func ForecastRetryAfterSec() int {
	return campaign.ForecastRetryAfterSec()
}

func parseFlowPaths(raw json.RawMessage) ([]FlowPathDTO, error) {
	return campaign.ParseFlowPaths(raw)
}

func buildCampaignFlowValidateResponse(paths []FlowPathDTO) FlowValidateResponseDTO {
	return campaign.BuildCampaignFlowValidateResponse(paths)
}

func attachCampaignPresentation(ctx context.Context, dto *CampaignDTO) {
	campaign.AttachCampaignPresentation(ctx, dto)
}

func clickQueryParamsFromRaw(raw []byte) map[string]string {
	return campaign.ClickQueryParamsFromRaw(raw)
}

func nonNilUUID(id uuid.UUID) *uuid.UUID {
	return campaign.NonNilUUID(id)
}

func ApplyOnboardingTemplate(key string) (CampaignWizardStored, error) {
	return campaign.ApplyOnboardingTemplate(key)
}

func (s *Service) cloneCampaignPlacementBlocks(ctx context.Context, sourceID, destID uuid.UUID) error {
	if s == nil || len(s.redisShards) == 0 {
		return nil
	}
	redisClient := s.redisClientForCampaign(sourceID)
	if redisClient == nil {
		return nil
	}
	key := domain.PlacementBlacklistKey(sourceID)
	placements, err := redisClient.HKeys(ctx, key).Result()
	if err != nil {
		return err
	}
	if len(placements) == 0 {
		return nil
	}
	destKey := domain.PlacementBlacklistKey(destID)
	for _, placementID := range placements {
		placementID = strings.TrimSpace(placementID)
		if placementID == "" {
			continue
		}
		if err := shardadmin.SyncGlobalHashFieldToAllShards(ctx, s.redisShards, destKey, placementID, "1", false); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) AutoscaleEnabled() bool {
	return s != nil && s.cfg != nil
}

func (s *Service) AutoscaleHighCTRThreshold() float64 {
	if s == nil || s.cfg == nil {
		return 0
	}
	return s.cfg.AutoscaleHighCTRThreshold
}

func (s *Service) AutoscaleLowCTRThreshold() float64 {
	if s == nil || s.cfg == nil {
		return 0
	}
	return s.cfg.AutoscaleLowCTRThreshold
}

func (s *Service) AutoscaleMinImpressions() int64 {
	if s == nil || s.cfg == nil {
		return 0
	}
	return s.cfg.AutoscaleMinImpressions
}

func (s *Service) AutoscaleMinRemainingBudget() int64 {
	if s == nil || s.cfg == nil {
		return 0
	}
	return s.cfg.AutoscaleMinRemainingBudget
}

func (s *Service) AutoscaleShiftAmount() int64 {
	if s == nil || s.cfg == nil {
		return 0
	}
	return s.cfg.AutoscaleShiftAmount
}

func (s *Service) AuditAutoscaleBudgetTransfer(ctx context.Context, q db.Querier, campaignID uuid.UUID, change campaign.AutoscaleBudgetAuditChange) {
	s.AuditLog(ctx, q, uuid.Nil, "AUTOSCALE_BUDGET_TRANSFER", "campaign", &campaignID, auditAutoscaleBudgetTransfer{
		OldBudget: change.OldBudget,
		NewBudget: change.NewBudget,
		CTR:       change.CTR,
		Target:    change.Target,
		Source:    change.Source,
	}, nil)
}

func (s *Service) MABMinImpressions() int64 {
	if s == nil || s.cfg == nil || s.cfg.MABMinImpressions <= 0 {
		return 1000
	}
	return s.cfg.MABMinImpressions
}

func (s *Service) MABLookbackDays() int {
	if s == nil || s.cfg == nil || s.cfg.MABLookbackDays <= 0 {
		return 90
	}
	return s.cfg.MABLookbackDays
}

func (s *Service) QueryMABCreativeStats(ctx context.Context, from, to time.Time) (map[uuid.UUID]campaign.CreativeMABStat, error) {
	if s == nil || s.clickhouseQuery == nil {
		return nil, nil
	}
	const query = `
SELECT
 campaign_id,
 creative_id,
 sum(impressions) AS impressions,
 sum(clicks) AS clicks
FROM (
 SELECT
 toString(campaign_id) AS campaign_id,
 nullIf(JSONExtractString(payload, 'creative_id'), '') AS creative_id,
 count() AS impressions,
 toUInt64(0) AS clicks
 FROM impressions
 WHERE created_at >= ? AND created_at < ?
 GROUP BY campaign_id, creative_id
 UNION ALL
 SELECT
 toString(campaign_id),
 nullIf(JSONExtractString(payload, 'creative_id'), ''),
 toUInt64(0),
 count() AS clicks
 FROM clicks
 WHERE created_at >= ? AND created_at < ?
 GROUP BY campaign_id, creative_id
)
GROUP BY campaign_id, creative_id`

	clickhouseCtx, cancel := reports.ClickHouseQueryContext(ctx)
	defer cancel()

	rows, err := s.clickhouseQuery.Query(clickhouseCtx, query, from, to, from, to)
	if err != nil {
		return nil, fmt.Errorf("mab creative stats query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[uuid.UUID]campaign.CreativeMABStat)
	for rows.Next() {
		var campaignID, creativeID string
		var impressions, clicks uint64
		if err := rows.Scan(&campaignID, &creativeID, &impressions, &clicks); err != nil {
			return nil, err
		}
		statKey, err := mabStatKey(campaignID, creativeID)
		if err != nil {
			continue
		}
		out[statKey] = campaign.CreativeMABStat{
			Impressions: int64(impressions),
			Clicks:      int64(clicks),
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func mabStatKey(campaignID, creativeID string) (uuid.UUID, error) {
	if creativeID != "" {
		return uuid.Parse(creativeID)
	}
	return uuid.Parse(campaignID)
}

func (s *Service) QueryFlowBanditStats(ctx context.Context, from, to time.Time) (
	map[uuid.UUID]map[uuid.UUID]flow.EntityBanditStat,
	map[uuid.UUID]map[uuid.UUID]flow.EntityBanditStat,
	error,
) {
	if s == nil || s.clickhouseQuery == nil {
		return nil, nil, nil
	}
	clickhouseCtx, cancel := reports.ClickHouseQueryContext(ctx)
	defer cancel()

	landerByCampaign, err := s.scanFlowBanditRows(clickhouseCtx, flowBanditLanderQuery, from, to)
	if err != nil {
		return nil, nil, err
	}
	offerByCampaign, err := s.scanFlowBanditRows(clickhouseCtx, flowBanditOfferQuery, from, to)
	if err != nil {
		return nil, nil, err
	}
	return landerByCampaign, offerByCampaign, nil
}

const flowBanditLanderQuery = `
SELECT
 toString(campaign_id) AS campaign_id,
 entity_id,
 sum(clicks) AS clicks,
 sum(conversions) AS conversions,
 sum(payout) AS payout
FROM (
 SELECT
 campaign_id,
 nullIf(JSONExtractString(payload, 'lander_id'), '') AS entity_id,
 count() AS clicks,
 toUInt64(0) AS conversions,
 toFloat64(0) AS payout
 FROM clicks
 WHERE created_at >= ? AND created_at < ?
 AND JSONExtractString(payload, 'lander_id') != ''
 GROUP BY campaign_id, entity_id
 UNION ALL
 SELECT
 campaign_id,
 nullIf(JSONExtractString(payload, 'lander_id'), '') AS entity_id,
 toUInt64(0),
 count(),
 sum(toFloat64OrZero(JSONExtractString(payload, 'payout')))
 FROM conversions
 WHERE created_at >= ? AND created_at < ?
 AND JSONExtractString(payload, 'lander_id') != ''
 GROUP BY campaign_id, entity_id
)
GROUP BY campaign_id, entity_id`

const flowBanditOfferQuery = `
SELECT
 toString(campaign_id) AS campaign_id,
 entity_id,
 sum(clicks) AS clicks,
 sum(conversions) AS conversions,
 sum(payout) AS payout
FROM (
 SELECT
 campaign_id,
 nullIf(JSONExtractString(payload, 'offer_id'), '') AS entity_id,
 count() AS clicks,
 toUInt64(0) AS conversions,
 toFloat64(0) AS payout
 FROM clicks
 WHERE created_at >= ? AND created_at < ?
 AND JSONExtractString(payload, 'offer_id') != ''
 GROUP BY campaign_id, entity_id
 UNION ALL
 SELECT
 campaign_id,
 nullIf(JSONExtractString(payload, 'offer_id'), '') AS entity_id,
 toUInt64(0),
 count(),
 sum(toFloat64OrZero(JSONExtractString(payload, 'payout')))
 FROM conversions
 WHERE created_at >= ? AND created_at < ?
 AND JSONExtractString(payload, 'offer_id') != ''
 GROUP BY campaign_id, entity_id
)
GROUP BY campaign_id, entity_id`

func (s *Service) scanFlowBanditRows(ctx context.Context, query string, from, to time.Time) (map[uuid.UUID]map[uuid.UUID]flow.EntityBanditStat, error) {
	rows, err := s.clickhouseQuery.Query(ctx, query, from, to, from, to)
	if err != nil {
		return nil, fmt.Errorf("flow bandit ch query: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := make(map[uuid.UUID]map[uuid.UUID]flow.EntityBanditStat)
	for rows.Next() {
		var campStr, entityStr string
		var clicks, conversions uint64
		var payout float64
		if err := rows.Scan(&campStr, &entityStr, &clicks, &conversions, &payout); err != nil {
			return nil, err
		}
		campID, err := uuid.Parse(campStr)
		if err != nil {
			continue
		}
		entityID, err := uuid.Parse(entityStr)
		if err != nil {
			continue
		}
		perCamp, ok := out[campID]
		if !ok {
			perCamp = make(map[uuid.UUID]flow.EntityBanditStat)
			out[campID] = perCamp
		}
		perCamp[entityID] = flow.EntityBanditStat{
			Clicks:      int64(clicks),
			Conversions: int64(conversions),
			Payout:      payout,
		}
	}
	return out, rows.Err()
}

func (s *Service) AutoscaleBudgets(ctx context.Context, syncWorkers []*domain.SyncWorker) error {
	return campaign.RunAutoscaleBudgetsTick(ctx, s, syncWorkers)
}

type campaignExperimentsHost struct {
	svc *Service
}

func (h campaignExperimentsHost) Pool() *pgxpool.Pool {
	return h.svc.GetPool()
}

func (h campaignExperimentsHost) CohortSnapshotOutboxPayload() ([]byte, error) {
	return coldpath.MarshalOutbox(outbox.CohortSnapshotPayload{Version: 1})
}

func (h campaignExperimentsHost) AuditCohortSnapshotChange(ctx context.Context, q db.Querier, experimentID uuid.UUID, change campaign.ExperimentCohortAuditChange, outboxEventID int64) {
	h.svc.AuditLog(ctx, q, uuid.Nil, "UPDATE_COHORT_SNAPSHOT", "experiment", &experimentID, auditCohortSnapshotChange{
		Name:     change.Name,
		Active:   change.Active,
		Variants: change.Variants,
	}, auditOutboxEventMeta{OutboxEventID: outboxEventID})
}

func (s *Service) UpsertExperimentCohort(ctx context.Context, spec campaign.ExperimentCohortSpec) error {
	return campaign.UpsertExperimentCohort(ctx, campaignExperimentsHost{svc: s}, spec)
}

type campaignImportExportHost struct {
	svc *Service
}

func (h campaignImportExportHost) Pool() *pgxpool.Pool {
	return h.svc.GetPool()
}

func (h campaignImportExportHost) AssertMediaBuyerCampaignAccess(ctx context.Context, row db.Campaign) error {
	return assertMediaBuyerCampaignAccess(ctx, row)
}

func (h campaignImportExportHost) GetFlow(ctx context.Context, flowID uuid.UUID) (FlowDTO, error) {
	return h.svc.GetFlow(ctx, flowID)
}

func (h campaignImportExportHost) AuditImportCampaign(ctx context.Context, q *db.Queries, campaignID uuid.UUID, change campaign.ImportCampaignAuditChange, meta campaign.ImportCampaignIdempotencyMeta) error {
	h.svc.AuditLog(ctx, q, uuid.Nil, "IMPORT_CAMPAIGN", "campaign", &campaignID, auditImportCampaignChange{Name: change.Name}, auditIdempotencyMeta{IdempotencyKey: meta.IdempotencyKey})
	return nil
}

func (h campaignImportExportHost) EmitCampaignLifecycleOutbox(ctx context.Context, q *db.Queries, campaignID uuid.UUID, status db.CampaignStatusType, budget int64) error {
	return h.svc.EmitCampaignLifecycleOutbox(ctx, q, campaignID, status, budget)
}

func (h campaignImportExportHost) PublishCampaignUpdate(ctx context.Context, campaignID string) {
	_ = h.svc.publishCampaignUpdate(ctx, campaignID)
}

func (h campaignImportExportHost) PublishFlowReload(ctx context.Context) {
	_ = h.svc.PublishFlowReload(ctx)
}

func (s *Service) ExportCampaign(ctx context.Context, campaignID uuid.UUID) (CampaignExportBundle, error) {
	return s.CampaignRuntime().ExportCampaign(ctx, campaignID)
}

func (s *Service) ImportCampaign(ctx context.Context, spec ImportCampaignSpec) (ImportCampaignResult, error) {
	return s.CampaignRuntime().ImportCampaign(ctx, spec)
}

func (s *Service) ImportMigrationCampaigns(ctx context.Context, spec ImportMigrationSpec) (ImportMigrationResult, error) {
	return campaign.ImportMigrationCampaigns(ctx, s, spec)
}

func (s *Service) PreviewMigrationPull(ctx context.Context, spec PullMigrationPreviewSpec) (migrationsource.PreviewResult, error) {
	return campaign.PreviewMigrationPull(ctx, s, spec)
}

func (s *Service) ImportMigrationPull(ctx context.Context, spec PullMigrationImportSpec) (ImportMigrationResult, error) {
	return campaign.ImportMigrationPull(ctx, s, spec)
}

type auditImportCampaignChange struct {
	Name string `json:"name"`
}

func (s *Service) PostbackEncryptionKey() []byte {
	return []byte("postback-encryption-secret-key32")
}

func (s *Service) TemplateCatalog(pool *pgxpool.Pool) *campaign.TemplateCatalog {
	if s == nil {
		return nil
	}
	if pool == nil {
		pool = s.pool
	}
	return campaign.NewTemplateCatalog(pool, s)
}

func (s *Service) ApplyCampaignTemplates(ctx context.Context, campaignID uuid.UUID, req campaign.ApplyCampaignTemplatesRequest) (campaign.ApplyCampaignTemplatesResult, error) {
	return s.TemplateCatalog(s.pool).ApplyCampaignTemplates(ctx, campaignID, req)
}

func (s *Service) AutomationRules() *automation.RulesService {
	if s == nil {
		return nil
	}
	return &automation.RulesService{
		Pool:       s.pool,
		ClickHouse: s.ClickHouseQuery(),
		EvalFloorMinutes: func() int {
			if s.cfg == nil {
				return 15
			}
			return s.cfg.Management.AutomationRulesIntervalMin
		},
		LicenseGate: s,
	}
}

func (s *Service) ValidateAutomationLicense(ctx context.Context, actions []automation.Action) error {
	for _, action := range actions {
		if action.Type != automation.ActionPlatformPause {
			continue
		}
		snap, err := licensing.LoadDeploymentSnapshot(ctx, s.pool)
		if err != nil || !snap.ModuleAllowed(func(f licensing.FeatureSet) bool { return f.AdPlatformCampaignAPIEnabled() }) {
			return fmt.Errorf("platform_pause requires ad_platform_campaign_api license")
		}
	}
	return nil
}

func (s *Service) CampaignFlowID(ctx context.Context, campaignID uuid.UUID) (string, error) {
	return s.campaignFlowID(ctx, campaignID)
}

func (s *Service) AttachCampaignBudgetApprovalState(ctx context.Context, dto *campaign.CampaignDTO) {
	platformadmin.AttachCampaignBudgetApprovalState(ctx, s, dto)
}

func (s *Service) AttachCampaignListBudgetApprovalStates(ctx context.Context, items []campaign.CampaignDTO) {
	platformadmin.AttachCampaignListBudgetApprovalStates(ctx, s, items)
}

func (s *Service) ApplyCampaignIngressCostPatch(ctx context.Context, campaignID uuid.UUID, cfg campaign.IngressCostConfigDTO) error {
	return campaign.ApplyIngressCostPatch(ctx, s, s.pool, campaignID, cfg)
}

func (s *Service) ApplyCampaignClickPresetPatch(ctx context.Context, campaignID uuid.UUID, templateID *string, queryParams *map[string]string) error {
	return campaign.ApplyClickPresetPatch(ctx, s, s.pool, campaignID, templateID, queryParams)
}

func (s *Service) ApplyCampaignBudgetPatch(ctx context.Context, q db.Querier, locked db.Campaign, budgetMicro int64) error {
	return campaign.ApplyBudgetPatch(ctx, s, q, locked, budgetMicro)
}

func (s *Service) HandleMediaBuyerBudgetIncrease(ctx context.Context, locked db.Campaign, userID uuid.UUID, newLimit int64) error {
	return platformadmin.HandleMediaBuyerBudgetIncrease(ctx, s, locked, userID, newLimit)
}

func (s *Service) ApplyCampaignSchedulePatch(ctx context.Context, q db.Querier, campaignID uuid.UUID, locked db.Campaign, startAt, endAt *time.Time, daypartHours []int16) error {
	return campaign.ApplySchedulePatch(ctx, s, q, campaignID, locked, startAt, endAt, daypartHours)
}

func (s *Service) ApplyCampaignStatusPatch(ctx context.Context, q db.Querier, locked db.Campaign, status db.CampaignStatusType, reason string, publishForce bool) error {
	return campaign.ApplyStatusPatch(ctx, s, q, locked, status, reason, publishForce)
}

func (s *Service) CloneCampaignPlacementBlocks(ctx context.Context, sourceID, destID uuid.UUID) error {
	return s.cloneCampaignPlacementBlocks(ctx, sourceID, destID)
}

func (s *Service) CloneCampaign(ctx context.Context, spec campaign.CloneCampaignSpec) (campaign.CloneCampaignResult, error) {
	return campaign.CloneCampaign(ctx, s, s.GetPool(), spec)
}

func (s *Service) ProxyAllowHTTPInsecure() bool {
	return s.cfg != nil && s.cfg.ProxyAllowHTTPInsecure
}

func (s *Service) PublishCampaignUpdate(ctx context.Context, campaignID string) {
	_ = s.publishCampaignUpdate(ctx, campaignID)
}

func (s *Service) ValidateFlowPaths(ctx context.Context, paths []campaign.FlowPathDTO) error {
	return flow.ValidatePathRefs(ctx, s, paths)
}

func (s *Service) EnforceCampaignPublishGate(ctx context.Context, campaignID uuid.UUID, row db.Campaign, force bool) error {
	return campaign.EnforcePublishGate(ctx, s, s.GetPool(), campaignID, row, force)
}

func (s *Service) ResumeCampaignForPublish(ctx context.Context, campaignID uuid.UUID, force bool) error {
	return s.CampaignRuntime().ResumeCampaignForPublish(ctx, campaignID, force)
}

func (s *Service) EnqueueCampaignOutbox(ctx context.Context, q db.Querier, eventType string, campaignID uuid.UUID, budgetLimit int64) error {
	payload, err := coldpath.MarshalOutbox(CampaignPayload{CampaignID: campaignID.String(), BudgetLimit: budgetLimit})
	if err != nil {
		return fmt.Errorf("marshal %s outbox payload: %w", eventType, err)
	}
	_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{EventType: eventType, Payload: payload})
	return err
}

func (s *Service) AuditCampaignPublishForce(ctx context.Context, campaignID uuid.UUID) error {
	return campaign.AuditPublishForce(ctx, s, s.GetPool(), campaignID)
}

func (s *Service) EnforceDeploymentLicenseCampaignCap(ctx context.Context) error {
	return s.enforceDeploymentLicenseCampaignCap(ctx)
}

func (s *Service) AssignCampaignBrand(ctx context.Context, campaignID, brandID uuid.UUID) error {
	return campaign.AssignCampaignBrand(ctx, s, s.pool, campaignID, brandID)
}

func (s *Service) EmitCampaignLifecycleOutbox(ctx context.Context, q db.Querier, campaignID uuid.UUID, status db.CampaignStatusType, budgetLimit int64) error {
	return campaign.EmitCampaignLifecycleOutbox(ctx, q, campaignID, status, budgetLimit)
}

func (s *Service) CampaignImportExportHost() campaign.ImportExportHost {
	return campaignImportExportHost{svc: s}
}

func (s *Service) CampaignRuntime() *campaign.Runtime {
	if s == nil {
		return nil
	}
	if s.campaignRuntime == nil {
		s.campaignRuntime = campaign.NewRuntime(s.pool, s)
		if s.clickhouseQuery != nil {
			s.campaignRuntime.SetClickHouseQuery(s.clickhouseQuery)
		}
	}
	return s.campaignRuntime
}

func formatOptionalText(t pgtype.Text) string {
	return campaign.FormatOptionalText(t)
}

func (s *Service) ListCampaigns(ctx context.Context, customerID uuid.UUID, status string, limit, offset int32) ([]CampaignDTO, int64, error) {
	return s.CampaignRuntime().ListCampaigns(ctx, customerID, status, limit, offset)
}

func (s *Service) GetCampaign(ctx context.Context, id uuid.UUID) (CampaignDTO, error) {
	return s.CampaignRuntime().GetCampaign(ctx, id)
}

func (s *Service) PatchCampaign(ctx context.Context, campaignID uuid.UUID, req PatchCampaignRequest) (CampaignDTO, error) {
	return s.CampaignRuntime().PatchCampaign(ctx, campaignID, req)
}

func (s *Service) ListCampaignEvents(ctx context.Context, campaignID uuid.UUID, limit, offset int32) ([]CampaignEventDTO, int64, error) {
	return s.CampaignRuntime().ListCampaignEvents(ctx, campaignID, limit, offset)
}

func (s *Service) ListStatusHistory(ctx context.Context, campaignID uuid.UUID, limit, offset int32) ([]StatusHistoryDTO, int64, error) {
	return s.CampaignRuntime().ListStatusHistory(ctx, campaignID, limit, offset)
}

func (s *Service) SetClickHouse(conn driver.Conn, cfg database.ClickHouseQueryConfig) {
	if conn != nil {
		s.clickhouseQuery = database.NewClickHouseQuery(conn, cfg)
		if cr := s.campaignRuntime; cr != nil {
			cr.SetClickHouseQuery(s.clickhouseQuery)
		}
	}
}

func (s *Service) SetClickHouseWrite(conn driver.Conn) {
	s.clickhouseWriteConn = conn
}

func (s *Service) UpdateCampaignPacing(ctx context.Context, campaignID uuid.UUID, newMode string) (CampaignDTO, error) {
	return s.CampaignRuntime().UpdateCampaignPacing(ctx, campaignID, newMode)
}

func (s *Service) GetCampaignStats(ctx context.Context, campaignID uuid.UUID, from, to time.Time, granularity string) (CampaignStatsDTO, error) {
	return s.CampaignRuntime().GetCampaignStats(ctx, campaignID, from, to, granularity)
}

func (s *Service) CreateCampaign(ctx context.Context, spec CampaignCreateSpec) (uuid.UUID, error) {
	return s.CampaignRuntime().CreateCampaign(ctx, spec)
}

func (s *Service) PauseCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error {
	return s.CampaignRuntime().PauseCampaign(ctx, campaignID, reason)
}

func (s *Service) PreviewPauseCampaign(ctx context.Context, campaignID uuid.UUID, reason string) (MutationPreview, error) {
	return campaign.PreviewPauseCampaign(ctx, s.pool, campaignID, reason)
}

func (s *Service) ResumeCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error {
	return s.CampaignRuntime().ResumeCampaign(ctx, campaignID, reason)
}

func (s *Service) PreviewResumeCampaign(ctx context.Context, campaignID uuid.UUID, reason string) (MutationPreview, error) {
	return campaign.PreviewResumeCampaign(ctx, s.pool, s, campaignID, reason)
}

func (s *Service) UpdateCampaignSchedule(ctx context.Context, campaignID uuid.UUID, startAt, endAt *time.Time, daypartHours []int16) error {
	return s.CampaignRuntime().UpdateCampaignSchedule(ctx, campaignID, startAt, endAt, daypartHours)
}

func (s *Service) clickHouseIngestionLag(ctx context.Context) (time.Duration, error) {
	return campaign.ClickHouseIngestionLag(ctx, s.clickhouseQuery)
}

func (s *Service) transitionCampaignStatus(ctx context.Context, q db.Querier, campaignID uuid.UUID, old, newStatus db.CampaignStatusType, reason string, budget int64) error {
	return campaign.TransitionCampaignStatus(ctx, s, q, campaignID, old, newStatus, reason, budget)
}

func (s *Service) RunCampaignSmoke(ctx context.Context, campaignID uuid.UUID) (CampaignSmokeResultDTO, error) {
	return campaign.RunCampaignSmoke(ctx, s, campaignID)
}

func (s *Service) SmokeServiceAvailable() bool {
	return s != nil && s.pool != nil
}

func (s *Service) AuthorizeCampaignSmoke(ctx context.Context, campaignID uuid.UUID) error {
	row, err := s.GetCampaignRow(ctx, campaignID)
	if err != nil {
		return err
	}
	return assertMediaBuyerCampaignAccess(ctx, row)
}

func (s *Service) TrackerPublicBaseURL() string {
	if s.cfg != nil {
		return strings.TrimSpace(s.cfg.LanderPublicBaseURL)
	}
	return ""
}

func (s *Service) EvaluateCampaignPublish(ctx context.Context, campaignID uuid.UUID) (CampaignPublishCheckDTO, error) {
	return s.CampaignRuntime().EvaluateCampaignPublish(ctx, campaignID)
}

func (s *Service) PublishCampaign(ctx context.Context, campaignID uuid.UUID, force bool) (CampaignDTO, error) {
	return s.CampaignRuntime().PublishCampaign(ctx, campaignID, force)
}

func (s *Service) GetCampaignIntegrationHealth(ctx context.Context, campaignID uuid.UUID) (IntegrationHealthDTO, error) {
	return campaign.GetCampaignIntegrationHealth(ctx, s.pool, s, campaignID)
}

func (s *Service) ListCampaignConversionMappings(ctx context.Context, campaignID uuid.UUID) ([]ConversionMappingDTO, error) {
	return campaign.ListCampaignConversionMappings(ctx, s.pool, campaignID)
}

func (s *Service) ReplaceCampaignConversionMappings(ctx context.Context, campaignID uuid.UUID, mappings []ConversionMappingDTO) ([]ConversionMappingDTO, error) {
	return campaign.ReplaceCampaignConversionMappings(ctx, s.pool, campaignID, mappings)
}

func (s *Service) AuditCampaignRevisionConflict(ctx context.Context, campaignID uuid.UUID, expectedRevision string) {
	campaign.AuditCampaignRevisionConflict(ctx, s.pool, s, campaignID, expectedRevision)
}

type supplyChainBridge struct {
	svc *Service
}

func (b supplyChainBridge) MapCampaignNotFound(err error) error {
	return mapNotFound(err, ErrCampaignNotFound)
}

func (b supplyChainBridge) AuditSupplyChainUpdate(ctx context.Context, q db.Querier, campaignID uuid.UUID, oldNodesJSON, newNodesJSON []byte) {
	var uid uuid.UUID
	if u, ok := GetUser(ctx); ok {
		uid = u.UserID
	}
	b.svc.AuditLog(ctx, q, uid, "UPDATE_CAMPAIGN_SUPPLY_CHAIN", "campaign", &campaignID, auditSupplyChainChange{
		OldNodes: json.RawMessage(oldNodesJSON),
		NewNodes: json.RawMessage(newNodesJSON),
	}, nil)
}

func (s *Service) GetCampaignSupplyChain(ctx context.Context, campaignID uuid.UUID) (CampaignSupplyChainDTO, error) {
	return campaign.GetCampaignSupplyChain(ctx, s.pool, supplyChainBridge{svc: s}, campaignID)
}

func (s *Service) UpdateCampaignSupplyChain(ctx context.Context, campaignID uuid.UUID, nodes []SupplyChainNode) (CampaignSupplyChainDTO, error) {
	return campaign.UpdateCampaignSupplyChain(ctx, s.pool, supplyChainBridge{svc: s}, campaignID, nodes)
}

func (s *Service) WizardStore() *campaign.WizardStore {
	if s == nil {
		return nil
	}
	if s.wizardStore == nil {
		s.wizardStore = campaign.NewWizardStore(s.pool, s)
	}
	return s.wizardStore
}

func (s *Service) CreateCampaignWizardSession(ctx context.Context, customerID uuid.UUID, templateKey string) (CampaignWizardSessionDTO, error) {
	return s.WizardStore().CreateCampaignWizardSession(ctx, customerID, templateKey)
}

func (s *Service) GetCampaignWizardSession(ctx context.Context, sessionID uuid.UUID) (CampaignWizardSessionDTO, error) {
	return s.WizardStore().GetCampaignWizardSession(ctx, sessionID)
}

func (s *Service) UpdateCampaignWizardSessionStep(ctx context.Context, sessionID uuid.UUID, step string, payload []byte) (CampaignWizardSessionDTO, error) {
	return s.WizardStore().UpdateCampaignWizardSessionStep(ctx, sessionID, step, payload)
}

func (s *Service) CommitCampaignWizardSession(ctx context.Context, sessionID uuid.UUID, idempotencyKey string, publish bool) (CampaignWizardCommitResult, error) {
	return s.WizardStore().CommitCampaignWizardSession(ctx, sessionID, idempotencyKey, publish)
}

func (s *Service) ApplyOnboardingTemplate(key string) (CampaignWizardStored, error) {
	return campaign.ApplyOnboardingTemplate(key)
}

func (s *Service) ImportBundledTemplate(ctx context.Context, schemaName string) error {
	entry, ok := integrationschema.FindCatalogEntry(schemaName)
	if !ok {
		return errValidation(fmt.Sprintf("integration schema %q not found in catalog", schemaName))
	}
	_, err := s.TemplateCatalog(s.pool).ImportCatalogEntry(ctx, entry)
	return err
}

func (s *Service) ApplyAffiliateNetworkTemplate(ctx context.Context, campaignID uuid.UUID, network, trackingDomain string) error {
	_, err := s.ApplyCampaignTemplates(ctx, campaignID, ApplyCampaignTemplatesRequest{
		AffiliateNetwork: network,
		TrackingDomain:   trackingDomain,
	})
	return err
}

func (s *Service) TrackingDomain(ctx context.Context, override string) string {
	if d := strings.TrimSpace(override); d != "" {
		return d
	}
	if cfg, _, err := s.GetPlatformConfig(ctx); err == nil {
		if d := strings.TrimSpace(cfg.TrackingDomain); d != "" {
			return d
		}
	}
	if s.cfg != nil {
		return strings.TrimSpace(s.cfg.LanderPublicBaseURL)
	}
	return ""
}

func (s *Service) InboundTargetURL(ctx context.Context, schemaName, trackingDomain string) (string, error) {
	schemaName = strings.TrimSpace(schemaName)
	if schemaName == "" {
		return "", nil
	}
	entry, ok := integrationschema.FindCatalogEntry(schemaName)
	if !ok {
		return "", errValidation(fmt.Sprintf("integration schema %q not found in catalog", schemaName))
	}
	_, kind, parsed, err := integrationschema.LoadBundledTemplate(entry)
	if err != nil {
		return "", errValidation(err.Error())
	}
	if kind != integrationschema.KindInboundTokens {
		return "", nil
	}
	inbound := parsed.(*integrationschema.InboundTokensSchema)
	return integrationschema.BuildInboundTrackingURL(s.TrackingDomain(ctx, trackingDomain), inbound), nil
}

func (s *Service) CampaignWorker() *campaign.Worker {
	if s == nil {
		return nil
	}
	if s.campaignWorker == nil {
		s.campaignWorker = campaign.NewWorker(s.CampaignRuntime(), s)
	}
	return s.campaignWorker
}

func (s *Service) MABInterval() time.Duration {
	if s == nil || s.cfg == nil || s.cfg.MABIntervalMs <= 0 {
		return 0
	}
	return time.Duration(s.cfg.MABIntervalMs) * time.Millisecond
}

func (s *Service) RunWithPostgresLow(ctx context.Context, fn func(context.Context) error) error {
	return s.withPostgresLow(ctx, fn)
}

func (s *Service) Pool() *pgxpool.Pool {
	return s.GetPool()
}

func (s *Service) PacingToleranceMargin() float64 {
	if s == nil || s.cfg == nil {
		return 0
	}
	return s.cfg.PacingToleranceMargin
}

func (s *Service) CampaignLocation(timezone string) *time.Location {
	return campaign.CampaignLocation(&s.timezoneLocationCache, timezone)
}

func (s *Service) PacingHourWeights(ctx context.Context) [24]float64 {
	return s.fetchPacingHourWeights(ctx)
}

func (s *Service) AuditPacingLoopAdjustment(ctx context.Context, q db.Querier, campaignID uuid.UUID, oldPacing, newPacing, spend, expected string) {
	s.AuditLog(ctx, q, uuid.Nil, "PACING_LOOP_ADJUSTMENT", "campaign", &campaignID, auditPacingLoopAdjustment{
		OldPacing: oldPacing,
		NewPacing: newPacing,
		Spend:     spend,
		Expected:  expected,
		Curve:     "daypart_weighted",
	}, nil)
}

func (s *Service) EmitBrandCreativesOutbox(ctx context.Context, q db.Querier, brandID uuid.UUID) error {
	return s.emitBrandCreativesOutbox(ctx, q, brandID)
}

func (s *Service) CreateCampaignTemplate(ctx context.Context, customerID uuid.UUID, name string, budgetLimit int64, pacing db.PacingModeType, dailyBudget int64, timezone string, freqLimit, freqWindow int32, targetCountries []string, brandID *uuid.UUID, daypartHours []int16) (uuid.UUID, error) {
	return s.CampaignRuntime().CreateCampaignTemplate(ctx, customerID, name, budgetLimit, pacing, dailyBudget, timezone, freqLimit, freqWindow, targetCountries, brandID, daypartHours)
}

func (s *Service) ListCampaignTemplates(ctx context.Context, customerID uuid.UUID, limit, offset int32) ([]CampaignTemplateDTO, int64, error) {
	return s.CampaignRuntime().ListCampaignTemplates(ctx, customerID, limit, offset)
}

func (s *Service) CreateCampaignFromTemplate(ctx context.Context, templateID uuid.UUID, customerID uuid.UUID, name string, budgetLimit *int64, idempotencyKey string) (uuid.UUID, error) {
	return s.CampaignRuntime().CreateCampaignFromTemplate(ctx, templateID, customerID, name, budgetLimit, idempotencyKey)
}

func (s *Service) SaveCampaignAsTemplate(ctx context.Context, campaignID uuid.UUID, templateName string) (uuid.UUID, error) {
	return s.CampaignRuntime().SaveCampaignAsTemplate(ctx, campaignID, templateName)
}

func (s *Service) ProcessScheduleTick(ctx context.Context) error {
	return s.CampaignWorker().ProcessScheduleTick(ctx)
}

func (s *Service) RunDeliveryOptimizerTick(ctx context.Context, syncWorkers []*domain.SyncWorker, runMAB bool) error {
	return s.CampaignWorker().RunDeliveryOptimizerTick(ctx, syncWorkers, runMAB)
}

func (s *Service) ClosedLoopPacingController(ctx context.Context, syncWorkers []*domain.SyncWorker) error {
	return s.CampaignWorker().ClosedLoopPacingController(ctx, syncWorkers)
}

func (s *Service) RunVPPPacingController(ctx context.Context) error {
	return campaign.RunVPPPacingController(ctx, s)
}

func (s *Service) CampaignShard(campaignID uuid.UUID) int {
	if s == nil || s.sharder == nil {
		return 0
	}
	return s.sharder.GetShard(campaignID)
}

func (s *Service) QueryVPPCampaignSamplesBatch(ctx context.Context, from, to time.Time, campaignIDs []uuid.UUID) (map[uuid.UUID][]campaign.VPPCampaignSample, error) {
	return campaign.QueryVPPCampaignSamplesBatch(ctx, s.clickhouseQuery, from, to, campaignIDs)
}

func (s *Service) DrainWaitTimeoutMs() int64 {
	if s == nil || s.cfg == nil || s.cfg.Lifecycle.WaitTimeoutMs <= 0 {
		return 100
	}
	return int64(s.cfg.Lifecycle.WaitTimeoutMs)
}

func (s *Service) FinalizeDrainingCampaign(ctx context.Context, q db.Querier, campaignID uuid.UUID, camp db.Campaign, reason string) error {
	return s.finalizeDrainingCampaign(ctx, q, campaignID, camp, reason)
}

func (s *Service) fetchPacingHourWeights(ctx context.Context) [24]float64 {
	if s.clickhouseQuery == nil {
		return campaign.UniformHourWeights()
	}
	lookbackEnd := time.Now().UTC().Truncate(time.Hour)
	lookbackStart := lookbackEnd.Add(-campaign.PacingLookbackDays * 24 * time.Hour)
	_, samples, err := reports.QueryForecastHourlySamples(ctx, forecastHost{svc: s}, lookbackStart, lookbackEnd, nil)
	if err != nil {
		return campaign.UniformHourWeights()
	}
	return reports.BuildHourWeights(samples)
}
