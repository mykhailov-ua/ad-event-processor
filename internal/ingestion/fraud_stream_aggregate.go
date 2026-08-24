package ingestion

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"

	redis "github.com/redis/go-redis/v9"
)

const (
	fraudAggTableSize     = 4096
	fraudAggTableMask     = fraudAggTableSize - 1
	fraudAggThreshold     = fraudRingUsable * 8 / 10
	fraudAggFlushInterval = 75 * time.Millisecond
	fraudAggMaxProbe      = 64
)

type fraudAggPrefixKind uint8

const (
	fraudAggPrefixNone  fraudAggPrefixKind = 0
	fraudAggPrefixV4    fraudAggPrefixKind = 1
	fraudAggPrefixV6_64 fraudAggPrefixKind = 2
	fraudAggPrefixV6_48 fraudAggPrefixKind = 3
)

type fraudAggCell struct {
	prefixKind   atomic.Uint32
	subnetPrefix atomic.Uint32
	ipv6Hi       atomic.Uint64
	ipv6Lo       atomic.Uint64
	reasonID     atomic.Uint32
	count        atomic.Uint64
}

type fraudAggKey struct {
	kind fraudAggPrefixKind
	v4   uint32
	v6Hi uint64
	v6Lo uint64
}

type fraudAggFlushEntry struct {
	kind   fraudAggPrefixKind
	subnet uint32
	v6Hi   uint64
	v6Lo   uint64
	reason uint8
	count  uint64
}

var fraudAggFlushPool = sync.Pool{
	New: func() any {
		s := make([]fraudAggFlushEntry, 0, 256)
		return &s
	},
}

func fraudAggregateExempt(evt *domain.Event) bool {
	if evt == nil || evt.FraudReason == "" {
		return false
	}
	if fraudReasonContainsCode(evt.FraudReason, FraudReasonCodeL3Blocklist) {
		return true
	}
	return countL1HighSignalsInReason(evt.FraudReason) >= 2
}

func countL1HighSignalsInReason(reason string) int {
	n := 0
	for id := FraudReasonID(1); id < fraudReasonCount; id++ {
		if FraudSignalFlags(id)&fraudSignalL1High == 0 {
			continue
		}
		if fraudReasonContainsCode(reason, FraudReasonCode(id)) {
			n++
		}
	}
	return n
}

func fraudReasonContainsCode(reason, code string) bool {
	if reason == "" || code == "" {
		return false
	}
	off := 0
	for off <= len(reason) {
		end := off
		for end < len(reason) && reason[end] != ',' {
			end++
		}
		if len(code) == end-off && reason[off:end] == code {
			return true
		}
		if end >= len(reason) {
			break
		}
		off = end + 1
	}
	return false
}

func primaryFraudReasonID(reason string) uint8 {
	if reason == "" {
		return 0
	}
	end := len(reason)
	for i := 0; i < len(reason); i++ {
		if reason[i] == ',' {
			end = i
			break
		}
	}
	for id := FraudReasonID(1); id < fraudReasonCount; id++ {
		code := FraudReasonCode(id)
		if len(code) == end && reason[:end] == code {
			return uint8(id)
		}
	}
	return 0
}

func ipv4Subnet24Prefix(ip string) (uint32, bool) {
	if len(ip) < 7 {
		return 0, false
	}
	var octets [4]uint32
	idx := 0
	val := uint32(0)
	for i := 0; i < len(ip) && idx < 4; i++ {
		c := ip[i]
		if c >= '0' && c <= '9' {
			val = val*10 + uint32(c-'0')
			if val > 255 {
				return 0, false
			}
			continue
		}
		if c == '.' {
			octets[idx] = val
			idx++
			val = 0
			continue
		}
		return 0, false
	}
	if idx != 3 {
		return 0, false
	}
	octets[3] = val
	addr := (octets[0] << 24) | (octets[1] << 16) | (octets[2] << 8) | octets[3]
	return addr & 0xFFFFFF00, true
}

func parseIPv6To128(ip string) (hi, lo uint64, ok bool) {
	if len(ip) < 2 {
		return 0, 0, false
	}
	start, end := 0, len(ip)
	for start < end && ip[start] == ' ' {
		start++
	}
	for end > start && ip[end-1] == ' ' {
		end--
	}
	if start >= end {
		return 0, 0, false
	}
	ip = ip[start:end]
	if ip[0] == '[' {
		if len(ip) < 3 || ip[len(ip)-1] != ']' {
			return 0, 0, false
		}
		ip = ip[1 : len(ip)-1]
	}
	colon := -1
	for i := 0; i < len(ip); i++ {
		if ip[i] == ':' {
			colon = i
			break
		}
	}
	if colon < 0 {
		return 0, 0, false
	}

	var groups [8]uint16
	n := 0
	i := 0
	doubleColon := -1

	for i < len(ip) && n < 8 {
		if ip[i] == ':' {
			if i+1 < len(ip) && ip[i+1] == ':' {
				if doubleColon >= 0 {
					return 0, 0, false
				}
				doubleColon = n
				i += 2
				if i == len(ip) {
					break
				}
				continue
			}
			i++
			continue
		}
		var val uint16
		startGroup := i
		for i < len(ip) && ip[i] != ':' {
			c := ip[i]
			switch {
			case c >= '0' && c <= '9':
				val = val*16 + uint16(c-'0')
			case c >= 'a' && c <= 'f':
				val = val*16 + uint16(c-'a'+10)
			case c >= 'A' && c <= 'F':
				val = val*16 + uint16(c-'A'+10)
			default:
				return 0, 0, false
			}
			i++
		}
		if i == startGroup {
			return 0, 0, false
		}
		groups[n] = val
		n++
	}

	if doubleColon >= 0 {
		zeros := 8 - n
		if zeros < 0 {
			return 0, 0, false
		}
		copy(groups[doubleColon+zeros:], groups[doubleColon:n])
		for j := doubleColon; j < doubleColon+zeros; j++ {
			groups[j] = 0
		}
		n = 8
	}
	if n != 8 {
		return 0, 0, false
	}

	hi = uint64(groups[0])<<48 | uint64(groups[1])<<32 | uint64(groups[2])<<16 | uint64(groups[3])
	lo = uint64(groups[4])<<48 | uint64(groups[5])<<32 | uint64(groups[6])<<16 | uint64(groups[7])
	return hi, lo, true
}

func ipv6Subnet64FromIP(ip string) (hi, lo uint64, ok bool) {
	hi, _, ok = parseIPv6To128(ip)
	if !ok {
		return 0, 0, false
	}
	return hi, 0, true
}

func ipv6Subnet48FromIP(ip string) (hi uint64, ok bool) {
	hi, lo, ok := parseIPv6To128(ip)
	if !ok {
		return 0, false
	}
	_ = lo
	return hi & 0xFFFFFFFFFFFF0000, true
}

func fraudAggHash(key fraudAggKey, reason uint8) uint32 {
	h := uint32(key.kind) * 0x9e3779b9
	h ^= key.v4
	h ^= uint32(key.v6Hi)
	h ^= uint32(key.v6Hi>>32) * 0x85ebca6b
	h ^= uint32(key.v6Lo)
	h ^= uint32(key.v6Lo>>32) * 0xc2b2ae35
	h ^= uint32(reason) * 0x27d4eb2f
	h ^= h >> 16
	return h & fraudAggTableMask
}

func (q *FraudStreamWriter) ringFill() uint64 {
	alloc := atomic.LoadUint64(&q.allocCursor)
	read := atomic.LoadUint64(&q.readCursor)
	if alloc <= read {
		return 0
	}
	return alloc - read
}

func (q *FraudStreamWriter) publishAggMode(fill uint64) {
	agg := uint32(0)
	if fill >= fraudAggThreshold || atomic.LoadUint32(&q.forceAgg) == 1 {
		agg = 1
	}
	prev := atomic.LoadUint32(&q.aggregating)
	if prev == agg {
		return
	}
	if atomic.CompareAndSwapUint32(&q.aggregating, prev, agg) {
		metrics.FraudStreamMode.WithLabelValues("aggregating").Set(float64(agg))
	}
}

func (q *FraudStreamWriter) aggTableFillRatio() float64 {
	occ := atomic.LoadUint64(&q.aggOccupied)
	return float64(occ) / float64(fraudAggTableSize)
}

func (q *FraudStreamWriter) aggregateEvent(evt *domain.Event) bool {
	reasonID := primaryFraudReasonID(evt.FraudReason)
	if reasonID == 0 {
		return false
	}

	if prefix, ok := ipv4Subnet24Prefix(evt.IP); ok {
		if q.aggIncrement(fraudAggKey{kind: fraudAggPrefixV4, v4: prefix}, reasonID) {
			metrics.FraudStreamAggregatedTotal.Inc()
			return true
		}
		metrics.FraudStreamAggregatedDropTotal.Inc()
		return false
	}

	aggregated := false
	if hi64, lo64, ok := ipv6Subnet64FromIP(evt.IP); ok {
		if q.aggIncrement(fraudAggKey{kind: fraudAggPrefixV6_64, v6Hi: hi64, v6Lo: lo64}, reasonID) {
			metrics.FraudStreamAggregatedTotal.Inc()
			aggregated = true
		} else {
			metrics.FraudStreamAggregatedDropTotal.Inc()
		}
	}
	if hi48, ok := ipv6Subnet48FromIP(evt.IP); ok {
		if q.aggIncrement(fraudAggKey{kind: fraudAggPrefixV6_48, v6Hi: hi48, v6Lo: 0}, reasonID) {
			metrics.FraudStreamAggregatedTotal.Inc()
			aggregated = true
		} else {
			metrics.FraudStreamAggregatedDropTotal.Inc()
		}
	}
	return aggregated
}

func (q *FraudStreamWriter) aggIncrement(key fraudAggKey, reasonID uint8) bool {
	start := fraudAggHash(key, reasonID)
	for probe := range fraudAggMaxProbe {
		idx := (start + uint32(probe)) & fraudAggTableMask
		cell := &q.aggSlots[idx]

		for {
			existingKind := fraudAggPrefixKind(cell.prefixKind.Load())
			if existingKind == fraudAggPrefixNone {
				if cell.prefixKind.CompareAndSwap(uint32(fraudAggPrefixNone), uint32(key.kind)) {
					cell.subnetPrefix.Store(key.v4)
					cell.ipv6Hi.Store(key.v6Hi)
					cell.ipv6Lo.Store(key.v6Lo)
					cell.reasonID.Store(uint32(reasonID))
					cell.count.Store(1)
					atomic.AddUint64(&q.aggOccupied, 1)
					metrics.FraudStreamAggTableFill.Set(q.aggTableFillRatio())
					return true
				}
				continue
			}
			if existingKind == key.kind &&
				cell.subnetPrefix.Load() == key.v4 &&
				cell.ipv6Hi.Load() == key.v6Hi &&
				cell.ipv6Lo.Load() == key.v6Lo &&
				uint8(cell.reasonID.Load()) == reasonID {
				cell.count.Add(1)
				return true
			}
			break
		}
	}
	return false
}

func (q *FraudStreamWriter) aggregateFlusher() {
	defer q.aggWg.Done()
	ticker := time.NewTicker(fraudAggFlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-q.stopCh:
			q.flushAggregates(true)
			return
		case <-ticker.C:
			q.flushAggregates(false)
		}
	}
}

func (q *FraudStreamWriter) flushAggregates(final bool) {
	entriesPtr := fraudAggFlushPool.Get().(*[]fraudAggFlushEntry)
	entries := *entriesPtr
	entries = entries[:0]

	windowStart := atomic.LoadInt64(&q.aggWindowStart)
	now := time.Now().UnixMilli()
	if windowStart == 0 {
		atomic.StoreInt64(&q.aggWindowStart, now)
		windowStart = now
	}
	windowMs := now - windowStart
	if windowMs <= 0 {
		windowMs = int64(fraudAggFlushInterval / time.Millisecond)
	}

	for i := range q.aggSlots {
		cell := &q.aggSlots[i]
		kind := fraudAggPrefixKind(cell.prefixKind.Load())
		if kind == fraudAggPrefixNone {
			continue
		}
		subnet := cell.subnetPrefix.Load()
		v6Hi := cell.ipv6Hi.Load()
		v6Lo := cell.ipv6Lo.Load()
		count := cell.count.Swap(0)
		if count == 0 {
			continue
		}
		reason := uint8(cell.reasonID.Load())
		entries = append(entries, fraudAggFlushEntry{
			kind:   kind,
			subnet: subnet,
			v6Hi:   v6Hi,
			v6Lo:   v6Lo,
			reason: reason,
			count:  count,
		})
		if cell.count.Load() == 0 {
			if cell.prefixKind.CompareAndSwap(uint32(kind), uint32(fraudAggPrefixNone)) {
				cell.subnetPrefix.Store(0)
				cell.ipv6Hi.Store(0)
				cell.ipv6Lo.Store(0)
				cell.reasonID.Store(0)
				atomic.AddUint64(&q.aggOccupied, ^uint64(0))
			}
		}
	}
	metrics.FraudStreamAggTableFill.Set(q.aggTableFillRatio())
	atomic.StoreInt64(&q.aggWindowStart, now)

	if len(entries) == 0 {
		*entriesPtr = entries
		fraudAggFlushPool.Put(entriesPtr)
		if final {
			return
		}
		return
	}

	ctx := context.Background()
	if q.useBroker && q.brokerSink != nil {
		payloads := make([][]byte, 0, len(entries))
		wraps := make([]*ByteSliceValue, 0, len(entries))
		bufs := make([]*[]byte, 0, len(entries))
		for _, e := range entries {
			data, wrap, bufPtr := marshalFraudAggregateEntry(e, windowMs)
			if data == nil {
				filterFraudStreamWriteErrors.Inc()
				continue
			}
			wraps = append(wraps, wrap)
			bufs = append(bufs, bufPtr)
			payloads = append(payloads, data)
		}
		if len(payloads) > 0 {
			if err := q.brokerSink.Produce(ctx, 0, payloads); err != nil {
				for range payloads {
					filterFraudStreamWriteErrors.Inc()
				}
			}
		}
		for i := range wraps {
			byteSliceValuePool.Put(wraps[i])
			byteBufPool.Put(bufs[i])
		}
		*entriesPtr = entries
		fraudAggFlushPool.Put(entriesPtr)
		return
	}

	redisClient := firstConnectedRedisShard(q.redisShards)
	if redisClient == nil {
		for range entries {
			filterFraudStreamWriteErrors.Inc()
		}
		*entriesPtr = entries
		fraudAggFlushPool.Put(entriesPtr)
		return
	}

	pipe := redisClient.Pipeline()
	for _, e := range entries {
		fillFraudAggregateValues(e, windowMs, q.aggValScratch[:])
		pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: q.stream,
			MaxLen: q.maxLen,
			Approx: true,
			Values: q.aggValScratch[:],
		})
	}
	if _, err := pipe.Exec(ctx); err != nil {
		for range entries {
			filterFraudStreamWriteErrors.Inc()
		}
	}

	*entriesPtr = entries
	fraudAggFlushPool.Put(entriesPtr)
}

func fillFraudAggregateValues(e fraudAggFlushEntry, windowMs int64, valSlice []any) {
	subnetStr := ""
	ipv6Str := ""
	switch e.kind {
	case fraudAggPrefixV4:
		subnetStr = formatIPv4Subnet24(e.subnet)
	case fraudAggPrefixV6_64:
		ipv6Str = formatIPv6Prefix(e.v6Hi, e.v6Lo, 64)
	case fraudAggPrefixV6_48:
		ipv6Str = formatIPv6Prefix(e.v6Hi, 0, 48)
	}
	reasonStr := FraudReasonCode(FraudReasonID(e.reason))
	countStr := strconv.FormatUint(e.count, 10)
	windowStr := strconv.FormatInt(windowMs, 10)

	valSlice[0] = "type"
	valSlice[1] = "fraud_aggregate"
	valSlice[2] = "subnet"
	valSlice[3] = subnetStr
	valSlice[4] = "ipv6_prefix"
	valSlice[5] = ipv6Str
	valSlice[6] = "fraud_reason"
	valSlice[7] = reasonStr
	valSlice[8] = "count"
	valSlice[9] = countStr
	valSlice[10] = "window_ms"
	valSlice[11] = windowStr
}

func formatIPv4Subnet24(prefix uint32) string {
	a := (prefix >> 24) & 0xFF
	b := (prefix >> 16) & 0xFF
	c := (prefix >> 8) & 0xFF
	return strconv.FormatUint(uint64(a), 10) + "." +
		strconv.FormatUint(uint64(b), 10) + "." +
		strconv.FormatUint(uint64(c), 10) + ".0/24"
}

func formatIPv6Prefix(hi, lo uint64, bits int) string {
	groups := [8]uint16{
		uint16(hi >> 48),
		uint16(hi >> 32),
		uint16(hi >> 16),
		uint16(hi),
		uint16(lo >> 48),
		uint16(lo >> 32),
		uint16(lo >> 16),
		uint16(lo),
	}
	switch bits {
	case 48:
		groups[3] = 0
		groups[4] = 0
		groups[5] = 0
		groups[6] = 0
		groups[7] = 0
	case 64:
		groups[4] = 0
		groups[5] = 0
		groups[6] = 0
		groups[7] = 0
	}

	var b []byte
	b = appendHexIPv6Group(b, groups[0])
	for i := 1; i < 8; i++ {
		b = append(b, ':')
		b = appendHexIPv6Group(b, groups[i])
	}
	b = append(b, '/')
	b = strconv.AppendInt(b, int64(bits), 10)
	return string(b)
}

func appendHexIPv6Group(b []byte, g uint16) []byte {
	if g == 0 {
		return append(b, '0')
	}
	start := len(b)
	for i := 0; i < 4; i++ {
		digit := (g >> (12 - i*4)) & 0xF
		if digit == 0 && len(b) == start {
			continue
		}
		if digit < 10 {
			b = append(b, byte('0'+digit))
		} else {
			b = append(b, byte('a'+digit-10))
		}
	}
	if len(b) == start {
		return append(b, '0')
	}
	return b
}
