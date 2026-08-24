//go:build linux

package licensing_test

import (
	"testing"

	"ad-event-processor/internal/licensing"
	"github.com/stretchr/testify/require"
)

func TestDecodeHWIDPath_knownIDs(t *testing.T) {
	t.Parallel()
	cases := []struct {
		id   uint8
		want string
	}{
		{licensing.HWIDPathDMIUUID(), "/sys/class/dmi/id/product_uuid"},
		{licensing.HWIDPathMachineID(), "/etc/machine-id"},
	}
	for _, tc := range cases {
		got := licensing.DecodeHWIDPathForTest(tc.id, nil)
		require.Equal(t, tc.want, got)
	}
}

func TestDecodeHWIDPath_netAddress(t *testing.T) {
	got := licensing.DecodeHWIDPathFromIDsForTest(
		licensing.HWIDPathNetPrefix(),
		[]byte("eth0"),
		licensing.HWIDPathNetAddressSuffix(),
	)
	require.Equal(t, "/sys/class/net/eth0/address", got)
}
