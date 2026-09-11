package platformadmin

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseTeamMetricsRange_holdoutUsesQueryWhenPresent(t *testing.T) {
	fromTS := "2024-01-01T00:00:00Z"
	toTS := "2024-01-07T00:00:00Z"
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/team/metrics?from="+fromTS+"&to="+toTS,
		http.NoBody,
	)

	from, to, err := parseTeamMetricsRange(req)
	require.NoError(t, err)
	require.Equal(t, fromTS, from.UTC().Format(time.RFC3339))
	require.Equal(t, toTS, to.UTC().Format(time.RFC3339))
}

func TestParseTeamMetricsRange_holdoutRejectsInvalidFrom(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/team/metrics?from=not-a-date&to=2024-01-07T00:00:00Z",
		http.NoBody,
	)

	_, _, err := parseTeamMetricsRange(req)
	require.Error(t, err)
}
