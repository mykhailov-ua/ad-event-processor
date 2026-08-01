package controlplane

import (
	"context"

	"espx/internal/controlplane/adminapi"
)

type rtbAdminService struct {
	svc *Service
}

func (a *rtbAdminService) ListRtbDeals(ctx context.Context) ([]adminapi.RtbDealDTO, error) {
	rows, err := a.svc.ListRtbDeals(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]adminapi.RtbDealDTO, len(rows))
	for i := range rows {
		out[i] = rtbDealToAdmin(rows[i])
	}
	return out, nil
}

func (a *rtbAdminService) GetRtbDeal(ctx context.Context, id int64) (adminapi.RtbDealDTO, error) {
	row, err := a.svc.GetRtbDeal(ctx, id)
	if err != nil {
		return adminapi.RtbDealDTO{}, err
	}
	return rtbDealToAdmin(row), nil
}

func (a *rtbAdminService) CreateRtbDeal(ctx context.Context, spec adminapi.RtbDealCreateSpec) (adminapi.RtbDealDTO, error) {
	row, err := a.svc.CreateRtbDeal(ctx, RtbDealCreateSpec{
		DealID:     spec.DealID,
		FloorMicro: spec.FloorMicro,
		GeoMask:    spec.GeoMask,
		CatMask:    spec.CatMask,
		Pacing:     spec.Pacing,
		Seats:      spec.Seats,
		CustomerID: spec.CustomerID,
	})
	if err != nil {
		return adminapi.RtbDealDTO{}, err
	}
	return rtbDealToAdmin(row), nil
}

func (a *rtbAdminService) UpdateRtbDeal(ctx context.Context, id int64, spec adminapi.RtbDealUpdateSpec) (adminapi.RtbDealDTO, error) {
	row, err := a.svc.UpdateRtbDeal(ctx, id, RtbDealUpdateSpec{
		DealID:     spec.DealID,
		FloorMicro: spec.FloorMicro,
		GeoMask:    spec.GeoMask,
		CatMask:    spec.CatMask,
		Pacing:     spec.Pacing,
		Seats:      spec.Seats,
		CustomerID: spec.CustomerID,
	})
	if err != nil {
		return adminapi.RtbDealDTO{}, err
	}
	return rtbDealToAdmin(row), nil
}

func (a *rtbAdminService) DeleteRtbDeal(ctx context.Context, id int64) error {
	return a.svc.DeleteRtbDeal(ctx, id)
}

func rtbDealToAdmin(row RtbDealDTO) adminapi.RtbDealDTO {
	return adminapi.RtbDealDTO{
		ID:         row.ID,
		DealID:     row.DealID,
		FloorMicro: row.FloorMicro,
		GeoMask:    row.GeoMask,
		CatMask:    row.CatMask,
		Pacing:     row.Pacing,
		Seats:      row.Seats,
		CustomerID: row.CustomerID,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}
