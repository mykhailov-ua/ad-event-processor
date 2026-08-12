package platformconfig

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisAddrsUDS_applianceFourShards(t *testing.T) {
	addrs := RedisAddrsUDS(RedisShardCountAppliance)
	assert.Equal(t, 4, RedisShardCountForAddrs(addrs))
	assert.NotContains(t, addrs, "redis-4.sock")
	assert.Contains(t, addrs, "redis-0.sock")
	assert.Contains(t, addrs, "redis-3.sock")
}

func TestRedisAddrsForProfile_singleVPSNeverSixShards(t *testing.T) {
	addrs := RedisAddrsForProfile(ProfileSingleVPS)
	require.Equal(t, RedisShardCountAppliance, RedisShardCountForAddrs(addrs))
	assert.NotContains(t, addrs, "redis-4")
	assert.NotContains(t, addrs, "redis-5")
}

func TestRenderComposeEnv_singleVPSRedisAddrs(t *testing.T) {
	out := string(RenderComposeEnv(Config{Profile: ProfileSingleVPS}))
	require.Contains(t, out, "REDIS_ADDRS=")
	line := ""
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, "REDIS_ADDRS=") {
			line = l
			break
		}
	}
	require.NotEmpty(t, line)
	addrs := strings.TrimPrefix(line, "REDIS_ADDRS=")
	assert.Equal(t, RedisShardCountAppliance, RedisShardCountForAddrs(addrs))
}
