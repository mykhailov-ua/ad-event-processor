package database

import (
	"time"

	"github.com/testcontainers/testcontainers-go/wait"
)

const postgresContainerStartupTimeout = 60 * time.Second

// PostgresContainerWaitStrategy waits for postgres:16-alpine to publish 5432 and log readiness once.
// Pairing port + log avoids the init-restart double-log flake under parallel integration runs.
func PostgresContainerWaitStrategy() wait.Strategy {
	return wait.ForAll(
		wait.ForListeningPort("5432/tcp").
			WithStartupTimeout(postgresContainerStartupTimeout),
		wait.ForLog("database system is ready to accept connections").
			WithOccurrence(1).
			WithStartupTimeout(postgresContainerStartupTimeout),
	).WithStartupTimeout(postgresContainerStartupTimeout)
}
