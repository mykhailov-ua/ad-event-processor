package ingestion

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	"ad-event-processor/internal/rtb"

	"github.com/redis/go-redis/v9"
)

var emptyFcapSnapshot = &rtb.FcapSnapshot{}

func (sw *SettingsWatcher) syncFcapCounts(ctx context.Context) {
	rdb := sw.pickHealthyShard()
	if rdb == nil {
		return
	}

	newCounts := make(map[uint64]uint32)
	prefix := "fcap:c:"

	for attempt := 0; attempt < len(sw.rdbs); attempt++ {
		cursor := uint64(0)
		ok := true
		for {
			keys, next, err := rdb.Scan(ctx, cursor, prefix+"*", 200).Result()
			if err != nil {
				slog.Warn("fcap snapshot scan failed, trying next shard", "error", err)
				ok = false
				break
			}
			for _, key := range keys {
				ingestFcapKey(newCounts, key, rdb, ctx)
			}
			cursor = next
			if cursor == 0 {
				break
			}
		}
		if ok {
			if len(newCounts) == 0 {
				sw.fcapSnap.Store(emptyFcapSnapshot)
			} else {
				sw.fcapSnap.Store(rtb.NewFcapSnapshot(newCounts))
			}
			return
		}
		rdb = sw.nextShardAfter(rdb)
		if rdb == nil {
			return
		}
		newCounts = make(map[uint64]uint32)
	}
}

func ingestFcapKey(counts map[uint64]uint32, key string, rdb redis.UniversalClient, ctx context.Context) {
	idx := strings.LastIndex(key, ":u:")
	if idx < 0 || idx+3 >= len(key) {
		return
	}
	prefix := key[:idx+3]
	userID := key[idx+3:]
	if userID == "" {
		return
	}
	valStr, err := rdb.Get(ctx, key).Result()
	if err != nil {
		return
	}
	val, err := strconv.ParseUint(valStr, 10, 32)
	if err != nil {
		return
	}
	lookup := rtb.FcapLookupKey(rtb.HashBytes64([]byte(prefix)), rtb.HashBytes64([]byte(userID)))
	counts[lookup] = uint32(val)
}
