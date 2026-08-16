package runtimepaths

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRuntimePaths(t *testing.T) {
	assert.Equal(t, "/run/ad-event-processor/redis/redis-0.sock", RedisSocket(0))
	assert.Equal(t, "/etc/ad-event-processor/secrets.env", SecretsEnvPath())
	assert.Equal(t, "/run/ad-event-processor/postgresql", PostgresSocketDir())
	assert.Equal(t, "/run/ad-event-processor/broker/gnet.sock", BrokerGnetSocket())
	assert.Equal(t, "/run/ad-event-processor/control/http.sock", ControlHTTPSocket())
	assert.Equal(t, "/run/ad-event-processor/tracker/tracker-0.sock", TrackerSocket(0))
}
