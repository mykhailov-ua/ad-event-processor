package netaddr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsUnixSocketPath(t *testing.T) {
	assert.True(t, IsUnixSocketPath("/run/ad-event-processor/redis/redis-0.sock"))
	assert.True(t, IsUnixSocketPath("redis.sock"))
	assert.False(t, IsUnixSocketPath("127.0.0.1:6379"))
}

func TestGnetListenURI(t *testing.T) {
	assert.Equal(t, "unix:///run/foo.sock", GnetListenURI("/run/foo.sock"))
	assert.Equal(t, "tcp://127.0.0.1:9092", GnetListenURI("127.0.0.1:9092"))
}

func TestRedisURLFromAddr(t *testing.T) {
	assert.Equal(t, "unix:///run/redis-0.sock?db=0", RedisURLFromAddr("/run/redis-0.sock", "", 0))
	assert.Equal(t, "redis://127.0.0.1:6379/0", RedisURLFromAddr("127.0.0.1:6379", "", 0))
}

func TestParseRedisURL_unix(t *testing.T) {
	rdb, err := ParseRedisURL("unix:///tmp/not-there.sock", "")
	require.NoError(t, err)
	require.NotNil(t, rdb)
	_ = rdb.Close()
}
