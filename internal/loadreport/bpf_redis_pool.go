package loadreport

import (
	"strconv"
	"strings"
)

func checkBPFRedisPoolPrometheus(prom *promClient, rateWindow string) []BPFGateCheck {
	rate := func(q string) string {
		return strings.ReplaceAll(q, "${window}", rateWindow)
	}

	var checks []BPFGateCheck

	missesRate := prom.scalarOrZero(rate(`sum(rate(ad_redis_pool_misses_total[${window}]))`))
	missVal, _ := strconv.ParseFloat(missesRate, 64)
	checks = append(checks, BPFGateCheck{
		Name:   "redis_pool_misses_rate",
		Value:  missesRate,
		Limit:  "0.5",
		OK:     missVal <= 0.5,
		Detail: "go-redis pool connection churn (new conns per second)",
	})

	timeoutsRate := prom.scalarOrZero(rate(`sum(rate(ad_redis_pool_timeouts_total[${window}]))`))
	timeoutVal, _ := strconv.ParseFloat(timeoutsRate, 64)
	checks = append(checks, BPFGateCheck{
		Name:   "redis_pool_timeouts_rate",
		Value:  timeoutsRate,
		Limit:  "0",
		OK:     timeoutVal <= 0,
		Detail: "go-redis pool wait timeouts",
	})

	return checks
}
