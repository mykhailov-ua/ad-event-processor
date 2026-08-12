package installer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInstallProfile_rejectsRemovedK8sProfile(t *testing.T) {
	err := (&InstallProfile{Type: Profile("k8s_k3s"), Interface: "eth0"}).Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid profile")
}
