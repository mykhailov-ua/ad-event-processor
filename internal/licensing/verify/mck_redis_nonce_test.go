package verify_test

import (
	"encoding/hex"
	"testing"

	"ad-event-processor/internal/licensing/verify"

	"github.com/stretchr/testify/require"
)

func TestRedisScriptNonceFromMCKWork_deterministic(t *testing.T) {
	var work [32]byte
	work[0] = 0xab
	work[31] = 0xcd

	n1 := verify.RedisScriptNonceFromMCKWork(work)
	n2 := verify.RedisScriptNonceFromMCKWork(work)
	require.Equal(t, n1, n2)
	require.NotEqual(t, [8]byte{}, n1)

	work[1] = 0x01
	n3 := verify.RedisScriptNonceFromMCKWork(work)
	require.NotEqual(t, n1, n3)
}

func TestRedisScriptNonceFromMCKWork_stretchVector_holdout(t *testing.T) {
	workHex := "7f8e9d0c1b2a3948574635445362718090a1b2c3d4e5f60718293a4b5c6d7e8f"
	workBytes, err := hex.DecodeString(workHex)
	require.NoError(t, err)
	require.Len(t, workBytes, 32)
	var work [32]byte
	copy(work[:], workBytes)

	got := verify.RedisScriptNonceFromMCKWork(work)
	require.Equal(t, "0b9c837c72658943", hex.EncodeToString(got[:]))
}
