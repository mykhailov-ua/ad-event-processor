package campaign

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ConversionMappingService interface {
	ListCampaignConversionMappings(ctx context.Context, campaignID uuid.UUID) ([]ConversionMappingDTO, error)
	ReplaceCampaignConversionMappings(ctx context.Context, campaignID uuid.UUID, mappings []ConversionMappingDTO) ([]ConversionMappingDTO, error)
}

func (h *CampaignsHTTPHandlers) registerConversionMappingRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func([]string, http.HandlerFunc) http.HandlerFunc) {
	if h == nil || h.ConversionMappings == nil {
		return
	}
	read := []string{"campaigns:read", "campaigns:read:masked"}
	mux.HandleFunc("GET /api/v1/campaigns/{id}/conversion-mappings", limit(perm(read, h.getConversionMappings)))
	mux.HandleFunc("PUT /api/v1/campaigns/{id}/conversion-mappings", limit(perm([]string{"campaigns:write"}, h.putConversionMappings)))
}

func (h *CampaignsHTTPHandlers) getConversionMappings(w http.ResponseWriter, r *http.Request) {
	campaignID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign id")
		return
	}
	if h.AuthorizeCampaignAccess != nil {
		if err := h.AuthorizeCampaignAccess(r, campaignID); err != nil {
			h.WriteHandlerError(w, err)
			return
		}
	}
	rows, err := h.ConversionMappings.ListCampaignConversionMappings(r.Context(), campaignID)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	if rows == nil {
		rows = []ConversionMappingDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, ConversionMappingListResponse{Mappings: rows})
}

func (h *CampaignsHTTPHandlers) putConversionMappings(w http.ResponseWriter, r *http.Request) {
	campaignID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign id")
		return
	}
	if h.AuthorizeCampaignAccess != nil {
		if err := h.AuthorizeCampaignAccess(r, campaignID); err != nil {
			h.WriteHandlerError(w, err)
			return
		}
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[ReplaceConversionMappingsRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	rows, err := h.ConversionMappings.ReplaceCampaignConversionMappings(r.Context(), campaignID, req.Mappings)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	if rows == nil {
		rows = []ConversionMappingDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, ConversionMappingListResponse{Mappings: rows})
}

func NormalizeConversionMappings(mappings []ConversionMappingDTO) ([]ConversionMappingDTO, error) {
	if len(mappings) == 0 {
		return []ConversionMappingDTO{}, nil
	}
	seen := make(map[string]struct{}, len(mappings))
	out := make([]ConversionMappingDTO, 0, len(mappings))
	for i := range mappings {
		row := mappings[i]
		status := strings.ToLower(strings.TrimSpace(row.InboundStatus))
		if status == "" {
			return nil, fmt.Errorf("inbound_status is required")
		}
		goal := strings.TrimSpace(row.GoalName)
		if goal == "" {
			goal = status
		}
		if row.PayoutMicro < 0 {
			return nil, fmt.Errorf("payout_micro must be non-negative")
		}
		if _, dup := seen[status]; dup {
			return nil, fmt.Errorf("duplicate inbound_status %q", status)
		}
		seen[status] = struct{}{}
		out = append(out, ConversionMappingDTO{
			InboundStatus: status,
			GoalName:      goal,
			PayoutMicro:   row.PayoutMicro,
		})
	}
	return out, nil
}

func ConversionMappingToDTO(row *db.CampaignConversionMapping) ConversionMappingDTO {
	if row == nil {
		return ConversionMappingDTO{}
	}
	return ConversionMappingDTO{
		InboundStatus: row.InboundStatus,
		GoalName:      row.GoalName,
		PayoutMicro:   row.PayoutMicro,
	}
}

func ListCampaignConversionMappings(ctx context.Context, pool *pgxpool.Pool, campaignID uuid.UUID) ([]ConversionMappingDTO, error) {
	if pool == nil {
		return nil, errServiceUnavailable()
	}
	rows, err := db.New(pool).ListConversionMappingsByCampaign(ctx, domain.ToUUID(campaignID))
	if err != nil {
		return nil, err
	}
	out := make([]ConversionMappingDTO, 0, len(rows))
	for i := range rows {
		out = append(out, ConversionMappingToDTO(&rows[i]))
	}
	return out, nil
}

func ReplaceCampaignConversionMappings(ctx context.Context, pool *pgxpool.Pool, campaignID uuid.UUID, mappings []ConversionMappingDTO) ([]ConversionMappingDTO, error) {
	if pool == nil {
		return nil, errServiceUnavailable()
	}
	normalized, err := NormalizeConversionMappings(mappings)
	if err != nil {
		return nil, err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := db.New(tx)
	if err := q.DeleteConversionMappingsByCampaign(ctx, domain.ToUUID(campaignID)); err != nil {
		return nil, err
	}
	for i := range normalized {
		row := &normalized[i]
		if err := q.InsertConversionMapping(ctx, db.InsertConversionMappingParams{
			CampaignID:    domain.ToUUID(campaignID),
			InboundStatus: row.InboundStatus,
			GoalName:      row.GoalName,
			PayoutMicro:   row.PayoutMicro,
		}); err != nil {
			return nil, fmt.Errorf("insert conversion mapping: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return normalized, nil
}
