package campaign

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ad-event-processor/internal/costsync"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const manualCostNetwork = "manual"

type ManualCampaignCostRequest struct {
	CostDate    string `json:"cost_date"`
	AmountMicro int64  `json:"amount_micro"`
	Currency    string `json:"currency"`
	PlacementID string `json:"placement_id,omitempty"`
}

type ManualCampaignCostResponse struct {
	CampaignID  string `json:"campaign_id"`
	CostDate    string `json:"cost_date"`
	AmountMicro int64  `json:"amount_micro"`
	Currency    string `json:"currency"`
	PlacementID string `json:"placement_id"`
}

func PutManualCampaignCost(
	ctx context.Context,
	pool *pgxpool.Pool,
	customerID uuid.UUID,
	campaignID uuid.UUID,
	req ManualCampaignCostRequest,
) (ManualCampaignCostResponse, error) {
	if pool == nil {
		return ManualCampaignCostResponse{}, errServiceUnavailable()
	}
	if customerID == uuid.Nil || campaignID == uuid.Nil {
		return ManualCampaignCostResponse{}, errValidation("customer_id and campaign_id are required")
	}
	if req.AmountMicro < 0 {
		return ManualCampaignCostResponse{}, errValidation("amount_micro must be non-negative")
	}
	costDate, err := time.Parse("2006-01-02", strings.TrimSpace(req.CostDate))
	if err != nil {
		return ManualCampaignCostResponse{}, errValidation("cost_date must be YYYY-MM-DD")
	}
	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if currency == "" {
		currency = "USD"
	}
	placementID := strings.TrimSpace(req.PlacementID)
	if placementID == "" {
		placementID = "manual"
	}
	line := costsync.CostLine{
		CustomerID:  customerID,
		CampaignID:  campaignID,
		Date:        costDate,
		Network:     manualCostNetwork,
		PlacementID: placementID,
		LineType:    costsync.LineTypeSpend,
		AmountMicro: req.AmountMicro,
		Currency:    currency,
	}
	imported, err := costsync.InsertManualCostLines(ctx, pool, []costsync.CostLine{line})
	if err != nil {
		return ManualCampaignCostResponse{}, err
	}
	if imported == 0 {
		return ManualCampaignCostResponse{}, fmt.Errorf("manual cost not written")
	}
	return ManualCampaignCostResponse{
		CampaignID:  campaignID.String(),
		CostDate:    costDate.Format("2006-01-02"),
		AmountMicro: req.AmountMicro,
		Currency:    currency,
		PlacementID: placementID,
	}, nil
}

func GetCampaignCustomerID(ctx context.Context, pool *pgxpool.Pool, campaignID uuid.UUID) (uuid.UUID, error) {
	if pool == nil {
		return uuid.Nil, errServiceUnavailable()
	}
	row, err := db.New(pool).GetCampaign(ctx, domain.ToUUID(campaignID))
	if err != nil {
		return uuid.Nil, err
	}
	if !row.CustomerID.Valid {
		return uuid.Nil, errValidation("campaign customer missing")
	}
	return uuid.UUID(row.CustomerID.Bytes), nil
}
