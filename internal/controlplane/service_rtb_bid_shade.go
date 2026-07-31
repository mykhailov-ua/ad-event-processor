package controlplane

import (
	"context"
	"fmt"

	"espx/internal/domain"
)

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
