package domainhealth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClassifySSL(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	require.Equal(t, SSLExpired, classifySSL(now.Add(-time.Hour), now))
	require.Equal(t, SSLExpiring, classifySSL(now.Add(7*24*time.Hour), now))
	require.Equal(t, SSLValid, classifySSL(now.Add(30*24*time.Hour), now))
}

func TestClassifyHTTP(t *testing.T) {
	t.Parallel()
	health, detail := classifyHTTP(200, 100, SSLValid)
	require.Equal(t, HealthHealthy, health)
	require.Contains(t, detail, "200")

	health, _ = classifyHTTP(200, 3000, SSLValid)
	require.Equal(t, HealthDegraded, health)

	health, _ = classifyHTTP(503, 50, SSLValid)
	require.Equal(t, HealthDown, health)
}

func TestProbePath(t *testing.T) {
	t.Parallel()
	require.Equal(t, "/healthz", ProbePath("admin"))
	require.Equal(t, "/health", ProbePath("tracking"))
	require.Equal(t, "/health", ProbePath("custom"))
}

func TestNormalizeRole(t *testing.T) {
	t.Parallel()
	require.Equal(t, "admin", NormalizeRole("ADMIN"))
	require.Equal(t, "custom", NormalizeRole("custom"))
	require.Equal(t, "tracking", NormalizeRole(""))
}
