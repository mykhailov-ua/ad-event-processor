package billingadmin

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"sync"

	"ad-event-processor/internal/costsync"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type CostSyncSnapshotDTO struct {
	Networks    []costsync.NetworkCredentialSchema `json:"networks"`
	Credentials []CostSyncCredentialDTO            `json:"credentials"`
	History     []CostSyncRunDTO                   `json:"history"`
}

func (h *CostSyncHTTPHandlers) listCostSyncCredentials(ctx context.Context, customerID string) ([]CostSyncCredentialDTO, error) {
	if h == nil || h.Pool == nil {
		return nil, errors.New("cost sync handler not configured")
	}
	q := db.New(h.Pool)
	var rows []db.CostSyncCredential
	var err error

	if customerID != "" {
		custID, parseErr := uuid.Parse(customerID)
		if parseErr != nil {
			return nil, parseErr
		}
		rows, err = q.ListCostSyncCredentialsByCustomer(ctx, pgtype.UUID{Bytes: custID, Valid: true})
	} else {
		rows, err = q.ListCostSyncCredentials(ctx)
	}
	if err != nil {
		return nil, err
	}

	dtos := make([]CostSyncCredentialDTO, 0, len(rows))
	for _, row := range rows {
		dto, err := costSyncCredentialDTO(row)
		if err != nil {
			return nil, err
		}
		dtos = append(dtos, dto)
	}
	return dtos, nil
}

func (h *CostSyncHTTPHandlers) listCostSyncHistoryRows(
	ctx context.Context,
	customerID string,
	limit int32,
	offset int32,
) ([]CostSyncRunDTO, error) {
	if h == nil || h.Pool == nil {
		return nil, errors.New("cost sync handler not configured")
	}
	var cust pgtype.UUID
	if customerID != "" {
		cid, err := uuid.Parse(customerID)
		if err != nil {
			return nil, err
		}
		cust = pgtype.UUID{Bytes: cid, Valid: true}
	}

	rows, err := db.New(h.Pool).ListCostSyncRuns(ctx, db.ListCostSyncRunsParams{
		Column1: cust,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, err
	}

	dtos := make([]CostSyncRunDTO, 0, len(rows))
	for _, row := range rows {
		dto := CostSyncRunDTO{
			ID:                  row.ID,
			CustomerID:          pgUUIDToString(row.CustomerID),
			Network:             row.Network,
			CostDate:            row.CostDate.Time.Format("2006-01-02"),
			Status:              row.Status,
			RowsImported:        row.RowsImported,
			TotalAmountUSDMicro: row.TotalAmountUsdMicro,
			TriggerSource:       row.TriggerSource,
			StartedAt:           row.StartedAt.Time,
		}
		if row.ErrorMessage.Valid {
			dto.ErrorMessage = row.ErrorMessage.String
		}
		if row.CompletedAt.Valid {
			t := row.CompletedAt.Time
			dto.CompletedAt = &t
		}
		dtos = append(dtos, dto)
	}
	return dtos, nil
}

func (h *CostSyncHTTPHandlers) getCostSyncSnapshot(w http.ResponseWriter, r *http.Request) {
	customerID := r.URL.Query().Get("customer_id")
	if customerID != "" {
		if _, err := uuid.Parse(customerID); err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
			return
		}
	}
	limit := int32(50)
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 32); err == nil && parsed > 0 {
			limit = int32(parsed)
		}
	}

	ctx := r.Context()
	snap := CostSyncSnapshotDTO{
		Networks: costsync.ListNetworkCredentialSchemas(),
	}
	var credErr, histErr error
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		snap.Credentials, credErr = h.listCostSyncCredentials(ctx, customerID)
	}()
	go func() {
		defer wg.Done()
		snap.History, histErr = h.listCostSyncHistoryRows(ctx, customerID, limit, 0)
	}()
	wg.Wait()

	if credErr != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", credErr.Error())
		return
	}
	if histErr != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", histErr.Error())
		return
	}
	if snap.Credentials == nil {
		snap.Credentials = []CostSyncCredentialDTO{}
	}
	if snap.History == nil {
		snap.History = []CostSyncRunDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, snap)
}
