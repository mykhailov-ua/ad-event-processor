package controlplane

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/rtb"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrRtbDealNotFound     = errors.New("rtb deal not found")
	ErrInvalidDealPacing   = rtb.ErrInvalidDealPacing
	ErrDuplicateDealID     = errors.New("deal_id already exists")
	ErrDealCustomerMissing = errors.New("customer not found")
	ErrInvalidDealSeats    = errors.New("seats must be at least 1")
)

type RtbCatalogReloadPayload struct {
	Trigger string `json:"trigger"`
}

func parseDealCustomerID(raw string) (uuid.UUID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return uuid.Nil, errValidation("customer_id is required")
	}
	return coldpath.ParseUUID(raw)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func normalizeDealSeats(seats int32) (int32, error) {
	if seats < 0 {
		return 0, ErrInvalidDealSeats
	}
	if seats == 0 {
		seats = 1
	}
	return seats, nil
}

func (s *Service) enqueueRtbCatalogReload(ctx context.Context, q db.Querier, trigger string) error {
	payload, err := coldpath.MarshalOutbox(RtbCatalogReloadPayload{Trigger: trigger})
	if err != nil {
		return err
	}
	_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType: "RELOAD_RTB_CATALOG",
		Payload:   payload,
	})
	return err
}

func (s *Service) PublishRtbCatalogReload(ctx context.Context) error {
	return publishControlChannelToAllShards(ctx, s.rdbs, domain.RtbCatalogReloadChannel(s.cfg), "reload")
}

func (s *Service) ListRtbDeals(ctx context.Context) ([]RtbDealDTO, error) {
	rows, err := db.New(s.GetPool()).ListRtbDeals(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]RtbDealDTO, len(rows))
	for i, row := range rows {
		out[i] = RtbDealDTO{
			ID:         row.ID,
			DealID:     row.DealID,
			FloorMicro: row.FloorMicro,
			GeoMask:    row.GeoMask,
			CatMask:    row.CatMask,
			Pacing:     rtb.DealPacingLabel(row.Pacing),
			Seats:      row.Seats,
			CustomerID: uuid.UUID(row.CustomerID.Bytes).String(),
			CreatedAt:  row.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:  row.UpdatedAt.Time.Format(time.RFC3339),
		}
	}
	return out, nil
}

func (s *Service) GetRtbDeal(ctx context.Context, id int64) (RtbDealDTO, error) {
	row, err := db.New(s.GetPool()).GetRtbDeal(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RtbDealDTO{}, ErrRtbDealNotFound
		}
		return RtbDealDTO{}, err
	}
	return RtbDealDTO{
		ID:         row.ID,
		DealID:     row.DealID,
		FloorMicro: row.FloorMicro,
		GeoMask:    row.GeoMask,
		CatMask:    row.CatMask,
		Pacing:     rtb.DealPacingLabel(row.Pacing),
		Seats:      row.Seats,
		CustomerID: uuid.UUID(row.CustomerID.Bytes).String(),
		CreatedAt:  row.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:  row.UpdatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (s *Service) CreateRtbDeal(ctx context.Context, spec RtbDealCreateSpec) (RtbDealDTO, error) {
	pacing, err := rtb.ParseDealPacingString(spec.Pacing)
	if err != nil {
		return RtbDealDTO{}, err
	}
	if strings.TrimSpace(spec.DealID) == "" {
		return RtbDealDTO{}, errValidation("deal_id is required")
	}
	if spec.FloorMicro < 0 {
		return RtbDealDTO{}, errValidation("floor_micro must be non-negative")
	}
	seats, err := normalizeDealSeats(spec.Seats)
	if err != nil {
		return RtbDealDTO{}, err
	}
	customerID, err := parseDealCustomerID(spec.CustomerID)
	if err != nil {
		return RtbDealDTO{}, err
	}

	var out RtbDealDTO
	err = pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		if _, err := q.GetCustomerByID(ctx, domain.ToUUID(customerID)); err != nil {
			return ErrDealCustomerMissing
		}
		row, err := q.CreateRtbDeal(ctx, db.CreateRtbDealParams{
			DealID:     strings.TrimSpace(spec.DealID),
			FloorMicro: spec.FloorMicro,
			GeoMask:    spec.GeoMask,
			CatMask:    spec.CatMask,
			Pacing:     pacing,
			CustomerID: domain.ToUUID(customerID),
			Seats:      seats,
		})
		if err != nil {
			if isUniqueViolation(err) {
				return ErrDuplicateDealID
			}
			return err
		}

		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}
		s.AuditLog(ctx, q, uid, "CREATE_RTB_DEAL", "rtb", nil, auditRtbDealCreateChange{
			DealID: row.DealID,
		}, nil)

		if err := s.enqueueRtbCatalogReload(ctx, q, "create_rtb_deal"); err != nil {
			return err
		}
		out = RtbDealDTO{
			ID:         row.ID,
			DealID:     row.DealID,
			FloorMicro: row.FloorMicro,
			GeoMask:    row.GeoMask,
			CatMask:    row.CatMask,
			Pacing:     rtb.DealPacingLabel(row.Pacing),
			Seats:      row.Seats,
			CustomerID: uuid.UUID(row.CustomerID.Bytes).String(),
			CreatedAt:  row.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:  row.UpdatedAt.Time.Format(time.RFC3339),
		}
		return nil
	})
	return out, err
}

func (s *Service) UpdateRtbDeal(ctx context.Context, id int64, spec RtbDealUpdateSpec) (RtbDealDTO, error) {
	pacing, err := rtb.ParseDealPacingString(spec.Pacing)
	if err != nil {
		return RtbDealDTO{}, err
	}
	if strings.TrimSpace(spec.DealID) == "" {
		return RtbDealDTO{}, errValidation("deal_id is required")
	}
	if spec.FloorMicro < 0 {
		return RtbDealDTO{}, errValidation("floor_micro must be non-negative")
	}
	seats, err := normalizeDealSeats(spec.Seats)
	if err != nil {
		return RtbDealDTO{}, err
	}
	customerID, err := parseDealCustomerID(spec.CustomerID)
	if err != nil {
		return RtbDealDTO{}, err
	}

	var out RtbDealDTO
	err = pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		if _, err := q.GetCustomerByID(ctx, domain.ToUUID(customerID)); err != nil {
			return ErrDealCustomerMissing
		}
		row, err := q.UpdateRtbDeal(ctx, db.UpdateRtbDealParams{
			ID:         id,
			DealID:     strings.TrimSpace(spec.DealID),
			FloorMicro: spec.FloorMicro,
			GeoMask:    spec.GeoMask,
			CatMask:    spec.CatMask,
			Pacing:     pacing,
			CustomerID: domain.ToUUID(customerID),
			Seats:      seats,
		})
		if err != nil {
			if isUniqueViolation(err) {
				return ErrDuplicateDealID
			}
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrRtbDealNotFound
			}
			return err
		}

		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}
		s.AuditLog(ctx, q, uid, "UPDATE_RTB_DEAL", "rtb", nil, auditRtbDealUpdateChange{
			ID:     id,
			DealID: row.DealID,
		}, nil)

		if err := s.enqueueRtbCatalogReload(ctx, q, "update_rtb_deal"); err != nil {
			return err
		}
		out = RtbDealDTO{
			ID:         row.ID,
			DealID:     row.DealID,
			FloorMicro: row.FloorMicro,
			GeoMask:    row.GeoMask,
			CatMask:    row.CatMask,
			Pacing:     rtb.DealPacingLabel(row.Pacing),
			Seats:      row.Seats,
			CustomerID: uuid.UUID(row.CustomerID.Bytes).String(),
			CreatedAt:  row.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:  row.UpdatedAt.Time.Format(time.RFC3339),
		}
		return nil
	})
	return out, err
}

func (s *Service) DeleteRtbDeal(ctx context.Context, id int64) error {
	return pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		row, err := q.GetRtbDeal(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrRtbDealNotFound
			}
			return err
		}
		if err := q.DeleteRtbDeal(ctx, id); err != nil {
			return err
		}

		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}
		s.AuditLog(ctx, q, uid, "DELETE_RTB_DEAL", "rtb", nil, auditRtbDealDeleteChange{
			ID:     id,
			DealID: row.DealID,
		}, nil)
		return s.enqueueRtbCatalogReload(ctx, q, "delete_rtb_deal")
	})
}

func (s *Service) SetRtbMode(ctx context.Context, mode string) error {
	norm, err := domain.NormalizeRtbModeSetting(mode)
	if err != nil {
		return err
	}
	return s.UpdateSettings(ctx, map[string]string{domain.SystemSettingRtbMode: norm})
}

func (s *Service) GetRtbMode(ctx context.Context, cfg *config.Config) string {
	settings, err := s.GetSettings(ctx)
	if err == nil {
		if v, ok := settings[domain.SystemSettingRtbMode]; ok && v != "" {
			return v
		}
	}
	if cfg != nil && cfg.RtbMode != "" {
		return cfg.RtbMode
	}
	return "off"
}

func ValidateRtbModeSetting(mode string) (string, error) {
	norm, err := domain.NormalizeRtbModeSetting(mode)
	if err != nil {
		return "", fmt.Errorf("invalid rtb mode: %w", err)
	}
	return norm, nil
}

type RtbBidShadeRequest struct {
	GeoHash      uint32 `json:"geo_hash"`
	DeviceType   uint8  `json:"device_type"`
	CategoryMask uint64 `json:"category_mask"`
	MinBidMicro  int64  `json:"min_bid_micro"`
}

type RtbBidShadeResponse struct {
	HasBid              bool    `json:"has_bid"`
	CampaignID          string  `json:"campaign_id,omitempty"`
	ClearingPriceMicro  int64   `json:"clearing_price_micro"`
	RecommendedBidMicro int64   `json:"recommended_bid_micro"`
	ShadeDeltaMicro     int64   `json:"shade_delta_micro"`
	NoBidReason         string  `json:"no_bid_reason,omitempty"`
	SecondPriceDeltaPct float64 `json:"second_price_delta_pct"`
}

func (s *Service) SimulateRtbBidShade(ctx context.Context, req RtbBidShadeRequest) (RtbBidShadeResponse, error) {
	out := RtbBidShadeResponse{}
	if s == nil || s.rtbBidShadeSim == nil {
		return out, fmt.Errorf("rtb bid shade simulator not configured")
	}
	sim, err := s.rtbBidShadeSim(ctx, s.pool, s.cfg, domain.RtbBidShadeInput{
		GeoHash:      req.GeoHash,
		DeviceType:   req.DeviceType,
		CategoryMask: req.CategoryMask,
		MinBidMicro:  req.MinBidMicro,
	})
	if err != nil {
		return out, err
	}
	out.HasBid = sim.HasBid
	out.CampaignID = sim.CampaignID
	out.ClearingPriceMicro = sim.ClearingPriceMicro
	out.RecommendedBidMicro = sim.RecommendedBidMicro
	out.ShadeDeltaMicro = sim.ShadeDeltaMicro
	out.NoBidReason = sim.NoBidReason
	out.SecondPriceDeltaPct = sim.SecondPriceDeltaPct
	return out, nil
}
