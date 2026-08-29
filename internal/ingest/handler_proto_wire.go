package ingest

import (
	"errors"
	"strconv"
	"sync/atomic"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/ingest/pb"
	"ad-event-processor/internal/metrics"

	"github.com/prometheus/client_golang/prometheus"
)

const ProtoMaxFields = 256

var (
	protoMaxFields      atomic.Int32
	errProtoFieldBudget = errors.New("protobuf field budget exceeded")
)

func init() {
	protoMaxFields.Store(int32(ProtoMaxFields))
}

func configureProtoMaxFields(cfg *config.Config) {
	if cfg == nil || cfg.ProtoMaxFields <= 0 {
		protoMaxFields.Store(int32(ProtoMaxFields))
		return
	}
	protoMaxFields.Store(int32(cfg.ProtoMaxFields))
}

func unmarshalAdEventVT(evt *pb.AdEvent, wire []byte) error {
	if evt == nil {
		return errProtoFieldBudget
	}
	if _, err := protoWireFieldCount(wire, int(protoMaxFields.Load())); err != nil {
		return err
	}
	return evt.UnmarshalVT(wire)
}

func protoWireFieldCount(wire []byte, maxFields int) (int, error) {
	off := 0
	n := len(wire)
	count := 0
	for off < n {
		tag, next, err := protoDecodeVarint(wire, off)
		if err != nil {
			return count, err
		}
		if tag == 0 {
			return count, errProtoFieldBudget
		}
		off = next
		count++
		if count > maxFields {
			return count, errProtoFieldBudget
		}
		wireType := tag & 7
		fieldNum := tag >> 3
		if fieldNum == 0 {
			return count, errProtoFieldBudget
		}
		switch wireType {
		case 0:
			_, off, err = protoDecodeVarint(wire, off)
		case 1:
			off += 8
			if off > n {
				return count, errProtoFieldBudget
			}
		case 2:
			var ln uint64
			ln, off, err = protoDecodeVarint(wire, off)
			if err != nil {
				return count, err
			}
			if ln > uint64(n-off) {
				return count, errProtoFieldBudget
			}
			off += int(ln)
		case 5:
			off += 4
			if off > n {
				return count, errProtoFieldBudget
			}
		default:
			return count, errProtoFieldBudget
		}
		if err != nil || off > n {
			return count, errProtoFieldBudget
		}
	}
	return count, nil
}

func protoDecodeVarint(wire []byte, off int) (uint64, int, error) {
	n := len(wire)
	if off >= n {
		return 0, off, errProtoFieldBudget
	}
	var val uint64
	shift := uint(0)
	for i := off; i < n; i++ {
		b := wire[i]
		if shift >= 64 {
			return 0, off, errProtoFieldBudget
		}
		val |= uint64(b&0x7f) << shift
		if b < 0x80 {
			return val, i + 1, nil
		}
		shift += 7
	}
	return 0, off, errProtoFieldBudget
}

func chaosProtoWireFieldFlood(n int) []byte {
	wire := make([]byte, 0, n*4)
	for i := range n {
		tag := uint64((i%200 + 1) << 3)
		wire = appendProtoVarint(wire, tag)
		wire = appendProtoVarint(wire, 1)
	}
	return wire
}

func appendProtoVarint(dst []byte, v uint64) []byte {
	for v >= 0x80 {
		dst = append(dst, byte(v)|0x80)
		v >>= 7
	}
	return append(dst, byte(v))
}

type PreboundTrackMetrics struct {
	throughputProto prometheus.Counter
	throughputJSON  prometheus.Counter

	decisionAccepted         prometheus.Counter
	decisionEmergencyBreaker prometheus.Counter
	decisionRateLimited      prometheus.Counter
	decisionDuplicate        prometheus.Counter
	decisionBudgetExhausted  prometheus.Counter
	decisionPacingLimit      prometheus.Counter
	decisionFrequencyCapped  prometheus.Counter
	decisionGeoBlocked       prometheus.Counter
	decisionScheduleBlocked  prometheus.Counter
	decisionCampaignNotFound prometheus.Counter
	decisionBidFloor         prometheus.Counter
	decisionFilterTimeout    prometheus.Counter
	decisionFraud            prometheus.Counter
	decisionConsentDenied    prometheus.Counter
	decisionInfraUnavailable prometheus.Counter
	decisionRegistryStale    prometheus.Counter
	decisionShardUnavailable prometheus.Counter
	decisionProducerOverload prometheus.Counter

	blockedEmergencyBreaker prometheus.Counter
	blockedRateLimit        prometheus.Counter
	blockedDuplicate        prometheus.Counter
	blockedBudget           prometheus.Counter
	blockedPacing           prometheus.Counter
	blockedFreq             prometheus.Counter
	blockedGeo              prometheus.Counter
	blockedSchedule         prometheus.Counter
	blockedCampaignNotFound prometheus.Counter
	blockedBidFloor         prometheus.Counter
	blockedFilterTimeout    prometheus.Counter
	blockedFraud            prometheus.Counter
	blockedConsent          prometheus.Counter
	blockedInfra            prometheus.Counter
	blockedRegistryStale    prometheus.Counter
	blockedShardUnavailable prometheus.Counter
	blockedProducerOverload prometheus.Counter
}

func NewPreboundTrackMetrics() PreboundTrackMetrics {
	return PreboundTrackMetrics{
		throughputProto: metrics.FilterThroughput.WithLabelValues("protobuf"),
		throughputJSON:  metrics.FilterThroughput.WithLabelValues("json"),

		decisionAccepted:         metrics.FilterDecisions.WithLabelValues("accepted"),
		decisionEmergencyBreaker: metrics.FilterDecisions.WithLabelValues("emergency_breaker"),
		decisionRateLimited:      metrics.FilterDecisions.WithLabelValues("rate_limited"),
		decisionDuplicate:        metrics.FilterDecisions.WithLabelValues("duplicate"),
		decisionBudgetExhausted:  metrics.FilterDecisions.WithLabelValues("budget_exhausted"),
		decisionPacingLimit:      metrics.FilterDecisions.WithLabelValues("pacing_limit"),
		decisionFrequencyCapped:  metrics.FilterDecisions.WithLabelValues("frequency_capped"),
		decisionGeoBlocked:       metrics.FilterDecisions.WithLabelValues("geo_blocked"),
		decisionScheduleBlocked:  metrics.FilterDecisions.WithLabelValues("schedule_blocked"),
		decisionCampaignNotFound: metrics.FilterDecisions.WithLabelValues("campaign_not_found"),
		decisionBidFloor:         metrics.FilterDecisions.WithLabelValues("bid_floor"),
		decisionFilterTimeout:    metrics.FilterDecisions.WithLabelValues("filter_timeout"),
		decisionFraud:            metrics.FilterDecisions.WithLabelValues("fraud"),
		decisionConsentDenied:    metrics.FilterDecisions.WithLabelValues("consent_denied"),
		decisionInfraUnavailable: metrics.FilterDecisions.WithLabelValues("infra_unavailable"),
		decisionRegistryStale:    metrics.FilterDecisions.WithLabelValues("registry_stale"),
		decisionShardUnavailable: metrics.FilterDecisions.WithLabelValues("shard_unavailable"),
		decisionProducerOverload: metrics.FilterDecisions.WithLabelValues("producer_overload"),

		blockedEmergencyBreaker: metrics.FilterBlockedTotal.WithLabelValues("emergency_breaker"),
		blockedRateLimit:        metrics.FilterBlockedTotal.WithLabelValues("rate_limit"),
		blockedDuplicate:        metrics.FilterBlockedTotal.WithLabelValues("duplicate"),
		blockedBudget:           metrics.FilterBlockedTotal.WithLabelValues("budget"),
		blockedPacing:           metrics.FilterBlockedTotal.WithLabelValues("pacing"),
		blockedFreq:             metrics.FilterBlockedTotal.WithLabelValues("freq"),
		blockedGeo:              metrics.FilterBlockedTotal.WithLabelValues("geo"),
		blockedSchedule:         metrics.FilterBlockedTotal.WithLabelValues("schedule"),
		blockedCampaignNotFound: metrics.FilterBlockedTotal.WithLabelValues("campaign_not_found"),
		blockedBidFloor:         metrics.FilterBlockedTotal.WithLabelValues("bid_floor"),
		blockedFilterTimeout:    metrics.FilterBlockedTotal.WithLabelValues("filter_timeout"),
		blockedFraud:            metrics.FilterBlockedTotal.WithLabelValues("fraud"),
		blockedConsent:          metrics.FilterBlockedTotal.WithLabelValues("consent_denied"),
		blockedInfra:            metrics.FilterBlockedTotal.WithLabelValues("infra_unavailable"),
		blockedRegistryStale:    metrics.FilterBlockedTotal.WithLabelValues("registry_stale"),
		blockedShardUnavailable: metrics.FilterBlockedTotal.WithLabelValues("shard_unavailable"),
		blockedProducerOverload: metrics.FilterBlockedTotal.WithLabelValues("producer_overload"),
	}
}

func newRedisLuaObservers(numShards int) []prometheus.Observer {
	if numShards <= 0 {
		numShards = 1
	}
	observers := make([]prometheus.Observer, numShards)
	for i := range observers {
		observers[i] = metrics.RedisLuaDuration.WithLabelValues(strconv.Itoa(i))
	}
	return observers
}

func newRedisLuaNoScriptCounters(numShards int) []prometheus.Counter {
	if numShards <= 0 {
		numShards = 1
	}
	counters := make([]prometheus.Counter, numShards)
	for i := range counters {
		counters[i] = metrics.RedisLuaNoScriptTotal.WithLabelValues(strconv.Itoa(i))
	}
	return counters
}

func incRedisLuaNoScript(counters []prometheus.Counter, shard int) {
	if shard >= 0 && shard < len(counters) {
		counters[shard].Inc()
		return
	}
	metrics.RedisLuaNoScriptTotal.WithLabelValues(strconv.Itoa(shard)).Inc()
}

func observeRedisLua(observers []prometheus.Observer, shard int, seconds float64) {
	if shard >= 0 && shard < len(observers) {
		observers[shard].Observe(seconds)
		return
	}
	metrics.RedisLuaDuration.WithLabelValues(strconv.Itoa(shard)).Observe(seconds)
}

func newRedisLuaTierObservers(numShards int) []prometheus.Observer {
	if numShards <= 0 {
		numShards = 1
	}
	observers := make([]prometheus.Observer, numShards)
	for i := range observers {
		observers[i] = metrics.RedisLuaFastDuration.WithLabelValues(strconv.Itoa(i))
	}
	return observers
}

func newRedisLuaPathCounters(numShards int, fast bool) []prometheus.Counter {
	if numShards <= 0 {
		numShards = 1
	}
	counters := make([]prometheus.Counter, numShards)
	for i := range counters {
		shard := strconv.Itoa(i)
		if fast {
			counters[i] = metrics.RedisLuaFastPathTotal.WithLabelValues(shard)
		} else {
			counters[i] = metrics.RedisLuaFullPathTotal.WithLabelValues(shard)
		}
	}
	return counters
}

func incRedisLuaTier(counters []prometheus.Counter, shard int) {
	if shard >= 0 && shard < len(counters) {
		counters[shard].Inc()
		return
	}
	metrics.RedisLuaFastPathTotal.WithLabelValues(strconv.Itoa(shard)).Inc()
}

func observeRedisLuaTier(observers []prometheus.Observer, shard int, seconds float64) {
	if shard >= 0 && shard < len(observers) {
		observers[shard].Observe(seconds)
		return
	}
	metrics.RedisLuaFastDuration.WithLabelValues(strconv.Itoa(shard)).Observe(seconds)
}

func newRedisOpsCounters(numShards int) []prometheus.Counter {
	if numShards <= 0 {
		numShards = 1
	}
	counters := make([]prometheus.Counter, numShards)
	for i := range counters {
		counters[i] = metrics.RedisOpsTotal.WithLabelValues(strconv.Itoa(i))
	}
	return counters
}

func incRedisOps(counters []prometheus.Counter, shard int) {
	if shard >= 0 && shard < len(counters) {
		counters[shard].Inc()
		return
	}
	metrics.RedisOpsTotal.WithLabelValues(strconv.Itoa(shard)).Inc()
}
