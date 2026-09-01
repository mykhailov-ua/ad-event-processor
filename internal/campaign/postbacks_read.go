package campaign

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
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
	return dtos, nil
}

func ListPostbackDlqEntries(ctx context.Context, pool *pgxpool.Pool) ([]PostbackDlqDTO, error) {
	if pool == nil {
		return nil, errors.New("postgres pool not configured")
	}
	dlqs, err := db.New(pool).ListPostbackDLQ(ctx)
	if err != nil {
		return nil, err
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
	return dtos, nil
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

func (h *PostbackHTTPHandlers) getPostbacksSnapshot(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var snap PostbacksSnapshotDTO
	var configsErr, dlqErr, statusErr error
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		snap.Configs, configsErr = ListPostbackConfigs(ctx, h.Pool)
	}()
	go func() {
		defer wg.Done()
		snap.Dlq, dlqErr = ListPostbackDlqEntries(ctx, h.Pool)
	}()
	go func() {
		defer wg.Done()
		snap.CampaignStatus, statusErr = ListPostbackCampaignStatusRows(ctx, h.Pool)
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
