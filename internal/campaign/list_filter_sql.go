package campaign

import (
	"strings"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func optionalCampaignSearchText(searchQuery string) pgtype.Text {
	searchQuery = strings.TrimSpace(searchQuery)
	if searchQuery == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: searchQuery, Valid: true}
}

func optionalCampaignPacingMode(pacingMode string) pgtype.Text {
	pacingMode = strings.TrimSpace(pacingMode)
	if pacingMode == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: pacingMode, Valid: true}
}

func campaignListSortField(sortField string) pgtype.Text {
	sortField = strings.TrimSpace(sortField)
	if sortField == "" {
		sortField = "updated_at"
	}
	return pgtype.Text{String: sortField, Valid: true}
}

func campaignListSortDesc(order string) pgtype.Bool {
	return pgtype.Bool{Bool: strings.EqualFold(strings.TrimSpace(order), "desc"), Valid: true}
}

func CampaignCountParamsFromFilter(filter ListCampaignsFilter) db.CountCampaignsParams {
	var customerID pgtype.UUID
	if filter.CustomerID != uuid.Nil {
		customerID = domain.ToUUID(filter.CustomerID)
	}
	var status pgtype.Text
	if filter.Status != "" {
		status = pgtype.Text{String: filter.Status, Valid: true}
	}
	var targetCountry pgtype.Text
	if filter.TargetCountry != "" {
		targetCountry = pgtype.Text{String: filter.TargetCountry, Valid: true}
	}
	return db.CountCampaignsParams{
		CustomerID:     customerID,
		Status:         status,
		OwnerUserID:    filter.OwnerUserID,
		TargetCountry:  targetCountry,
		BudgetMinMicro: filter.BudgetMinMicro,
		BudgetMaxMicro: filter.BudgetMaxMicro,
		SearchQuery:    optionalCampaignSearchText(filter.SearchQuery),
		PacingMode:     optionalCampaignPacingMode(filter.PacingMode),
	}
}

func CampaignCountStatusTotalsParamsFromFilter(filter ListCampaignsFilter) db.CountCampaignsStatusTotalsParams {
	var customerID pgtype.UUID
	if filter.CustomerID != uuid.Nil {
		customerID = domain.ToUUID(filter.CustomerID)
	}
	var targetCountry pgtype.Text
	if filter.TargetCountry != "" {
		targetCountry = pgtype.Text{String: filter.TargetCountry, Valid: true}
	}
	return db.CountCampaignsStatusTotalsParams{
		CustomerID:     customerID,
		OwnerUserID:    filter.OwnerUserID,
		TargetCountry:  targetCountry,
		BudgetMinMicro: filter.BudgetMinMicro,
		BudgetMaxMicro: filter.BudgetMaxMicro,
		SearchQuery:    optionalCampaignSearchText(filter.SearchQuery),
		PacingMode:     optionalCampaignPacingMode(filter.PacingMode),
	}
}

func CampaignListKeysParamsFromFilter(filter ListCampaignsFilter) db.ListCampaignListKeysForFilterParams {
	var customerID pgtype.UUID
	if filter.CustomerID != uuid.Nil {
		customerID = domain.ToUUID(filter.CustomerID)
	}
	var status pgtype.Text
	if filter.Status != "" {
		status = pgtype.Text{String: filter.Status, Valid: true}
	}
	var targetCountry pgtype.Text
	if filter.TargetCountry != "" {
		targetCountry = pgtype.Text{String: filter.TargetCountry, Valid: true}
	}
	return db.ListCampaignListKeysForFilterParams{
		CustomerID:     customerID,
		Status:         status,
		OwnerUserID:    filter.OwnerUserID,
		TargetCountry:  targetCountry,
		BudgetMinMicro: filter.BudgetMinMicro,
		BudgetMaxMicro: filter.BudgetMaxMicro,
		SearchQuery:    optionalCampaignSearchText(filter.SearchQuery),
		PacingMode:     optionalCampaignPacingMode(filter.PacingMode),
	}
}

func CampaignCountFlowsParamsFromFilter(filter ListCampaignsFilter) db.CountCampaignFlowsForFilterParams {
	var customerID pgtype.UUID
	if filter.CustomerID != uuid.Nil {
		customerID = domain.ToUUID(filter.CustomerID)
	}
	var status pgtype.Text
	if filter.Status != "" {
		status = pgtype.Text{String: filter.Status, Valid: true}
	}
	var targetCountry pgtype.Text
	if filter.TargetCountry != "" {
		targetCountry = pgtype.Text{String: filter.TargetCountry, Valid: true}
	}
	return db.CountCampaignFlowsForFilterParams{
		CustomerID:     customerID,
		Status:         status,
		OwnerUserID:    filter.OwnerUserID,
		TargetCountry:  targetCountry,
		BudgetMinMicro: filter.BudgetMinMicro,
		BudgetMaxMicro: filter.BudgetMaxMicro,
		SearchQuery:    optionalCampaignSearchText(filter.SearchQuery),
		PacingMode:     optionalCampaignPacingMode(filter.PacingMode),
	}
}

func campaignListBaseParams(filter ListCampaignsFilter) (
	customerID pgtype.UUID,
	status pgtype.Text,
	targetCountry pgtype.Text,
) {
	if filter.CustomerID != uuid.Nil {
		customerID = domain.ToUUID(filter.CustomerID)
	}
	if filter.Status != "" {
		status = pgtype.Text{String: filter.Status, Valid: true}
	}
	if filter.TargetCountry != "" {
		targetCountry = pgtype.Text{String: filter.TargetCountry, Valid: true}
	}
	return customerID, status, targetCountry
}

func CampaignListParamsFromFilter(filter ListCampaignsFilter) db.ListCampaignsParams {
	customerID, status, targetCountry := campaignListBaseParams(filter)
	return db.ListCampaignsParams{
		Limit:          filter.Limit,
		Offset:         filter.Offset,
		CustomerID:     customerID,
		Status:         status,
		OwnerUserID:    filter.OwnerUserID,
		TargetCountry:  targetCountry,
		BudgetMinMicro: filter.BudgetMinMicro,
		BudgetMaxMicro: filter.BudgetMaxMicro,
		SearchQuery:    optionalCampaignSearchText(filter.SearchQuery),
		PacingMode:     optionalCampaignPacingMode(filter.PacingMode),
		SortField:      campaignListSortField(filter.SortField),
		SortDesc:       campaignListSortDesc(filter.SortOrder),
	}
}

func CampaignListSortedByStatsParamsFromFilter(filter ListCampaignsFilter) db.ListCampaignsSortedByStatsParams {
	customerID, status, targetCountry := campaignListBaseParams(filter)
	return db.ListCampaignsSortedByStatsParams{
		Limit:          filter.Limit,
		Offset:         filter.Offset,
		CustomerID:     customerID,
		Status:         status,
		OwnerUserID:    filter.OwnerUserID,
		TargetCountry:  targetCountry,
		BudgetMinMicro: filter.BudgetMinMicro,
		BudgetMaxMicro: filter.BudgetMaxMicro,
		SearchQuery:    optionalCampaignSearchText(filter.SearchQuery),
		PacingMode:     optionalCampaignPacingMode(filter.PacingMode),
		SortField:      campaignListSortField(filter.SortField),
		SortDesc:       campaignListSortDesc(filter.SortOrder),
		StatsFrom:      filter.StatsFrom,
		StatsTo:        filter.StatsTo,
	}
}
