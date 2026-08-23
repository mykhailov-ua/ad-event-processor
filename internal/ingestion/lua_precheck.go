package ingestion

import (
	"context"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/licensing"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

const (
	luaReturnDailyQuota   int64 = 12
	luaReturnPlacement    int64 = 14
	luaReturnTierDegraded int64 = 20
	luaReturnFraudSignal  int64 = 21

	luaPrecheckIngressTTLSec = 28 * 3600
	luaDegradeThresholdNs    = int64(2_000_000)
)

func luaBranchLabel(res int64) string {
	switch res {
	case 0:
		return "ok"
	case 1:
		return "rate"
	case 2:
		return "duplicate"
	case 3:
		return "budget"
	case 4:
		return "pacing"
	case 5:
		return "freq"
	case 6:
		return "ttc_low"
	case 7:
		return "ttc_missing"
	case 10:
		return "ttc_bypass"
	case 11:
		return "migration_fence"
	case luaReturnDailyQuota:
		return "daily_quota"
	case luaReturnPlacement:
		return "placement"
	case luaReturnTierDegraded:
		return "tier_degraded"
	case luaReturnFraudSignal:
		return "fraud_signal"
	default:
		return "accept"
	}
}

var (
	luaPrecheckIngressTTLAny any = luaPrecheckIngressTTLSec
	luaDegradeThresholdAny   any = luaDegradeThresholdNs
)

var (
	placementIgnoredKeyVal = StringVal{s: "fcap:ignored"}
	ingressIgnoredKeyVal   = StringVal{s: "fcap:ignored"}
)

var maxRPDAnyCache [8192]any

func maxRPDAsAny(v uint64) any {
	if v == 0 {
		return zeroAny
	}
	if int(v) < len(maxRPDAnyCache) {
		return maxRPDAnyCache[v]
	}
	return v
}

type entitlementsLookup interface {
	GetEntitlements(customerID uuid.UUID) (licensing.Entitlements, bool)
}

type luaPrecheckScratch struct {
	wIngress, wPlacement bufWrapper
}

func (f *UnifiedFilter) entitlementsMaxRPD(custID uuid.UUID) uint64 {
	lookup, ok := f.registry.(entitlementsLookup)
	if !ok {
		return 0
	}
	ent, ok := lookup.GetEntitlements(custID)
	if !ok || ent.Limits.MaxRequestsPerDay == 0 {
		return 0
	}
	return ent.Limits.MaxRequestsPerDay
}

func appendCampaignIngressDayKey(dst []byte, campaignID uuid.UUID, regionCode uint8, customerID uuid.UUID, t time.Time) []byte {
	dst = appendCampaignHashTag(dst[:0], campaignID)
	dst = append(dst, "ingress:day:"...)
	if regionCode > 0 {
		dst = append(dst, hexByte(regionCode>>4), hexByte(regionCode&0x0f), ':')
	}
	dst = appendUUID(dst, customerID)
	dst = append(dst, ':')
	return appendDate(dst, t)
}

func (f *UnifiedFilter) SetFraudBlacklistFilter(bl *FraudBlacklistFilter) {
	if f != nil {
		f.fraudBL = bl
	}
}

// SetIngressRPDHandledExternally skips campaign ingress INCR when EntitlementsFilter already ran.
func (f *UnifiedFilter) SetIngressRPDHandledExternally(v bool) {
	if f != nil {
		f.ingressRPDHandledExternally = v
	}
}

func (f *UnifiedFilter) applyLuaGoPrechecks(
	ctx context.Context,
	evt *domain.Event,
	campInfo *domain.Campaign,
	rdb redis.UniversalClient,
	now time.Time,
) error {
	if f.ingressRPDHandledExternally {
		return nil
	}
	return f.checkIngressRPDGo(ctx, evt, campInfo, rdb, now)
}

func (f *UnifiedFilter) checkIngressRPDGo(
	ctx context.Context,
	evt *domain.Event,
	campInfo *domain.Campaign,
	rdb redis.UniversalClient,
	now time.Time,
) error {
	maxRPD := f.entitlementsMaxRPD(campInfo.CustomerID)
	if maxRPD == 0 || rdb == nil {
		return nil
	}
	var keyBuf []byte
	keyBuf = appendCampaignIngressDayKey(keyBuf, evt.CampaignID, f.regionCode, campInfo.CustomerID, now)
	redisKey := unsafeString(keyBuf)
	pipe := rdb.Pipeline()
	incr := pipe.Incr(ctx, redisKey)
	pipe.Expire(ctx, redisKey, time.Duration(luaPrecheckIngressTTLSec)*time.Second)
	if _, err := pipe.Exec(ctx); err != nil {
		return nil
	}
	if uint64(incr.Val()) > maxRPD {
		return ErrDailyQuotaExceeded
	}
	return nil
}

func fillLuaIgnoredPrecheckKeys(keyArgs []any, ingressIdx, placementIdx int) {
	keyArgs[ingressIdx] = &ingressIgnoredKeyVal
	keyArgs[placementIdx] = &placementIgnoredKeyVal
}
