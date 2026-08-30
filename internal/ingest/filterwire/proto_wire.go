package filterwire

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
	ErrProtoFieldBudget = errProtoFieldBudget
	errProtoFieldBudget = errors.New("protobuf field budget exceeded")
)

func init() {
	protoMaxFields.Store(int32(ProtoMaxFields))
}

func ConfigureProtoMaxFields(cfg *config.Config) {
	if cfg == nil || cfg.ProtoMaxFields <= 0 {
		protoMaxFields.Store(int32(ProtoMaxFields))
		return
	}
	protoMaxFields.Store(int32(cfg.ProtoMaxFields))
}

func UnmarshalAdEventVT(evt *pb.AdEvent, wire []byte) error {
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

func ChaosProtoWireFieldFlood(n int) []byte {
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

func NewRedisLuaObservers(numShards int) []prometheus.Observer {
	if numShards <= 0 {
		numShards = 1
	}
	observers := make([]prometheus.Observer, numShards)
	for i := range observers {
		observers[i] = metrics.RedisLuaDuration.WithLabelValues(strconv.Itoa(i))
	}
	return observers
}

func NewRedisLuaNoScriptCounters(numShards int) []prometheus.Counter {
	if numShards <= 0 {
		numShards = 1
	}
	counters := make([]prometheus.Counter, numShards)
	for i := range counters {
		counters[i] = metrics.RedisLuaNoScriptTotal.WithLabelValues(strconv.Itoa(i))
	}
	return counters
}

func IncRedisLuaNoScript(counters []prometheus.Counter, shard int) {
	if shard >= 0 && shard < len(counters) {
		counters[shard].Inc()
		return
	}
	metrics.RedisLuaNoScriptTotal.WithLabelValues(strconv.Itoa(shard)).Inc()
}

func ObserveRedisLua(observers []prometheus.Observer, shard int, seconds float64) {
	if shard >= 0 && shard < len(observers) {
		observers[shard].Observe(seconds)
		return
	}
	metrics.RedisLuaDuration.WithLabelValues(strconv.Itoa(shard)).Observe(seconds)
}

func NewRedisLuaTierObservers(numShards int) []prometheus.Observer {
	if numShards <= 0 {
		numShards = 1
	}
	observers := make([]prometheus.Observer, numShards)
	for i := range observers {
		observers[i] = metrics.RedisLuaFastDuration.WithLabelValues(strconv.Itoa(i))
	}
	return observers
}

func NewRedisLuaPathCounters(numShards int, fast bool) []prometheus.Counter {
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

func IncRedisLuaTier(counters []prometheus.Counter, shard int) {
	if shard >= 0 && shard < len(counters) {
		counters[shard].Inc()
		return
	}
	metrics.RedisLuaFastPathTotal.WithLabelValues(strconv.Itoa(shard)).Inc()
}

func ObserveRedisLuaTier(observers []prometheus.Observer, shard int, seconds float64) {
	if shard >= 0 && shard < len(observers) {
		observers[shard].Observe(seconds)
		return
	}
	metrics.RedisLuaFastDuration.WithLabelValues(strconv.Itoa(shard)).Observe(seconds)
}

func NewRedisOpsCounters(numShards int) []prometheus.Counter {
	if numShards <= 0 {
		numShards = 1
	}
	counters := make([]prometheus.Counter, numShards)
	for i := range counters {
		counters[i] = metrics.RedisOpsTotal.WithLabelValues(strconv.Itoa(i))
	}
	return counters
}

func IncRedisOps(counters []prometheus.Counter, shard int) {
	if shard >= 0 && shard < len(counters) {
		counters[shard].Inc()
		return
	}
	metrics.RedisOpsTotal.WithLabelValues(strconv.Itoa(shard)).Inc()
}
