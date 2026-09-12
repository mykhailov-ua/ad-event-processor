package database

import (
	"time"

	"github.com/testcontainers/testcontainers-go/wait"
)

const postgresContainerStartupTimeout = 60 * time.Second

// PostgresContainerWaitStrategy waits for postgres:16-alpine to publish 5432 and log readiness once.
// Pairing port + log avoids the init-restart double-log flake under parallel integration runs.
func PostgresContainerWaitStrategy() wait.Strategy {
	return wait.ForAll( //nolint:staticcheck // SA1019: WithDeadline not on pinned testcontainers wait API
		wait.ForListeningPort("5432/tcp").
			WithStartupTimeout(postgresContainerStartupTimeout), //nolint:staticcheck // SA1019: testcontainers wait API; WithDeadline unavailable on pinned version
		wait.ForLog("database system is ready to accept connections").
			WithOccurrence(1).
			WithStartupTimeout(postgresContainerStartupTimeout), //nolint:staticcheck
	).WithStartupTimeout(postgresContainerStartupTimeout) //nolint:staticcheck
}
