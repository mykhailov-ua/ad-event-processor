package unified

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"ad-event-processor/internal/domain"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// ipRateLimitRaceRedis stubs EVALSHA so the holdout hammers concurrent Check, not Redis I/O.
type ipRateLimitRaceRedis struct {
	redis.UniversalClient
}

func (c *ipRateLimitRaceRedis) FilterEvalFast(cmd *redis.Cmd) (int64, error) {
	return 1, nil
}

// Holdout: one shared IPRateLimiter must stay race-free when Check runs from many goroutines.
// Revert to process-wide lastIP/keyBuf/redisCmd cache and this fails under -race.
//
// Verify:
// go test ./internal/filter/unified/ -race -run TestIPRateLimiter_sharedConcurrentCheck_noDataRace_holdout -count=1
func TestIPRateLimiter_sharedConcurrentCheck_noDataRace_holdout(t *testing.T) {
	mr := miniredis.RunT(t)
	base := redis.NewUniversalClient(&redis.UniversalOptions{Addrs: []string{mr.Addr()}})
	redisClient := &ipRateLimitRaceRedis{UniversalClient: base}
	limiter := NewIPRateLimiter(redisClient, 1_000_000, time.Minute)
	ctx := context.Background()

	const workers = 24
	const iters = 3_000

	var wg sync.WaitGroup
	wg.Add(workers)
	for w := range workers {
		go func(worker int) {
			defer wg.Done()
			evt := &domain.Event{}
			for i := range iters {
				evt.IP = fmt.Sprintf("203.0.113.%d", (worker*iters+i)%250)
				_ = limiter.Check(ctx, evt)
			}
		}(w)
	}
	wg.Wait()
}
