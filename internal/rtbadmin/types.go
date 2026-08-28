package rtbadmin

import (
	"context"
	"time"

	"ad-event-processor/pkg/platformconfig"
)

type DealDTO struct {
	ID         int64  `json:"id"`
	DealID     string `json:"deal_id"`
	FloorMicro int64  `json:"floor_micro"`
	GeoMask    int64  `json:"geo_mask"`
	CatMask    int64  `json:"cat_mask"`
	Pacing     string `json:"pacing"`
	Seats      int32  `json:"seats"`
	CustomerID string `json:"customer_id"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type DealCreateSpec struct {
	DealID     string `json:"deal_id"`
	FloorMicro int64  `json:"floor_micro"`
	GeoMask    int64  `json:"geo_mask"`
	CatMask    int64  `json:"cat_mask"`
	Pacing     string `json:"pacing"`
	Seats      int32  `json:"seats"`
	CustomerID string `json:"customer_id"`
}

type DealUpdateSpec struct {
	DealID     string `json:"deal_id"`
	FloorMicro int64  `json:"floor_micro"`
	GeoMask    int64  `json:"geo_mask"`
	CatMask    int64  `json:"cat_mask"`
	Pacing     string `json:"pacing"`
	Seats      int32  `json:"seats"`
	CustomerID string `json:"customer_id"`
}

type DealService interface {
	ListRtbDeals(ctx context.Context) ([]DealDTO, error)
	GetRtbDeal(ctx context.Context, id int64) (DealDTO, error)
	CreateRtbDeal(ctx context.Context, spec DealCreateSpec) (DealDTO, error)
	UpdateRtbDeal(ctx context.Context, id int64, spec DealUpdateSpec) (DealDTO, error)
	DeleteRtbDeal(ctx context.Context, id int64) error
}

type ShadowDiffSnapshotDTO struct {
	Window            string  `json:"window"`
	Source            string  `json:"source"`
	ShadowEvals       uint64  `json:"shadow_evals"`
	ShadowWinnerMatch uint64  `json:"shadow_winner_match"`
	ShadowMismatch    uint64  `json:"shadow_winner_mismatch"`
	ShadowNoBid       uint64  `json:"shadow_no_bid"`
	LiveWouldAccept   uint64  `json:"live_would_accept"`
	LiveWouldReject   uint64  `json:"live_would_reject"`
	ParityMatch       uint64  `json:"parity_match"`
	ParityRate        float64 `json:"parity_rate"`
	MismatchRate      float64 `json:"mismatch_rate"`
}

type LiveGateDTO struct {
	Ready   bool                  `json:"ready"`
	Reasons []string              `json:"reasons,omitempty"`
	Shadow  ShadowDiffSnapshotDTO `json:"shadow"`
}

type RuntimeConfigReader interface {
	RtbMode() string
	RtbEnabled() bool
	RtbExchangeNoBidMode() string
}

type PlatformConfigReader func(ctx context.Context) (platformconfig.Config, error)

type ReconcileCHFunc func(ctx context.Context, requestID string, window time.Duration) (bids, wins uint64, spendMicro int64, ok bool)

type FloorSuggestionDTO struct {
	PlacementID         string  `json:"placement_id"`
	DealID              string  `json:"deal_id"`
	CurrentFloorMicro   int64   `json:"current_floor_micro"`
	SuggestedFloorMicro int64   `json:"suggested_floor_micro"`
	WinRate             float64 `json:"win_rate"`
	SampleN             int64   `json:"sample_n"`
	FloorBucketMicro    int64   `json:"floor_bucket_micro"`
	ComputedAt          string  `json:"computed_at"`
}

type FloorsApplyRequest struct {
	PlacementIDs []string `json:"placement_ids,omitempty"`
}

type FloorsApplyResult struct {
	DryRun      bool                 `json:"dry_run"`
	Applied     int                  `json:"applied"`
	Suggestions []FloorSuggestionDTO `json:"suggestions"`
	OutboxRows  int                  `json:"outbox_rows"`
}

type FloorOptimizer interface {
	ApplyRtbFloorSuggestions(ctx context.Context, dryRun bool, placementIDs []string) (FloorsApplyResult, error)
}

type BidFloorRecommendationDTO struct {
	DealID           string  `json:"deal_id"`
	BaseFloorMicro   int64   `json:"base_floor_micro"`
	RecommendedMicro int64   `json:"recommended_floor_micro"`
	WinRate          float64 `json:"win_rate"`
	SampleN          uint64  `json:"sample_n"`
}
