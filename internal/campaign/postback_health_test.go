package campaign

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPostbackHealthStatus_thresholds(t *testing.T) {
	okRow := PostbackHealthRowDTO{SuccessRate24h: ptrFloat(99.5)}
	require.Equal(t, "ok", postbackHealthStatus(okRow, 100))

	failRate := PostbackHealthRowDTO{SuccessRate24h: ptrFloat(94.9)}
	require.Equal(t, "fail", postbackHealthStatus(failRate, 50))

	failDLQ := PostbackHealthRowDTO{SuccessRate24h: ptrFloat(100), DLQPendingCount: 1}
	require.Equal(t, "fail", postbackHealthStatus(failDLQ, 10))

	warnNoTraffic := PostbackHealthRowDTO{}
	require.Equal(t, "warn", postbackHealthStatus(warnNoTraffic, 0))
}

func ptrFloat(v float64) *float64 {
	return &v
}
