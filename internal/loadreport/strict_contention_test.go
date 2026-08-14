package loadreport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStrictContentionVerdicts_trackerP99Fail(t *testing.T) {
	base := &StrictContentionSnapshot{TrackerP99Ms: "40", SyncLagMaxSec: "0"}
	treat := &StrictContentionSnapshot{
		TrackerP99Ms:          "85",
		SyncLagMaxSec:         "3",
		LocalQuotaBlockPerSec: "2",
		RedisLuaP99MaxMs:      "12",
	}
	verdicts := strictContentionVerdicts(base, treat)
	require.Contains(t, verdicts[0], "FAIL")
	require.Contains(t, joinVerdicts(verdicts), "SIGNAL: sync_lag")
	require.Contains(t, joinVerdicts(verdicts), "local_quota_block")
	require.Contains(t, joinVerdicts(verdicts), "Redis Lua p99")
}

func joinVerdicts(v []string) string {
	var s string
	for _, line := range v {
		s += line + "\n"
	}
	return s
}
