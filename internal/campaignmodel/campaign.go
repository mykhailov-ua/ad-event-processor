package campaignmodel

import (
	"time"

	"github.com/google/uuid"
)

type CampaignStatus string

const (
	CampaignStatusActive    CampaignStatus = "ACTIVE"
	CampaignStatusPaused    CampaignStatus = "PAUSED"
	CampaignStatusExhausted CampaignStatus = "EXHAUSTED"
)

type PacingMode string

const (
	PacingModeAsap PacingMode = "ASAP"
	PacingModeEven PacingMode = "EVEN"
	PacingModeVpp  PacingMode = "VPP"
)

type Campaign struct {
	ID                  uuid.UUID
	CustomerID          uuid.UUID
	IDStr               string
	CustomerIDStr       string
	IDStrAny            any
	CustomerIDStrAny    any
	BrandFcapKey        string
	Name                string
	Status              CampaignStatus
	PacingMode          PacingMode
	DailyBudgetMicroAny any
	Timezone            string
	FreqLimitAny        any
	FreqWindowAny       any
	BudgetCampaignKey   string
	CampaignSyncKey     string
	CustomerSyncKey     string
	FcapKeyPrefix       string
	DailySpendKeyPrefix string

	MigrationGen  int64
	RoutingEpoch  int64
	HasTriplet    bool
	PrimaryAShard int16
	PrimaryBShard int16
	ReserveShard  int16
	HEma          float64
	CEma          float64

	BrandID          *uuid.UUID
	BudgetLimit      int64
	CurrentSpend     int64
	DailyBudget      int64
	DailyBudgetMicro int64
	ReserveMicro     int64
	Location         *time.Location
	TargetCountries  map[string]struct{}

	FreqLimit  int32
	FreqWindow int32

	StartAt      *time.Time
	EndAt        *time.Time
	DaypartHours map[int16]struct{}

	RequireConsentPurposes int16

	FraudThresholdPass    uint8
	FraudThresholdSuspect uint8
	FraudThresholdIVT     uint8
	FraudThresholdBlock   uint8
	GhostIVTEnabled       bool
	BehaviorFlags         BehaviorFlags
}

func (c *Campaign) LuaRoutingEpoch() int64 {
	if c == nil {
		return 0
	}
	if c.RoutingEpoch > c.MigrationGen {
		return c.RoutingEpoch
	}
	return c.MigrationGen
}

type BehaviorFlags uint32

const (
	BehaviorRequireImp BehaviorFlags = 1 << iota
	BehaviorLowTTC
	BehaviorVelIP
	BehaviorVelUser
	BehaviorConvFast
	BehaviorSeqGap
	BehaviorDwellProxy
)

const (
	DefaultFraudThresholdPass    uint8 = 30
	DefaultFraudThresholdSuspect uint8 = 60
	DefaultFraudThresholdIVT     uint8 = 80
	DefaultFraudThresholdBlock   uint8 = 100
)
