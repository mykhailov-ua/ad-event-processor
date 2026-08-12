package installer

import (
	"strings"
	"testing"

	"github.com/bidshard/ad-event-processor/pkg/platformconfig"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderComposeEnv_singleVPSFourRedisShards(t *testing.T) {
	profile := &InstallProfile{
		Type:          ProfileSingleVPS,
		IngressSchema: IngressSchemaOpenRTB3,
	}
	out := string(renderComposeEnv(profile))
	line := redisAddrsLine(out)
	require.NotEmpty(t, line)
	addrs := strings.TrimPrefix(line, "REDIS_ADDRS=")
	assert.Equal(t, platformconfig.RedisShardCountAppliance, platformconfig.RedisShardCountForAddrs(addrs))
	assert.NotContains(t, addrs, "redis-4")
}

func TestRenderComposeEnv_composeDevFourRedisShards(t *testing.T) {
	profile := &InstallProfile{Type: ProfileComposeDev, IngressSchema: IngressSchemaOpenRTB3}
	out := string(renderComposeEnv(profile))
	line := redisAddrsLine(out)
	require.NotEmpty(t, line)
	addrs := strings.TrimPrefix(line, "REDIS_ADDRS=")
	assert.Equal(t, platformconfig.RedisShardCountAppliance, platformconfig.RedisShardCountForAddrs(addrs))
}

func redisAddrsLine(env string) string {
	for _, l := range strings.Split(env, "\n") {
		if strings.HasPrefix(l, "REDIS_ADDRS=") {
			return l
		}
	}
	return ""
}
