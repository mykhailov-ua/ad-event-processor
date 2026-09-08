package fraudadmin

import (
	"github.com/redis/go-redis/v9"
)

type ProbeClusterHost interface {
	ModeratorCorpusHost
	ProbeClusterRedis() redis.UniversalClient
	ProbeClusterExportEnabled() bool
}
