package campaign

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostbacksSnapshotDTO struct {
	Configs        []PostbackConfigDTO         `json:"configs"`
	Dlq            []PostbackDlqDTO            `json:"dlq"`
	CampaignStatus []PostbackCampaignStatusDTO `json:"campaignStatus"`
}

func ListPostbackConfigs(ctx context.Context, pool *pgxpool.Pool) ([]PostbackConfigDTO, error) {
	if pool == nil {
		return nil, errors.New("postgres pool not configured")
	}
	configs, err := db.New(pool).ListPostbackConfigs(ctx)
	if err != nil {
		return nil, err
	}
	return postbackConfigDTOsFromRows(configs), nil
}

func ListPostbackConfigsForCampaigns(ctx context.Context, pool *pgxpool.Pool, campaignIDs []uuid.UUID) ([]PostbackConfigDTO, error) {
	if pool == nil {
		return nil, errors.New("postgres pool not configured")
	}
	if len(campaignIDs) == 0 {
		return []PostbackConfigDTO{}, nil
	}
	pgIDs := make([]pgtype.UUID, len(campaignIDs))
	for i, id := range campaignIDs {
		pgIDs[i] = domain.ToUUID(id)
	}
	configs, err := db.New(pool).ListPostbackConfigsByCampaignIDs(ctx, pgIDs)
	if err != nil {
		return nil, err
	}
	return postbackConfigDTOsFromRows(configs), nil
}

func postbackConfigDTOsFromRows(configs []db.PostbackConfig) []PostbackConfigDTO {
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
	return dtos
}

func ListPostbackDlqEntries(ctx context.Context, pool *pgxpool.Pool) ([]PostbackDlqDTO, error) {
	if pool == nil {
		return nil, errors.New("postgres pool not configured")
	}
	dlqs, err := db.New(pool).ListPostbackDLQ(ctx)
	if err != nil {
		return nil, err
	}
	return postbackDlqDTOsFromRows(dlqs), nil
}

func ListPostbackDlqEntriesForCampaigns(ctx context.Context, pool *pgxpool.Pool, campaignIDs []uuid.UUID) ([]PostbackDlqDTO, error) {
	all, err := ListPostbackDlqEntries(ctx, pool)
	if err != nil {
		return nil, err
	}
	if len(campaignIDs) == 0 {
		return []PostbackDlqDTO{}, nil
	}
	allowed := campaignIDSet(campaignIDs)
	out := make([]PostbackDlqDTO, 0, len(all))
	for _, dto := range all {
		id, err := uuid.Parse(dto.CampaignID)
		if err != nil {
			continue
		}
		if _, ok := allowed[id]; ok {
			out = append(out, dto)
		}
	}
	return out, nil
}

func postbackDlqDTOsFromRows(dlqs []db.PostbackDlq) []PostbackDlqDTO {
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
	return dtos
}

func ListPostbackCampaignStatusRows(ctx context.Context, pool *pgxpool.Pool) ([]PostbackCampaignStatusDTO, error) {
	if pool == nil {
		return nil, errors.New("postgres pool not configured")
	}
	rows, err := db.New(pool).ListPostbackCampaignStatus(ctx)
	if err != nil {
		return nil, err
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
	return out, nil
}

func ListPostbackCampaignStatusForCampaigns(ctx context.Context, pool *pgxpool.Pool, campaignIDs []uuid.UUID) ([]PostbackCampaignStatusDTO, error) {
	all, err := ListPostbackCampaignStatusRows(ctx, pool)
	if err != nil {
		return nil, err
	}
	if len(campaignIDs) == 0 {
		return []PostbackCampaignStatusDTO{}, nil
	}
	allowed := campaignIDSet(campaignIDs)
	out := make([]PostbackCampaignStatusDTO, 0, len(all))
	for _, row := range all {
		id, err := uuid.Parse(row.CampaignID)
		if err != nil {
			continue
		}
		if _, ok := allowed[id]; ok {
			out = append(out, row)
		}
	}
	return out, nil
}

func campaignIDSet(ids []uuid.UUID) map[uuid.UUID]struct{} {
	out := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		out[id] = struct{}{}
	}
	return out
}

func (h *PostbackHTTPHandlers) postbackCampaignScope(ctx context.Context) (allCampaigns bool, ids []uuid.UUID, err error) {
	return SessionScopedCampaignIDs(ctx, h.Pool)
}

func (h *PostbackHTTPHandlers) authorizePostbackCampaign(w http.ResponseWriter, r *http.Request, campaignID uuid.UUID) bool {
	if h.AuthorizeCampaignAccess == nil {
		return true
	}
	if err := h.AuthorizeCampaignAccess(r, campaignID); err != nil {
		h.writePostbackError(w, err)
		return false
	}
	return true
}

func (h *PostbackHTTPHandlers) writePostbackError(w http.ResponseWriter, err error) {
	if h.WriteServiceError != nil {
		h.WriteServiceError(w, err)
		return
	}
	if errors.Is(err, ErrForbidden) {
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden")
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
}

func (h *PostbackHTTPHandlers) listScopedPostbackConfigs(ctx context.Context) ([]PostbackConfigDTO, error) {
	allCampaigns, ids, err := h.postbackCampaignScope(ctx)
	if err != nil {
		return nil, err
	}
	if allCampaigns {
		return ListPostbackConfigs(ctx, h.Pool)
	}
	return ListPostbackConfigsForCampaigns(ctx, h.Pool, ids)
}

func (h *PostbackHTTPHandlers) listScopedPostbackDlq(ctx context.Context) ([]PostbackDlqDTO, error) {
	allCampaigns, ids, err := h.postbackCampaignScope(ctx)
	if err != nil {
		return nil, err
	}
	if allCampaigns {
		return ListPostbackDlqEntries(ctx, h.Pool)
	}
	return ListPostbackDlqEntriesForCampaigns(ctx, h.Pool, ids)
}

func (h *PostbackHTTPHandlers) listScopedPostbackCampaignStatus(ctx context.Context) ([]PostbackCampaignStatusDTO, error) {
	allCampaigns, ids, err := h.postbackCampaignScope(ctx)
	if err != nil {
		return nil, err
	}
	if allCampaigns {
		return ListPostbackCampaignStatusRows(ctx, h.Pool)
	}
	return ListPostbackCampaignStatusForCampaigns(ctx, h.Pool, ids)
}

func (h *PostbackHTTPHandlers) listScopedPostbackHealth(ctx context.Context) ([]PostbackHealthRowDTO, error) {
	allCampaigns, ids, err := h.postbackCampaignScope(ctx)
	if err != nil {
		return nil, err
	}
	if allCampaigns {
		return ListPostbackHealthRows(ctx, h.Pool)
	}
	return ListPostbackHealthRowsForCampaigns(ctx, h.Pool, ids)
}

func (h *PostbackHTTPHandlers) getPostbackHealth(w http.ResponseWriter, r *http.Request) {
	rows, err := h.listScopedPostbackHealth(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if rows == nil {
		rows = []PostbackHealthRowDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, PostbackHealthResponseDTO{
		Rows:                      rows,
		AlertThresholdSuccessRate: postbackHealthAlertSuccessRate,
		RunbookPath:               "/docs/INTEGRATIONS.md#postback-health",
	})
}

func (h *PostbackHTTPHandlers) getPostbacksSnapshot(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var snap PostbacksSnapshotDTO
	var configsErr, dlqErr, statusErr error
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		snap.Configs, configsErr = h.listScopedPostbackConfigs(ctx)
	}()
	go func() {
		defer wg.Done()
		snap.Dlq, dlqErr = h.listScopedPostbackDlq(ctx)
	}()
	go func() {
		defer wg.Done()
		snap.CampaignStatus, statusErr = h.listScopedPostbackCampaignStatus(ctx)
	}()
	wg.Wait()

	if err := errors.Join(configsErr, dlqErr, statusErr); err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if snap.Configs == nil {
		snap.Configs = []PostbackConfigDTO{}
	}
	if snap.Dlq == nil {
		snap.Dlq = []PostbackDlqDTO{}
	}
	if snap.CampaignStatus == nil {
		snap.CampaignStatus = []PostbackCampaignStatusDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, snap)
}
