package filter

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

type offerClickCounts struct {
	daily int64
	total int64
}

func offerClickCapDailyKey(offerID uuid.UUID, date string) string {
	return fmt.Sprintf("%s:clicks:daily:%s", offerID.String(), date)
}

func offerClickCapTotalKey(offerID uuid.UUID) string {
	return fmt.Sprintf("%s:clicks:total", offerID.String())
}

func offerClickCapExceeded(c offerClickCounts, dailyCap, totalCap int32) bool {
	if totalCap > 0 && c.total >= int64(totalCap) {
		return true
	}
	if dailyCap > 0 && c.daily >= int64(dailyCap) {
		return true
	}
	return false
}

func offerFullyCapped(
	offerID uuid.UUID,
	capDaily, capTotal *int32,
	convCounts map[uuid.UUID]offerConversionCounts,
	clicksDaily, clicksTotal int32,
	clickCounts map[uuid.UUID]offerClickCounts,
) bool {
	if offerIsCapped(offerID, capDaily, capTotal, convCounts) {
		return true
	}
	return offerClickCapExceeded(clickCounts[offerID], clicksDaily, clicksTotal)
}

func loadOfferClickCounts(ctx context.Context, rdb redis.UniversalClient, offerIDs []uuid.UUID, now time.Time) map[uuid.UUID]offerClickCounts {
	out := make(map[uuid.UUID]offerClickCounts)
	if rdb == nil || len(offerIDs) == 0 {
		return out
	}
	date := now.UTC().Format("2006-01-02")
	keys := make([]string, 0, len(offerIDs)*2)
	index := make([]uuid.UUID, 0, len(offerIDs)*2)
	for _, id := range offerIDs {
		if id == uuid.Nil {
			continue
		}
		keys = append(keys, offerClickCapDailyKey(id, date))
		index = append(index, id)
		keys = append(keys, offerClickCapTotalKey(id))
		index = append(index, id)
	}
	vals, err := rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return out
	}
	for i, raw := range vals {
		if raw == nil {
			continue
		}
		var n int64
		switch v := raw.(type) {
		case string:
			if v == "" {
				continue
			}
			parsed, parseErr := parseRedisInt64(v)
			if parseErr != nil {
				continue
			}
			n = parsed
		case int64:
			n = v
		default:
			continue
		}
		id := index[i]
		c := out[id]
		if i%2 == 0 {
			c.daily = n
		} else {
			c.total = n
		}
		out[id] = c
	}
	return out
}

func parseRedisInt64(s string) (int64, error) {
	var n int64
	for i := range len(s) {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid integer")
		}
		n = n*10 + int64(c-'0')
	}
	return n, nil
}

// ReserveOfferClick increments Redis click counters and reports whether the offer remains eligible.
func ReserveOfferClick(ctx context.Context, rdb redis.UniversalClient, offerID uuid.UUID, dailyCap, totalCap int32, now time.Time) (bool, error) {
	if offerID == uuid.Nil || (dailyCap <= 0 && totalCap <= 0) {
		return true, nil
	}
	if rdb == nil {
		return true, nil
	}
	date := now.UTC().Format("2006-01-02")
	dailyKey := offerClickCapDailyKey(offerID, date)
	totalKey := offerClickCapTotalKey(offerID)

	pipe := rdb.Pipeline()
	dailyCmd := pipe.Incr(ctx, dailyKey)
	totalCmd := pipe.Incr(ctx, totalKey)
	if dailyCap > 0 {
		pipe.Expire(ctx, dailyKey, 48*time.Hour)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return false, err
	}
	daily := dailyCmd.Val()
	total := totalCmd.Val()
	over := (dailyCap > 0 && daily > int64(dailyCap)) || (totalCap > 0 && total > int64(totalCap))
	if over {
		rollback := rdb.Pipeline()
		rollback.Decr(ctx, dailyKey)
		rollback.Decr(ctx, totalKey)
		_, _ = rollback.Exec(ctx)
		return false, nil
	}
	return true, nil
}
