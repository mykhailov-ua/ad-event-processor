package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestSetControlOutboxQueueMetrics_dualPublish(t *testing.T) {
	SetControlOutboxQueueMetrics(7, 42.5)
	require.Equal(t, 7.0, testutil.ToFloat64(ControlOutboxPendingTotal))
	require.Equal(t, 7.0, testutil.ToFloat64(ManagementOutboxPendingTotal))
	require.Equal(t, 42.5, testutil.ToFloat64(ControlOutboxOldestPendingSeconds))
	require.Equal(t, 42.5, testutil.ToFloat64(ManagementOutboxOldestPendingSeconds))
}

func TestControlPublishHelpers_dualPublish(t *testing.T) {
	IncControlOpsAlertEnqueueFailures()
	AddControlOpsAlertEnqueueFailures(2)
	require.Equal(t, 3.0, testutil.ToFloat64(ControlOpsAlertEnqueueFailuresTotal))
	require.Equal(t, 3.0, testutil.ToFloat64(ManagementOpsAlertEnqueueFailuresTotal))

	AddControlCommissionsCollected(1.5)
	require.Equal(t, 1.5, testutil.ToFloat64(ControlCommissionsCollectedTotal))
	require.Equal(t, 1.5, testutil.ToFloat64(CommissionsCollectedTotal))

	AddControlBalanceTopup("USD", 10)
	require.Equal(t, 10.0, testutil.ToFloat64(ControlBalanceTopupsTotal.WithLabelValues("USD")))
	require.Equal(t, 10.0, testutil.ToFloat64(BalanceTopupsTotal.WithLabelValues("USD")))

	SetControlActiveCampaigns(12)
	require.Equal(t, 12.0, testutil.ToFloat64(ControlActiveCampaigns))
	require.Equal(t, 12.0, testutil.ToFloat64(ActiveCampaigns))
}
