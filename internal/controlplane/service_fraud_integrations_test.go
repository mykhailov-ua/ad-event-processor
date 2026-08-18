package controlplane

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPostbackConfigConfigured(t *testing.T) {
	require.True(t, postbackConfigConfigured("webhook", "https://example.com/pb", 0))
	require.False(t, postbackConfigConfigured("webhook", "", 0))
	require.True(t, postbackConfigConfigured("facebook", "https://graph.facebook.com", 32))
	require.False(t, postbackConfigConfigured("facebook", "https://graph.facebook.com", 0))
}

func TestDeriveFraudIntegrationHealth(t *testing.T) {
	now := time.Now()
	require.Equal(t, "unconfigured", deriveFraudIntegrationHealth(false, 0, nil))
	require.Equal(t, "failing", deriveFraudIntegrationHealth(true, 2, &now))
	require.Equal(t, "idle", deriveFraudIntegrationHealth(true, 0, nil))
	require.Equal(t, "configured", deriveFraudIntegrationHealth(true, 0, &now))
}
