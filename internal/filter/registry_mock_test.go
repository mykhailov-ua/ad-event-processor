package filter

import (
	"context"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type MockRepo struct {
	db.Querier
	ids     []pgtype.UUID
	err     error
	budgets map[uuid.UUID]db.GetCampaignBudgetRow
	full    map[uuid.UUID]db.GetCampaignFullRow
}

func (m *MockRepo) GetCampaignFull(ctx context.Context, id pgtype.UUID) (db.GetCampaignFullRow, error) {
	if m.err != nil {
		return db.GetCampaignFullRow{}, m.err
	}
	uid := uuid.UUID(id.Bytes)
	if m.full != nil {
		if row, ok := m.full[uid]; ok {
			return row, nil
		}
	}
	if m.budgets != nil {
		if row, ok := m.budgets[uid]; ok {
			return db.GetCampaignFullRow{
				ID:           row.ID,
				CustomerID:   row.CustomerID,
				BudgetLimit:  row.BudgetLimit,
				CurrentSpend: row.CurrentSpend,
				Status:       row.Status,
			}, nil
		}
	}
	return db.GetCampaignFullRow{
		ID:           id,
		CustomerID:   id,
		BudgetLimit:  1000,
		CurrentSpend: 100,
		Status:       db.CampaignStatusTypeACTIVE,
	}, nil
}

func (m *MockRepo) ListActiveCampaigns(ctx context.Context) ([]db.ListActiveCampaignsRow, error) {
	var res []db.ListActiveCampaignsRow
	for _, id := range m.ids {
		res = append(res, db.ListActiveCampaignsRow{
			ID:         id,
			CustomerID: id,
			Status:     db.CampaignStatusTypeACTIVE,
		})
	}
	return res, m.err
}

func (m *MockRepo) GetCampaignBudget(ctx context.Context, id pgtype.UUID) (db.GetCampaignBudgetRow, error) {
	if m.err != nil {
		return db.GetCampaignBudgetRow{}, m.err
	}
	uid := uuid.UUID(id.Bytes)
	if m.budgets != nil {
		if row, ok := m.budgets[uid]; ok {
			return row, nil
		}
	}
	return db.GetCampaignBudgetRow{
		ID:           id,
		CustomerID:   id,
		BudgetLimit:  1000,
		CurrentSpend: 100,
		Status:       db.CampaignStatusTypeACTIVE,
	}, nil
}
