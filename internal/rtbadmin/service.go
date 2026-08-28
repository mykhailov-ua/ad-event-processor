package rtbadmin

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/rtb"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Service struct {
	host Host
}

func NewService(host Host) *Service {
	return &Service{host: host}
}

func parseDealCustomerID(host Host, raw string) (uuid.UUID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return uuid.Nil, host.ErrValidation("customer_id is required")
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

func (s *Service) ListRtbDeals(ctx context.Context) ([]DealDTO, error) {
	rows, err := db.New(s.host.Pool()).ListRtbDeals(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]DealDTO, len(rows))
	for i, row := range rows {
		out[i] = dealRowToDTO(row)
	}
	return out, nil
}

func (s *Service) GetRtbDeal(ctx context.Context, id int64) (DealDTO, error) {
	row, err := db.New(s.host.Pool()).GetRtbDeal(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DealDTO{}, ErrRtbDealNotFound
		}
		return DealDTO{}, err
	}
	return dealRowToDTO(row), nil
}

func (s *Service) CreateRtbDeal(ctx context.Context, spec DealCreateSpec) (DealDTO, error) {
	pacing, err := rtb.ParseDealPacingString(spec.Pacing)
	if err != nil {
		return DealDTO{}, err
	}
	if strings.TrimSpace(spec.DealID) == "" {
		return DealDTO{}, s.host.ErrValidation("deal_id is required")
	}
	if spec.FloorMicro < 0 {
		return DealDTO{}, s.host.ErrValidation("floor_micro must be non-negative")
	}
	seats, err := normalizeDealSeats(spec.Seats)
	if err != nil {
		return DealDTO{}, err
	}
	customerID, err := parseDealCustomerID(s.host, spec.CustomerID)
	if err != nil {
		return DealDTO{}, err
	}

	var out DealDTO
	err = pgx.BeginFunc(ctx, s.host.Pool(), func(tx pgx.Tx) error {
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

		adminID := s.host.ActorUserID(ctx)
		s.host.AuditCreateRtbDeal(ctx, q, adminID, row.DealID)

		if err := s.host.EnqueueRtbCatalogReload(ctx, q, "create_rtb_deal"); err != nil {
			return err
		}
		out = dealRowToDTO(row)
		return nil
	})
	return out, err
}

func (s *Service) UpdateRtbDeal(ctx context.Context, id int64, spec DealUpdateSpec) (DealDTO, error) {
	pacing, err := rtb.ParseDealPacingString(spec.Pacing)
	if err != nil {
		return DealDTO{}, err
	}
	if strings.TrimSpace(spec.DealID) == "" {
		return DealDTO{}, s.host.ErrValidation("deal_id is required")
	}
	if spec.FloorMicro < 0 {
		return DealDTO{}, s.host.ErrValidation("floor_micro must be non-negative")
	}
	seats, err := normalizeDealSeats(spec.Seats)
	if err != nil {
		return DealDTO{}, err
	}
	customerID, err := parseDealCustomerID(s.host, spec.CustomerID)
	if err != nil {
		return DealDTO{}, err
	}

	var out DealDTO
	err = pgx.BeginFunc(ctx, s.host.Pool(), func(tx pgx.Tx) error {
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

		adminID := s.host.ActorUserID(ctx)
		s.host.AuditUpdateRtbDeal(ctx, q, adminID, id, row.DealID)

		if err := s.host.EnqueueRtbCatalogReload(ctx, q, "update_rtb_deal"); err != nil {
			return err
		}
		out = dealRowToDTO(row)
		return nil
	})
	return out, err
}

func (s *Service) DeleteRtbDeal(ctx context.Context, id int64) error {
	return pgx.BeginFunc(ctx, s.host.Pool(), func(tx pgx.Tx) error {
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

		adminID := s.host.ActorUserID(ctx)
		s.host.AuditDeleteRtbDeal(ctx, q, adminID, id, row.DealID)
		return s.host.EnqueueRtbCatalogReload(ctx, q, "delete_rtb_deal")
	})
}

func (s *Service) SetRtbMode(ctx context.Context, mode string) error {
	norm, err := domain.NormalizeRtbModeSetting(mode)
	if err != nil {
		return err
	}
	return s.host.UpdateSettings(ctx, map[string]string{domain.SystemSettingRtbMode: norm})
}

func (s *Service) GetRtbMode(ctx context.Context) string {
	settings, err := s.host.GetSettings(ctx)
	if err == nil {
		if v, ok := settings[domain.SystemSettingRtbMode]; ok && v != "" {
			return v
		}
	}
	if cfg := s.host.Config(); cfg != nil && cfg.RtbMode != "" {
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

type BidShadeRequest struct {
	GeoHash      uint32 `json:"geo_hash"`
	DeviceType   uint8  `json:"device_type"`
	CategoryMask uint64 `json:"category_mask"`
	MinBidMicro  int64  `json:"min_bid_micro"`
}

type BidShadeResponse struct {
	HasBid              bool    `json:"has_bid"`
	CampaignID          string  `json:"campaign_id,omitempty"`
	ClearingPriceMicro  int64   `json:"clearing_price_micro"`
	RecommendedBidMicro int64   `json:"recommended_bid_micro"`
	ShadeDeltaMicro     int64   `json:"shade_delta_micro"`
	NoBidReason         string  `json:"no_bid_reason,omitempty"`
	SecondPriceDeltaPct float64 `json:"second_price_delta_pct"`
}

func (s *Service) SimulateRtbBidShade(ctx context.Context, req BidShadeRequest) (BidShadeResponse, error) {
	out := BidShadeResponse{}
	sim, err := s.host.SimulateBidShade(ctx, domain.RtbBidShadeInput{
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

func dealRowToDTO(row db.RtbDeal) DealDTO {
	return DealDTO{
		ID:         row.ID,
		DealID:     row.DealID,
		FloorMicro: row.FloorMicro,
		GeoMask:    row.GeoMask,
		CatMask:    row.CatMask,
		Pacing:     rtb.DealPacingLabel(row.Pacing),
		Seats:      row.Seats,
		CustomerID: uuid.UUID(row.CustomerID.Bytes).String(),
		CreatedAt:  row.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:  row.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
}
