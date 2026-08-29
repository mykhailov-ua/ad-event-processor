//go:build linux

package verify_test

import (
	"testing"

	"ad-event-processor/internal/licensing/verify"

	"github.com/stretchr/testify/require"
)

func TestDecodeHWIDPath_knownIDs(t *testing.T) {
	t.Parallel()
	cases := []struct {
		id   uint8
		want string
	}{
		{verify.HWIDPathDMIUUID(), "/sys/class/dmi/id/product_uuid"},
		{verify.HWIDPathMachineID(), "/etc/machine-id"},
	}
	for _, tc := range cases {
		got := verify.DecodeHWIDPathForTest(tc.id, nil)
		require.Equal(t, tc.want, got)
	}
}

func TestDecodeHWIDPath_netAddress(t *testing.T) {
	got := verify.DecodeHWIDPathFromIDsForTest(
		verify.HWIDPathNetPrefix(),
		[]byte("eth0"),
		verify.HWIDPathNetAddressSuffix(),
	)
	require.Equal(t, "/sys/class/net/eth0/address", got)
}
