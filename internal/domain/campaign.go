package domain

import (
	"strings"
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

type ConnTypePolicy string

const (
	ConnTypeBlockVPNHosting ConnTypePolicy = "block_vpn_hosting"
	ConnTypeMobileOnly      ConnTypePolicy = "mobile_only"
	ConnTypeResidentialOnly ConnTypePolicy = "residential_only"

	SocialInAppConnTypePolicy = ConnTypeMobileOnly
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
	SilentRejectEnabled   bool
	BehaviorFlags         BehaviorFlags

	RetargetSegmentID uuid.UUID
	SegmentTTLHours   int32
	SegmentIncludeID  uuid.UUID
	SegmentExcludeID  uuid.UUID

	SafePageURL              string
	SafePageEnabled          bool
	CanvasRetestEnabled      bool
	CgnatIPPolicyEnabled     bool
	AcceptLangGeoEnabled     bool
	JSONSerializationEnabled bool
	AttestationEnabled       bool
	AttestationMode          AttestationMode
	AttestationTTLSec        int32
	DmrEnabled               bool

	CIDRBlockEnabled bool

	ProxyVPNBlockEnabled bool

	ModeratorIntelEnabled bool

	ReviewTrafficAction ReviewTrafficAction

	TLSFingerprintBlockEnabled bool

	SocialInAppEnabled bool

	ConnTypePolicy ConnTypePolicy

	LinkSigningEnabled bool
	LinkSigningTTLSec  int32

	ClickDelivery      string
	ProxyUpstreamURL   string
	ProxyRewriteAssets bool

	IngressCost IngressCostConfig
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
	BehaviorRoughPacing
	BehaviorHighVolumeDebit
)

const HighVolumeDebitSubShards = 4

func (c *Campaign) DebitSubShardCount() int {
	if c == nil || c.BehaviorFlags&BehaviorHighVolumeDebit == 0 {
		return 0
	}
	return HighVolumeDebitSubShards
}

func (c *Campaign) RoughPacingEnabled() bool {
	if c == nil {
		return false
	}
	return c.BehaviorFlags&BehaviorRoughPacing != 0
}

const (
	DefaultFraudThresholdPass    uint8 = 30
	DefaultFraudThresholdSuspect uint8 = 60
	DefaultFraudThresholdIVT     uint8 = 80
	DefaultFraudThresholdBlock   uint8 = 100

	FraudPresetConservative          = "conservative"
	FraudPresetBalanced              = "balanced"
	FraudPresetAggressive            = "aggressive"
	FraudPresetEnhancedDefense       = "enhanced_defense"
	FraudPresetEnhancedDefenseLegacy = "gray_market" // wire/DB alias for enhanced_defense
	FraudPresetSocialInApp           = "social_in_app"
)

func IsEnhancedDefenseFraudPreset(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case FraudPresetEnhancedDefense, FraudPresetEnhancedDefenseLegacy:
		return true
	default:
		return false
	}
}

func IsSocialInAppFraudPreset(name string) bool {
	return strings.EqualFold(strings.TrimSpace(name), FraudPresetSocialInApp)
}

func ResolveFraudPreset(name string) (pass, suspect, ivt, block uint8, ok bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case FraudPresetConservative:
		return 40, 70, 90, 100, true
	case FraudPresetBalanced:
		return DefaultFraudThresholdPass, DefaultFraudThresholdSuspect, DefaultFraudThresholdIVT, DefaultFraudThresholdBlock, true
	case FraudPresetAggressive:
		return 20, 45, 65, 85, true
	case FraudPresetEnhancedDefense, FraudPresetEnhancedDefenseLegacy:
		return 20, 45, 65, 85, true
	case FraudPresetSocialInApp:
		return DefaultFraudThresholdPass, DefaultFraudThresholdSuspect, DefaultFraudThresholdIVT, DefaultFraudThresholdBlock, true
	default:
		return 0, 0, 0, 0, false
	}
}
