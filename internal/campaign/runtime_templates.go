package campaign

import (
	"context"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func (r *Runtime) CreateCampaignTemplate(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	budgetLimit int64,
	pacing db.PacingModeType,
	dailyBudget int64,
	timezone string,
	freqLimit, freqWindow int32,
	targetCountries []string,
	brandID *uuid.UUID,
	daypartHours []int16,
) (uuid.UUID, error) {
	if r == nil {
		return uuid.Nil, errServiceUnavailable()
	}
	return createCampaignTemplate(ctx, r.poolOrNil(), customerID, name, budgetLimit, pacing, dailyBudget, timezone, freqLimit, freqWindow, targetCountries, brandID, daypartHours)
}

func (r *Runtime) ListCampaignTemplates(ctx context.Context, customerID uuid.UUID, limit, offset int32) ([]CampaignTemplateDTO, int64, error) {
	return listCampaignTemplates(ctx, r.poolOrNil(), customerID, limit, offset)
}

func (r *Runtime) CreateCampaignFromTemplate(ctx context.Context, templateID, customerID uuid.UUID, name string, budgetLimit *int64, idempotencyKey string) (uuid.UUID, error) {
	if r == nil || r.effects == nil {
		return uuid.Nil, errServiceUnavailable()
	}
	return createCampaignFromTemplate(ctx, r.poolOrNil(), r, templateID, customerID, name, budgetLimit, idempotencyKey)
}

func (r *Runtime) SaveCampaignAsTemplate(ctx context.Context, campaignID uuid.UUID, templateName string) (uuid.UUID, error) {
	if r == nil || r.effects == nil {
		return uuid.Nil, errServiceUnavailable()
	}
	return saveCampaignAsTemplate(ctx, r.poolOrNil(), r.effects, r, campaignID, templateName)
}

func createCampaignTemplate(
	ctx context.Context,
	pool *pgxpool.Pool,
	customerID uuid.UUID,
	name string,
	budgetLimit int64,
	pacing db.PacingModeType,
	dailyBudget int64,
	timezone string,
	freqLimit, freqWindow int32,
	targetCountries []string,
	brandID *uuid.UUID,
	daypartHours []int16,
) (uuid.UUID, error) {
	if err := validateDaypartHours(daypartHours); err != nil {
		return uuid.Nil, err
	}
	if pool == nil {
		return uuid.Nil, errServiceUnavailable()
	}
	templateID, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, err
	}

	var brandParam pgtype.UUID
	if brandID != nil {
		brandParam = domain.ToUUID(*brandID)
	}

	_, err = db.New(pool).CreateCampaignTemplate(ctx, db.CreateCampaignTemplateParams{
		ID:              domain.ToUUID(templateID),
		CustomerID:      domain.ToUUID(customerID),
		Name:            name,
		BudgetLimit:     budgetLimit,
		PacingMode:      pacing,
		DailyBudget:     dailyBudget,
		Timezone:        timezone,
		FreqLimit:       freqLimit,
		FreqWindow:      freqWindow,
		TargetCountries: countriesOrEmpty(targetCountries),
		BrandID:         brandParam,
		DaypartHours:    DaypartOrEmpty(daypartHours),
	})
	return templateID, err
}

func listCampaignTemplates(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID, limit, offset int32) ([]CampaignTemplateDTO, int64, error) {
	if pool == nil {
		return nil, 0, errServiceUnavailable()
	}
	q := db.New(pool)
	cid := domain.ToUUID(customerID)
	listParams := db.ListCampaignTemplatesParams{
		CustomerID: cid,
		Limit:      limit,
		Offset:     offset,
	}
	return coldpath.PaginatedList(
		func() (int64, error) { return q.CountCampaignTemplates(ctx, cid) },
		func() ([]db.CampaignTemplate, error) { return q.ListCampaignTemplates(ctx, listParams) },
		campaignTemplateToDTO,
	)
}

func campaignTemplateToDTO(t db.CampaignTemplate) CampaignTemplateDTO {
	countries := t.TargetCountries
	if countries == nil {
		countries = []string{}
	}
	hours := t.DaypartHours
	if hours == nil {
		hours = []int16{}
	}
	var brandID string
	if t.BrandID.Valid {
		brandID = uuid.UUID(t.BrandID.Bytes).String()
	}
	return CampaignTemplateDTO{
		ID:              uuid.UUID(t.ID.Bytes).String(),
		CustomerID:      uuid.UUID(t.CustomerID.Bytes).String(),
		Name:            t.Name,
		BudgetLimit:     formatCampaignMicro(t.BudgetLimit),
		PacingMode:      string(t.PacingMode),
		DailyBudget:     formatCampaignMicro(t.DailyBudget),
		Timezone:        t.Timezone,
		FreqLimit:       t.FreqLimit,
		FreqWindow:      t.FreqWindow,
		TargetCountries: countries,
		BrandID:         brandID,
		DaypartHours:    hours,
		CreatedAt:       t.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:       t.UpdatedAt.Time.Format(time.RFC3339),
	}
}

type templateCampaignCreator interface {
	CreateCampaign(ctx context.Context, spec CreateCampaignSpec) (uuid.UUID, error)
	CreateCampaignTemplate(
		ctx context.Context,
		customerID uuid.UUID,
		name string,
		budgetLimit int64,
		pacing db.PacingModeType,
		dailyBudget int64,
		timezone string,
		freqLimit, freqWindow int32,
		targetCountries []string,
		brandID *uuid.UUID,
		daypartHours []int16,
	) (uuid.UUID, error)
}

func createCampaignFromTemplate(
	ctx context.Context,
	pool *pgxpool.Pool,
	creator templateCampaignCreator,
	templateID, customerID uuid.UUID,
	name string,
	budgetLimit *int64,
	idempotencyKey string,
) (uuid.UUID, error) {
	if pool == nil || creator == nil {
		return uuid.Nil, errServiceUnavailable()
	}
	tmpl, err := db.New(pool).GetCampaignTemplate(ctx, domain.ToUUID(templateID))
	if err != nil {
		return uuid.Nil, mapCampaignNotFound(err, ErrTemplateNotFound)
	}
	if uuid.UUID(tmpl.CustomerID.Bytes) != customerID {
		return uuid.Nil, ErrTemplateBelongsToAnotherCustomer
	}

	limit := tmpl.BudgetLimit
	if budgetLimit != nil {
		limit = *budgetLimit
	}
	if name == "" {
		name = tmpl.Name
	}

	var brandID *uuid.UUID
	if tmpl.BrandID.Valid {
		id := uuid.UUID(tmpl.BrandID.Bytes)
		brandID = &id
	}

	return creator.CreateCampaign(ctx, CreateCampaignSpec{
		CustomerID:       customerID,
		BrandID:          brandID,
		Name:             name,
		BudgetLimitMicro: limit,
		PacingMode:       string(tmpl.PacingMode),
		DailyBudgetMicro: tmpl.DailyBudget,
		Timezone:         tmpl.Timezone,
		FreqLimit:        tmpl.FreqLimit,
		FreqWindow:       tmpl.FreqWindow,
		TargetCountries:  tmpl.TargetCountries,
		DaypartHours:     tmpl.DaypartHours,
		TemplateID:       &templateID,
		IdempotencyKey:   idempotencyKey,
	})
}

func saveCampaignAsTemplate(
	ctx context.Context,
	pool *pgxpool.Pool,
	fx Effects,
	creator templateCampaignCreator,
	campaignID uuid.UUID,
	templateName string,
) (uuid.UUID, error) {
	if fx == nil || creator == nil {
		return uuid.Nil, errServiceUnavailable()
	}
	camp, err := fx.GetCampaignRow(ctx, campaignID)
	if err != nil {
		return uuid.Nil, err
	}
	if templateName == "" {
		templateName = camp.Name + " template"
	}
	var brandID *uuid.UUID
	if camp.BrandID.Valid {
		id := uuid.UUID(camp.BrandID.Bytes)
		brandID = &id
	}
	hours := camp.DaypartHours
	if hours == nil {
		hours = []int16{}
	}
	return creator.CreateCampaignTemplate(ctx,
		uuid.UUID(camp.CustomerID.Bytes),
		templateName,
		camp.BudgetLimit,
		camp.PacingMode,
		camp.DailyBudget,
		camp.Timezone,
		camp.FreqLimit.Int32,
		camp.FreqWindow.Int32,
		camp.TargetCountries,
		brandID,
		hours,
	)
}
