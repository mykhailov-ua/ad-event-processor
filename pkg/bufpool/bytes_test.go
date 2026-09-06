package bufpool

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetPutBytes_roundTrip(t *testing.T) {
	ptr := GetBytes(1024)
	require.GreaterOrEqual(t, cap(*ptr), 1024)
	*ptr = append(*ptr, 1, 2, 3)
	PutBytes(ptr, DefaultMaxCap)
	ptr2 := GetBytes(0)
	require.Equal(t, 0, len(*ptr2))
}

func TestPutBytes_dropsOversized(t *testing.T) {
	buf := make([]byte, DefaultMaxCap+1)
	ptr := &buf
	PutBytes(ptr, DefaultMaxCap)
	ptr2 := GetBytes(0)
	require.LessOrEqual(t, cap(*ptr2), DefaultMaxCap)
}
